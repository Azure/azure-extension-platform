// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package vmextension

import (
	"testing"

	"github.com/Azure/azure-extension-platform/pkg/extensionerrors"
	"github.com/stretchr/testify/require"
)

func Test_initializationInfoValidate(t *testing.T) {
	// Empty name
	_, err := GetInitializationInfo("", "5.0", true, testEnableCallback)
	require.Equal(t, extensionerrors.ErrArgCannotBeNullOrEmpty, err)

	// Empty version
	_, err = GetInitializationInfo("yaba", "", true, testEnableCallback)
	require.Equal(t, extensionerrors.ErrArgCannotBeNullOrEmpty, err)

	// Null enable callback
	_, err = GetInitializationInfo("yaba", "5.0", true, nil)
	require.Equal(t, extensionerrors.ErrArgCannotBeNull, err)
}

func Test_initializationInfoDefaults(t *testing.T) {
	ii, err := GetInitializationInfo("yaba", "5.0", true, testEnableCallback)
	require.NoError(t, err, "Error from initialization")
	require.Equal(t, "yaba", ii.Name)
	require.Equal(t, "5.0", ii.Version)
	require.Equal(t, true, ii.SupportsDisable)
	require.Equal(t, true, ii.RequiresSeqNoChange)
	require.Equal(t, 52, ii.InstallExitCode)
	require.Equal(t, 3, ii.OtherExitCode)
	require.Equal(t, "", ii.LogFileNamePattern)
}

func testEnableCallback(ext *VMExtension) (string, error) {
	return "blah", nil
}

func testEnableCallbackReadSettings(ext *VMExtension) (string, error) {
	_, err := ext.GetSettings()
	return "trying to read settings", err
}
