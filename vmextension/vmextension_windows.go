package vmextension

import (
	"os"
	"path"
)

// GetOSName returns the name of the OS
func getOSName() (name string) {
	return "Windows"
}

func getDataFolder(name string, version string) string {
	systemDriveFolder := os.Getenv("SystemDrive")
	return path.Join(systemDriveFolder, "Packages\\Plugins", name, version, "Downloads")
}
