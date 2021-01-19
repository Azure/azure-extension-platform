package vmextension

import (
	"fmt"
	"os"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

var (
	updateNormalCallbackCalled bool
	updateErrorCallbackCalled  bool
)

type evilInstallDependencies struct {
	mkdirCalled          bool
	removeAllCalled      bool
	statErrorToReturn    error
	installErrorToReturn error
}

func (eid *evilInstallDependencies) mkdirAll(path string, perm os.FileMode) error {
	eid.mkdirCalled = true
	return eid.installErrorToReturn
}

func (eid *evilInstallDependencies) removeAll(path string) error {
	eid.removeAllCalled = true
	return eid.installErrorToReturn
}

func (eid *evilInstallDependencies) stat(name string) (os.FileInfo, error) {
	return nil, eid.statErrorToReturn
}

func Test_updateCallback(t *testing.T) {
	ext := createTestVMExtension()
	createDirsForVMExtension(ext)
	defer cleanupDirsForVMExtension(ext)

	// Callback succeeds
	updateNormalCallbackCalled = false
	updateErrorCallbackCalled = false
	ext.exec.updateCallback = testUpdateCallbackNormal
	_, err := update(ext)
	require.NoError(t, err, "Update callback failed")
	require.True(t, updateNormalCallbackCalled)

	// Callback returns an error
	ext.exec.disableCallback = testUpdateCallbackError
	_, err = disable(ext)
	require.Error(t, err, "oh no. The world is ending, but styling prevents me from using end punctuation or caps")
	require.True(t, updateErrorCallbackCalled)
}

func Test_installAlreadyExists(t *testing.T) {
	ext := createTestVMExtension()

	installDependency = &evilInstallDependencies{mkdirCalled: false, statErrorToReturn: nil}
	defer resetDependencies()

	_, err := install(ext)
	require.NoError(t, err, "install failed")
	require.False(t, installDependency.(*evilInstallDependencies).mkdirCalled)
}

func Test_installSuccess(t *testing.T) {
	ext := createTestVMExtension()

	installDependency = &evilInstallDependencies{mkdirCalled: false, statErrorToReturn: os.ErrNotExist, installErrorToReturn: nil}
	defer resetDependencies()

	_, err := install(ext)
	require.NoError(t, err, "install failed")
	require.True(t, installDependency.(*evilInstallDependencies).mkdirCalled)
}

func Test_installFailToMakeDir(t *testing.T) {
	ext := createTestVMExtension()

	installDependency = &evilInstallDependencies{}
	defer resetDependencies()

	installDependency = &evilInstallDependencies{mkdirCalled: false, statErrorToReturn: os.ErrNotExist, installErrorToReturn: errors.New("something happened")}
	defer resetDependencies()

	_, err := install(ext)
	require.Error(t, err, installDependency.(*evilInstallDependencies).installErrorToReturn)
	require.True(t, installDependency.(*evilInstallDependencies).mkdirCalled)
}

func Test_installFileExistFails(t *testing.T) {
	ext := createTestVMExtension()

	installDependency = &evilInstallDependencies{}
	defer resetDependencies()

	installDependency = &evilInstallDependencies{mkdirCalled: false, statErrorToReturn: errors.New("bad permissions"), installErrorToReturn: os.ErrNotExist}
	defer resetDependencies()

	_, err := install(ext)
	require.Error(t, err, installDependency.(*evilInstallDependencies).statErrorToReturn)
	require.False(t, installDependency.(*evilInstallDependencies).mkdirCalled)
}

func Test_uninstallAlreadyGone(t *testing.T) {
	ext := createTestVMExtension()

	installDependency = &evilInstallDependencies{removeAllCalled: false, statErrorToReturn: os.ErrNotExist}
	defer resetDependencies()

	_, err := uninstall(ext)
	require.NoError(t, err, "uninstall failed")
	require.False(t, installDependency.(*evilInstallDependencies).removeAllCalled)
}

func Test_uninstallSuccess(t *testing.T) {
	ext := createTestVMExtension()

	installDependency = &evilInstallDependencies{removeAllCalled: false, statErrorToReturn: nil, installErrorToReturn: nil}
	defer resetDependencies()

	_, err := uninstall(ext)
	require.NoError(t, err, "uninstall failed")
	require.True(t, installDependency.(*evilInstallDependencies).removeAllCalled)
}

func Test_uninstallFailToRemoveDir(t *testing.T) {
	ext := createTestVMExtension()

	installDependency = &evilInstallDependencies{}
	defer resetDependencies()

	installDependency = &evilInstallDependencies{removeAllCalled: false, statErrorToReturn: nil, installErrorToReturn: errors.New("something happened")}
	defer resetDependencies()

	_, err := uninstall(ext)
	require.Error(t, err, installDependency.(*evilInstallDependencies).installErrorToReturn)
	require.True(t, installDependency.(*evilInstallDependencies).removeAllCalled)
}

func Test_uninstallFileExistFails(t *testing.T) {
	ext := createTestVMExtension()

	installDependency = &evilInstallDependencies{}
	defer resetDependencies()

	installDependency = &evilInstallDependencies{removeAllCalled: false, statErrorToReturn: errors.New("bad permissions"), installErrorToReturn: os.ErrNotExist}
	defer resetDependencies()

	_, err := uninstall(ext)
	require.Error(t, err, installDependency.(*evilInstallDependencies).statErrorToReturn)
	require.False(t, installDependency.(*evilInstallDependencies).removeAllCalled)
}

func testUpdateCallbackNormal(ext *VMExtension) error {
	updateNormalCallbackCalled = true
	return nil
}

func testUpdateCallbackError(ext *VMExtension) error {
	updateErrorCallbackCalled = true
	return fmt.Errorf("oh no. The world is ending, but styling prevents me from using end punctuation or caps")
}
