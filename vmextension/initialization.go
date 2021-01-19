package vmextension

import (
	"github.com/Azure/azure-extension-platform/pkg/extensionerrors"
)

// CallbackFunc is used for a non-Enable operation callback
type CallbackFunc func(ext *VMExtension) error

// EnableCallbackFunc is used for Enable operation callbacks
type EnableCallbackFunc func(ext *VMExtension) (string, error)

// InitializationInfo is passed by the extension to specify how the framework should run
type InitializationInfo struct {
	Name                string             // The name of the extension, without the Linux or Windows suffix
	Version             string             // The version of the extension
	SupportsDisable     bool               // True if we should automatically disable the extension if Disable is called
	RequiresSeqNoChange bool               // True if Enable will only execute if the sequence number changes
	InstallExitCode     int                // Exit code to use for the install case
	OtherExitCode       int                // Exit code to use for all other cases
	EnableCallback      EnableCallbackFunc // Called for the enable operation
	DisableCallback     CallbackFunc       // Called for the disable operation. Only set this if the extension wants a callback.
	UpdateCallback      CallbackFunc       // Called for the update operation. If nil, then update is not supported.
}

// GetInitializationInfo returns a new InitializationInfo object
func GetInitializationInfo(name string, version string, requiresSeqNoChange bool, enableCallback EnableCallbackFunc) (*InitializationInfo, error) {
	if len(name) < 1 || len(version) < 1 {
		return nil, extensionerrors.ErrArgCannotBeNullOrEmpty
	}

	if enableCallback == nil {
		return nil, extensionerrors.ErrArgCannotBeNull
	}

	return &InitializationInfo{
		Name:                name,
		Version:             version,
		SupportsDisable:     true,
		RequiresSeqNoChange: requiresSeqNoChange,
		InstallExitCode:     52,
		OtherExitCode:       3,
		EnableCallback:      enableCallback,
		DisableCallback:     nil,
		UpdateCallback:      nil,
	}, nil
}
