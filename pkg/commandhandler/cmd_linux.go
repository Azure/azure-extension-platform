package commandhandler

import (
	"fmt"
	"github.com/pkg/errors"
	"io"
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

func execDontWait(cmd, workdir string) (int, error) {
	return execCommon(cmd, workdir, nil, nil, func(c *exec.Cmd) error {
		return c.Start()
	})
}

func execCommon(cmd, workdir string, stdout, stderr io.WriteCloser, execFunctionToCall func(*exec.Cmd)(error)) (int, error) {
	c := exec.Command("/bin/sh", "-c", cmd)
	c.Dir = workdir
	c.Stdout = stdout
	c.Stderr = stderr

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
