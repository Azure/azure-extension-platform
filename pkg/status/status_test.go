package status

import (
	"encoding/json"
	"github.com/D1v38om83r/azure-extension-platform/pkg/testhelpers"
	"io/ioutil"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	statusTestDirectory = "./statustest"
)

func Test_statusMsgSucceededWithString(t *testing.T) {
	s := StatusMsg("yaba", StatusSuccess, "")
	require.Equal(t, "yaba succeeded", s)
}

func Test_statusMsgFailedWithMsg(t *testing.T) {
	s := StatusMsg("", StatusError, "flipper")
	require.Equal(t, " failed: flipper", s)
}

func Test_statusMsgInProgressEmpty(t *testing.T) {
	s := StatusMsg("", StatusTransitioning, "")
	require.Equal(t, " in progress", s)
}

func Test_statusMsgOther(t *testing.T) {
	s := StatusMsg("yaba", "flooper", "flop")
	require.Equal(t, "yaba: flop", s)
}

func Test_statusMsgFull(t *testing.T) {
	s := StatusMsg("yaba", StatusSuccess, "flop")
	require.Equal(t, "yaba succeeded: flop", s)
}

func Test_newStatus(t *testing.T) {
	report := New(StatusError, "WorldDomination", "bow before the unit test!")
	require.NotNil(t, report)
	require.Equal(t, 1, len(report))
	require.Equal(t, "WorldDomination", report[0].Status.Operation)
	require.Equal(t, StatusError, report[0].Status.Status)
}

func Test_statusSaveFolderDoesntExist(t *testing.T) {
	report := New(StatusSuccess, "flip", "flop")
	err := report.Save("./flopperdoodle", 5)
	require.Error(t, err)
}

func Test_statusSaveNewFile(t *testing.T) {
	report := New(StatusSuccess, "flip", "flop")
	testhelpers.CleanupTestDirectory(t, statusTestDirectory)
	err := report.Save(statusTestDirectory, 5)
	require.NoError(t, err, "save report failed")

	filePath := path.Join(statusTestDirectory, "5.status")
	b, err := ioutil.ReadFile(filePath)
	require.NoError(t, err, "ReadFile failed")

	var r StatusReport
	err = json.Unmarshal(b, &r)
	require.NoError(t, err, "Unmarshal failed")
	require.Equal(t, 1, len(r))
	require.Equal(t, "flip", report[0].Status.Operation)
	require.Equal(t, StatusSuccess, report[0].Status.Status)
}

func Test_statusSaveExistingFile(t *testing.T) {
	report := New(StatusSuccess, "flip", "flop")
	testhelpers.CleanupTestDirectory(t, statusTestDirectory)
	err := report.Save(statusTestDirectory, 7)
	require.NoError(t, err, "save report failed")
	err = report.Save(statusTestDirectory, 7)
	require.NoError(t, err, "second ave report failed")
}



