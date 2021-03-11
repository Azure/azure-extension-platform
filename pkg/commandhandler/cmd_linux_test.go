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
