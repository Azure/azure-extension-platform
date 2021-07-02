package commandhandler

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"path"
	"strings"
	"testing"
)

const (
	lineReturnCharacter = "\n"
	commandNotExistReturnCode = 127
)


func TestEchoCommand2(t *testing.T) {
	defer cleanupTest()
	cmd := New()
	retCode, err := cmd.Execute("echo \"Hello 1\" \"Hello 2\"", workingDir, workingDir,true, extensionLogger)
	assert.NoError(t, err)
	assert.Equal(t, 0, retCode, "return code should be 0")
	fileBytes, err :=  ioutil.ReadFile(path.Join(workingDir, "stdout"))
	assert.NoError(t, err)
	stdoutResult := strings.TrimSuffix(strings.TrimSuffix(string(fileBytes), lineReturnCharacter), " ")
	assert.Equal(t, "Hello 1 Hello 2", stdoutResult)
}

func TestCommandWithEnvironmentVariable(t *testing.T){
	defer cleanupTest()
	cmd := New()
	var params map[string]interface{}
	json.Unmarshal([]byte(`{"BAR": "Hello World"}`), &params)
	retCode, err := cmd.ExecuteWithEnvVariables("echo $CustomAction_BAR", workingDir, workingDir, true, extensionLogger, &params)

	assert.NoError(t, err, "command execution should succeed")
	assert.Equal(t, 0, retCode, "return code should be 0")
	fileInfo, err := ioutil.ReadFile(path.Join(workingDir, "stdout"))
	assert.NoError(t, err, "stdout file should be read")
	assert.Contains(t, string(fileInfo), "Hello World", "stdout message should be as expected")
}

func TestCommandWithEnvironmentVariableQuotes(t *testing.T){
	defer cleanupTest()
	cmd := New()
	var params map[string]interface{}
	json.Unmarshal([]byte(`{"BAR": "\"Hello World\""}`), &params)
	retCode, err := cmd.ExecuteWithEnvVariables("echo $CustomAction_BAR", workingDir, workingDir, true, extensionLogger, &params)

	assert.NoError(t, err, "command execution should succeed")
	assert.Equal(t, 0, retCode, "return code should be 0")
	fileInfo, err := ioutil.ReadFile(path.Join(workingDir, "stdout"))
	assert.NoError(t, err, "stdout file should be read")
	assert.Contains(t, string(fileInfo), "\"Hello World\"", "stdout message should be as expected")
}

func TestCommandWithTwoEnvironmentVariables(t *testing.T){
	defer cleanupTest()
	cmd := New()
	var params map[string]interface{}
	json.Unmarshal([]byte(`{"FOO": "bizz", "BAR": "buzz"}`), &params)
	retCode, err := cmd.ExecuteWithEnvVariables("printenv", workingDir, workingDir, true, extensionLogger, &params)

	assert.NoError(t, err, "command execution should succeed")
	assert.Equal(t, 0, retCode, "return code should be 0")
	fileInfo, err := ioutil.ReadFile(path.Join(workingDir, "stdout"))
	assert.NoError(t, err, "stdout file should be read")
	assert.Contains(t, string(fileInfo), "CustomAction_FOO=bizz", "stdout message should be as expected")
	assert.Contains(t, string(fileInfo), "CustomAction_BAR=buzz", "stdout message should be as expected")
}