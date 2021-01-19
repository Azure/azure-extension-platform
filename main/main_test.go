package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/Azure/azure-extension-platform/pkg/constants"
	"github.com/Azure/azure-extension-platform/pkg/exithelper"
	"github.com/Azure/azure-extension-platform/pkg/extensionerrors"
	"github.com/Azure/azure-extension-platform/pkg/handlerenv"
	"github.com/Azure/azure-extension-platform/pkg/logging"
	"github.com/Azure/azure-extension-platform/pkg/seqno"
	"github.com/Azure/azure-extension-platform/pkg/settings"
	"github.com/Azure/azure-extension-platform/pkg/status"
	"github.com/Azure/azure-extension-platform/vmextension"
	"github.com/stretchr/testify/assert"
)

var testdir = path.Join(".", "testenv")
var heartbeatFile = path.Join(testdir, "heartbeat")
var statusFolder = path.Join(testdir, "status")
var configFolder = path.Join(testdir, "config")
var logFolder = path.Join(testdir, "log")
var dataFolder = path.Join(testdir, "data")

var settingsData = `
{
	"runtimeSettings": [
		{
			"handlerSettings": {
				"publicSettings": {
					"commandToExecute": "echo 'hello world'"
				},
				"protectedSettings": null,
				"protectedSettingsCertThumbprint": null
			}
		}
	]
}
`

type mockVMExtensionEnvironmentManager struct {
	handlerEnvironment *handlerenv.HandlerEnvironment
	currentSeqNum      *uint
	setSequenceNumber  uint
}

func (mock *mockVMExtensionEnvironmentManager) GetHandlerEnvironment(name string, version string) (*handlerenv.HandlerEnvironment, error) {
	return mock.handlerEnvironment, nil
}

func (mock *mockVMExtensionEnvironmentManager) FindSeqNum(el *logging.ExtensionLogger, configFolder string) (uint, error) {
	return seqno.FindSeqNum(el, configFolder)
}

func (mock *mockVMExtensionEnvironmentManager) GetCurrentSequenceNumber(el *logging.ExtensionLogger, retriever seqno.ISequenceNumberRetriever, name, version string) (uint, error) {
	if mock.currentSeqNum == nil {
		return 0, extensionerrors.ErrNoSettingsFiles
	} else {
		return *mock.currentSeqNum, nil
	}
}
func (*mockVMExtensionEnvironmentManager) GetHandlerSettings(el *logging.ExtensionLogger, he *handlerenv.HandlerEnvironment, seqNo uint) (*settings.HandlerSettings, error) {
	return settings.GetHandlerSettings(el, he, seqNo)
}
func (mock *mockVMExtensionEnvironmentManager) SetSequenceNumberInternal(extensionName, extensionVersion string, seqNo uint) error {
	if mock.currentSeqNum == nil {
		mock.currentSeqNum = new(uint)
	}
	*mock.currentSeqNum = seqNo
	return nil
}

var enableCallbackCalled = false

var mockEnableCallbackFunc vmextension.EnableCallbackFunc = func(ext *vmextension.VMExtension) (string, error) {
	enableCallbackCalled = true
	return fmt.Sprintf("enable callback called, settings %v", ext.Settings.PublicSettings), nil
}

func mockInitializationFunc(name string, version string, requiresSeqNoChange bool, enableCallback vmextension.EnableCallbackFunc) (*vmextension.InitializationInfo, error) {
	return &vmextension.InitializationInfo{
		Name:                "testExtension",
		Version:             "0.0.0.1",
		SupportsDisable:     true,
		RequiresSeqNoChange: requiresSeqNoChange,
		InstallExitCode:     52,
		OtherExitCode:       3,
		EnableCallback:      mockEnableCallbackFunc,
		DisableCallback:     nil,
		UpdateCallback:      nil,
	}, nil
}

