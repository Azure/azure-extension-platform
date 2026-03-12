// filepath: /home/anasanc/repos/azure-extension-platform/pkg/extensionpolicysettings/extensionpolicysettings_test.go
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package extensionpolicysettings

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"testing"

	"github.com/Azure/azure-extension-platform/pkg/extensionerrors"
	"github.com/stretchr/testify/require"
)

const extensionRuntimePolicySettingsFilePath = "./testutils/runtime_policy.json"

// This is a sample struct for an example extension's policy settings.
// Each extension will define their own struct that implements the ExtensionPolicySettings interface according to their needs.
type TestPolicy struct {
	RequiresSigning string   `json:"requireSigning"`
	AllowedScripts  []string `json:"allowedScripts"`
}

func (tp TestPolicy) ValidateFormat() error {
	// In a real extension, you would implement logic to validate the policy was correctly loaded.
	return nil
}

type InvalidFormatPolicy struct {
	Value string `json:"value"`
}

func (p InvalidFormatPolicy) ValidateFormat() error {
	return fmt.Errorf("invalid format for test")
}

func TestNewExtensionPolicySettingsManager_EmptyPath(t *testing.T) {
	manager, err := NewExtensionPolicySettingsManager[TestPolicy]("")
	require.ErrorIs(t, err, extensionerrors.ErrEmptyPolicyFilePath)
	require.Nil(t, manager)
}

func TestLoadExtensionPolicySettings_NilManager(t *testing.T) {
	var manager *ExtensionPolicySettingsManager[TestPolicy]
	err := manager.LoadExtensionPolicySettings()
	require.Error(t, err)
	require.Contains(t, err.Error(), "manager is nil")
}

func TestNewExtensionPolicySettingsManager(t *testing.T) {
	// Create a new ExtensionPolicySettingsManager
	manager, err := NewExtensionPolicySettingsManager[TestPolicy](extensionRuntimePolicySettingsFilePath)
	require.NoError(t, err)
	require.NotNil(t, manager)
	require.Equal(t, extensionRuntimePolicySettingsFilePath, manager.settingsFilePath)
	require.Nil(t, manager.settings) // settings should not be loaded until LoadExtensionPolicySettings is called
}

func TestLoadExtensionPolicySettings_EmptyManagerPath(t *testing.T) {
	manager := &ExtensionPolicySettingsManager[TestPolicy]{}
	err := manager.LoadExtensionPolicySettings()
	require.ErrorIs(t, err, extensionerrors.ErrEmptyPolicyFilePath)
}

func TestLoadExtensionPolicySettings_ValidateFormatFailure(t *testing.T) {
	manager, err := NewExtensionPolicySettingsManager[InvalidFormatPolicy](extensionRuntimePolicySettingsFilePath)
	require.NoError(t, err)

	validJSONButInvalidFormat := `{"value":"anything"}`
	require.NoError(t, writeToFile(extensionRuntimePolicySettingsFilePath, validJSONButInvalidFormat))
	defer cleanupFile(extensionRuntimePolicySettingsFilePath)

	err = manager.LoadExtensionPolicySettings()
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid format")
}

func TestLoadExtensionPolicySettings(t *testing.T) {
	// Setup test parameters
	manager, err := NewExtensionPolicySettingsManager[TestPolicy](extensionRuntimePolicySettingsFilePath)
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
	manager, err := NewExtensionPolicySettingsManager[TestPolicy](extensionRuntimePolicySettingsFilePath)
	require.NoError(t, err)
	validPolicyContent := `{
		"requireSigning": "true",
		"allowedScripts": []
	}`
	require.NoError(t, writeToFile(extensionRuntimePolicySettingsFilePath, validPolicyContent))
	defer cleanupFile(extensionRuntimePolicySettingsFilePath)

	// Call LoadExtensionPolicySettings and check for errors
	_, err = manager.GetSettings()
	require.ErrorIs(t, err, extensionerrors.ErrPolicyNotYetLoaded) // should return an error because settings have not been loaded yet
	err = manager.LoadExtensionPolicySettings()
	require.NoError(t, err)
	require.NotNil(t, manager.settings)
	require.Equal(t, "true", manager.settings.RequiresSigning)

	// Call GetSettings and check for errors
	settings, err := manager.GetSettings()
	require.NoError(t, err)
	require.NotNil(t, settings)
	require.Equal(t, "true", settings.RequiresSigning)
	require.Empty(t, settings.AllowedScripts)
}

func TestValidateValueInAllowlist(t *testing.T) {
	require.ErrorIs(t, ValidateValueInAllowlist("x", []string{}), extensionerrors.ErrPolicyAllowlistEmpty)
	require.NoError(t, ValidateValueInAllowlist("b", []string{"a", "b", "c"}))
	require.ErrorIs(t, ValidateValueInAllowlist("z", []string{"a", "b", "c"}), extensionerrors.ErrItemNotInAllowlist)
}

