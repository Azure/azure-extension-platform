// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package vmextension

import (
	"github.com/Azure/azure-extension-platform/pkg/extensionerrors"
	"github.com/Azure/azure-extension-platform/pkg/status"
)

// CallbackFunc is used for a non-Enable operation callback
type CallbackFunc func(ext *VMExtension) error

// EnableCallbackFunc is used for Enable operation callbacks
type EnableCallbackFunc func(ext *VMExtension) (string, error)

// InitializationInfo is passed by the extension to specify how the framework should run
type InitializationInfo struct {
	Name                  string                        // The name of the extension, without the Linux or Windows suffix
	Version               string                        // The version of the extension
	SupportsDisable       bool                          // True if we should automatically disable the extension if Disable is called
	SupportsResetState    bool                          // True if we should remove all contents of all folder when ResetState is called
	RequiresSeqNoChange   bool                          // True if Enable will only execute if the sequence number changes
	InstallExitCode       int                           // Exit code to use for the install case
	OtherExitCode         int                           // Exit code to use for all other cases
	EnableCallback        EnableCallbackFunc            // Called for the enable operation
	DisableCallback       CallbackFunc                  // Called for the Disable operation. Only set this if the extension wants a callback.
	UpdateCallback        CallbackFunc                  // Called for the Update operation. If nil, then update is not supported.
	ResetStateCallback    CallbackFunc                  // Called for the ResetState operation. Only set this if the extension wants a callback.
	InstallCallback       CallbackFunc                  // Called for the Install operation. Only set this if the extension wants a callback.
	UninstallCallback     CallbackFunc                  // Called for the Uninstall operation. Only set this if the extension wants a callback.
	CustomStatusFormatter status.StatusMessageFormatter // Provide a function to format the status message. If nil default formatting behavior will be preserved.
	LogFileNameFormat     string                        // Default format to use for log files; Eg: "<name_pattern>%v"
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
		SupportsResetState:  true,
		RequiresSeqNoChange: requiresSeqNoChange,
		InstallExitCode:     52,
		OtherExitCode:       3,
		EnableCallback:      enableCallback,
		DisableCallback:     nil,
		UpdateCallback:      nil,
		ResetStateCallback:  nil,
		InstallCallback:     nil,
		UninstallCallback:   nil,
		LogFileNameFormat:   "",
	}, nil
}
