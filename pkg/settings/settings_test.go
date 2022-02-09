// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package settings

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/Azure/azure-extension-platform/pkg/constants"
	"github.com/Azure/azure-extension-platform/pkg/extensionerrors"
	"github.com/Azure/azure-extension-platform/pkg/handlerenv"
	"github.com/Azure/azure-extension-platform/pkg/logging"

	"github.com/stretchr/testify/require"
)

const (
	testSeqNo      = 5
	testThumbprint = "0b612f714b8defd5592bd45da9fe8cc5bbba3648"
	testdir        = "testdir"
)

func Test_settingsFileDoesntExist(t *testing.T) {
	he := getTestHandlerEnvironment()
	el := logging.New(nil)
	os.Remove(getTestSettingsFileName(he))
	_, err := GetHandlerSettings(el, he, 5)
	require.Equal(t, extensionerrors.ErrInvalidSettingsFile, err)
}

func Test_settingsEmptyFile(t *testing.T) {
	he := getTestHandlerEnvironment()
	err := initHandlerEnvironmentDirs(he)
	defer cleanuphandlerEnvDir(he)
	require.NoError(t, err)

	contents := []byte("")
	err = ioutil.WriteFile(getTestSettingsFileName(he), contents, 0644)
	require.NoError(t, err, "WriteFile failed")

	el := logging.New(nil)
	hs, err := GetHandlerSettings(el, he, testSeqNo)
	require.NoError(t, err, "getHandlerSettings failed")
	require.NotNil(t, hs)
	require.Empty(t, hs.PublicSettings)
	require.Empty(t, hs.ProtectedSettings)
}

func Test_settingsCannotParseSettings(t *testing.T) {
	he := getTestHandlerEnvironment()
	err := initHandlerEnvironmentDirs(he)
	defer cleanuphandlerEnvDir(he)
	contents := []byte("flarfegnugen")
	err = ioutil.WriteFile(getTestSettingsFileName(he), contents, 0644)
	require.NoError(t, err, "WriteFile failed")

	el := logging.New(nil)
	_, err = GetHandlerSettings(el, he, testSeqNo)
	require.Equal(t, extensionerrors.ErrInvalidSettingsFile, err)
}

func Test_settingsNoProtectedSettings(t *testing.T) {
	he := getTestHandlerEnvironment()
	err := initHandlerEnvironmentDirs(he)
	defer cleanuphandlerEnvDir(he)
	el := logging.New(nil)
	settingsFile := getTestSettingsFileName(he)
	writeSettingsToFile(t, testThumbprint, "", 1, settingsFile)

	hs, err := GetHandlerSettings(el, he, testSeqNo)
	require.NoError(t, err)
	validateHandlerSettings(t, hs)
}

func Test_settingsNoThumbprint(t *testing.T) {
	he := getTestHandlerEnvironment()
	err := initHandlerEnvironmentDirs(he)
	defer cleanuphandlerEnvDir(he)
	el := logging.New(nil)
	settingsFile := getTestSettingsFileName(he)
	writeSettingsToFile(t, "", "eWFiYQ==", 1, settingsFile)

	_, err = GetHandlerSettings(el, he, testSeqNo)
	require.Equal(t, extensionerrors.ErrNoCertificateThumbprint, err)
}

func Test_settingsCannotDecodeProtectedSettings(t *testing.T) {
	he := getTestHandlerEnvironment()
	err := initHandlerEnvironmentDirs(he)
	defer cleanuphandlerEnvDir(he)
	el := logging.New(nil)
	settingsFile := getTestSettingsFileName(he)
	writeSettingsToFile(t, testThumbprint, "&(*@#&JH", 1, settingsFile)

	_, err = GetHandlerSettings(el, he, testSeqNo)
	require.Equal(t, extensionerrors.ErrInvalidProtectedSettingsData, err)
}

func Test_settingsNoRuntimeSettings(t *testing.T) {
	he := getTestHandlerEnvironment()
	err := initHandlerEnvironmentDirs(he)
	defer cleanuphandlerEnvDir(he)
	el := logging.New(nil)
	settingsFile := getTestSettingsFileName(he)
	writeSettingsToFile(t, testThumbprint, "", 0, settingsFile)

	_, err = GetHandlerSettings(el, he, testSeqNo)
	require.Equal(t, extensionerrors.ErrInvalidSettingsRuntimeSettingsCount, err)
}

func Test_settingsTooManyRuntimeSettings(t *testing.T) {
	he := getTestHandlerEnvironment()
	err := initHandlerEnvironmentDirs(he)
	defer cleanuphandlerEnvDir(he)
	el := logging.New(nil)
	settingsFile := getTestSettingsFileName(he)
	writeSettingsToFile(t, testThumbprint, "", 2, settingsFile)

	_, err = GetHandlerSettings(el, he, testSeqNo)
	require.Equal(t, extensionerrors.ErrInvalidSettingsRuntimeSettingsCount, err)
}

func validateHandlerSettings(t *testing.T, hs *HandlerSettings) {
	require.NotNil(t, hs)
	v := make(map[string]interface{})
	jsonBytes := []byte(hs.PublicSettings)
	err := json.Unmarshal(jsonBytes, &v)
	require.NoError(t, err, "json unmarshal failed")
	flopper := v["Flopper"]
	require.NotEmpty(t, flopper)
	require.Equal(t, "flop", flopper)
}

func getTestSettingsFileName(he *handlerenv.HandlerEnvironment) string {
	return filepath.Join(he.ConfigFolder, fmt.Sprintf("%d%s", testSeqNo, settingsFileSuffix))
}

func getTestHandlerEnvironment() *handlerenv.HandlerEnvironment {

	return &handlerenv.HandlerEnvironment{
		HeartbeatFile: "./heartbeat.txt",
		StatusFolder:  path.Join(".", testdir, "./status/"),
		ConfigFolder:  path.Join(".", testdir, "./config/"),
		LogFolder:     path.Join(".", testdir, "./log/"),
		DataFolder:    path.Join(".", testdir, "./data/"),
	}
}

func cleanuphandlerEnvDir(he *handlerenv.HandlerEnvironment) {
	os.RemoveAll(testdir)
}

func initHandlerEnvironmentDirs(handlerEnv *handlerenv.HandlerEnvironment) error {
	err := os.MkdirAll(handlerEnv.StatusFolder, constants.FilePermissions_UserOnly_ReadWriteExecute)
	err2 := os.MkdirAll(handlerEnv.ConfigFolder, constants.FilePermissions_UserOnly_ReadWriteExecute)
	err = extensionerrors.CombineErrors(err, err2)
	err2 = os.MkdirAll(handlerEnv.LogFolder, constants.FilePermissions_UserOnly_ReadWriteExecute)
	err = extensionerrors.CombineErrors(err, err2)
	err2 = os.MkdirAll(handlerEnv.DataFolder, constants.FilePermissions_UserOnly_ReadWriteExecute)
	return extensionerrors.CombineErrors(err, err2)
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
