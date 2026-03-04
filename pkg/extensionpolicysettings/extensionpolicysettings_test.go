// filepath: /home/anasanc/repos/azure-extension-platform/pkg/extensionpolicysettings/extensionpolicysettings_test.go
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package extensionpolicysettings

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"testing"

	"github.com/Azure/azure-extension-platform/pkg/extensionerrors"
	"github.com/Azure/azure-extension-platform/pkg/logging"
	"github.com/stretchr/testify/require"
)

var extensionLogger = logging.New(nil)

const extensionRuntimePolicySettingsFilePath = "./testutils/runtime_policy.json"

// This is a sample struct for an example extension's policy settings. Each extension will define their own struct that implements the ExtensionPolicySettings interface according to their needs.
type TestPolicy struct {
	RequiresSigning string   `json:"requireSigning"`
	AllowedScripts  []string `json:"allowedScripts"`
}

func (tp TestPolicy) ValidateFormat() error {
	// In a real extension, you would implement logic to validate the policy was correctly loaded.
	return nil
}

func TestNewExtensionPolicySettingsManager(t *testing.T) {
	// Create a new ExtensionPolicySettingsManager
	manager, err := NewExtensionPolicySettingsManager[TestPolicy](extensionRuntimePolicySettingsFilePath, extensionLogger)
	require.NoError(t, err)
	require.NotNil(t, manager)
	require.Equal(t, extensionRuntimePolicySettingsFilePath, manager.settingsFilePath)
	require.Equal(t, extensionLogger, manager.logger)
	require.Nil(t, manager.settings) // settings should not be loaded until LoadExtensionPolicySettings is called
}

func TestLoadExtensionPolicySettings(t *testing.T) {
	// Setup test parameters
	manager, err := NewExtensionPolicySettingsManager[TestPolicy](extensionRuntimePolicySettingsFilePath, extensionLogger)
	require.NoError(t, err)

	// Test cases:
	// 1. Valid policy file: we should be able to load the settings without error
	validPolicyContent := `{
		"requireSigning": "true",
		"allowedScripts": []
	}`
	writeToFile(extensionRuntimePolicySettingsFilePath, validPolicyContent)
	defer cleanupFile(extensionRuntimePolicySettingsFilePath)

	// Call LoadExtensionPolicySettings and check for errors
	err = manager.LoadExtensionPolicySettings()
	require.NoError(t, err)
	require.NotNil(t, manager.settings)
	require.Equal(t, "true", manager.settings.RequiresSigning)
	require.Empty(t, manager.settings.AllowedScripts)

	// 2. Invalid policy file (e.g. not valid json): we should get an error when trying to load the settings
	invalidPolicyContent := `{`
	writeToFile(extensionRuntimePolicySettingsFilePath, invalidPolicyContent)
	err = manager.LoadExtensionPolicySettings()
	require.Error(t, err)

	// 3. Empty policy file: we should get an error indicating the policy file is empty
	writeToFile(extensionRuntimePolicySettingsFilePath, "")
	err = manager.LoadExtensionPolicySettings()
	require.ErrorIs(t, err, extensionerrors.ErrEmptyPolicyFile)

	// 5. Locked policy file: we should get an error indicating the file cannot be accessed.
	// modify the file permissions to simulate a locked file (read-only file)
	os.Chmod(extensionRuntimePolicySettingsFilePath, 0200) // write-only permissions
	err = manager.LoadExtensionPolicySettings()
	require.Error(t, err)

	// 5. Missing policy file: we should get an error indicating the policy file is missing
	cleanupFile(extensionRuntimePolicySettingsFilePath)
	err = manager.LoadExtensionPolicySettings()
	require.ErrorIs(t, err, extensionerrors.ErrMissingPolicyFile)
}

func TestGetSettings(t *testing.T) {
	// Setup test parameters
	manager, err := NewExtensionPolicySettingsManager[TestPolicy](extensionRuntimePolicySettingsFilePath, extensionLogger)
	require.NoError(t, err)
	validPolicyContent := `{
		"requireSigning": "true",
		"allowedScripts": []
	}`
	writeToFile(extensionRuntimePolicySettingsFilePath, validPolicyContent)
	defer cleanupFile(extensionRuntimePolicySettingsFilePath) // Clean up after test

	// Call LoadExtensionPolicySettings and check for errors
	err = manager.LoadExtensionPolicySettings()
	require.NoError(t, err)
	require.NotNil(t, manager.settings)
	require.Equal(t, "true", manager.settings.RequiresSigning)

	// Call GetSettings and check for errors
	settings := manager.GetSettings()
	require.NotNil(t, settings)
}

func TestValidateAgainstAllowlist(t *testing.T) {
	// Setup test parameters
	manager, err := NewExtensionPolicySettingsManager[TestPolicy](extensionRuntimePolicySettingsFilePath, extensionLogger)
	require.NoError(t, err)
	defer cleanupFile(extensionRuntimePolicySettingsFilePath) // Clean up after test

	script1Hash := hashHelper("./testutils/testscripts/script1.sh")
	script2Hash := hashHelper("./testutils/testscripts/script2.sh")
	//script3Hash := hashHelper("./testutils/testscripts/script3.sh")

	// Some scripts are allowed
	validPolicyContent := fmt.Sprintf(`{
    "requireSigning": "true",
    "allowedScripts": ["%s", "%s"]
	}`, script1Hash, script2Hash)
	writeToFile(extensionRuntimePolicySettingsFilePath, validPolicyContent)

	// Call LoadExtensionPolicySettings and check for errors
	err = manager.LoadExtensionPolicySettings()
	require.NoError(t, err)
	require.NotNil(t, manager.settings)
	require.Equal(t, "true", manager.settings.RequiresSigning)
	require.NotEmpty(t, manager.settings.AllowedScripts)

	require.NoError(t, ValidateFileHashInAllowlist(manager.logger, "./testutils/testscripts/script1.sh", manager.settings.AllowedScripts, HashTypeSHA256))
	require.NoError(t, ValidateFileHashInAllowlist(manager.logger, "./testutils/testscripts/script2.sh", manager.settings.AllowedScripts, HashTypeSHA256))
	require.ErrorIs(t, ValidateFileHashInAllowlist(manager.logger, "./testutils/testscripts/script3.sh", manager.settings.AllowedScripts, HashTypeSHA256), extensionerrors.ErrItemNotInAllowlist)
	require.ErrorIs(t, ValidateFileHashInAllowlist(manager.logger, "", manager.settings.AllowedScripts, HashTypeSHA256), extensionerrors.ErrEmptyFilepathToValidate)
	require.Error(t, ValidateFileHashInAllowlist(manager.logger, "./testutils/testscripts/missing.sh", manager.settings.AllowedScripts, HashTypeSHA256))

	// Now, empty list.
	require.ErrorIs(t, ValidateFileHashInAllowlist(manager.logger, "./testutils/testscripts/script1.sh", []string{}, HashTypeSHA256), extensionerrors.ErrPolicyAllowlistEmpty)
}

func writeToFile(filePath, content string) {
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		panic(err)
	}
}

func cleanupFile(path string) {
	// Do not remove missingPolicyFilePath as it simulates a missing file
	if _, err := os.Stat(path); err == nil {
		os.Remove(path)
	}
}

func hashHelper(filePath string) string {
	contents, err := os.ReadFile(filePath)

	if err != nil {
		panic(err)
	}

	hash := sha256.Sum256([]byte(contents))
	hashStr := hex.EncodeToString(hash[:])
	return hashStr
}
