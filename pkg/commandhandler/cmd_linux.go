package commandhandler

import (
	"fmt"
	"github.com/pkg/errors"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"syscall"
)

func execWait(cmd, workdir string, stdout, stderr io.WriteCloser) (int, error) {
	defer stdout.Close()
	defer stderr.Close()
	return execCommonWithEnvVariables(workdir, stdout, stderr, func(c *exec.Cmd) error {
		return c.Run()
	}, "", cmd)
}

func execWaitWithEnvVariables(cmd, workdir string, stdout, stderr io.WriteCloser, params string) (int, error) {
	defer stdout.Close()
	defer stderr.Close()
	return execCommonWithEnvVariables(workdir, stdout, stderr, func(c *exec.Cmd) error {
		return c.Run()
	}, params, cmd)
}

func execDontWait(cmd, workdir string) (int, error) {
	// passing '&' as a trailing parameter to /bin/sh in addition (*exec.Command).Start() to will double fork and prevent zombie processes
	return execCommonWithEnvVariables(workdir, os.Stdout, os.Stderr, func(c *exec.Cmd) error {
		return c.Start()
	}, "", cmd, "&")
}

func execDontWaitWithEnvVariables(cmd, workdir string, params string) (int, error) {
	return execCommonWithEnvVariables(workdir, os.Stdout, os.Stderr, func(c *exec.Cmd) error {
		return c.Start()
	}, params, cmd, "&")
}

func execCommonWithEnvVariables(workdir string, stdout, stderr io.WriteCloser, execMethodToCall func(*exec.Cmd) error, params string, args ...string) (int, error) {

	args = append([]string{"-c"}, args...)
	c := exec.Command("/bin/sh", args...)
	c.Dir = workdir
	c.Stdout = stdout
	c.Stderr = stderr
	c.Env = os.Environ()

	if len(params) > 0 {
		var parameters map[string]interface{}
		err := json.Unmarshal([]byte(params), &parameters)
		if err != nil {
			return 1, errors.Wrapf(err, "failed to deserialize parameters")
		}
		for name, value := range parameters {
			envVar := string("CustomAction_"+name+"="+value.(string))
			c.Env = append(c.Env, envVar)
		}
	}

	err := execMethodToCall(c)
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