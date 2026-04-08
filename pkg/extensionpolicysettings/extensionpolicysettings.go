package extensionpolicysettings

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Azure/azure-extension-platform/pkg/extensionerrors"
	"github.com/Azure/azure-extension-platform/pkg/hashutils"
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

// These are the hash types supported by this package; they extend the definitions from utils.
// However, they must be explicitly mapped instead of just reusing hashutils.HashType because
// the hash types we support will be more restricted than the ones in hashutils, which may grow over time.
type HashType hashutils.HashType

const (
	HashTypeNone   HashType = 0
	HashTypeSHA1   HashType = 1
	HashTypeSHA256 HashType = 2
)

// Validation Helper Functions
func ValidateValueInAllowlist(value string, allowlist []string) error {
	if len(allowlist) == 0 {
		return extensionerrors.ErrPolicyAllowlistEmpty
	}

	for _, allowlistValue := range allowlist {
		// Although a hash value wouldn't have whitespace we trim spaces for other use cases of this function.
		trimmedAllowlistValue := strings.TrimSpace(allowlistValue)
		trimmedValue := strings.TrimSpace(value)
		if strings.EqualFold(trimmedValue, trimmedAllowlistValue) {
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

	if hashOpt == HashTypeNone {
		// If no hashing is needed, we can directly validate the file content against the allowlist.
		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file %s for validation: %w", filePath, err)
		}
		return ValidateValueInAllowlist(string(content), allowlist)
	}

	hashAlg, err := hashutils.GetHashAlgorithm(hashutils.HashType(hashOpt))
	if err != nil {
		return fmt.Errorf("error occured when getting hash algorithm for file %s: %w", filePath, err)
	}
	value, err := hashutils.ComputeFileHash(filePath, hashAlg)
	if err != nil {
		return fmt.Errorf("error occured when hashing contents of file %s for validation: %w", filePath, err)
	}
	return ValidateValueInAllowlist(value, allowlist)
}
