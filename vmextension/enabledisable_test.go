package vmextension

import (
	"fmt"
	"os"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/require"
)

var (
	normalCallbackCalled bool
	errorCallbackCalled  bool
)

type evilDisableDependencies struct{}

func (evilDisableDependencies) writeFile(filename string, data []byte, perm os.FileMode) error {
	return errNotFound
}

func (evilDisableDependencies) remove(name string) error {
	return errNotFound
}

func Test_disableCallback(t *testing.T) {
	ctx := log.NewSyncLogger(log.NewLogfmtLogger(os.Stdout))
	ext := createTestVMExtension()
	ext.exec.supportsDisable = false

	// Callback succeeds
	normalCallbackCalled = false
	errorCallbackCalled = false
	ext.exec.disableCallback = testDisableCallbackNormal
	_, err := disable(ctx, ext)
	require.NoError(t, err, "Disable callback failed")
	require.True(t, normalCallbackCalled)

	// Callback returns an error
	ext.exec.disableCallback = testDisableCallbackError
	_, err = disable(ctx, ext)
	require.Error(t, err, "oh no. The world is ending, but styling prevents me from using end punctuation or caps")
	require.True(t, errorCallbackCalled)
}

func Test_disableNotSupported(t *testing.T) {
	ctx := log.NewSyncLogger(log.NewLogfmtLogger(os.Stdout))
	ext := createTestVMExtension()
	ext.exec.supportsDisable = false
	ext.exec.disableCallback = nil

	normalCallbackCalled = false
	errorCallbackCalled = false
	_, err := disable(ctx, ext)
	require.NoError(t, err, "Disable failed")
	require.False(t, normalCallbackCalled)
	require.False(t, errorCallbackCalled)
}

func Test_setDisabled(t *testing.T) {
	ctx := log.NewSyncLogger(log.NewLogfmtLogger(os.Stdout))
	ext := createTestVMExtension()
	createDirsForVMExtension(ext)
	defer cleanupDirsForVMExtension(ext)
	ext.exec.supportsDisable = true

	// Enable
	require.NoError(t, setDisabled(ctx, ext, false), "Enabling failed")
	require.False(t, isDisabled(ctx, ext))

	// Disable
	require.NoError(t, setDisabled(ctx, ext, true), "Disabling failed")
	require.True(t, isDisabled(ctx, ext))

	// Disable when already disabled
	require.NoError(t, setDisabled(ctx, ext, true), "Disabling failed")
	require.True(t, isDisabled(ctx, ext))

	// Enable when disabled
	require.NoError(t, setDisabled(ctx, ext, false), "Enabling failed")
	require.False(t, isDisabled(ctx, ext))

	// Enable when already enabled
	require.NoError(t, setDisabled(ctx, ext, false), "Enabling failed")
	require.False(t, isDisabled(ctx, ext))
}

func Test_disabling(t *testing.T) {
	ctx := log.NewSyncLogger(log.NewLogfmtLogger(os.Stdout))
	ext := createTestVMExtension()
	createDirsForVMExtension(ext)
	defer cleanupDirsForVMExtension(ext)
	ext.exec.supportsDisable = true

	// Ensure we're enabled first
	require.NoError(t, setDisabled(ctx, ext, false), "Enabling failed")
	require.False(t, isDisabled(ctx, ext))

	// Now disable from the disable method
	_, err := disable(ctx, ext)
	defer setDisabled(ctx, ext, false)
	require.NoError(t, err, "Disable failed")
	require.True(t, isDisabled(ctx, ext))

	// Disable again
	_, err = disable(ctx, ext)
	require.NoError(t, err, "Disable failed")
	require.True(t, isDisabled(ctx, ext))
}

func Test_cannotSetDisabled(t *testing.T) {
	ctx := log.NewSyncLogger(log.NewLogfmtLogger(os.Stdout))
	ext := createTestVMExtension()
	ext.exec.supportsDisable = true
	disableDependency = evilDisableDependencies{}
	defer resetDependencies()

	_, err := disable(ctx, ext)
	require.Equal(t, errNotFound, err)
}

func Test_cannotReenable(t *testing.T) {
	ctx := log.NewSyncLogger(log.NewLogfmtLogger(os.Stdout))
	ext := createTestVMExtension()
	createDirsForVMExtension(ext)
	defer cleanupDirsForVMExtension(ext)
	ext.exec.supportsDisable = true

	// Disable the extension
	err := setDisabled(ctx, ext, true)
	require.NoError(t, err, "setDisabled failed")

	// Prevent the extension from being reenabled
	disableDependency = evilDisableDependencies{}
	defer setDisabled(ctx, ext, false) // Needs to be here because defer is LIFO
	defer resetDependencies()

	// Attempt to reenable the extension
	err = setDisabled(ctx, ext, false)
	require.Error(t, err, errNotFound)
}

func resetDependencies() {
	installDependency = &installDependencyImpl{}
	disableDependency = &disableDependencyImpl{}
}

func testDisableCallbackNormal(ctx log.Logger, ext *VMExtension) error {
	normalCallbackCalled = true
	return nil
}

func testDisableCallbackError(ctx log.Logger, ext *VMExtension) error {
	errorCallbackCalled = true
	return fmt.Errorf("oh no. The world is ending, but styling prevents me from using end punctuation or caps")
}
