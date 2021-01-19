package logging

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/Azure/azure-extension-platform/pkg/constants"
	"github.com/Azure/azure-extension-platform/pkg/handlerenv"
	"github.com/stretchr/testify/require"
)

var logtestdir string

func TestMain(m *testing.M) {
	testdir, err := ioutil.TempDir("", "logtest")
	if err != nil {
		return
	}

	err = os.MkdirAll(testdir, constants.FilePermissions_UserOnly_ReadWriteExecute)
	if err != nil {
		return
	}

	logtestdir = testdir
	exitVal := m.Run()
	os.RemoveAll(logtestdir)

	os.Exit(exitVal)
}

func Test_noHandlerEnvironment(t *testing.T) {
	el := New(nil)
	el.Info("blah")
	el.Close()

	dir, _ := ioutil.ReadDir(logtestdir)
	require.Equal(t, 0, len(dir))
}

func Test_cannotCreateFile(t *testing.T) {
	he := getHandlerEnvironment(logtestdir)
	os.RemoveAll(logtestdir)
	defer os.MkdirAll(logtestdir, constants.FilePermissions_UserOnly_ReadWriteExecute)
	el := New(he)

	// Verify nothing blows up
	el.Info("blah")
	el.Close()
}

func Test_normalTrace(t *testing.T) {
	he := getHandlerEnvironment(logtestdir)
	el := New(he)

	el.Info("this is a %s", "test")
	el.Warn("something weird %s", "happened")
	el.Error("we ran out of cupcakes")
	el.Close()

	dir, _ := ioutil.ReadDir(logtestdir)
	require.Equal(t, 1, len(dir))
}

func getHandlerEnvironment(eventsFolder string) *handlerenv.HandlerEnvironment {
	return &handlerenv.HandlerEnvironment{
		HeartbeatFile: "",
		StatusFolder:  "",
		ConfigFolder:  "",
		LogFolder:     logtestdir,
		DataFolder:    "",
		EventsFolder:  "",
	}
}
