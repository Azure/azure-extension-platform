// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package commandhandler

import (
	"github.com/Azure/azure-extension-platform/pkg/constants"
	"github.com/Azure/azure-extension-platform/pkg/logging"
	"github.com/pkg/errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

type ICommandHandler interface {
	Execute(command string, workingDir, logDir string, waitForCompletion bool, el *logging.ExtensionLogger) (returnCode int, err error)
}
type ICommandHandlerWithEnvVariables interface {
	ExecuteWithEnvVariables(command string, workingDir, logDir string, waitForCompletion bool, el *logging.ExtensionLogger, params *map[string]string) (returnCode int, err error)
}

func (commandHandler *CommandHandler) ExecuteWithEnvVariables(command string, workingDir, logDir string, waitForCompletion bool, el *logging.ExtensionLogger, params *map[string]string) (returnCode int, err error) {
	return execCmdInDirWithAction(command, workingDir, logDir, waitForCompletion, el, params)
}

type CommandHandler struct {
}

func New() *CommandHandler {
	return &CommandHandler{}
}

func (commandHandler *CommandHandler) Execute(command string, workingDir, logDir string, waitForCompletion bool, el *logging.ExtensionLogger) (returnCode int, err error) {
	return execCmdInDirWithAction(command, workingDir, logDir, waitForCompletion, el, nil)
}

var execWaitFunctionToCall func(cmd string, workingDir string, stdout, stderr io.WriteCloser) (int, error) = execWait
var execDontWaitFunctionToCall func(cmd string, workingDir string) (int, error) = execDontWait

var execWaitFunctionWithParams = execWaitWithEnvVariables
var execDontWaitFunctionWithParams = execDontWaitWithEnvVariables

func execCmdInDirWithAction(cmd, workingDir, logDir string, waitForCompletion bool, el *logging.ExtensionLogger, params *map[string]string) (int, error) {
	var exitCode int
	var execErr error
	err := os.MkdirAll(workingDir, constants.FilePermissions_UserOnly_ReadWriteExecute)
	if err != nil {
		return -1, errors.Wrapf(err, "error while creating/accessing directory %s", workingDir)
	}

	if waitForCompletion {
		outFileName, errFileName := logPaths(logDir)
		outF, err := os.OpenFile(outFileName, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, constants.FilePermissions_UserOnly_ReadWrite)
		if err != nil {
			return -1, errors.Wrapf(err, "failed to open stdout file")
		}

		errF, err := os.OpenFile(errFileName, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, constants.FilePermissions_UserOnly_ReadWrite)
		if err != nil {
			return -1, errors.Wrapf(err, "failed to open stderr file")
		}

		exitCode, execErr = execWaitFunctionWithParams(cmd, workingDir, outF, errF, params)

		// add the output of the command to the log file
		el.Info("command: %s", cmd)
		stdOutFile, err2 := os.OpenFile(outFileName, os.O_RDONLY, constants.FilePermissions_UserOnly_ReadWrite)
		if err2 == nil {
			el.InfoFromStream("stdout:", stdOutFile)
			stdOutFile.Close()
		}

		stdErrFile, err3 := os.OpenFile(errFileName, os.O_RDONLY, constants.FilePermissions_UserOnly_ReadWrite)
		if err3 == nil {
			el.InfoFromStream("stderr:", stdErrFile)
			stdErrFile.Close()
		}

	} else {

		exitCode, execErr = execDontWaitFunctionWithParams(cmd, workingDir, params)

	}

	return exitCode, execErr
}

// logPaths returns stdout and stderr file paths for the specified output
// directory. It does not create the files.
func logPaths(dir string) (stdout string, stderr string) {
	stdout = filepath.Join(dir, "stdout")
	stderr = filepath.Join(dir, "stderr")
	return
}
func addEnvVariables(params *map[string]string, command *exec.Cmd) {
	if params != nil && len(*params) > 0 {
		for name, value := range *params {
			envVar := string("CustomAction_" + name + "=" + value)
			(command).Env = append((command).Env, envVar)
		}
	}
}
