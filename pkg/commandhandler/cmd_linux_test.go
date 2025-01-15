// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package commandhandler

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	lineReturnCharacter       = "\n"
	commandNotExistReturnCode = 127
)

func TestEchoCommand2(t *testing.T) {
	defer cleanupTest()
	cmd := New()
	retCode, err := cmd.Execute("echo \"Hello 1\" \"Hello 2\"", workingDir, workingDir, true, extensionLogger)
	assert.NoError(t, err)
	assert.Equal(t, 0, retCode, "return code should be 0")
	fileBytes, err := ioutil.ReadFile(path.Join(workingDir, "stdout"))
	assert.NoError(t, err)
	stdoutResult := strings.TrimSuffix(strings.TrimSuffix(string(fileBytes), lineReturnCharacter), " ")
	assert.Equal(t, "Hello 1 Hello 2", stdoutResult)
}

func TestCommandWithEnvironmentVariable(t *testing.T) {
	defer cleanupTest()
	cmd := New()
	var params map[string]string
	json.Unmarshal([]byte(`{"BAR": "Hello World"}`), &params)
	retCode, err := cmd.ExecuteWithEnvVariables("echo $CustomAction_BAR", workingDir, workingDir, true, extensionLogger, &params)

	assert.NoError(t, err, "command execution should succeed")
	assert.Equal(t, 0, retCode, "return code should be 0")
	fileInfo, err := ioutil.ReadFile(path.Join(workingDir, "stdout"))
	assert.NoError(t, err, "stdout file should be read")
	assert.Contains(t, string(fileInfo), "Hello World", "stdout message should be as expected")
}

func TestCommandWithEnvironmentVariableQuotes(t *testing.T) {
	defer cleanupTest()
	cmd := New()
	var params map[string]string
	json.Unmarshal([]byte(`{"BAR": "\"Hello World\""}`), &params)
	retCode, err := cmd.ExecuteWithEnvVariables("echo $CustomAction_BAR", workingDir, workingDir, true, extensionLogger, &params)

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
	retCode, err := cmd.ExecuteWithEnvVariables("printenv", workingDir, workingDir, true, extensionLogger, &params)

	assert.NoError(t, err, "command execution should succeed")
	assert.Equal(t, 0, retCode, "return code should be 0")
	fileInfo, err := ioutil.ReadFile(path.Join(workingDir, "stdout"))
	assert.NoError(t, err, "stdout file should be read")
	assert.Contains(t, string(fileInfo), "CustomAction_FOO=bizz", "stdout message should be as expected")
	assert.Contains(t, string(fileInfo), "CustomAction_BAR=buzz", "stdout message should be as expected")
}

func TestCommandWithEnvironmentVariableNil(t *testing.T) {
	defer cleanupTest()
	cmd := New()

	retCode, err := cmd.ExecuteWithEnvVariables("echo $CustomAction_FOO \n", workingDir, workingDir, true, extensionLogger, nil)

	assert.NoError(t, err, "command execution should succeed")
	assert.Equal(t, 0, retCode, "return code should be 0")
	fileInfo, err := ioutil.ReadFile(path.Join(workingDir, "stdout"))
	assert.NoError(t, err, "stdout file should be read")
	assert.Contains(t, string(fileInfo), "", "stdout message should be as expected")
}

func TestCommandWithEnvironmentVariableEmpty(t *testing.T) {
	defer cleanupTest()
	cmd := New()
	var params map[string]string
	json.Unmarshal([]byte(`{}`), &params)
	retCode, err := cmd.ExecuteWithEnvVariables("echo $CustomAction_FOO \n", workingDir, workingDir, true, extensionLogger, &params)

	assert.NoError(t, err, "command execution should succeed")
	assert.Equal(t, 0, retCode, "return code should be 0")
	fileInfo, err := ioutil.ReadFile(path.Join(workingDir, "stdout"))
	assert.NoError(t, err, "stdout file should be read")
	assert.Contains(t, string(fileInfo), "", "stdout message should be as expected")
}

func TestNonExistingCommand(t *testing.T) {
	defer cleanupTest()
	cmd := New()
	retcode, err := cmd.Execute("command_does_not_exist", workingDir, workingDir, true, extensionLogger)
	assert.Equal(t, commandNotExistReturnCode, retcode)
	assert.Error(t, err, "command execution should fail")
	assert.Contains(t, err.Error(), "command_does_not_exist: not found", "error returned by cmd.Execute should include stderr")

	fileInfo, err := os.ReadFile(path.Join(workingDir, "stderr"))
	assert.NoError(t, err, "stderr file should be read")
	assert.Contains(t, string(fileInfo), "command_does_not_exist: not found", "stderr message should be as expected")
}
