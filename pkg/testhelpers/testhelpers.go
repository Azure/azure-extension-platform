// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package testhelpers

import (
	"github.com/Azure/azure-extension-platform/pkg/constants"
	"github.com/stretchr/testify/require"
	"os"
	"path"
	"testing"
)

func CleanupTestDirectory(t *testing.T, testDirectory string) {
	// Create the directory if it doesn't already exist
	_ = os.Mkdir(testDirectory, constants.FilePermissions_UserOnly_ReadWriteExecute)

	// Open the directory and read all its files.
	dirRead, err := os.Open(testDirectory)
	require.NoError(t, err, "os.Open failed")
	dirFiles, err := dirRead.Readdir(0)
	require.NoError(t, err, "Readdir failed")

	// Loop over the directory's files.
	for index := range dirFiles {
		fileToDelete := dirFiles[index]
		fullPath := path.Join(testDirectory, fileToDelete.Name())
		err = os.Remove(fullPath)
		require.NoError(t, err, "os.Remove failed")
	}
}

