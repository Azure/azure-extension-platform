// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package logging

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

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

func Test_fileName(t *testing.T) {
	clearLogTestDir()
	he := getHandlerEnvironment(logtestdir)
	el := NewWithName(he, "yaba_%v.log")

	el.Info("this is a %s", "test")
	el.Close()

	dir, _ := ioutil.ReadDir(logtestdir)
	require.Equal(t, 1, len(dir))
	require.True(t, strings.HasPrefix(dir[0].Name(), "yaba_"))
	require.True(t, strings.HasSuffix(dir[0].Name(), ".log"))
}

func Test_normalTrace(t *testing.T) {
	clearLogTestDir()
	he := getHandlerEnvironment(logtestdir)
	el := New(he)

	el.Info("this is a %s", "test")
	el.Warn("something weird %s", "happened")
	el.Error("we ran out of cupcakes")
	el.Close()

	files, _ := ioutil.ReadDir(logtestdir)
	require.Equal(t, 1, len(files))

	fullpath := path.Join(logtestdir, files[0].Name())
	bytes, err := ioutil.ReadFile(fullpath)
	assert.NoError(t, err)
	contents := string(bytes)

	lines := strings.Split(contents, "\n")
	assert.Contains(t, lines[0], "this is a test")
	assert.Contains(t, lines[1], "something weird happened")
	assert.Contains(t, lines[2], "we ran out of cupcakes")
}

func Test_loggingFromStream(t *testing.T) {
	he := getHandlerEnvironment(logtestdir)
	el := New(he)

	dateTimeRegexPattern := "[0-9]{4}/[0-9]{2}/[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2} "

	errorRegexPattern := logLevelError + dateTimeRegexPattern
	infoRegexPattern := logLevelInfo + dateTimeRegexPattern
	warnRegexPattern := logLevelWarning + dateTimeRegexPattern

	errorMessagePrefix := "error message prefix"
	infoMessagePrefix := "info message prefix"
	warnMessagePrefix := "warning message prefix"
	errorMessage := "this is an error message"
	infoMessage := "this is an info message"
	warningMessage := "this is a warning message"

	message1 := "message 1"
	message2 := "message 2"

	errorFilePath := path.Join(logtestdir, "errorfile")
	errorFile, err := createFileAndWriteMessage(errorFilePath, errorMessage)
	assert.NoError(t, err)

	infoFilePath := path.Join(logtestdir, "infoFile")
	infoFile, err := createFileAndWriteMessage(infoFilePath, infoMessage)
	assert.NoError(t, err)

	warnFilePath := path.Join(logtestdir, "warnFile")
	warnFile, err := createFileAndWriteMessage(warnFilePath, warningMessage)
	assert.NoError(t, err)

	el.Error(message1)
	el.ErrorFromStream(errorMessagePrefix, errorFile)
	el.Error(message2)
	el.InfoFromStream(infoMessagePrefix, infoFile)
	el.WarnFromStream(warnMessagePrefix, warnFile)
	errorFile.Close()
	infoFile.Close()
	warnFile.Close()

	fileContents, err := readContentsOfMostRecentLogFile(logtestdir)
	assert.NoError(t, err)
	assert.Regexp(t, errorRegexPattern+message1, fileContents)
	assert.Regexp(t, errorRegexPattern+errorMessagePrefix, fileContents)
	assert.True(t, strings.Contains(fileContents, errorMessage))
	assert.Regexp(t, errorRegexPattern+message2, fileContents)
	assert.Regexp(t, infoRegexPattern+infoMessagePrefix, fileContents)
	assert.True(t, strings.Contains(fileContents, infoMessage))
	assert.Regexp(t, warnRegexPattern+warnMessagePrefix, fileContents)
	assert.True(t, strings.Contains(fileContents, warningMessage))
}

func createFileAndWriteMessage(fileName, message string) (*os.File, error) {
	file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, constants.FilePermissions_UserOnly_ReadWrite)
	if err != nil {
		return file, err
	}

	file.Write([]byte(message))
	file.Seek(0, io.SeekStart)
	return file, err
}

func readContentsOfMostRecentLogFile(logfileDir string) (string, error) {
	dir, err := ioutil.ReadDir(logfileDir)
	if err != nil {
		return "", err
	}

	var mostRecentLogFilename = ""
	modTime, err := time.Parse(time.RFC3339, "2006-01-02T15:04:05Z")
	if err != nil {
		return "", err
	}
	for _, fileInfo := range dir {
		if !fileInfo.IsDir() && strings.HasPrefix(fileInfo.Name(), "log") {
			if fileInfo.ModTime().After(modTime) {
				modTime = fileInfo.ModTime()
				mostRecentLogFilename = fileInfo.Name()
			}
		}
	}
	if mostRecentLogFilename == "" {
		return "", fmt.Errorf("did not find log file under %s", logfileDir)
	}
	fileToRead := path.Join(logtestdir, mostRecentLogFilename)
	fileContents, err := ioutil.ReadFile(fileToRead)
	fileContentsString := string(fileContents)
	return fileContentsString, nil
}

func clearLogTestDir() {
	dir, _ := ioutil.ReadDir(logtestdir)
	for _, d := range dir {
		os.RemoveAll(path.Join([]string{logtestdir, d.Name()}...))
	}
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