func writeSettingsFile(settingsFileContent string, settingsDir string, seqNo uint) error {
	settingFileName := fmt.Sprintf("%d.settings", seqNo)
	settingsFilePath := path.Join(settingsDir, settingFileName)
	return ioutil.WriteFile(settingsFilePath, []byte(settingsFileContent), constants.FilePermissions_UserOnly_ReadWrite)
}

var handlerEnv = handlerenv.HandlerEnvironment{
	HeartbeatFile: heartbeatFile,
	ConfigFolder:  configFolder,
	DataFolder:    dataFolder,
	LogFolder:     logFolder,
	StatusFolder:  statusFolder,
}

type mockExiter struct{}

func (mockExiter) Exit(exitCode int) {
	panic(exitCode)
}

func initialize(t *testing.T) {
	// create all mock dirs
	err := os.MkdirAll(testdir, constants.FilePermissions_UserOnly_ReadWriteExecute)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = os.MkdirAll(handlerEnv.ConfigFolder, constants.FilePermissions_UserOnly_ReadWriteExecute)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = os.MkdirAll(handlerEnv.DataFolder, constants.FilePermissions_UserOnly_ReadWriteExecute)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = os.MkdirAll(handlerEnv.LogFolder, constants.FilePermissions_UserOnly_ReadWriteExecute)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = os.MkdirAll(handlerEnv.StatusFolder, constants.FilePermissions_UserOnly_ReadWriteExecute)
	if err != nil {
		t.Fatal(err.Error())
	}

	// place a legit settings file
	err = writeSettingsFile(settingsData, configFolder, 0)
	if err != nil {
		t.Fatal(err.Error())
	}

	enableCallbackCalled = false

	getInitializationInfoFuncToCall = mockInitializationFunc
	getVMExtensionFuncToCall = func(initInfo *vmextension.InitializationInfo) (ext *vmextension.VMExtension, _ error) {
		manager := mockVMExtensionEnvironmentManager{
			currentSeqNum:      nil,
			handlerEnvironment: &handlerEnv,
		}
		return vmextension.GetVMExtensionForTesting(initInfo, &manager)
	}

	exithelper.Exiter = &mockExiter{}
}

func testStatusFile(statusDir string, seqNo uint, requiredStatus status.StatusType) error {
	statusFilePath := path.Join(statusDir, fmt.Sprintf("%d.status", seqNo))
	statusFileBytes, err := ioutil.ReadFile(statusFilePath)
	if err != nil {
		return err
	}
	statusReport := make(status.StatusReport, 0)
	err = json.Unmarshal(statusFileBytes, &statusReport)
	if err != nil {
		return err
	}

	lenStatusReport := len(statusReport)
	if lenStatusReport != 1 {
		return fmt.Errorf("unexpected number of status items encountered, expected 1, got %d", lenStatusReport)
	}
	if statusReport[0].Status.Status != requiredStatus {
		return fmt.Errorf("unexpected status, expected %s, got %s", requiredStatus, statusReport[0].Status.Status)
	}

	return nil
}

func cleanupTest() {
	os.RemoveAll(testdir)
}

func TestMainFirstInstall(t *testing.T) {
	initialize(t)
	defer cleanupTest()
	os.Args = []string{"testprogram", "install"}
	err := getExtensionAndRun()
	assert.NoError(t, err)
}

func TestMainFirstEnable(t *testing.T) {
	initialize(t)
	defer cleanupTest()
	os.Args = []string{"testprogram", "enable"}
	err := getExtensionAndRun()
	assert.NoError(t, err)
	assert.True(t, enableCallbackCalled, "enable callback must be called")
	err = testStatusFile(handlerEnv.StatusFolder, 0, status.StatusSuccess)
	assert.NoError(t, err)
}

