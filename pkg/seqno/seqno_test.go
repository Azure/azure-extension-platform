package seqno

import (
	"github.com/Azure/azure-extension-platform/pkg/extensionerrors"
	"github.com/Azure/azure-extension-platform/pkg/testhelpers"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/require"
)

var (
	sequenceNumberTestFolder = "./flooperflop"
)

type mockSequenceNumberRetriever struct {
	returnSeqNo uint
	returnError error
}

func (snr mockSequenceNumberRetriever) GetSequenceNumber(name, version string) (uint, error) {
	return snr.returnSeqNo, snr.returnError
}

func Test_getCurrentSequenceNumberNotFound(t *testing.T) {
	retriever := mockSequenceNumberRetriever{returnSeqNo: 0, returnError: extensionerrors.ErrNotFound}
	ctx := log.NewSyncLogger(log.NewLogfmtLogger(os.Stdout))
	seqNo, err := GetCurrentSequenceNumber(ctx, retriever, "yaba", "5.0")
	require.Equal(t, uint(0), seqNo)
	require.Nil(t, err)
}

func Test_getCurrentSequenceNumberOtherError(t *testing.T) {
	retriever := mockSequenceNumberRetriever{returnSeqNo: 0, returnError: extensionerrors.ErrInvalidSettingsFile}
	ctx := log.NewSyncLogger(log.NewLogfmtLogger(os.Stdout))
	_, err := GetCurrentSequenceNumber(ctx, retriever, "yaba", "5.0")
	require.Equal(t, extensionerrors.ErrInvalidSettingsFile, err)
}

func Test_getCurrentSequenceNumberFound(t *testing.T) {
	retriever := mockSequenceNumberRetriever{returnSeqNo: 42, returnError: nil}
	ctx := log.NewSyncLogger(log.NewLogfmtLogger(os.Stdout))
	seqNo, err := GetCurrentSequenceNumber(ctx, retriever, "yaba", "5.0")
	require.NoError(t, err, "getCurrentSequenceNumber failed")
	require.Equal(t, uint(42), seqNo)
}

func Test_findSeqNoFolderDoesntExist(t *testing.T) {
	ctx := log.NewSyncLogger(log.NewLogfmtLogger(os.Stdout))
	seqNo, err := FindSeqNum(ctx, "./yabamonster")
	require.Equal(t, uint(0), seqNo)
	require.Error(t, err)
}

func Test_findSeqNoFilesInDifferentOrder(t *testing.T) {
	ctx := log.NewSyncLogger(log.NewLogfmtLogger(os.Stdout))
	testhelpers.CleanupTestDirectory(t, sequenceNumberTestFolder)
	// sleep after the creation of each file so that there is a difference in time of creation
	// findSeqNum should return the most recently created file, sleep it necessary to ensure that the creation times
	// are different enough
	writeSequenceNumberFile(t, sequenceNumberTestFolder, "5")
	time.Sleep(5 * time.Millisecond)
	writeSequenceNumberFile(t, sequenceNumberTestFolder, "4")
	time.Sleep(5 * time.Millisecond)
	writeSequenceNumberFile(t, sequenceNumberTestFolder, "3")
	time.Sleep(5 * time.Millisecond)
	writeSequenceNumberFile(t, sequenceNumberTestFolder, "2")

	seqNo, err := FindSeqNum(ctx, sequenceNumberTestFolder)
	require.NoError(t, err, "findSeqNum failed")
	require.Equal(t, uint(2), seqNo)
}

func Test_findSeqNoNoFilesInFolder(t *testing.T) {
	testhelpers.CleanupTestDirectory(t, sequenceNumberTestFolder)
	ctx := log.NewSyncLogger(log.NewLogfmtLogger(os.Stdout))

	seqNo, err := FindSeqNum(ctx, sequenceNumberTestFolder)
	require.Equal(t, extensionerrors.ErrNoSettingsFiles, err)
	require.Equal(t, uint(0), seqNo)
}

func Test_findSeqNoInvalidFileName(t *testing.T) {
	ctx := log.NewSyncLogger(log.NewLogfmtLogger(os.Stdout))
	testhelpers.CleanupTestDirectory(t, sequenceNumberTestFolder)
	writeSequenceNumberFile(t, sequenceNumberTestFolder, "0")
	writeSequenceNumberFile(t, sequenceNumberTestFolder, "yaba")

	seqNo, err := FindSeqNum(ctx, sequenceNumberTestFolder)
	require.Equal(t, extensionerrors.ErrInvalidSettingsFileName, err)
	require.Equal(t, uint(0), seqNo)
}

