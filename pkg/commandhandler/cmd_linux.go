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
	return execCommon(workdir, stdout, stderr, func(c *exec.Cmd) error {
		return c.Run()
	}, cmd)
}

func execWaitWithParams(cmd, workdir string, stdout, stderr io.WriteCloser, params string) (int, error) {
	defer stdout.Close()
	defer stderr.Close()
	return execCommonWithParams(workdir, stdout, stderr, func(c *exec.Cmd) error {
		return c.Run()
	}, params, cmd)
}

func execDontWait(cmd, workdir string) (int, error) {
	// passing '&' as a trailing parameter to /bin/sh in addition (*exec.Command).Start() to will double fork and prevent zombie processes
	return execCommon(workdir, os.Stdout, os.Stderr, func(c *exec.Cmd) error {
		return c.Start()
	}, cmd, "&")
}

func execDontWaitWithParams(cmd, workdir string, params string) (int, error) {
	return execCommonWithParams(workdir, os.Stdout, os.Stderr, func(c *exec.Cmd) error {
		return c.Start()
	}, params, cmd, "&")
}

func execCommon(workdir string, stdout, stderr io.WriteCloser, execMethodToCall func(*exec.Cmd) error, args ...string) (int, error) {
	args = append([]string{"-c"}, args...)
	c := exec.Command("/bin/sh", args...)
	c.Dir = workdir
	c.Stdout = stdout
	c.Stderr = stderr

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

func execCommonWithParams(workdir string, stdout, stderr io.WriteCloser, execMethodToCall func(*exec.Cmd) error, params string, args ...string) (int, error) {
	//get parameter strings for environment variables
	var parameters map[string]interface{}
	json.Unmarshal([]byte(params), &parameters)

	//exports := []string{}
	//for name, value := range parameters {
	//	exports = append(exports, string("export "+name+"="+value.(string)+";"))
	//}
	//exports = append(exports, "-c")
	fmt.Println(parameters)
	args = append([]string{"-c"}, args...)
	c := exec.Command("/bin/sh", args...)
	c.Dir = workdir
	c.Stdout = stdout
	c.Stderr = stderr
	c.Env = os.Environ()

	for  name, value := range parameters {
		///Would this be cleaner with os.Setenv?
		//envVar := string("CustomAction_"+p.ParameterName+"="+p.ParameterValue)
		//c.Env = append(os.Environ(), envVar)
		envVar := string("CustomAction_"+name+"="+value.(string))
		fmt.Println(name, value.(string))
		c.Env = append(c.Env, envVar)
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