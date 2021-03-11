package extensionlauncher

import (
	"encoding/json"
	"github.com/Azure/azure-extension-platform/pkg/constants"
	"github.com/Azure/azure-extension-platform/pkg/handlerenv"
	"github.com/Azure/azure-extension-platform/pkg/logging"
	"github.com/Azure/azure-extension-platform/pkg/status"
	"github.com/Azure/azure-extension-platform/vmextension"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var testDir, statusFolder, logFolder string
var handlerEnv *handlerenv.HandlerEnvironment

var el *logging.ExtensionLogger

func testInit(t *testing.T) {
	testD, err := filepath.Abs("testdir")
	assert.NoError(t, err, "should be able to get absolute path")
	testDir = testD
	err = os.MkdirAll(testDir, constants.FilePermissions_UserOnly_ReadWriteExecute)
	assert.NoError(t, err, "should be able to create testdir")
	// create status and log folders
	statusFolder = path.Join(testDir, "status")
	err = os.MkdirAll(statusFolder, constants.FilePermissions_UserOnly_ReadWriteExecute)
	assert.NoError(t, err, "should be able to create status folder")
	logFolder = path.Join(testDir, "log")
	err = os.MkdirAll(logFolder, constants.FilePermissions_UserOnly_ReadWriteExecute)
	assert.NoError(t, err, "should be able to create log folder")
	handlerEnv = &handlerenv.HandlerEnvironment{
		StatusFolder: statusFolder,
		LogFolder:    logFolder,
	}
	el = logging.New(handlerEnv)
}

func testCleanup() {
	el.Close()
	// give it time to cleanup the stdout and stderr files
	time.Sleep(1*time.Second)
	err := os.RemoveAll(testDir)
	if err != nil {
		el.Warn("could not remove testdir %s", err.Error())
	}
}

func TestWriteTransitioningStatus(t *testing.T) {
	testInit(t)
	defer testCleanup()
	writeTransitioningStatusAndStartExtensionAsASeparateProcess("testExtension", "1.0.0.0", "cmd", "enable", handlerEnv, el)
	statusFile := path.Join(statusFolder, "0.status")
	_, err := os.Stat(statusFolder)
	assert.NoError(t, err, "status file should exist")
	statusFileBytes, err := ioutil.ReadFile(statusFile)
	assert.NoError(t, err, "should be able to read status file")
	statusReport := status.StatusReport{}
	err = json.Unmarshal(statusFileBytes, &statusReport)
	assert.NoError(t, err, "should be able to deserialize status file")
	assert.Equal(t, 1, len(statusReport), "there should be 1 status item in the status report")
	assert.Equal(t, status.StatusTransitioning, statusReport[0].Status.Status, "status should be transitioning")
	assert.True(t, vmextension.EnableOperation.ToPascalCaseName() == statusReport[0].Status.Operation, "operation should be enable")
}

func TestExistingStatusFileIsNotOverwritten(t *testing.T) {
	testInit(t)
	defer testCleanup()
	statusFileContentString := "I already exist"
	statusFile := path.Join(statusFolder, "0.status")
	err := ioutil.WriteFile(statusFile, []byte(statusFileContentString), constants.FilePermissions_UserOnly_ReadWrite)
	assert.NoError(t, err, "should be able to write mock status file")
	writeTransitioningStatusAndStartExtensionAsASeparateProcess("testExtension", "1.0.0.0", "cmd", "enable", handlerEnv, el)
	readStatusFileContentBytes, err := ioutil.ReadFile(statusFile)
	assert.NoError(t, err, "should be able to read status file")
	assert.Contains(t, string(readStatusFileContentBytes), statusFileContentString, "status file content should not be overwritten")

	// also test the logs
	dirInfo, err := ioutil.ReadDir(logFolder)
	assert.NoError(t, err, "should be able to read log folder")

	var logFileFound = false
	for _, dirContent := range dirInfo{
		if strings.Contains(dirContent.Name(), "log_") && !dirContent.IsDir(){
			logFileFound = true
			logFileContent, err := ioutil.ReadFile(path.Join(logFolder, dirContent.Name()))
			assert.NoError(t, err, "should be able to read log file")
			assert.Contains(t, string(logFileContent), "status file already exists, will not create new status file", "info message for skipping creation of new transitioning status file expected")
		}
	}
	assert.True(t, logFileFound, "log file should be found")
}