func Test_findSeqNoFilesSameTimestamp(t *testing.T) {
	ctx := log.NewSyncLogger(log.NewLogfmtLogger(os.Stdout))
	testhelpers.CleanupTestDirectory(t, sequenceNumberTestFolder)
	timeStamp := time.Now()
	writeSequenceNumberFileTs(t, sequenceNumberTestFolder, "3", timeStamp)
	writeSequenceNumberFileTs(t, sequenceNumberTestFolder, "2", timeStamp)
	writeSequenceNumberFileTs(t, sequenceNumberTestFolder, "1", timeStamp)
	writeSequenceNumberFileTs(t, sequenceNumberTestFolder, "0", timeStamp)

	seqNo, err := FindSeqNum(ctx, sequenceNumberTestFolder)
	require.NoError(t, err, "findSeqNum failed")
	require.Equal(t, uint(3), seqNo)
}

func Test_findSeqNoFilesSameTimestampOneInvalid(t *testing.T) {
	ctx := log.NewSyncLogger(log.NewLogfmtLogger(os.Stdout))
	testhelpers.CleanupTestDirectory(t, sequenceNumberTestFolder)
	timeStamp := time.Now()
	writeSequenceNumberFileTs(t, sequenceNumberTestFolder, "3", timeStamp)
	writeSequenceNumberFileTs(t, sequenceNumberTestFolder, "2", timeStamp)
	writeSequenceNumberFileTs(t, sequenceNumberTestFolder, "yaba", timeStamp)
	writeSequenceNumberFileTs(t, sequenceNumberTestFolder, "0", timeStamp)

	seqNo, err := FindSeqNum(ctx, sequenceNumberTestFolder)
	require.Equal(t, extensionerrors.ErrInvalidSettingsFileName, err)
	require.Equal(t, uint(0), seqNo)
}

func Test_findSeqNoVeryDifferentNumbers(t *testing.T) {
	ctx := log.NewSyncLogger(log.NewLogfmtLogger(os.Stdout))
	testhelpers.CleanupTestDirectory(t, sequenceNumberTestFolder)
	writeSequenceNumberFile(t, sequenceNumberTestFolder, "0")
	writeSequenceNumberFile(t, sequenceNumberTestFolder, "117")
	writeSequenceNumberFile(t, sequenceNumberTestFolder, "2942")
	writeSequenceNumberFile(t, sequenceNumberTestFolder, "35749")

	seqNo, err := FindSeqNum(ctx, sequenceNumberTestFolder)
	require.NoError(t, err, "findSeqNum failed")
	require.Equal(t, uint(35749), seqNo)
}

func Test_findSeqNoOnlyOneFile(t *testing.T) {
	ctx := log.NewSyncLogger(log.NewLogfmtLogger(os.Stdout))
	testhelpers.CleanupTestDirectory(t, sequenceNumberTestFolder)
	writeSequenceNumberFile(t, sequenceNumberTestFolder, "157")

	seqNo, err := FindSeqNum(ctx, sequenceNumberTestFolder)
	require.NoError(t, err, "findSeqNum failed")
	require.Equal(t, uint(157), seqNo)
}

func Test_findSeqNoNormalExecution(t *testing.T) {
	ctx := log.NewSyncLogger(log.NewLogfmtLogger(os.Stdout))
	testhelpers.CleanupTestDirectory(t, sequenceNumberTestFolder)
	writeSequenceNumberFile(t, sequenceNumberTestFolder, "0")
	writeSequenceNumberFile(t, sequenceNumberTestFolder, "1")
	writeSequenceNumberFile(t, sequenceNumberTestFolder, "2")
	writeSequenceNumberFile(t, sequenceNumberTestFolder, "3")

	seqNo, err := FindSeqNum(ctx, sequenceNumberTestFolder)
	require.NoError(t, err, "findSeqNum failed")
	require.Equal(t, uint(3), seqNo)
}

func writeSequenceNumberFileTs(t *testing.T, testDirectory string, name string, timeStamp time.Time) {
	fullPath := writeSequenceNumberFile(t, testDirectory, name)
	err := os.Chtimes(fullPath, timeStamp, timeStamp)
	require.NoError(t, err, "Chtimes failed")
}

func writeSequenceNumberFile(t *testing.T, testDirectory string, name string) string {
	fullName := name + ".settings"
	fullPath := path.Join(testDirectory, fullName)
	data := []byte("this doesn't matter")
	err := ioutil.WriteFile(fullPath, data, 0644)
	require.NoError(t, err, "WriteFile failed")

	return fullPath
}
