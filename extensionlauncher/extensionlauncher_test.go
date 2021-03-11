package extensionlauncher

import (
	"encoding/json"
	"fmt"
	"github.com/Azure/azure-extension-platform/pkg/constants"
	"github.com/Azure/azure-extension-platform/pkg/handlerenv"
	"github.com/Azure/azure-extension-platform/pkg/logging"
	"github.com/Azure/azure-extension-platform/pkg/status"
	"github.com/Azure/azure-extension-platform/vmextension"
	"github.com/stretchr/testify/assert"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

var testDir string

var statusFolder string
var logFolder string
var el = logging.New(nil)

// implementation of logging.IExtensionLogger
type mockExtensionLogger struct {
	InfoStream    strings.Builder
	WarningStream strings.Builder
	ErrorStream   strings.Builder
}

func (mel *mockExtensionLogger) Error(format string, v ...interface{}) {
	mel.ErrorStream.WriteString(fmt.Sprintf(format, v ...))
	mel.ErrorStream.WriteString(constants.NewLineCharacter)
	fmt.Printf(format, v ...)
}
func (mel *mockExtensionLogger) Warn(format string, v ...interface{}) {
	mel.WarningStream.WriteString(fmt.Sprintf(format, v ...))
	mel.WarningStream.WriteString(constants.NewLineCharacter)
	fmt.Printf(format, v ...)
}
func (mel *mockExtensionLogger) Info(format string, v ...interface{}) {
	mel.InfoStream.WriteString(fmt.Sprintf(format, v ...))
	mel.InfoStream.WriteString(constants.NewLineCharacter)
	fmt.Printf(format, v ...)

}
func (mel *mockExtensionLogger) ErrorFromStream(prefix string, streamReader io.Reader) {
	bytes := make([]byte, 0, 1024)
	bytesRead, _ := streamReader.Read(bytes)
	mel.ErrorStream.Write(bytes[:bytesRead])
	mel.ErrorStream.WriteString(constants.NewLineCharacter)
	fmt.Print(string(bytes[:bytesRead]))
}
func (mel *mockExtensionLogger) WarnFromStream(prefix string, streamReader io.Reader) {
	bytes := make([]byte, 0, 1024)
	bytesRead, _ := streamReader.Read(bytes)
	mel.WarningStream.Write(bytes[:bytesRead])
	mel.WarningStream.WriteString(constants.NewLineCharacter)
	fmt.Print(string(bytes[:bytesRead]))
}
func (mel *mockExtensionLogger) InfoFromStream(prefix string, streamReader io.Reader) {
	bytes := make([]byte, 0, 1024)
	bytesRead, _ := streamReader.Read(bytes)
	mel.InfoStream.Write(bytes[:bytesRead])
	mel.InfoStream.WriteString(constants.NewLineCharacter)
	fmt.Print(string(bytes[:bytesRead]))
}

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
}

func testCleanup() {
	os.RemoveAll(testDir)
}

func TestWriteTransitioningStatus(t *testing.T) {
	testInit(t)
	defer testCleanup()
	oldGetHandlerEnvFuncToUse := getHandlerEnvFuncToUse
	defer func() { getHandlerEnvFuncToUse = oldGetHandlerEnvFuncToUse }()

	// create status and log folders

	getHandlerEnvFuncToUse = func(name, version string) (he *handlerenv.HandlerEnvironment, _ error) {
		return &handlerenv.HandlerEnvironment{
			StatusFolder: statusFolder,
			LogFolder:    logFolder,
		}, nil
	}

	writeTransitioningStatusAndStartExtensionAsASeparateProcess("testExtension", "1.0.0.0", "testExtensionExe", "enable")
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
