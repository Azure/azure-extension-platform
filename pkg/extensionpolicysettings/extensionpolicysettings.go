package extensionpolicysettings

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"os"
	"io/ioutil"

	"github.com/Azure/azure-extension-platform/pkg/extensionerrors"
	"github.com/Azure/azure-extension-platform/pkg/logging"
	"github.com/Azure/azure-extension-platform/pkg/status"
)

const extensionPolicySettingsFileName = "ExtensionRuntimePolicy.json" // Lourdes come back to this. This name is consistent with the filename defined by linux GA.

// ScriptType is consistent with the ScriptType defined in CRP.


// want:
// extendion policy settings struct passed in by client
// must satisfy validate (like the actual policy is properly set, or no)

type ExtensionPolicySettings interface {
	Validate() error
}

type ExtensionPolicySettingsManager[T ExtensionPolicySettings] struct {
	settingsFilePath 	string
	logger           	logging.Logger
	settings 			T
}

func NewExtensionPolicySettingsManager[T ExtensionPolicySettings](configFolder string, logger logging.ILogger) (*ExtensionPolicySettingsManager[T], error) {
	settingsFilePath := filepath.Join(configFolder, extensionPolicySettingsFileName)
	return &ExtensionPolicySettingsManager[T]{
		settingsFilePath: settingsFilePath,
		logger: logger, // settings is not loaded until LoadExtensionPolicySettings is called
	}, nil
}

func (epsm *ExtensionPolicySettingsManager[T]) LoadExtensionPolicySettings() error {
	epsm.logger.LogInfo(fmt.Sprintf("Loading extension policy settings from file: %s", epsm.settingsFilePath))

	// If an extension has a default policy configuration in case the file does not exist, they should handle that logic before calling this function.
	if _, err := os.Stat(epsm.settingsFilePath); os.IsNotExist(err) {
		return fmt.Errorf("extension policy settings file does not exist at path: %s", epsm.settingsFilePath)
	} else if err != nil {
		return fmt.Errorf("error checking extension policy settings file: %w", err)
	}
	
	fileContent, err := ioutil.ReadFile(epsm.settingsFilePath)
	if err != nil {
		return fmt.Errorf("failed to read extension policy settings file: %w", err) // lourdes: check if this is the correct error handling pattern for your project
	}

	if len(fileContent) == 0 {
		return fmt.Errorf("extension policy settings file is empty")
	}

	var settings T 
	err := json.Unmarshal(fileContent, &settings)
	if err != nil {
		return fmt.Errorf("failed to unmarshal extension policy settings: %w", err)
	}

	if err := settings.Validate(); err != nil {
		return fmt.Errorf("extension policy settings validation failed: %w", err)
	}

	epsm.settings = settings
	epsm.logger.LogInfo("Extension policy settings loaded and validated successfully.")
	return nil
}

func (epsm *ExtensionPolicySettingsManager[T]) GetSettings() (T, error) {
	return epsm.settings, nil
}

// Validation Helper Functions
type fileInputType int
const (
	filepath fileInputType = iota
	fileContents
)

type hashType int
const (
	noHash hashType = iota
	sha1
	sha256
)

func ValidateAgainstAllowlist(logger logging.ILogger, value string, allowlist []string, inputOpt fileInputType, hashOpt hashType) (bool, error) {
	// If extensions want special behavior when a list is empty, they should handle that before calling this function. For security purposes, we want to make sure that if an allowlist is expected, it should not be empty.
	if allowlist == nil || len(allowlist) == 0 {
		return false, fmt.Errorf("allowlist is empty")
	}

	// first, make sure we have the content we're working with
	if inputOpt == filepath {
		if value == "" {
			return false, fmt.Errorf("file path cannot be empty")
		}
		content, err := ioutil.ReadFile(value)
		if err != nil {
			return false, fmt.Errorf("failed to read file for validation: %w", err)
		}
		value = string(content) // lourdes: this replaces the file path with the file content.
	}

	if value == "" {
		return false, fmt.Errorf("contents of file to validate cannot be empty") // lourdes: or maybe they can? man idk
	}

	// second, handle the hash scenario.

	if hashOpt != noHash {
		logger.LogInfo("Computing hash of the value for validation.")
		value, err := ComputeFileHash(logger, value, hashOpt)
		if err != nil {
			return false, fmt.Errorf("failed to compute hash for validation: %w", err)
		}
		logger.LogInfo(fmt.Sprintf("Computed hash value: %s", value))
	}
	
	// finally, check if the value (or its hash) is in the allowlist.
	for _, allowlistValue := range allowlist {
		if value == allowlistValue {
			logger.LogInfo("Validation successful: file is in the allowlist.")
			return true, nil
		}
	}

	logger.LogInfo("Validation failed: file is not in the allowlist.")
	return false, nil
}

// ComputeFileHash computes the hash of a file or leaves string as is.
func ComputeFileHash(logger logging.ILogger, contents string, hashOpt hashType) (string, error) {
    logger.Info("Computing hash for file contents")
    
    if contents == "" {
        return "", fmt.Errorf("contents cannot be empty")
    }
    
    var hashStr string
    switch hashOpt {
    case sha1:
        hash := sha1.Sum([]byte(contents))
        hashStr = hex.EncodeToString(hash[:])
    case sha256:
        hash := sha256.Sum256([]byte(contents))
        hashStr = hex.EncodeToString(hash[:])
    default:
        return "", fmt.Errorf("Invalid hash option")
    }
    
    logger.Info("Computed hash: %s", hashStr)
    return hashStr, nil
}

// func ValidateFileEmbeddedSignature() error {
// }

// func ValidateCatalogSignature() error {
// }
