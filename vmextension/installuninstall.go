package vmextension

import (
	"os"

	"github.com/go-kit/kit/log"
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

func update(ctx log.Logger, ext *VMExtension) (string, error) {
	ctx.Log("event", "update")

	// The only thing we do for update is call the callback if we have one
	if ext.exec.updateCallback != nil {
		err := ext.exec.updateCallback(ctx, ext)
		if err != nil {
			ctx.Log("message", "Update failed", "error", err)
		}
	}

	return "", nil
}

func install(ctx log.Logger, ext *VMExtension) (string, error) {
	// Create the data directory if it doesn't exist
	exists, err := doesFileExistInstallDependency(ext.HandlerEnv.DataFolder)
	if err != nil {
		return "", err
	}

	if !exists {
		ctx.Log("event", "Creating data dir", "path", ext.HandlerEnv.DataFolder)
		if err := installDependency.mkdirAll(ext.HandlerEnv.DataFolder, 0755); err != nil {
			return "", errors.Wrap(err, "failed to create data dir")
		}

		ctx.Log("event", "created data dir", "path", ext.HandlerEnv.DataFolder)
	}

	ctx.Log("event", "installed")
	return "", nil
}

func uninstall(ctx log.Logger, ext *VMExtension) (string, error) {
	exists, err := doesFileExistInstallDependency(ext.HandlerEnv.DataFolder)
	if err != nil {
		return "", err
	}

	if exists {
		ctx.Log("event", "removing data dir", "path", ext.HandlerEnv.DataFolder)
		ctx = log.With(ctx, "path", ext.HandlerEnv.DataFolder)
		if err := installDependency.removeAll(ext.HandlerEnv.DataFolder); err != nil {
			return "", errors.Wrap(err, "failed to delete data dir")
		}
		ctx.Log("event", "removed data dir")
	}

	ctx.Log("event", "uninstalled")
	return "", nil
}
