package vmextension

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

var (
	installDependency installDependencies = &installDependencyImpl{}
)

type installDependencies interface {
	mkdirAll(string, os.FileMode) error
	removeAll(string) error
	stat(string) (os.FileInfo, error)
}

type installDependencyImpl struct{}

func (*installDependencyImpl) mkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (*installDependencyImpl) removeAll(path string) error {
	return os.RemoveAll(path)
}

func (*installDependencyImpl) stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func doesFileExistInstallDependency(filePath string) (bool, error) {
	_, err := installDependency.stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return true, err
	}

	return true, nil
}

func resetState(ext *VMExtension) (string, error) {
	ext.ExtensionLogger.Info("resetState called")

	// Remove all files in the data directory
	err := removeDirectoryContents(ext.HandlerEnv.DataFolder)
	if err != nil {
		ext.ExtensionLogger.Error("Removing data directory contents failed: %v", err)
	}

	// Call the callback if we have one
	if ext.exec.resetStateCallBack != nil {
		err := ext.exec.resetStateCallBack(ext)
		if err != nil {
			ext.ExtensionLogger.Error("ResetState failed: %v", err)
		}
	}

	return "", nil
}

func removeDirectoryContents(dir string) error {
	if dir == "" {
		return nil
	}

	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()

	// This is best effort. Readdirnames will return the directories it managed
	// to read if there is an error
	names, _ := d.Readdirnames(-1)
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}

	return nil
}

func update(ext *VMExtension) (string, error) {
	ext.ExtensionLogger.Info("update called")

	// The only thing we do for update is call the callback if we have one
	if ext.exec.updateCallback != nil {
		err := ext.exec.updateCallback(ext)
		if err != nil {
			ext.ExtensionLogger.Error("Update failed: %v", err)
		}
	}

	return "", nil
}

func install(ext *VMExtension) (string, error) {
	// Create the data directory if it doesn't exist
	exists, err := doesFileExistInstallDependency(ext.HandlerEnv.DataFolder)
	if err != nil {
		return "", err
	}

	if !exists {
		ext.ExtensionLogger.Info("Creating data dir %v", ext.HandlerEnv.DataFolder)
		if err := installDependency.mkdirAll(ext.HandlerEnv.DataFolder, 0755); err != nil {
			return "", errors.Wrap(err, "failed to create data dir")
		}

		ext.ExtensionLogger.Info("Created data dir %s", ext.HandlerEnv.DataFolder)
	}

	// Call the callback if we have one
	if ext.exec.installCallback != nil {
		err := ext.exec.installCallback(ext)
		if err != nil {
			ext.ExtensionLogger.Error("Install failed: %v", err)
		}
	}

	ext.ExtensionLogger.Info("installed")
	return "", nil
}

func uninstall(ext *VMExtension) (string, error) {
	exists, err := doesFileExistInstallDependency(ext.HandlerEnv.DataFolder)
	if err != nil {
		return "", err
	}

	if exists {
		ext.ExtensionLogger.Info("Removing data dir %v", ext.HandlerEnv.DataFolder)
		if err := installDependency.removeAll(ext.HandlerEnv.DataFolder); err != nil {
			return "", errors.Wrap(err, "failed to delete data dir")
		}
		ext.ExtensionLogger.Info("removed data dir")
	}

	// Call the callback if we have one
	if ext.exec.uninstallCallback != nil {
		err := ext.exec.uninstallCallback(ext)
		if err != nil {
			ext.ExtensionLogger.Error("Uninstall failed: %v", err)
		}
	}

	ext.ExtensionLogger.Info("uninstalled")
	return "", nil
}
