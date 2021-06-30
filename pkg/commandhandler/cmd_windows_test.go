package commandhandler

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"
)

const (
	lineReturnCharacter = "\r\n"
	commandNotExistReturnCode = 1
)

func TestQuotedCommandWorksCorrectly(t *testing.T){
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

func TestQuotedCommandWorksCorrectly2(t *testing.T){
	defer cleanupTest()
	cmd := New()
	retCode, err := cmd.Execute("echo \"Hello World\"", workingDir, workingDir, true, extensionLogger)
	assert.NoError(t, err, "command execution should succeed")
	assert.Equal(t, 0, retCode, "return code should be 0")
	fileInfo, err := ioutil.ReadFile(path.Join(workingDir, "stdout"))
	assert.NoError(t, err, "stdout file should be read")
	assert.Contains(t, string(fileInfo), "\"Hello World\"", "stdout message should be as expected")
}

func TestDoesntWaitForCompletion(t *testing.T){
	defer cleanupTest()
	cmd := New()
	startTime := time.Now()
	_, err := cmd.Execute("powershell.exe -command \"Start-Sleep -Seconds 5; 'sleep complete' | out-file testDoesntWait.txt\"", workingDir, workingDir, false, extensionLogger)
	assert.NoError(t, err, "should be able to execute command")
	endTime := time.Now()
	duration := endTime.Sub(startTime)
	assert.Less(t, duration, time.Second, "execute shouldn't block")
}

func TestCommandWithEnvironmentVariable(t *testing.T){
	defer cleanupTest()
	cmd := New()
	params := `{"FOO": "Hello World"}`
	retCode, err := cmd.ExecuteWithEnvVariables("echo %CustomAction_FOO% \n", workingDir, workingDir, true, extensionLogger, params)

	assert.NoError(t, err, "command execution should succeed")
	assert.Equal(t, 0, retCode, "return code should be 0")
	fileInfo, err := ioutil.ReadFile(path.Join(workingDir, "stdout"))
	assert.NoError(t, err, "stdout file should be read")
	assert.Contains(t, string(fileInfo), "Hello World", "stdout message should be as expected")
}

func TestCommandWithEnvironmentVariableQuotes(t *testing.T){
	defer cleanupTest()
	cmd := New()
	params := `{"FOO": "\"Hello World\""}`
	retCode, err := cmd.ExecuteWithEnvVariables("echo %CustomAction_FOO% \n", workingDir, workingDir, true, extensionLogger, params)

	assert.NoError(t, err, "command execution should succeed")
	assert.Equal(t, 0, retCode, "return code should be 0")
	fileInfo, err := ioutil.ReadFile(path.Join(workingDir, "stdout"))
	assert.NoError(t, err, "stdout file should be read")
	assert.Contains(t, string(fileInfo), "\"Hello World\"", "stdout message should be as expected")
}

func TestCommandWithEnvironmentVariable2(t *testing.T){
	defer cleanupTest()
	cmd := New()
	params := `{"FOO": "bizz", "BAR": "buzz"}`
	retCode, err := cmd.ExecuteWithEnvVariables("set", workingDir, workingDir, true, extensionLogger, params)

	assert.NoError(t, err, "command execution should succeed")
	assert.Equal(t, 0, retCode, "return code should be 0")
	fileInfo, err := ioutil.ReadFile(path.Join(workingDir, "stdout"))
	assert.NoError(t, err, "stdout file should be read")
	assert.Contains(t, string(fileInfo), "CustomAction_FOO=bizz", "stdout message should contain environment variable")
	assert.Contains(t, string(fileInfo), "CustomAction_BAR=buzz", "stdout message should contain environment variable")
}

func TestDoesntWaitForCompletionEnvironmentVariable(t *testing.T){
	defer cleanupTest()
	cmd := New()
	startTime := time.Now()
	params := `{"TEST_FILE": "testDoesntWait.txt"}`
	_, err := cmd.ExecuteWithEnvVariables("powershell.exe -command \"Start-Sleep -Seconds 5; 'sleep complete' | out-file %CustomAction_TEST_FILE%\"", workingDir, workingDir, false, extensionLogger, params)
	assert.NoError(t, err, "should be able to execute command")
	endTime := time.Now()
	duration := endTime.Sub(startTime)
	assert.Less(t, duration, time.Second, "execute shouldn't block")
}