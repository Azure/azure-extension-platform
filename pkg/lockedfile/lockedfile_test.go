// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package lockedfile

import (
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"testing"
	"time"

	"github.com/Azure/azure-extension-platform/pkg/constants"
	"github.com/stretchr/testify/assert"
)

const testdir = "./testdir"

var testFilePath = path.Join(testdir, "temp.lockedfile")

var lastOpenedRegex = regexp.MustCompile("\"LastOpened\":\"([^\\\"]+)\"")
var lastClosedRegex = regexp.MustCompile("\"LastClosed\":\"([^\\\"]+)\"")

func initializeTest(t *testing.T) {
	err := os.MkdirAll(testdir, constants.FilePermissions_UserOnly_ReadWriteExecute)
	assert.NoError(t, err)
}

func TestLockFileMetadata(t *testing.T) {
	initializeTest(t)
	var lf ILockedFile
	lf, err := New(testFilePath, time.Second)
	assert.NoError(t, err)
	lf.Close()
	lastOpened, lastClosed, err := getLastOpenedAndLastClosedTime(testFilePath)
	assert.NoError(t, err)
	assert.True(t, lastClosed.After(lastOpened), "lastClosed should be after than lastOpened")
}

func getLastOpenedAndLastClosedTime(filePath string) (lastOpened, lastClosed time.Time, err error) {
	// read locked file to ensure that the timestamps are correct
	// windows cannot handle shrinking sizes properly, so the timestamps have to be read with regex instead of json.Umarshall
	bytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return
	}

	fileContentString := string(bytes)

	groups := lastOpenedRegex.FindAllStringSubmatch(fileContentString, -1)
	lastOpened, err = time.Parse(time.RFC3339Nano, groups[0][1])
	if err != nil {
		return
	}
	groups = lastClosedRegex.FindAllStringSubmatch(fileContentString, -1)
	lastClosed, err = time.Parse(time.RFC3339Nano, groups[0][1])
	return
}
