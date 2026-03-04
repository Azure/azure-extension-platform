package extensionpolicysettings

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/Azure/azure-extension-platform/pkg/extensionerrors"
	"github.com/Azure/azure-extension-platform/pkg/logging"
)

type ExtensionPolicySettings interface {
	ValidateFormat() error
}

type ExtensionPolicySettingsManager[T ExtensionPolicySettings] struct {
	settingsFilePath string
	logger           logging.ILogger
	settings         *T
}

func NewExtensionPolicySettingsManager[T ExtensionPolicySettings](policyFilePath string, logger logging.ILogger) (*ExtensionPolicySettingsManager[T], error) {
	if policyFilePath == "" {
		logger.Error("Policy file path is empty. ExtensionPolicySettingsManager may not function correctly.")
		return nil, fmt.Errorf("policy file path cannot be empty")
	}
	return &ExtensionPolicySettingsManager[T]{
		settingsFilePath: policyFilePath,
		logger:           logger, // settings is not loaded until LoadExtensionPolicySettings is called
	}, nil
}

func (epsm *ExtensionPolicySettingsManager[T]) LoadExtensionPolicySettings() error {
	if (epsm == nil) || (epsm.logger == nil) {
		return fmt.Errorf("invalid ExtensionPolicySettingsManager: manager or logger is nil")
	}
	if epsm.settingsFilePath == "" {
		return fmt.Errorf("invalid ExtensionPolicySettingsManager: settings file path is empty")
	}
	epsm.logger.Info(fmt.Sprintf("Loading extension policy settings from file: %s", epsm.settingsFilePath))

	// If an extension has a default policy configuration in case the file does not exist, they should handle that logic before calling this function.
	if _, err := os.Stat(epsm.settingsFilePath); os.IsNotExist(err) {
		return extensionerrors.ErrMissingPolicyFile
	} else if err != nil {
		return fmt.Errorf("error checking extension policy settings file: %w", err)
	}

	// Read the file content: check: what if the file is locked? What if we don't have permissions to read?

	fileContent, err := os.ReadFile(epsm.settingsFilePath)
	if err != nil {
		return fmt.Errorf("failed to read extension policy settings file: %w", err) // Should we have retry logic?
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
		return fmt.Errorf("extension policy settings validation failed: %w", err)
	}

	epsm.settings = settings
	epsm.logger.Info("Extension policy settings loaded and validated successfully.")
	return nil
}

func (epsm *ExtensionPolicySettingsManager[T]) GetSettings() *T {
	if epsm.settings == nil {
		epsm.logger.Info("Extension policy settings have not been loaded yet. Returning nil.")
	}
	return epsm.settings
}

// Validation Helper Functions
type HashType int

const (
	HashTypeNone HashType = iota
	HashTypeSHA1
	HashTypeSHA256
)

func ValidateValueInAllowlist(logger logging.ILogger, value string, allowlist []string) error {
	if len(allowlist) == 0 {
		return extensionerrors.ErrPolicyAllowlistEmpty
	}

	for _, allowlistValue := range allowlist {
		if value == allowlistValue {
			logger.Info("Validation successful: item is in the allowlist.")
			return nil
		}
	}
	logger.Info("validation failed: item is not in the allowlist.")
	return extensionerrors.ErrItemNotInAllowlist
}

func ValidateFileHashInAllowlist(logger logging.ILogger, filePath string, allowlist []string, hashOpt HashType) error {
	if len(allowlist) == 0 {
		return extensionerrors.ErrPolicyAllowlistEmpty
	}

	if filePath == "" {
		return extensionerrors.ErrEmptyFilepathToValidate
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s for validation: %w", filePath, err)
	}

	value := string(content) // What if content is empty? Do we want to treat that as an error or just compute the hash of an empty string? For now, we'll compute the hash of an empty string, but this is something to consider based on the specific use case and security requirements.

	if hashOpt != HashTypeNone {
		logger.Info(fmt.Sprintf("Computing hash of file %s for validation.", filePath))
		value, err := ComputeFileHash(logger, value, hashOpt)
		if err != nil {
			return fmt.Errorf("failed to compute hash for file %s for validation: %w", filePath, err)
		}
		logger.Info(fmt.Sprintf("Computed hash value for file %s: %s", filePath, value))
		return ValidateValueInAllowlist(logger, value, allowlist)
	}

	return ValidateValueInAllowlist(logger, value, allowlist)
}

// ComputeFileHash computes the hash of a file or leaves string as is.
func ComputeFileHash(logger logging.ILogger, contents string, hashOpt HashType) (string, error) {
	logger.Info("Computing hash for file contents")

	if contents == "" {
		return "", extensionerrors.ErrContentsToValidateEmpty
	}

	var hashStr string
	switch hashOpt {
	case HashTypeSHA1:
		hash := sha1.Sum([]byte(contents))
		hashStr = hex.EncodeToString(hash[:])
	default:
		hash := sha256.Sum256([]byte(contents))
		hashStr = hex.EncodeToString(hash[:])
	}

	logger.Info(fmt.Sprintf("Computed hash: %s", hashStr))
	return hashStr, nil
}
