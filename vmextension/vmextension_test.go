package vmextension

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path"
	"testing"
	"time"

	"github.com/Azure/azure-extension-platform/pkg/exithelper"
	"github.com/Azure/azure-extension-platform/pkg/extensionerrors"
	"github.com/Azure/azure-extension-platform/pkg/handlerenv"
	"github.com/Azure/azure-extension-platform/pkg/logging"
	"github.com/Azure/azure-extension-platform/pkg/seqno"
	"github.com/Azure/azure-extension-platform/pkg/settings"
	"github.com/Azure/azure-extension-platform/pkg/status"
	"github.com/stretchr/testify/require"
)

var (
	statusTestDirectory = "./statustest"
)

type mockGetVMExtensionEnvironmentManager struct {
	seqNo                         uint
	currentSeqNo                  uint
	he                            *handlerenv.HandlerEnvironment
	hs                            *settings.HandlerSettings
	getHandlerEnvironmentError    error
	findSeqNumError               error
	getCurrentSequenceNumberError error
	getHandlerSettingsError       error
	setSequenceNumberError        error
}

func (mm *mockGetVMExtensionEnvironmentManager) GetHandlerEnvironment(name string, version string) (he *handlerenv.HandlerEnvironment, _ error) {
	if mm.getHandlerEnvironmentError != nil {
		return he, mm.getHandlerEnvironmentError
	}

	return mm.he, nil
}

func (mm *mockGetVMExtensionEnvironmentManager) FindSeqNum(el *logging.ExtensionLogger, configFolder string) (uint, error) {
	if mm.findSeqNumError != nil {
		return 0, mm.findSeqNumError
	}

	return mm.seqNo, nil
}

func (mm *mockGetVMExtensionEnvironmentManager) GetCurrentSequenceNumber(el *logging.ExtensionLogger, retriever seqno.ISequenceNumberRetriever, name, version string) (uint, error) {
	if mm.getCurrentSequenceNumberError != nil {
		return 0, mm.getCurrentSequenceNumberError
	}

	return mm.currentSeqNo, nil
}

func (mm *mockGetVMExtensionEnvironmentManager) GetHandlerSettings(el *logging.ExtensionLogger, he *handlerenv.HandlerEnvironment, seqNo uint) (hs *settings.HandlerSettings, _ error) {
	if mm.getHandlerSettingsError != nil {
		return hs, mm.getHandlerSettingsError
	}

	return mm.hs, nil
}

func (mm *mockGetVMExtensionEnvironmentManager) SetSequenceNumberInternal(extensionName, extensionVersion string, seqNo uint) error {
	if mm.setSequenceNumberError != nil {
		return mm.setSequenceNumberError
	}

	return nil
}

func Test_reportStatusShouldntReport(t *testing.T) {
	ext := createTestVMExtension()
	c := cmd{nil, "Install", false, 99}
	ext.HandlerEnv.StatusFolder = statusTestDirectory
	ext.RequestedSequenceNumber = 45

	err := reportStatus(ext, status.StatusSuccess, c, "msg")
	require.NoError(t, err, "reportStatus failed")
	_, err = os.Stat(path.Join(statusTestDirectory, "45.status"))
	require.True(t, os.IsNotExist(err), "File exists when we don't expect it to")
}

func Test_reportStatusCouldntSave(t *testing.T) {
	ext := createTestVMExtension()
	c := cmd{nil, "Install", true, 99}
	ext.HandlerEnv.StatusFolder = "./yabamonster"
	ext.RequestedSequenceNumber = 45

	err := reportStatus(ext, status.StatusSuccess, c, "msg")
	require.Error(t, err)
}

func Test_reportStatusSaved(t *testing.T) {
	ext := createTestVMExtension()

	c := cmd{nil, "Install", true, 99}
	ext.HandlerEnv.StatusFolder = statusTestDirectory
	ext.RequestedSequenceNumber = 45

	createDirsForVMExtension(ext)
	defer cleanupDirsForVMExtension(ext)

	err := reportStatus(ext, status.StatusSuccess, c, "msg")
	require.NoError(t, err, "reportStatus failed")
	_, err = os.Stat(path.Join(statusTestDirectory, "45.status"))
	require.NoError(t, err, "File doesn't exist")
}

