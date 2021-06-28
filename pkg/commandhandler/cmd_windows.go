package commandhandler

import (
	"encoding/json"
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
	return execCommon(cmd, workdir, stdout, stderr, func(c *exec.Cmd) error {
		return c.Run()
	})
}

func execWaitWithParams(cmd, workdir string, stdout, stderr io.WriteCloser, params string) (int, error) {
	defer stdout.Close()
	defer stderr.Close()
	return execCommonWithParams(cmd ,workdir, stdout, stderr, func(c *exec.Cmd) error {
		return c.Run()
	}, params)
}

func execDontWait(cmd, workdir string) (int, error) {
	return execCommon(cmd, workdir, nil, nil, func(c *exec.Cmd) error {
		return c.Start()
	})
}

func execDontWaitWithParams(cmd, workdir string, params string) (int, error) {
	return execCommonWithParams(cmd, workdir, nil, nil, func(c *exec.Cmd) error {
		return c.Start()
	}, params)
}

func execCommon(cmd, workdir string, stdout, stderr io.WriteCloser, execFunctionToCall func(*exec.Cmd)(error)) (int, error) {
	c := exec.Command("cmd")
	c.Dir = workdir
	c.Stdout = stdout
	c.Stderr = stderr
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

func execCommonWithParams(cmd, workdir string, stdout, stderr io.WriteCloser, execFunctionToCall func(*exec.Cmd) error, params string) (int, error) {
	var parameters map[string]interface{}
	json.Unmarshal([]byte(params), &parameters)
	c := exec.Command("cmd")
	c.Dir = workdir
	c.Stdout = stdout
	c.Stderr = stderr
	c.Env = os.Environ()

	for name, value := range parameters {
		envVar := string("CustomAction_"+name+"="+value.(string))
		c.Env = append(c.Env, envVar)
	}

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
