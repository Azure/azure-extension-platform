package extensionevents

import "golang.org/x/sys/windows"

func getThreadID() string {
	return string(windows.GetCurrentThreadId())
}