func Test_getVMExtensionNilValues(t *testing.T) {
	_, err := GetVMExtension(nil)
	require.Equal(t, extensionerrors.ErrArgCannotBeNull, err)

	initInfo := &InitializationInfo{Name: ""}
	_, err = GetVMExtension(initInfo)
	require.Equal(t, extensionerrors.ErrArgCannotBeNullOrEmpty, err)

	initInfo = &InitializationInfo{Name: "yaba", Version: ""}
	_, err = GetVMExtension(initInfo)
	require.Equal(t, extensionerrors.ErrArgCannotBeNullOrEmpty, err)

	initInfo = &InitializationInfo{Name: "yaba", Version: "1.0", EnableCallback: nil}
	_, err = GetVMExtension(initInfo)
	require.Equal(t, extensionerrors.ErrArgCannotBeNull, err)
}

func Test_getVMExtensionGetHandlerEnvironmentError(t *testing.T) {
	myerr := errors.New("cannot handle the environment")

	ii, _ := GetInitializationInfo("yaba", "5.0", true, testEnableCallback)
	mm := &mockGetVMExtensionEnvironmentManager{getHandlerEnvironmentError: myerr}
	_, err := getVMExtensionInternal(ii, mm)
	require.Equal(t, myerr, err)
}

func Test_getVMExtensionCannotFindSeqNo(t *testing.T) {
	mm := createMockVMExtensionEnvironmentManager()
	mm.findSeqNumError = errors.New("the sequence number annoys me")
	ii, _ := GetInitializationInfo("yaba", "5.0", true, testEnableCallback)

	_, err := getVMExtensionInternal(ii, mm)
	require.Error(t, err)
}

func Test_getVMExtensionCannotReadCurrentSeqNo(t *testing.T) {
	mm := createMockVMExtensionEnvironmentManager()
	mm.getCurrentSequenceNumberError = errors.New("the current sequence number is beyond our comprehension")
	ii, _ := GetInitializationInfo("yaba", "5.0", true, testEnableCallback)

	_, err := getVMExtensionInternal(ii, mm)
	require.Error(t, err)
}

func Test_getVMExtensionUpdateSupport(t *testing.T) {
	// Update disabled
	mm := createMockVMExtensionEnvironmentManager()
	ii, _ := GetInitializationInfo("yaba", "5.0", true, testEnableCallback)
	ext, err := getVMExtensionInternal(ii, mm)
	require.NoError(t, err, "getVMExtensionInternal failed")
	require.NotNil(t, ext)

	// Verify this is a noop
	updateNormalCallbackCalled = false
	cmd := ext.exec.cmds["update"]
	require.NotNil(t, cmd)
	_, err = cmd.f(ext)
	require.NoError(t, err, "updateCallback failed")
	require.False(t, updateNormalCallbackCalled)

	// Update enabled
	ii.UpdateCallback = testUpdateCallbackNormal
	ext, err = getVMExtensionInternal(ii, mm)
	require.NoError(t, err, "getVMExtensionInternal failed")
	require.NotNil(t, ext)

	// Verify this is not a noop
	cmd = ext.exec.cmds["update"]
	require.NotNil(t, cmd)
	_, err = cmd.f(ext)
	require.NoError(t, err, "updateCallback failed")
	require.True(t, updateNormalCallbackCalled)
}

