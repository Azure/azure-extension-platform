package settings

import (
	"encoding/json"
	"fmt"
	"github.com/D1v38om83r/azure-extension-platform/pkg/extensionerrors"
	"github.com/D1v38om83r/azure-extension-platform/pkg/handlerenv"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/require"
)

const (
	testSeqNo      = 5
	testThumbprint = "0b612f714b8defd5592bd45da9fe8cc5bbba3648"
)

func Test_settingsFileDoesntExist(t *testing.T) {
	he := getTestHandlerEnvironment()
	ctx := log.NewSyncLogger(log.NewLogfmtLogger(os.Stdout))
	os.Remove(getTestSettingsFileName(he))
	_, err := GetHandlerSettings(ctx, he, 5)
	require.Equal(t, extensionerrors.ErrInvalidSettingsFile, err)
}

func Test_settingsEmptyFile(t *testing.T) {
	he := getTestHandlerEnvironment()
	contents := []byte("")
	err := ioutil.WriteFile(getTestSettingsFileName(he), contents, 0644)
	require.NoError(t, err, "WriteFile failed")

	ctx := log.NewSyncLogger(log.NewLogfmtLogger(os.Stdout))
	hs, err := GetHandlerSettings(ctx, he, testSeqNo)
	require.NoError(t, err, "getHandlerSettings failed")
	require.NotNil(t, hs)
	require.Nil(t, hs.PublicSettings)
	require.Nil(t, hs.ProtectedSettings)
}

func Test_settingsCannotParseSettings(t *testing.T) {
	he := getTestHandlerEnvironment()
	contents := []byte("flarfegnugen")
	err := ioutil.WriteFile(getTestSettingsFileName(he), contents, 0644)
	require.NoError(t, err, "WriteFile failed")

	ctx := log.NewSyncLogger(log.NewLogfmtLogger(os.Stdout))
	_, err = GetHandlerSettings(ctx, he, testSeqNo)
	require.Equal(t, extensionerrors.ErrInvalidSettingsFile, err)
}

func Test_settingsNoProtectedSettings(t *testing.T) {
	he := getTestHandlerEnvironment()
	ctx := log.NewSyncLogger(log.NewLogfmtLogger(os.Stdout))
	settingsFile := getTestSettingsFileName(he)
	writeSettingsToFile(t, testThumbprint, "", 1, settingsFile)

	hs, err := GetHandlerSettings(ctx, he, testSeqNo)
	require.NoError(t, err)
	validateHandlerSettings(t, hs)
}

func Test_settingsNoThumbprint(t *testing.T) {
	he := getTestHandlerEnvironment()
	ctx := log.NewSyncLogger(log.NewLogfmtLogger(os.Stdout))
	settingsFile := getTestSettingsFileName(he)
	writeSettingsToFile(t, "", "eWFiYQ==", 1, settingsFile)

	_, err := GetHandlerSettings(ctx, he, testSeqNo)
	require.Equal(t, extensionerrors.ErrNoCertificateThumbprint, err)
}

func Test_settingsCannotDecodeProtectedSettings(t *testing.T) {
	he := getTestHandlerEnvironment()
	ctx := log.NewSyncLogger(log.NewLogfmtLogger(os.Stdout))
	settingsFile := getTestSettingsFileName(he)
	writeSettingsToFile(t, testThumbprint, "&(*@#&JH", 1, settingsFile)

	_, err := GetHandlerSettings(ctx, he, testSeqNo)
	require.Equal(t, extensionerrors.ErrInvalidProtectedSettingsData, err)
}

func Test_settingsNoRuntimeSettings(t *testing.T) {
	he := getTestHandlerEnvironment()
	ctx := log.NewSyncLogger(log.NewLogfmtLogger(os.Stdout))
	settingsFile := getTestSettingsFileName(he)
	writeSettingsToFile(t, testThumbprint, "", 0, settingsFile)

	_, err := GetHandlerSettings(ctx, he, testSeqNo)
	require.Equal(t, extensionerrors.ErrInvalidSettingsRuntimeSettingsCount, err)
}

func Test_settingsTooManyRuntimeSettings(t *testing.T) {
	he := getTestHandlerEnvironment()
	ctx := log.NewSyncLogger(log.NewLogfmtLogger(os.Stdout))
	settingsFile := getTestSettingsFileName(he)
	writeSettingsToFile(t, testThumbprint, "", 2, settingsFile)

	_, err := GetHandlerSettings(ctx, he, testSeqNo)
	require.Equal(t, extensionerrors.ErrInvalidSettingsRuntimeSettingsCount, err)
}

func validateHandlerSettings(t *testing.T, hs *HandlerSettings) {
	require.NotNil(t, hs)
	flopperRaw := hs.PublicSettings["Flopper"]
	require.NotNil(t, flopperRaw)
	flopper, ok := flopperRaw.(string)
	require.True(t, ok)
	require.Equal(t, "flop", flopper)
}

func getTestSettingsFileName(he *handlerenv.HandlerEnvironment) string {
	return filepath.Join(he.ConfigFolder, fmt.Sprintf("%d%s", testSeqNo, settingsFileSuffix))
}

func getTestHandlerEnvironment() *handlerenv.HandlerEnvironment {
	return &handlerenv.HandlerEnvironment{
		HeartbeatFile: "./heartbeat.txt",
		StatusFolder:  "./status/",
		ConfigFolder:  "./config/",
		LogFolder:     "./log/",
		DataFolder:    "./data/",
	}
}

func writeSettingsToFile(t *testing.T, thumbprint string, protectedSettings string, runtimeSettingsCount int, fileName string) {
	// Create the directory for the file if it doesn't exist
	fileDir := filepath.Dir(fileName)
	_ = os.Mkdir(fileDir, os.ModePerm)

	publicSettings := make(map[string]interface{})
	publicSettings["Flipper"] = "flip"
	publicSettings["Flopper"] = "flop"
	hs := handlerSettings{PublicSettings: publicSettings, ProtectedSettingsBase64: protectedSettings, SettingsCertThumbprint: thumbprint}
	hsf := handlerSettingsFile{}

	hsContainer := handlerSettingsContainer{HandlerSettings: hs}
	hsContainerArray := make([]handlerSettingsContainer, runtimeSettingsCount)
	for i := 0; i < runtimeSettingsCount; i++ {
		hsContainerArray[i] = hsContainer
	}

	hsf.RuntimeSettings = hsContainerArray

	file, err := json.MarshalIndent(hsf, "", " ")
	require.NoError(t, err)
	err = ioutil.WriteFile(fileName, file, 0644)
	require.NoError(t, err)
}
