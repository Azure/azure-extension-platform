package extensionevents

import (
	"golang.org/x/sys/unix"
)

func getThreadID() string {
	return string(unix.Gettid())
}
