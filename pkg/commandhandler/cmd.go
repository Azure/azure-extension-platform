package commandhandler

import (
	"github.com/Azure/azure-extension-platform/pkg/constants"
	"github.com/Azure/azure-extension-platform/pkg/logging"
	"github.com/pkg/errors"
	"io"
	"os"
	"path/filepath"
)

type ICommandHandler interface {
	Execute(command string, workingDir string, waitForCompletion bool, el logging.IExtensionLogger) (returnCode int, err error)
}

type CommandHandler struct {
}

func New() *CommandHandler {
	return &CommandHandler{}
}

func (commandHandler *CommandHandler) Execute(command string, workingDir string, waitForCompletion bool, el logging.IExtensionLogger) (returnCode int, err error) {
	return execCmdInDir(command, workingDir, waitForCompletion, el)
}

type internalExecFunction func (cmd string, workingDir string, stdout, stderr io.WriteCloser) (int, error)

var execWaitFunctionToCall internalExecFunction = execWait
var execDontWaitFunctionToCall internalExecFunction = execDontWait

// execCmdInDir executes the given command in given directory and saves output
// to ./stdout and ./stderr files (truncates files if exists, creates them if not
// with 0600/-rw------- permissions).
//
// Ideally, we execute commands only once per sequence number in custom-script-extension,
// and save their output under /var/lib/waagent/<dir>/download/<seqnum>/*.
func execCmdInDir(cmd, workingDir string, waitForCompletion bool, el logging.IExtensionLogger) (int, error) {
	err := os.MkdirAll(workingDir, constants.FilePermissions_UserOnly_ReadWriteExecute)
	if err != nil {
		return -1, errors.Wrapf(err, "error while creating/accessing directory %s", workingDir)
	}
	outFn, errFn := logPaths(workingDir)

	outF, err := os.OpenFile(outFn, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, constants.FilePermissions_UserOnly_ReadWrite)
	if err != nil {
		return -1, errors.Wrapf(err, "failed to open stdout file")
	}
	errF, err := os.OpenFile(errFn, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, constants.FilePermissions_UserOnly_ReadWrite)
	if err != nil {
		return -1, errors.Wrapf(err, "failed to open stderr file")
	}

	var exitCode int
	var execErr error

	if waitForCompletion {
		exitCode, execErr = execWaitFunctionToCall(cmd, workingDir, outF, errF)
	} else {
		exitCode, execErr = execDontWaitFunctionToCall(cmd, workingDir, outF, errF)
	}

	// add the output of the command to the log file
	el.Info("command: %s", cmd)
	stdOutFile, err2 := os.OpenFile(outFn, os.O_RDONLY, constants.FilePermissions_UserOnly_ReadWrite)
	if err2 == nil{
		el.InfoFromStream("stdout:", stdOutFile)
		stdOutFile.Close()
	}
	stdErrFile, err3 := os.OpenFile(errFn, os.O_RDONLY, constants.FilePermissions_UserOnly_ReadWrite)
	if err3 == nil {
		el.InfoFromStream("stderr:", stdErrFile)
		stdErrFile.Close()
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
