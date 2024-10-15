// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package commandhandler

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	lineReturnCharacter       = "\r\n"
	commandNotExistReturnCode = 1
)

func TestQuotedCommandWorksCorrectly(t *testing.T) {
	defer cleanupTest()
	cmd := New()
	retCode, err := cmd.Execute("dir \"C:\\Program Files\"", workingDir, workingDir, true, extensionLogger)
	assert.NoError(t, err, "command execution should succeed")
	assert.Zero(t, retCode, "return code should be 0")
	fileInfo, err := os.Stat(path.Join(workingDir, "stderr"))
	assert.NoError(t, err, "os.Stat should succeed")
	assert.Zero(t, fileInfo.Size(), "stderr file size should be 0")

	fileInfo, err = os.Stat(path.Join(workingDir, "stdout"))
	assert.NoError(t, err, "os.Stat should succeed")
	assert.NotZero(t, fileInfo.Size(), "stdout file size should not be 0")
}

func TestQuotedCommandWorksCorrectly2(t *testing.T) {
	defer cleanupTest()
	cmd := New()
	retCode, err := cmd.Execute("echo \"Hello World\"", workingDir, workingDir, true, extensionLogger)
	assert.NoError(t, err, "command execution should succeed")
	assert.Equal(t, 0, retCode, "return code should be 0")
	fileInfo, err := ioutil.ReadFile(path.Join(workingDir, "stdout"))
	assert.NoError(t, err, "stdout file should be read")
	assert.Contains(t, string(fileInfo), "\"Hello World\"", "stdout message should be as expected")
}

func TestDoesntWaitForCompletion(t *testing.T) {
	defer cleanupTest()
	cmd := New()
	startTime := time.Now()
	_, err := cmd.Execute("powershell.exe -command \"Start-Sleep -Seconds 5; 'sleep complete' | out-file testDoesntWait.txt\"", workingDir, workingDir, false, extensionLogger)
	assert.NoError(t, err, "should be able to execute command")
	endTime := time.Now()
	duration := endTime.Sub(startTime)
	assert.Less(t, duration, time.Second, "execute shouldn't block")
}

func TestCommandWithEnvironmentVariable(t *testing.T) {
	defer cleanupTest()
	cmd := New()
	var params map[string]string
	json.Unmarshal([]byte(`{"FOO": "Hello World"}`), &params)
	retCode, err := cmd.ExecuteWithEnvVariables("echo %CustomAction_FOO% \n", workingDir, workingDir, true, extensionLogger, &params)

	assert.NoError(t, err, "command execution should succeed")
	assert.Equal(t, 0, retCode, "return code should be 0")
	fileInfo, err := ioutil.ReadFile(path.Join(workingDir, "stdout"))
	assert.NoError(t, err, "stdout file should be read")
	assert.Contains(t, string(fileInfo), "Hello World", "stdout message should be as expected")
}

func TestCommandWithEnvironmentVariableEmpty(t *testing.T) {
	defer cleanupTest()
	cmd := New()
	var params map[string]string
	json.Unmarshal([]byte(`{}`), &params)
	retCode, err := cmd.ExecuteWithEnvVariables("echo %CustomAction_FOO% \n", workingDir, workingDir, true, extensionLogger, &params)

	assert.NoError(t, err, "command execution should succeed")
	assert.Equal(t, 0, retCode, "return code should be 0")
	fileInfo, err := ioutil.ReadFile(path.Join(workingDir, "stdout"))
	assert.NoError(t, err, "stdout file should be read")
	assert.Contains(t, string(fileInfo), "", "stdout message should be as expected")
}

func TestCommandWithEnvironmentVariableNil(t *testing.T) {
	defer cleanupTest()
	cmd := New()

	retCode, err := cmd.ExecuteWithEnvVariables("echo %CustomAction_FOO% \n", workingDir, workingDir, true, extensionLogger, nil)

	assert.NoError(t, err, "command execution should succeed")
	assert.Equal(t, 0, retCode, "return code should be 0")
	fileInfo, err := ioutil.ReadFile(path.Join(workingDir, "stdout"))
	assert.NoError(t, err, "stdout file should be read")
	assert.Contains(t, string(fileInfo), "%CustomAction_FOO%", "stdout message should be as expected")
}

func TestCommandWithEnvironmentVariableQuotes(t *testing.T) {
	defer cleanupTest()
	cmd := New()

	var params map[string]string
	json.Unmarshal([]byte(`{"FOO": "\"Hello World\""}`), &params)
	retCode, err := cmd.ExecuteWithEnvVariables("echo %CustomAction_FOO% \n", workingDir, workingDir, true, extensionLogger, &params)

	assert.NoError(t, err, "command execution should succeed")
	assert.Equal(t, 0, retCode, "return code should be 0")
	fileInfo, err := ioutil.ReadFile(path.Join(workingDir, "stdout"))
	assert.NoError(t, err, "stdout file should be read")
	assert.Contains(t, string(fileInfo), "\"Hello World\"", "stdout message should be as expected")
}

func TestCommandWithTwoEnvironmentVariables(t *testing.T) {
	defer cleanupTest()
	cmd := New()

	var params map[string]string
	json.Unmarshal([]byte(`{"FOO": "bizz", "BAR": "buzz"}`), &params)
	retCode, err := cmd.ExecuteWithEnvVariables("set", workingDir, workingDir, true, extensionLogger, &params)

	assert.NoError(t, err, "command execution should succeed")
	assert.Equal(t, 0, retCode, "return code should be 0")
	fileInfo, err := ioutil.ReadFile(path.Join(workingDir, "stdout"))
	assert.NoError(t, err, "stdout file should be read")
	assert.Contains(t, string(fileInfo), "CustomAction_FOO=bizz", "stdout message should contain environment variable")
	assert.Contains(t, string(fileInfo), "CustomAction_BAR=buzz", "stdout message should contain environment variable")
}

func TestDoesntWaitForCompletionEnvironmentVariable(t *testing.T) {
	defer cleanupTest()
	cmd := New()
	startTime := time.Now()

	var params map[string]string
	json.Unmarshal([]byte(`{"TEST_FILE": "testDoesntWait.txt"}`), &params)

	_, err := cmd.ExecuteWithEnvVariables("powershell.exe -command \"Start-Sleep -Seconds 5; 'sleep complete' | out-file %CustomAction_TEST_FILE%\"", workingDir, workingDir, false, extensionLogger, &params)
	assert.NoError(t, err, "should be able to execute command")
	endTime := time.Now()
	duration := endTime.Sub(startTime)
	assert.Less(t, duration, time.Second, "execute shouldn't block")
}

func TestNonExistingCommand(t *testing.T) {
	defer cleanupTest()
	cmd := New()
	retcode, err := cmd.Execute("command_does_not_exist", workingDir, workingDir, true, extensionLogger)
	assert.Equal(t, commandNotExistReturnCode, retcode)
	assert.Error(t, err, "command execution should fail")
	assert.Contains(t, err.Error(), "is not recognized as an internal or external command", "error returned by cmd.Execute should include stderr")

	fileInfo, err := os.ReadFile(path.Join(workingDir, "stderr"))
	assert.NoError(t, err, "stderr file should be read")
	assert.Contains(t, string(fileInfo), "is not recognized as an internal or external command", "stderr message should be as expected")
}
