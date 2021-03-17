package extensionlauncher

import (
	"fmt"
	"github.com/stretchr/testify/assert"
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
	runExecutableAsIndependentProcess("bash", fmt.Sprintf("-c \"sleep 5s; echo '%s' > '%s'\"", message, filePath), testDir, testDir, el)
	endTime := time.Now()
	duration := endTime.Sub(startTime)
	assert.Less(t, duration, time.Second, "the call to runExecutableAsIndependentProcess should not block current execution")

	// wait for the process to complete and check if the file was written
	testContentsOfFile(t, filePath, message)
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
	runExecutableAsIndependentProcess("bash", fmt.Sprintf("-c \"printenv > '%s'\"", filePath), testDir, testDir, el)
	testContentsOfFile(t, filePath, testEnvKey)
	testContentsOfFile(t, filePath, currentTime)
}