func Test_getVMExtensionDisableSupport(t *testing.T) {
	// Disbable disabled
	mm := createMockVMExtensionEnvironmentManager()
	ii, _ := GetInitializationInfo("yaba", "5.0", true, testEnableCallback)
	ii.SupportsDisable = false
	ext, err := getVMExtensionInternal(ii, mm)
	require.NoError(t, err, "getVMExtensionInternal failed")
	require.NotNil(t, ext)

	createDirsForVMExtension(ext)
	defer cleanupDirsForVMExtension(ext)

	// Verify this is a noop
	err = setDisabled(ext, false)
	require.NoError(t, err, "setDisabled failed")
	cmd := ext.exec.cmds["disable"]
	require.NotNil(t, cmd)
	_, err = cmd.f(ext)
	require.NoError(t, err, "disable cmd failed")
	require.False(t, isDisabled(ext))

	// Disable enabled
	ii.SupportsDisable = true
	ext, err = getVMExtensionInternal(ii, mm)
	require.NoError(t, err, "getVMExtensionInternal failed")
	require.NotNil(t, ext)

	// Verify this is not a noop
	cmd = ext.exec.cmds["disable"]
	require.NotNil(t, cmd)
	_, err = cmd.f(ext)
	defer setDisabled(ext, false)
	require.NoError(t, err, "disable cmd failed")
	require.True(t, isDisabled(ext))
}

func Test_getVMExtensionCannotGetSettings(t *testing.T) {
	mm := createMockVMExtensionEnvironmentManager()
	mm.getHandlerSettingsError = errors.New("the settings exist only in a parallel dimension")
	ii, _ := GetInitializationInfo("yaba", "5.0", true, testEnableCallback)

	_, err := getVMExtensionInternal(ii, mm)
	require.Equal(t, mm.getHandlerSettingsError, err)
}

func Test_getVMExtensionNormalOperation(t *testing.T) {
	mm := createMockVMExtensionEnvironmentManager()
	ii, _ := GetInitializationInfo("yaba", "5.0", true, testEnableCallback)

	ext, err := getVMExtensionInternal(ii, mm)
	require.NoError(t, err, "getVMExtensionInternal failed")
	require.NotNil(t, ext)
}

