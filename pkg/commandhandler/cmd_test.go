// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package commandhandler

import (
	"github.com/Azure/azure-extension-platform/pkg/logging"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"
)

var workingDir = path.Join(".", "testdir", "currentWorkingDir")

func cleanupTest() {
	os.RemoveAll(workingDir)
}

var extensionLogger = logging.New(nil)

func TestEchoCommand(t *testing.T) {
	defer cleanupTest()
	cmd := New()
	retCode, err := cmd.Execute("echo 1 2 3 4", workingDir, workingDir, true, extensionLogger)
	assert.NoError(t, err)
	assert.Equal(t, 0, retCode, "return code should be 0")
	fileBytes, err :=  ioutil.ReadFile(path.Join(workingDir, "stdout"))
	assert.NoError(t, err)
	stdoutResult := strings.TrimSuffix(strings.TrimSuffix(string(fileBytes), lineReturnCharacter), " ")
	assert.Equal(t, "1 2 3 4", stdoutResult)
}


func TestStderr(t *testing.T) {
	defer cleanupTest()
	cmd := New()
	retCode, err := cmd.Execute("echo 1 2 3 4 1>&2", workingDir, workingDir,true, extensionLogger)
	assert.NoError(t, err)
	assert.Equal(t, 0, retCode, "return code should be 0")
	fileBytes, err :=  ioutil.ReadFile(path.Join(workingDir, "stderr"))
	assert.NoError(t, err)
	stdoutResult := strings.TrimSuffix(strings.TrimSuffix(string(fileBytes), lineReturnCharacter), " ")
	assert.Equal(t, "1 2 3 4", stdoutResult)
}

func TestNonExistingCommand(t *testing.T) {
	defer cleanupTest()
	cmd := New()
	retcode, err := cmd.Execute("command_does_not_exist", workingDir, workingDir,true, extensionLogger)
	assert.Equal(t, commandNotExistReturnCode, retcode)
	assert.Error(t, err)
}


