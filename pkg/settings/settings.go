package settings

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/Azure/azure-extension-platform/pkg/decrypt"
	"github.com/Azure/azure-extension-platform/pkg/extensionerrors"
	"github.com/Azure/azure-extension-platform/pkg/handlerenv"
	"io/ioutil"
	"path/filepath"

	"github.com/go-kit/kit/log"
)

const (
	settingsFileSuffix = ".settings"
	disableFileName    = "disabled"
)

// HandlerSettings contains the decrypted settings for the extension
type HandlerSettings struct {
	PublicSettings    map[string]interface{}
	ProtectedSettings string
}

// handlerSettings is an internal structure used to deserialize the file
type handlerSettings struct {
	PublicSettings          map[string]interface{} `json:"publicSettings"`
	ProtectedSettingsBase64 string                 `json:"protectedSettings"`
	SettingsCertThumbprint  string                 `json:"protectedSettingsCertThumbprint"`
}

type handlerSettingsFile struct {
	RuntimeSettings []handlerSettingsContainer `json:"runtimeSettings"`
}

type handlerSettingsContainer struct {
	HandlerSettings handlerSettings `json:"handlerSettings"`
}

// getHandlerSettings reads and parses the handler's settings in an OS independent manner
func GetHandlerSettings(ctx log.Logger, he *handlerenv.HandlerEnvironment, seqNo uint) (hs *HandlerSettings, _ error) {
	// The file will be under the config folder with the path {seqNo}.settings
	settingsFileName := filepath.Join(he.ConfigFolder, fmt.Sprintf("%d%s", seqNo, settingsFileSuffix))
	parsedHs, err := parseHandlerSettingsFile(ctx, settingsFileName)
	if err != nil {
		return hs, err
	}

	protectedSettings, err := unmarshalProtectedSettings(ctx, he.ConfigFolder, parsedHs)
	if err != nil {
		return hs, err
	}

	hs = &HandlerSettings{
		PublicSettings:    parsedHs.PublicSettings,
		ProtectedSettings: protectedSettings,
	}

	return hs, nil
}

// unmarshalProtectedSettings decodes the protected settings from handler
// runtime settings JSON file, decrypts it using the certificates and unmarshals
// into the given struct v.
func unmarshalProtectedSettings(ctx log.Logger, configFolder string, hs handlerSettings) (string, error) {
	if hs.ProtectedSettingsBase64 == "" {
		// No protected settings
		return nil, nil
	}
	if hs.SettingsCertThumbprint == "" {
		ctx.Log("message", "parseHandlerSettingsFile failed", "error", extensionerrors.ErrNoCertificateThumbprint)
		return nil, extensionerrors.ErrNoCertificateThumbprint
	}

	decoded, err := base64.StdEncoding.DecodeString(hs.ProtectedSettingsBase64)
	if err != nil {
		ctx.Log("message", "parseHandlerSettingsFile failed", "error", fmt.Errorf("failed to decode base64: %v", err))
		return nil, extensionerrors.ErrInvalidProtectedSettingsData
	}

	v, err := decrypt.DecryptProtectedSettings(configFolder, hs.SettingsCertThumbprint, decoded)
	return v, err
}

// parseHandlerSettings parses a handler settings file (e.g. 0.settings) and
// returns it as a structured object.
func parseHandlerSettingsFile(ctx log.Logger, path string) (h handlerSettings, _ error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		ctx.Log("message", "parseHandlerSettingsFile failed", "error", fmt.Errorf("Error reading %s: %v", path, err))
		return h, extensionerrors.ErrInvalidSettingsFile
	}
	if len(b) == 0 { // if no config is specified, we get an empty file
		return h, nil
	}

	var f handlerSettingsFile
	if err := json.Unmarshal(b, &f); err != nil {
		ctx.Log("message", "parseHandlerSettingsFile failed", "error", fmt.Errorf("error parsing json: %v", err))
		return h, extensionerrors.ErrInvalidSettingsFile
	}
	if len(f.RuntimeSettings) != 1 {
		ctx.Log("message", "parseHandlerSettingsFile failed", "error", fmt.Errorf("wrong runtimeSettings count. expected:1, got:%d", len(f.RuntimeSettings)))
		return h, extensionerrors.ErrInvalidSettingsRuntimeSettingsCount
	}

	return f.RuntimeSettings[0].HandlerSettings, nil
}
