package exithelper

import "os"

type IExitHelper interface {
	Exit(int)
}

type ExitHelper struct{}

func (*ExitHelper) Exit(exitCode int) {
	os.Exit(exitCode)
}

var Exiter IExitHelper = &ExitHelper{}
