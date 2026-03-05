package extensionpolicysettings

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/Azure/azure-extension-platform/pkg/extensionerrors"
)

type ExtensionPolicySettings interface {
	ValidateFormat() error
}

type ExtensionPolicySettingsManager[T ExtensionPolicySettings] struct {
	settingsFilePath string
	settings         *T
}

func NewExtensionPolicySettingsManager[T ExtensionPolicySettings](policyFilePath string) (*ExtensionPolicySettingsManager[T], error) {
	if policyFilePath == "" {
		return nil, extensionerrors.ErrEmptyPolicyFilePath
	}
	return &ExtensionPolicySettingsManager[T]{
		settingsFilePath: policyFilePath,
	}, nil
}

func (epsm *ExtensionPolicySettingsManager[T]) LoadExtensionPolicySettings() error {
	if epsm == nil {
		return fmt.Errorf("invalid ExtensionPolicySettingsManager: manager is nil")
	}
	if epsm.settingsFilePath == "" {
		return extensionerrors.ErrEmptyPolicyFilePath
	}

	// If an extension has a default policy configuration in case the file does not exist, they should handle that logic before calling this function.
	if _, err := os.Stat(epsm.settingsFilePath); os.IsNotExist(err) {
		return extensionerrors.ErrMissingPolicyFile
	} else if err != nil {
		return fmt.Errorf("error checking extension policy settings file: %w", err)
	}

	fileContent, err := os.ReadFile(epsm.settingsFilePath)
	if err != nil {
		return fmt.Errorf("failed to read extension policy settings file: %w", err) // TODO: Add retry logic if appropriate.
	}

	if len(fileContent) == 0 {
		return extensionerrors.ErrEmptyPolicyFile
	}

	var settings *T = new(T)
	if err := json.Unmarshal(fileContent, settings); err != nil {
		return fmt.Errorf("failed to unmarshal extension policy settings: %w", err)
	}

	// Extensions themselves must decide the criteria for valid policy settings (i.e., if they can be null etc.).
	if err := (*settings).ValidateFormat(); err != nil {
		return fmt.Errorf("extension policy loaded, but invalid format: %w", err)
	}

	epsm.settings = settings
	return nil
}

func (epsm *ExtensionPolicySettingsManager[T]) GetSettings() (*T, error) {
	if epsm.settings == nil {
		return nil, extensionerrors.ErrPolicyNotYetLoaded
	}
	return epsm.settings, nil
}

// Validation Helper Functions
type HashType int

const (
	HashTypeNone HashType = iota
	HashTypeSHA1
	HashTypeSHA256
)

func ValidateValueInAllowlist(value string, allowlist []string) error {
	if len(allowlist) == 0 {
		return extensionerrors.ErrPolicyAllowlistEmpty
	}

	for _, allowlistValue := range allowlist {
		if value == allowlistValue {
			return nil
		}
	}
	return extensionerrors.ErrItemNotInAllowlist
}

// This function is the entry point for most use cases: it takes in the filepath, reads the content, and
// determines if the content is allowlisted. If hashOpt is not HashTypeNone, it will compute the hash of the file content.
// If extensions don't want to validate a filepath but a value directly, they can call ValidateValueInAllowlist,
// which this function calls.
func ValidateFileHashInAllowlist(filePath string, allowlist []string, hashOpt HashType) error {
	if len(allowlist) == 0 {
		return extensionerrors.ErrPolicyAllowlistEmpty
	}

	if filePath == "" {
		return extensionerrors.ErrEmptyFilepathToValidate
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file to validate does not exist: %w", err)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s for validation: %w", filePath, err)
	}

	value := string(content)

	if hashOpt != HashTypeNone {
		value, err := ComputeFileHash(value, hashOpt)
		if err != nil {
			return fmt.Errorf("error occured when hashing contents of file %s for validation: %w", filePath, err)
		}
		return ValidateValueInAllowlist(value, allowlist)
	}

	return ValidateValueInAllowlist(value, allowlist)
}

// ComputeFileHash computes the hash of a file or leaves string as is.
func ComputeFileHash(contents string, hashOpt HashType) (string, error) {
	var hashStr string
	switch hashOpt {
	case HashTypeSHA1:
		hash := sha1.Sum([]byte(contents))
		hashStr = hex.EncodeToString(hash[:])
	default:
		hash := sha256.Sum256([]byte(contents))
		hashStr = hex.EncodeToString(hash[:])
	}

	return hashStr, nil
}
