// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package vmextension

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
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
	normalCallbackCalled = false
	errorCallbackCalled = false
	ext.exec.updateCallback = testCallbackNormal
	_, err := update(ext)
	require.NoError(t, err, "Update callback failed")
	require.True(t, normalCallbackCalled)

	// Callback returns an error, but it isn't propagated
	ext.exec.updateCallback = testCallbackError
	_, err = update(ext)
	require.NoError(t, err)
	require.True(t, errorCallbackCalled)
}

func Test_installCallback(t *testing.T) {
	ext := createTestVMExtension()
	createDirsForVMExtension(ext)
	defer cleanupDirsForVMExtension(ext)

	// Callback succeeds
	normalCallbackCalled = false
	errorCallbackCalled = false
	ext.exec.installCallback = testCallbackNormal
	_, err := install(ext)
	require.NoError(t, err, "Install callback failed")
	require.True(t, normalCallbackCalled)

	// Callback returns an error, but it isn't propagated
	ext.exec.installCallback = testCallbackError
	_, err = install(ext)
	require.NoError(t, err)
	require.True(t, errorCallbackCalled)
}

func Test_uninstallCallback(t *testing.T) {
	ext := createTestVMExtension()
	createDirsForVMExtension(ext)
	defer cleanupDirsForVMExtension(ext)

	// Callback succeeds
	normalCallbackCalled = false
	errorCallbackCalled = false
	ext.exec.uninstallCallback = testCallbackNormal
	_, err := uninstall(ext)
	require.NoError(t, err, "Uninstall callback failed")
	require.True(t, normalCallbackCalled)

	// Callback returns an error, but it isn't propagated
	ext.exec.uninstallCallback = testCallbackError
	_, err = uninstall(ext)
	require.NoError(t, err)
	require.True(t, errorCallbackCalled)
}

func Test_resetStateCallback(t *testing.T) {
	ext := createTestVMExtension()
	createDirsForVMExtension(ext)
	defer cleanupDirsForVMExtension(ext)

	// Callback succeeds
	normalCallbackCalled = false
	errorCallbackCalled = false
	ext.exec.resetStateCallBack = testCallbackNormal
	_, err := resetState(ext)
	require.NoError(t, err, "Uninstall callback failed")
	require.True(t, normalCallbackCalled)

	// Callback returns an error, but it isn't propagated
	ext.exec.resetStateCallBack = testCallbackError
	_, err = resetState(ext)
	require.NoError(t, err)
	require.True(t, errorCallbackCalled)
}

func Test_resetStateRemoveFiles(t *testing.T) {
	ext := createTestVMExtension()
	createDirsForVMExtension(ext)
	defer cleanupDirsForVMExtension(ext)

	// Create a file that we'll delete in resetState
	filePath := path.Join(ext.HandlerEnv.DataFolder, "flarple.txt")
	emptyFile, err := os.Create(filePath)
	require.NoError(t, err)
	log.Println(emptyFile)
	emptyFile.Close()

	// Call resetState and verify the file is deleted
	ext.exec.supportsResetState = true
	_, err = resetState(ext)
	require.NoError(t, err)

	files, _ := ioutil.ReadDir(ext.HandlerEnv.DataFolder)
	for _, _ = range files {
		require.Fail(t, "Data directory file was not deleted")
	}
}

func Test_resetStateCannotOpenDirectory(t *testing.T) {
	ext := createTestVMExtension()

	// ResetState will encounter an error deleting the directory, but won't propagate it
	ext.exec.supportsResetState = true
	_, err := resetState(ext)
	require.NoError(t, err)
}

func Test_resetStateCannotRemoveFile(t *testing.T) {
	ext := createTestVMExtension()
	createDirsForVMExtension(ext)
	defer cleanupDirsForVMExtension(ext)

	// Create a file that we'll try to delete in resetState
	filePath := path.Join(ext.HandlerEnv.DataFolder, "blarp.txt")
	emptyFile, err := os.Create(filePath)
	require.NoError(t, err)
	log.Println(emptyFile)
	defer emptyFile.Close()

	// Call resetState, which will fail to delete the file because it's opened
	// but the error won't propagate
	ext.exec.supportsResetState = true
	_, err = resetState(ext)
	require.NoError(t, err)
}

func Test_resetStateEmptyDataDirectory(t *testing.T) {
	ext := createTestVMExtension()
	ext.HandlerEnv.DataFolder = ""

	ext.exec.supportsResetState = true
	_, err := resetState(ext)
	require.NoError(t, err)
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

func testCallbackNormal(ext *VMExtension) error {
	normalCallbackCalled = true
	return nil
}

func testCallbackError(ext *VMExtension) error {
	errorCallbackCalled = true
	return fmt.Errorf("oh no. The world is ending, but styling prevents me from using end punctuation or caps")
}