func TestHigherSequenceNumberIsExecuted(t *testing.T) {
	initialize(t)
	defer cleanupTest()
	os.Args = []string{"testprogram", "enable"}
	err := getExtensionAndRun()
	assert.NoError(t, err)
	assert.True(t, enableCallbackCalled, "enable callback must be called")
	err = testStatusFile(handlerEnv.StatusFolder, 0, status.StatusSuccess)
	assert.NoError(t, err)

	// write a new settings file
	writeSettingsFile(settingsData, handlerEnv.ConfigFolder, 1)
	enableCallbackCalled = false
	err = getExtensionAndRun()
	assert.NoError(t, err)
	assert.True(t, enableCallbackCalled, "enable callback must be called")
	err = testStatusFile(handlerEnv.StatusFolder, 1, status.StatusSuccess)
	assert.NoError(t, err)
}

func TestTransitioningStatus(t *testing.T) {
	initialize(t)
	defer cleanupTest()
	mockEnableCallbackFunc := func(ext *vmextension.VMExtension) (string, error) {
		return "testStatusFile", testStatusFile(ext.HandlerEnv.StatusFolder, ext.RequestedSequenceNumber, status.StatusTransitioning)
	}

	getInitializationInfoFuncToCall = func(name string, version string, requiresSeqNoChange bool, enableCallback vmextension.EnableCallbackFunc) (*vmextension.InitializationInfo, error) {
		return &vmextension.InitializationInfo{
			Name:                "testExtension",
			Version:             "0.0.0.1",
			SupportsDisable:     true,
			RequiresSeqNoChange: requiresSeqNoChange,
			InstallExitCode:     52,
			OtherExitCode:       3,
			EnableCallback:      mockEnableCallbackFunc,
			DisableCallback:     nil,
			UpdateCallback:      nil,
		}, nil
	}

	os.Args = []string{"testprogram", "enable"}
	err := getExtensionAndRun()
	assert.NoError(t, err)
	err = testStatusFile(handlerEnv.StatusFolder, 0, status.StatusSuccess)
	assert.NoError(t, err)
}

func TestSameSequenceNumberIsExecutedTwiceIfRequiresSeqNoChangeIsFalse(t *testing.T) {
	initialize(t)
	getInitializationInfoFuncToCall = func(name string, version string, requiresSeqNoChange bool, enableCallback vmextension.EnableCallbackFunc) (*vmextension.InitializationInfo, error) {
		return &vmextension.InitializationInfo{
			Name:                "testExtension",
			Version:             "0.0.0.1",
			SupportsDisable:     true,
			RequiresSeqNoChange: false,
			InstallExitCode:     52,
			OtherExitCode:       3,
			EnableCallback:      mockEnableCallbackFunc,
			DisableCallback:     nil,
			UpdateCallback:      nil,
		}, nil
	}
	var zero uint = 0
	getVMExtensionFuncToCall = func(initInfo *vmextension.InitializationInfo) (ext *vmextension.VMExtension, _ error) {
		manager := mockVMExtensionEnvironmentManager{
			currentSeqNum:      &zero,
			handlerEnvironment: &handlerEnv,
		}
		return vmextension.GetVMExtensionForTesting(initInfo, &manager)
	}
	defer cleanupTest()
	os.Args = []string{"testprogram", "enable"}
	defer func() {
		r := recover()
		if r != nil {
			t.Failed()
		}
	}()
	err := getExtensionAndRun()
	assert.NoError(t, err)
}

func TestSameSequenceNumberIsNotExecutedTwice(t *testing.T) {
	initialize(t)
	var zero uint = 0
	getVMExtensionFuncToCall = func(initInfo *vmextension.InitializationInfo) (ext *vmextension.VMExtension, _ error) {
		manager := mockVMExtensionEnvironmentManager{
			currentSeqNum:      &zero,
			handlerEnvironment: &handlerEnv,
		}
		return vmextension.GetVMExtensionForTesting(initInfo, &manager)
	}
	defer cleanupTest()
	os.Args = []string{"testprogram", "enable"}
	defer func() {
		r := recover()
		if r != nil {
			assert.EqualValues(t, 0, r, "ExitCode should be zero")
		}
	}()
	getExtensionAndRun()
}
