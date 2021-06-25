package commandhandler

import (
	"fmt"
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
	params := "{\"FOO\": \"Hello World\"}"
	retCode, err := cmd.ExecuteWithEnvVariable("echo %CustomAction_FOO%", workingDir, workingDir, true, extensionLogger, params)

	assert.Contains(t, os.Environ(), "CustomAction_FOO=Hello World")
	assert.NoError(t, err, "command execution should succeed")
	assert.Equal(t, 0, retCode, "return code should be 0")
	fileInfo, err := ioutil.ReadFile(path.Join(workingDir, "stdout"))
	assert.NoError(t, err, "stdout file should be read")
	assert.Contains(t, string(fileInfo), "Hello World", "stdout message should be as expected")
}

func TestCommandWithCLParameters(t *testing.T){
	defer cleanupTest()
	cmd := New()
	params := "[{\"name\": \"FOO\",\"value\": \"Hello World\"}]"
	for _,i := range(os.Environ()) {
		fmt.Println(i)
	}
	retCode, err := cmd.ExecuteWithEnvVariable("echo %CustomAction_FOO%", workingDir, workingDir, true, extensionLogger, params)
	assert.Contains(t, os.Environ(), "CustomAction_FOO=\"Hello World\"")
	assert.NoError(t, err, "command execution should succeed")
	assert.Equal(t, 0, retCode, "return code should be 0")
	fileInfo, err := ioutil.ReadFile(path.Join(workingDir, "stdout"))
	assert.NoError(t, err, "stdout file should be read")
	assert.Contains(t, string(fileInfo), "\"Hello World\"", "stdout message should be as expected")
}
