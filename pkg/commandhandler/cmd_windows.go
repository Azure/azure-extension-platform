package commandhandler

import (
	"fmt"
	"github.com/pkg/errors"
	"io"
	"os"
	"os/exec"
	"syscall"
)

func execWait(cmd, workdir string, stdout, stderr io.WriteCloser) (int, error) {
	defer stdout.Close()
	defer stderr.Close()
	return execCommonWithEnvVariables(cmd ,workdir, stdout, stderr, func(c *exec.Cmd) error {
		return c.Run()
	}, nil)
}

func execWaitWithEnvVariables(cmd, workdir string, stdout, stderr io.WriteCloser, params *map[string]interface{}) (int, error) {
	defer stdout.Close()
	defer stderr.Close()
	return execCommonWithEnvVariables(cmd ,workdir, stdout, stderr, func(c *exec.Cmd) error {
		return c.Run()
	}, params)
}

func execDontWait(cmd, workdir string) (int, error) {
	return execCommonWithEnvVariables(cmd, workdir, nil, nil, func(c *exec.Cmd) error {
		return c.Start()
	}, nil)
}

func execDontWaitWithEnvVariables(cmd, workdir string, params *map[string]interface{}) (int, error) {
	return execCommonWithEnvVariables(cmd, workdir, nil, nil, func(c *exec.Cmd) error {
		return c.Start()
	}, params)
}


func execCommonWithEnvVariables(cmd, workdir string, stdout, stderr io.WriteCloser, execFunctionToCall func(*exec.Cmd) error, params *map[string]interface{}) (int, error) {

	c := exec.Command("cmd")
	c.Dir = workdir
	c.Stdout = stdout
	c.Stderr = stderr
	c.Env = os.Environ()

	//if params!= nil && len(*params) > 0 {
	//
	//	for name, value := range *params {
	//		envVar := string("CustomAction_"+name+"="+value.(string))
	//		c.Env = append(c.Env, envVar)
	//	}
	//}

	addEnvVariables(params, c)

	// don't pass the args in exec.Command because
	// On Windows, processes receive the whole command line as a single string
	// and do their own parsing. Command combines and quotes Args into a command
	// line string with an algorithm compatible with applications using
	// CommandLineToArgvW (which is the most common way). Notable exceptions are
	// msiexec.exe and cmd.exe (and thus, all batch files), which have a different
	// unquoting algorithm. In these or other similar cases, you can do the
	// quoting yourself and provide the full command line in SysProcAttr.CmdLine,
	// leaving Args empty.
	c.SysProcAttr = &syscall.SysProcAttr{CmdLine:"/C " + cmd}

	err := execFunctionToCall(c)
	exitErr, ok := err.(*exec.ExitError)
	if ok {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			code := status.ExitStatus()
			return code, fmt.Errorf("command terminated with exit status=%d", code)
		}
	}
	if err != nil {
		return 1, errors.Wrapf(err, "failed to execute command")
	}
	return 0, nil
}
