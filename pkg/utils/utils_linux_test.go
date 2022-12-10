// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	scriptsDirectory         = "./downloadTemp"
	runtimeSettingsDirectory = "./configTemp"
	scriptFileName           = "script.sh"
	extensionName            = "RunCommandName"
	scriptContent            = "sample script content"
	settingsFileContent      = "sample settings content"
)

func Test_TryClearRuntimeSettingsAndDeleteScriptsExceptMostRecent(t *testing.T) {
	// Clear scripts and runtimeSettings directories
	os.RemoveAll(scriptsDirectory)
	os.RemoveAll(runtimeSettingsDirectory)

	var filePermission os.FileMode = 666

	// Create scriptsDirectory and runtimeSettingsDirectory
	os.Mkdir(scriptsDirectory, filePermission)
	os.Mkdir(runtimeSettingsDirectory, filePermission)

	// Create scripts for 3 sequence numbers (0, 1, 2)
	scriptDir0 := filepath.Join(scriptsDirectory, extensionName, "0")
	os.MkdirAll(scriptDir0, filePermission)
	scriptFile0 := filepath.Join(scriptDir0, scriptFileName)
	scriptFileDesc0, _ := os.Create(scriptFile0)
	fmt.Fprintln(scriptFileDesc0, scriptContent)

	scriptDir1 := filepath.Join(scriptsDirectory, extensionName, "1")
	os.MkdirAll(scriptDir1, filePermission)
	scriptFile1 := filepath.Join(scriptDir1, scriptFileName)
	scriptFileDesc1, _ := os.Create(scriptFile1)
	fmt.Fprintln(scriptFileDesc1, scriptContent)

	scriptDir2 := filepath.Join(scriptsDirectory, extensionName, "2")
	os.MkdirAll(scriptDir2, filePermission)
	scriptFile2 := filepath.Join(scriptDir2, scriptFileName)
	scriptFileDesc2, _ := os.Create(scriptFile2)
	fmt.Fprintln(scriptFileDesc2, scriptContent)

	// create runtimeSettings for 3 sequence numbers (0,1,2)
	settingsFile0 := filepath.Join(runtimeSettingsDirectory, "0.settings")
	settingsFileDesc0, _ := os.Create(settingsFile0)
	fmt.Fprintln(settingsFileDesc0, settingsFileContent)

	settingsFile1 := filepath.Join(runtimeSettingsDirectory, "1.settings")
	settingsFileDesc1, _ := os.Create(settingsFile1)
	fmt.Fprintln(settingsFileDesc1, settingsFileContent)

	settingsFile2 := filepath.Join(runtimeSettingsDirectory, "2.settings")
	settingsFileDesc2, _ := os.Create(settingsFile2)
	fmt.Fprintln(settingsFileDesc2, settingsFileContent)

	// Delete scripts and empty runtimeSettings except for last sequence number
	err := TryClearExtensionScriptsDirectoriesAndSettingsFilesExceptMostRecent(scriptsDirectory,
		runtimeSettingsDirectory,
		"RunCommandName", // extensionName
		2,                // most recent sequence number
		"\\d+.settings",  // regex to identify runtime settings files for all sequence numbers
		"%d.settings")    // Format string to construct last runtime settings file which need to be skipped
	require.Equal(t, nil, err)

}
