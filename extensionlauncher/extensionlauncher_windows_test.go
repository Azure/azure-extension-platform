package extensionlauncher

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"
)

func TestRunExecutableAsIndependentProcess(t *testing.T) {
	testInit(t)
	defer testCleanup()
	fileName := "message.txt"
	filePath := path.Join(testDir, fileName)
	message := "sleep complete"
	startTime := time.Now()
	RunExecutableAsIndependentProcess("powershell.exe", fmt.Sprintf("-command \"Start-Sleep -s 5; '%s' | Out-File -FilePath '%s' -Encoding utf8\"", message, filePath), testDir, el)
	endTime := time.Now()
	duration := endTime.Sub(startTime)
	assert.Less(t, duration, time.Second, "the call to RunExecutableAsIndependentProcess should not block current execution")

	// wait for the process to complete and check if the file was written
	time.Sleep(time.Second * 7)
	fileContents, err := ioutil.ReadFile(filePath)
	assert.NoError(t, err, "the %s file should exist and be readable", fileName)
	assert.Contains(t, string(fileContents), message, "the contents of file %s should contain the message %s", fileName, message)
}

func TestEnvironmentVariablesAreProperlyPassed(t *testing.T){
	testInit(t)
	defer testCleanup()
	testEnvKey := "TestEnvKey"
	currentTime := time.Now().Format(time.RFC3339Nano)
	err := os.Setenv(testEnvKey, currentTime)
	assert.NoError(t, err, "should be able to set environment variable")
	fileName := "envVariables.txt"
	filePath := path.Join(testDir, fileName)
	RunExecutableAsIndependentProcess("powershell.exe", fmt.Sprintf("-command \"Get-ChildItem -path Env: | Out-File -FilePath '%s' -Encoding utf8\"", filePath), testDir, el)
	time.Sleep(2 *time.Second)
	fileContents, err := ioutil.ReadFile(filePath)
	assert.NoError(t, err, "the %s file should exist and be readable", fileName)
	assert.Contains(t, string(fileContents), testEnvKey, "the contents of file %s should contain the environment variable key %s", fileName, testEnvKey)
	assert.Contains(t, string(fileContents), currentTime, "the contents of file %s should contain the environment variable value %s", fileName, currentTime)
}