func TestValidateAgainstAllowlist(t *testing.T) {
	// Setup test parameters
	manager, err := NewExtensionPolicySettingsManager[TestPolicy](extensionRuntimePolicySettingsFilePath)
	require.NoError(t, err)
	defer cleanupFile(extensionRuntimePolicySettingsFilePath) // Clean up after test

	script1Hash, err := hashHelper("./testutils/testscripts/script1.sh", TestHashTypeSha256)
	require.NoError(t, err)
	script2Hash, err := hashHelper("./testutils/testscripts/script2.sh", TestHashTypeSha256)
	require.NoError(t, err)
	// Skip computing script3 hash because it will not be allowed..
	script4Hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" // pre-computed hash of the empty string
	script5Hash, err := hashHelper("./testutils/testscripts/script5.sh", TestHashTypeSha1)
	require.NoError(t, err)

	// Some scripts are allowed
	validPolicyContent := fmt.Sprintf(`{
    "requireSigning": "true",
    "allowedScripts": ["%s", "%s", "%s", "%s"]
	}`, script1Hash, script2Hash, script4Hash, script5Hash)
	require.NoError(t, writeToFile(extensionRuntimePolicySettingsFilePath, validPolicyContent))

	// Call LoadExtensionPolicySettings and check for errors
	err = manager.LoadExtensionPolicySettings()
	require.NoError(t, err)
	require.NotNil(t, manager.settings)
	require.Equal(t, "true", manager.settings.RequiresSigning)
	require.NotEmpty(t, manager.settings.AllowedScripts)

	require.NoError(t, ValidateFileHashInAllowlist("./testutils/testscripts/script1.sh", manager.settings.AllowedScripts, HashTypeSHA256))
	require.NoError(t, ValidateFileHashInAllowlist("./testutils/testscripts/script2.sh", manager.settings.AllowedScripts, HashTypeSHA256))
	require.ErrorIs(t, ValidateFileHashInAllowlist("./testutils/testscripts/script3.sh", manager.settings.AllowedScripts, HashTypeSHA256), extensionerrors.ErrItemNotInAllowlist)
	require.NoError(t, ValidateFileHashInAllowlist("./testutils/testscripts/script5.sh", manager.settings.AllowedScripts, HashTypeSHA1))

	// Empty filepath
	require.ErrorIs(t, ValidateFileHashInAllowlist("", manager.settings.AllowedScripts, HashTypeSHA256), extensionerrors.ErrEmptyFilepathToValidate)
	// Missing file
	require.Error(t, ValidateFileHashInAllowlist("./testutils/testscripts/missing.sh", manager.settings.AllowedScripts, HashTypeSHA256))
	// Now, empty list.
	require.ErrorIs(t, ValidateFileHashInAllowlist("./testutils/testscripts/script1.sh", []string{}, HashTypeSHA256), extensionerrors.ErrPolicyAllowlistEmpty)
	// Empty file
	require.NoError(t, ValidateFileHashInAllowlist("./testutils/testscripts/script4.sh", manager.settings.AllowedScripts, HashTypeSHA256))

}

func TestValidateFileHashInAllowlist_HashTypeNone_UsesRawContent(t *testing.T) {
	filePath := "./testutils/raw_content_test.txt"
	content := "raw-content-123"

	require.NoError(t, writeToFile(filePath, content))
	defer cleanupFile(filePath)

	require.NoError(t, ValidateFileHashInAllowlist(filePath, []string{content}, HashTypeNone))
	require.ErrorIs(t, ValidateFileHashInAllowlist(filePath, []string{"different-content"}, HashTypeNone), extensionerrors.ErrItemNotInAllowlist)
}

func TestComputeFileHash(t *testing.T) {
	input := "abc"

	sha1Expected := sha1.Sum([]byte(input))
	sha256Expected := sha256.Sum256([]byte(input))

	gotSHA1, err := ComputeFileHash(input, HashTypeSHA1)
	require.NoError(t, err)
	require.Equal(t, hex.EncodeToString(sha1Expected[:]), gotSHA1)

	gotSHA256, err := ComputeFileHash(input, HashTypeSHA256)
	require.NoError(t, err)
	require.Equal(t, hex.EncodeToString(sha256Expected[:]), gotSHA256)

	// Current behavior: unknown hash type falls back to SHA256.
	gotUnknown, err := ComputeFileHash(input, HashType(999))
	require.NoError(t, err)
	require.Equal(t, hex.EncodeToString(sha256Expected[:]), gotUnknown)
}

// Helper functions for tests
func writeToFile(filePath, content string) error {
	err := os.WriteFile(filePath, []byte(content), 0644)
	return err
}

func cleanupFile(path string) {
	if _, err := os.Stat(path); err == nil {
		os.Remove(path)
	}
}

type TestHashType int

const (
	TestHashTypeSha1 TestHashType = iota
	TestHashTypeSha256
)

func hashHelper(filePath string, hashOpt TestHashType) (string, error) {
	contents, err := os.ReadFile(filePath)

	if err != nil {
		return "", err
	}

	var hashStr string
	switch hashOpt {
	case TestHashTypeSha1:
		hash := sha1.New()
		hash.Write(contents)
		hashStr = hex.EncodeToString(hash.Sum(nil))
	case TestHashTypeSha256:
		hash := sha256.New()
		hash.Write(contents)
		hashStr = hex.EncodeToString(hash.Sum(nil))
	}
	return hashStr, nil
}