func Test_parseCommandWrongArgsCount(t *testing.T) {
	if os.Getenv("DIE_PROCESS_DIE") == "1" {
		mm := createMockVMExtensionEnvironmentManager()
		ii, _ := GetInitializationInfo("yaba", "5.0", true, testEnableCallback)
		ext, _ := getVMExtensionInternal(ii, mm)

		args := make([]string, 1)
		args[0] = "install"
		ext.parseCmd(args)
		return
	}

	// Verify that the process exits
	cmd := exec.Command(os.Args[0], "-test.run=Test_parseCommandWrongArgsCount")
	cmd.Env = append(os.Environ(), "DIE_PROCESS_DIE=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}

func Test_parseCommandUnsupportedOperation(t *testing.T) {
	if os.Getenv("DIE_PROCESS_DIE") == "1" {
		mm := createMockVMExtensionEnvironmentManager()
		ii, _ := GetInitializationInfo("yaba", "5.0", true, testEnableCallback)
		ext, _ := getVMExtensionInternal(ii, mm)

		args := make([]string, 2)
		args[0] = "processname_dont_care"
		args[1] = "flipperdoodle"
		ext.parseCmd(args)
		return
	}

	// Verify that the process exits
	cmd := exec.Command(os.Args[0], "-test.run=Test_parseCommandUnsupportedOperation")
	cmd.Env = append(os.Environ(), "DIE_PROCESS_DIE=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}

func Test_parseCommandNormalOperation(t *testing.T) {
	mm := createMockVMExtensionEnvironmentManager()
	ii, _ := GetInitializationInfo("yaba", "5.0", true, testEnableCallback)
	ext, _ := getVMExtensionInternal(ii, mm)

	args := make([]string, 2)
	args[0] = "processname_dont_care"
	args[1] = "install"
	cmd := ext.parseCmd(args)
	require.NotNil(t, cmd)
}

func Test_enableNoSeqNoChangeButRequired(t *testing.T) {
	if os.Getenv("DIE_PROCESS_DIE") == "1" {
		mm := createMockVMExtensionEnvironmentManager()
		mm.currentSeqNo = mm.seqNo
		ii, _ := GetInitializationInfo("yaba", "5.0", true, testEnableCallback)
		ii.RequiresSeqNoChange = true
		ext, _ := getVMExtensionInternal(ii, mm)

		enable(ext)
		exithelper.Exiter.Exit(2) // enable above should exit the process cleanly. If it doesn't, fail.
	}

	// Verify that the process exits
	cmd := exec.Command(os.Args[0], "-test.run=Test_enableNoSeqNoChangeButRequired")
	cmd.Env = append(os.Environ(), "DIE_PROCESS_DIE=1")
	err := cmd.Run()
	if _, ok := err.(*exec.ExitError); !ok {
		return
	}
	t.Fatal("Process didn't shut cleanly as expected")
}

func Test_reenableExtension(t *testing.T) {
	mm := createMockVMExtensionEnvironmentManager()
	ii, _ := GetInitializationInfo("yaba", "5.0", true, testEnableCallback)
	ii.SupportsDisable = true
	ext, _ := getVMExtensionInternal(ii, mm)

	createDirsForVMExtension(ext)
	defer cleanupDirsForVMExtension(ext)
	resetDependencies()

	err := setDisabled(ext, true)
	//defer setDisabled(ext, false)
	time.Sleep(1000 * time.Millisecond)
	require.NoError(t, err, "setDisabled failed")
	_, err = enable(ext)
	time.Sleep(1000 * time.Millisecond)
	require.NoError(t, err, "enable failed")
	require.False(t, isDisabled(ext))
}

func Test_reenableExtensionFails(t *testing.T) {
	mm := createMockVMExtensionEnvironmentManager()
	ii, _ := GetInitializationInfo("yaba", "5.0", true, testEnableCallback)
	ii.SupportsDisable = true
	ext, _ := getVMExtensionInternal(ii, mm)

	createDirsForVMExtension(ext)
	defer cleanupDirsForVMExtension(ext)

	err := setDisabled(ext, true)
	defer setDisabled(ext, false)
	require.NoError(t, err, "setDisabled failed")
	disableDependency = evilDisableDependencies{}
	defer resetDependencies()
	msg, err := enable(ext)
	require.NoError(t, err) // We let the extension continue if we fail to reenable it
	require.Equal(t, "blah", msg)
}

func Test_enableCallbackFails(t *testing.T) {
	mm := createMockVMExtensionEnvironmentManager()
	ii, _ := GetInitializationInfo("yaba", "5.0", true, testFailEnableCallback)
	ext, _ := getVMExtensionInternal(ii, mm)

	_, err := enable(ext)
	require.Equal(t, extensionerrors.ErrMustRunAsAdmin, err)
}

func Test_enableCallbackSucceeds(t *testing.T) {
	mm := createMockVMExtensionEnvironmentManager()
	ii, _ := GetInitializationInfo("yaba", "5.0", true, testEnableCallback)
	ext, _ := getVMExtensionInternal(ii, mm)

	msg, err := enable(ext)
	require.NoError(t, err, "enable failed")
	require.Equal(t, "blah", msg)
}

func Test_doFailToWriteSequenceNumber(t *testing.T) {
	mm := createMockVMExtensionEnvironmentManager()
	mm.setSequenceNumberError = extensionerrors.ErrMustRunAsAdmin
	ii, _ := GetInitializationInfo("yaba", "5.0", true, testEnableCallback)
	ext, _ := getVMExtensionInternal(ii, mm)

	// We log but continue if we fail to write the sequence number
	oldArgs := os.Args
	defer putBackArgs(oldArgs)
	os.Args = make([]string, 2)
	os.Args[0] = "dontcare"
	os.Args[1] = "enable"
	ext.Do()
}

func Test_doCommandFails(t *testing.T) {
	if os.Getenv("DIE_PROCESS_DIE") == "1" {
		mm := createMockVMExtensionEnvironmentManager()
		ii, _ := GetInitializationInfo("yaba", "5.0", true, testFailEnableCallback)
		ext, _ := getVMExtensionInternal(ii, mm)

		oldArgs := os.Args
		defer putBackArgs(oldArgs)
		os.Args = make([]string, 2)
		os.Args[0] = "dontcare"
		os.Args[1] = "enable"
		ext.Do()
		return
	}

	// Verify that the process exits
	cmd := exec.Command(os.Args[0], "-test.run=Test_doCommandFails")
	cmd.Env = append(os.Environ(), "DIE_PROCESS_DIE=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("process ran with err %v, want exit status 3", err)
}

func Test_doCommandSucceeds(t *testing.T) {
	mm := createMockVMExtensionEnvironmentManager()
	ii, _ := GetInitializationInfo("yaba", "5.0", true, testEnableCallback)
	ext, _ := getVMExtensionInternal(ii, mm)

	oldArgs := os.Args
	defer putBackArgs(oldArgs)
	os.Args = make([]string, 2)
	os.Args[0] = "dontcare"
	os.Args[1] = "enable"
	ext.Do()
}

func putBackArgs(args []string) {
	os.Args = args
}

func testFailEnableCallback(ext *VMExtension) (string, error) {
	return "", extensionerrors.ErrMustRunAsAdmin
}

func getTestHandlerEnvironment() *handlerenv.HandlerEnvironment {
	return &handlerenv.HandlerEnvironment{
		HeartbeatFile: path.Join(".", "testdir", "heartbeat.txt"),
		StatusFolder:  path.Join(".", "testdir", "status"),
		ConfigFolder:  path.Join(".", "testdir", "config"),
		LogFolder:     path.Join(".", "testdir", "log"),
		DataFolder:    path.Join(".", "testdir", "data"),
	}
}

var one uint = 1

func createTestVMExtension() *VMExtension {
	return &VMExtension{
		Name:                    "yaba",
		Version:                 "5.0",
		RequestedSequenceNumber: 2,
		CurrentSequenceNumber:   &one,
		HandlerEnv:              getTestHandlerEnvironment(),
		Settings:                &settings.HandlerSettings{},
		ExtensionLogger:         logging.New(nil),
		exec: &executionInfo{
			requiresSeqNoChange: true,
			supportsDisable:     true,
			enableCallback:      testEnableCallback,
			disableCallback:     testDisableCallbackNormal,
			updateCallback:      nil,
			cmds:                nil,
		},
	}
}

func createMockVMExtensionEnvironmentManager() *mockGetVMExtensionEnvironmentManager {
	publicSettings := make(map[string]interface{})
	publicSettings["Flipper"] = "flip"
	publicSettings["Flopper"] = "flop"
	publiSettingJsonBytes, _ := json.Marshal(publicSettings)
	hs := &settings.HandlerSettings{PublicSettings: string(publiSettingJsonBytes), ProtectedSettings: ""}
	he := getTestHandlerEnvironment()

	return &mockGetVMExtensionEnvironmentManager{
		seqNo:        5,
		currentSeqNo: 4,
		hs:           hs,
		he:           he,
	}
}

func createDirsForVMExtension(vmExt *VMExtension) error {
	if err := os.MkdirAll(vmExt.HandlerEnv.StatusFolder, 0700); err != nil {
		return err
	}
	if err := os.MkdirAll(vmExt.HandlerEnv.ConfigFolder, 0700); err != nil {
		return err
	}
	if err := os.MkdirAll(vmExt.HandlerEnv.LogFolder, 0700); err != nil {
		return err
	}
	return os.MkdirAll(vmExt.HandlerEnv.DataFolder, 0700)
}

func cleanupDirsForVMExtension(vmExt *VMExtension) (combinedError error) {
	combinedError = extensionerrors.CombineErrors(combinedError, os.RemoveAll(vmExt.HandlerEnv.StatusFolder))
	combinedError = extensionerrors.CombineErrors(combinedError, os.RemoveAll(vmExt.HandlerEnv.ConfigFolder))
	combinedError = extensionerrors.CombineErrors(combinedError, os.RemoveAll(vmExt.HandlerEnv.LogFolder))
	combinedError = extensionerrors.CombineErrors(combinedError, os.RemoveAll(vmExt.HandlerEnv.DataFolder))
	return
}
