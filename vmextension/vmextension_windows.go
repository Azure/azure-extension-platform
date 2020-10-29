package vmextension

import (
	"fmt"
	"github.com/Azure/azure-extension-platform/pkg/handlerenv"
	"golang.org/x/sys/windows/registry"
	"os"
	"path"
)

const (
	sequenceNumberKeyName = "SequenceNumber"
	heartBeatFileKeyName  = "HeartBeatFile"
	statusFolderKeyName   = "StatusFolder"
)

// GetOSName returns the name of the OS
func getOSName() (name string) {
	return "Windows"
}

func getExtensionKeyName(name string, version string) (keyName string) {
	return fmt.Sprintf("Software\\Microsoft\\Windows Azure\\HandlerState\\%s_%s", name, version)
}

// GetHandlerEnv reads the directory information from the registry
func getHandlerEnvironment(name string, version string) (he *handlerenv.HandlerEnvironment, _ error) {
	extensionKeyName := getExtensionKeyName(name, version)
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, extensionKeyName, registry.QUERY_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			// This may happen if the extension isn't installed. Return a uniform error indicating this.
			return he, errNotFound
		}

		return he, fmt.Errorf("VmExtension: Cannot open sequence registry key due to '%v'", err)
	}
	defer k.Close()

	heartBeatFile, _, err := k.GetStringValue(heartBeatFileKeyName)
	if err != nil {
		return he, fmt.Errorf("VmExtension: Cannot read heartbeat file name due to '%v'", err)
	}

	statusFolder, _, err := k.GetStringValue(statusFolderKeyName)
	if err != nil {
		return he, fmt.Errorf("VmExtension: Cannot read status folder name due to '%v'", err)
	}

	// Config folder is at %SYSTEMDRIVE%\Packages\Plugins\{extension name}\{extension version}\RuntimeSettings
	systemDriveFolder := os.Getenv("SystemDrive")
	configFolder := path.Join(systemDriveFolder, "Packages\\Plugins", name, version, "RuntimeSettings")
	dataFolder := path.Join(systemDriveFolder, "Packages\\Plugins", name, version, "Downloads")

	// Logs folder is at %SYSTEMDRIVE%\WindowsAzure\Logs\Plugins\{extension name}\{extension version}
	logFolder := path.Join(systemDriveFolder, "WindowsAzure\\Logs\\Plugins", name, version)

	he = &handlerenv.HandlerEnvironment{
		HeartbeatFile: heartBeatFile,
		StatusFolder:  statusFolder,
		ConfigFolder:  configFolder,
		LogFolder:     logFolder,
		DataFolder:    dataFolder,
	}

	return he, nil
}
