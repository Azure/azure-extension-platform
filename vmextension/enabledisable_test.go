// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package vmextension

import (
	"fmt"
	"github.com/Azure/azure-extension-platform/pkg/extensionerrors"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	normalCallbackCalled bool
	errorCallbackCalled  bool
)

type evilDisableDependencies struct{}

func (evilDisableDependencies) writeFile(filename string, data []byte, perm os.FileMode) error {
	return extensionerrors.ErrNotFound
}

func (evilDisableDependencies) remove(name string) error {
	return extensionerrors.ErrNotFound
}

func Test_disableCallback(t *testing.T) {
	ext := createTestVMExtension()
	ext.exec.supportsDisable = false

	// Callback succeeds
	normalCallbackCalled = false
	errorCallbackCalled = false
	ext.exec.disableCallback = testDisableCallbackNormal
	_, err := disable(ext)
	require.NoError(t, err, "Disable callback failed")
	require.True(t, normalCallbackCalled)

	// Callback returns an error
	ext.exec.disableCallback = testDisableCallbackError
	_, err = disable(ext)
	require.Error(t, err, "oh no. The world is ending, but styling prevents me from using end punctuation or caps")
	require.True(t, errorCallbackCalled)
}

func Test_disableNotSupported(t *testing.T) {
	ext := createTestVMExtension()
	ext.exec.supportsDisable = false
	ext.exec.disableCallback = nil

	normalCallbackCalled = false
	errorCallbackCalled = false
	_, err := disable(ext)
	require.NoError(t, err, "Disable failed")
	require.False(t, normalCallbackCalled)
	require.False(t, errorCallbackCalled)
}

func Test_setDisabled(t *testing.T) {
	ext := createTestVMExtension()
	createDirsForVMExtension(ext)
	defer cleanupDirsForVMExtension(ext)
	ext.exec.supportsDisable = true

	// Enable
	require.NoError(t, setDisabled(ext, false), "Enabling failed")
	require.False(t, isDisabled(ext))

	// Disable
	require.NoError(t, setDisabled(ext, true), "Disabling failed")
	require.True(t, isDisabled(ext))

	// Disable when already disabled
	require.NoError(t, setDisabled(ext, true), "Disabling failed")
	require.True(t, isDisabled(ext))

	// Enable when disabled
	require.NoError(t, setDisabled(ext, false), "Enabling failed")
	require.False(t, isDisabled(ext))

	// Enable when already enabled
	require.NoError(t, setDisabled(ext, false), "Enabling failed")
	require.False(t, isDisabled(ext))
}

func Test_disabling(t *testing.T) {
	ext := createTestVMExtension()
	createDirsForVMExtension(ext)
	defer cleanupDirsForVMExtension(ext)
	ext.exec.supportsDisable = true

	// Ensure we're enabled first
	require.NoError(t, setDisabled(ext, false), "Enabling failed")
	require.False(t, isDisabled(ext))

	// Now disable from the disable method
	_, err := disable(ext)
	defer setDisabled(ext, false)
	require.NoError(t, err, "Disable failed")
	require.True(t, isDisabled(ext))

	// Disable again
	_, err = disable(ext)
	require.NoError(t, err, "Disable failed")
	require.True(t, isDisabled(ext))
}

func Test_cannotSetDisabled(t *testing.T) {
	ext := createTestVMExtension()
	ext.exec.supportsDisable = true
	disableDependency = evilDisableDependencies{}
	defer resetDependencies()

	_, err := disable(ext)
	require.Equal(t, extensionerrors.ErrNotFound, err)
}

func Test_cannotReenable(t *testing.T) {
	ext := createTestVMExtension()
	createDirsForVMExtension(ext)
	defer cleanupDirsForVMExtension(ext)
	ext.exec.supportsDisable = true

	// Disable the extension
	err := setDisabled(ext, true)
	require.NoError(t, err, "setDisabled failed")

	// Prevent the extension from being reenabled
	disableDependency = evilDisableDependencies{}
	defer setDisabled(ext, false) // Needs to be here because defer is LIFO
	defer resetDependencies()

	// Attempt to reenable the extension
	err = setDisabled(ext, false)
	require.Error(t, err, extensionerrors.ErrNotFound)
}

func resetDependencies() {
	installDependency = &installDependencyImpl{}
	disableDependency = &disableDependencyImpl{}
}

func testDisableCallbackNormal(ext *VMExtension) error {
	normalCallbackCalled = true
	return nil
}

func testDisableCallbackError(ext *VMExtension) error {
	errorCallbackCalled = true
	return fmt.Errorf("oh no. The world is ending, but styling prevents me from using end punctuation or caps")
}

