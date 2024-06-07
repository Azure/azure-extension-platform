// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package extensionevents

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/Azure/azure-extension-platform/pkg/constants"
	"github.com/Azure/azure-extension-platform/pkg/handlerenv"
	"github.com/Azure/azure-extension-platform/pkg/logging"
	"github.com/stretchr/testify/require"
)

var eventstestdir string

func TestMain(m *testing.M) {
	testdir, err := ioutil.TempDir("", "eventtest")
	if err != nil {
		return
	}

	err = os.MkdirAll(testdir, constants.FilePermissions_UserOnly_ReadWriteExecute)
	if err != nil {
		return
	}

	eventstestdir = testdir
	exitVal := m.Run()
	os.RemoveAll(eventstestdir)

	os.Exit(exitVal)
}

func Test_cantWriteFile(t *testing.T) {
	el := logging.New(nil)
	he := getHandlerEnvironment(eventstestdir)
	eem := New(el, he)
	os.RemoveAll(eventstestdir)
	defer os.MkdirAll(eventstestdir, constants.FilePermissions_UserOnly_ReadWriteExecute)

	// Verify nothing blows up
	eem.LogCriticalEvent("blah", "blah")
}

func Test_noEventsFolder(t *testing.T) {
	el := logging.New(nil)
	he := getHandlerEnvironment("")
	eem := New(el, he)

	// Write an event and verify no file is generated
	eem.LogCriticalEvent("blah", "blah")
	dir, _ := ioutil.ReadDir(eventstestdir)
	require.Equal(t, 0, len(dir))
}

func Test_logAllTypes(t *testing.T) {
	el := logging.New(nil)
	he := getHandlerEnvironment(eventstestdir)
	eem := New(el, he)
	eem.prefix = "(chipmunk) "
	defer clearEventTestDir()

	duration, _ := time.ParseDuration("100ms")
	eem.LogCriticalEvent("critical", "critical message")
	time.Sleep(duration)
	eem.LogErrorEvent("error", "error message")
	time.Sleep(duration)
	eem.LogInformationalEvent("informational", "informational message")
	time.Sleep(duration)
	eem.LogVerboseEvent("verbose", "verbose message")
	time.Sleep(duration)
	eem.LogWarningEvent("warning", "warning message")

	dir, _ := ioutil.ReadDir(eventstestdir)
	require.Equal(t, 5, len(dir))

	verifyEventFile(t, dir[0].Name(), "Critical", "(chipmunk) critical message")
	verifyEventFile(t, dir[1].Name(), "Error", "(chipmunk) error message")
	verifyEventFile(t, dir[2].Name(), "Informational", "(chipmunk) informational message")
	verifyEventFile(t, dir[3].Name(), "Verbose", "(chipmunk) verbose message")
	verifyEventFile(t, dir[4].Name(), "Warning", "(chipmunk) warning message")
}

func Test_NoTaskName(t *testing.T) {
	el := logging.New(nil)
	he := getHandlerEnvironment(eventstestdir)
	eem := New(el, he)
	defer clearEventTestDir()

	eem.LogCriticalEvent("", "critical message")

	dir, _ := ioutil.ReadDir(eventstestdir)
	require.Equal(t, 1, len(dir))

	verifyEventFile(t, dir[0].Name(), "Critical", "critical message")
}

func verifyEventFile(t *testing.T, fileName string, expectedLevel string, expectedMessage string) {
	require.Equal(t, ".json", filepath.Ext(fileName))
	openedFile, err := os.Open(path.Join(eventstestdir, fileName))
	require.NoError(t, err)
	defer openedFile.Close()

	b, err := ioutil.ReadAll(openedFile)
	require.NoError(t, err)
	var ee extensionEvent
	err = json.Unmarshal(b, &ee)
	require.NoError(t, err)

	require.Equal(t, expectedLevel, ee.EventLevel)
	require.Equal(t, expectedMessage, ee.Message)
}

func clearEventTestDir() {
	dir, _ := ioutil.ReadDir(eventstestdir)
	for _, d := range dir {
		os.RemoveAll(path.Join([]string{eventstestdir, d.Name()}...))
	}
}

func getHandlerEnvironment(eventsFolder string) *handlerenv.HandlerEnvironment {
	return &handlerenv.HandlerEnvironment{
		HeartbeatFile: "",
		StatusFolder:  "",
		ConfigFolder:  "",
		LogFolder:     "",
		DataFolder:    "",
		EventsFolder:  eventsFolder,
	}
}
