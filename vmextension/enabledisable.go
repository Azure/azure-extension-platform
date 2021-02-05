package vmextension

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"syscall"

	"github.com/Azure/azure-extension-platform/pkg/status"

	"github.com/Azure/azure-extension-platform/pkg/exithelper"
)

const disabledFileName = "disable"

var (
	disableDependency disableDependencies = &disableDependencyImpl{}
)

func enable(ext *VMExtension) (string, error) {
	// If the sequence number has not changed and we require it to, then exit
	// remember the sequence number
	// execute the command
	enableCmd, exists := ext.exec.cmds["enable"]
	if !exists {
		msg := "extension does not have an enable command"
		ext.ExtensionLogger.Error(msg)
		reportStatus(ext, status.StatusError, enableCmd, msg)
		return msg, fmt.Errorf(msg)
	}
	requestedSequenceNumber, err := ext.GetRequestedSequenceNumber()
	if err != nil {
		msg := "could not determine requested sequence number"
		ext.ExtensionLogger.Error("%s: %v", msg, err)
		reportStatus(ext, status.StatusError, enableCmd, err.Error()+msg)
		return msg, err
	}

	if ext.exec.requiresSeqNoChange && ext.CurrentSequenceNumber != nil && requestedSequenceNumber <= *ext.CurrentSequenceNumber {
		ext.ExtensionLogger.Info("sequence number has not increased. Exiting.")
		exithelper.Exiter.Exit(0)
	}

	ext.ExtensionLogger.Info("Running operation %v for seqNo %v", strings.ToLower(enableCmd.name), requestedSequenceNumber)
	reportStatus(ext, status.StatusTransitioning, enableCmd, "")

	err = ext.exec.manager.SetSequenceNumberInternal(ext.Name, ext.Version, requestedSequenceNumber)
	if err != nil {
		msg := "failed to write new sequence number"
		ext.ExtensionLogger.Warn("%s: %v", msg, err)
		// execution is not stopped by design
	}

	if ext.exec.supportsDisable && isDisabled(ext) {
		// The sequence number has changed and we're disabled, so re-enable the extension
		ext.ExtensionLogger.Info("re-enabling the extension")
		err := setDisabled(ext, false)
		if err != nil {
			// Note: we don't return here because the least we can do is let the extension do its stuff
			ext.ExtensionLogger.Error("Could not re-enable the extension: %v", err)
		}
	}

	// execute the command, save its error
	msg, runErr := ext.exec.enableCallback(ext)
	if runErr != nil {
		ext.ExtensionLogger.Error("Enable failed: %v", runErr)
		reportStatus(ext, status.StatusError, enableCmd, runErr.Error()+msg)
	} else {
		ext.ExtensionLogger.Info("Enable succeeded")
		reportStatus(ext, status.StatusSuccess, enableCmd, msg)
	}

	return msg, runErr
}

type disableDependencies interface {
	writeFile(string, []byte, os.FileMode) error
	remove(name string) error
}

type disableDependencyImpl struct{}

func (*disableDependencyImpl) writeFile(filename string, data []byte, perm os.FileMode) error {
	return ioutil.WriteFile(filename, data, perm)
}

func (*disableDependencyImpl) remove(name string) error {
	return os.Remove(name)
}

func doesFileExistDisableDependency(filePath string) (bool, error) {
	_, err := installDependency.stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return true, err
	}

	return true, nil
}

func disable(ext *VMExtension) (string, error) {
	disableCmd, exists := ext.exec.cmds["disable"]
	if ! exists {
		msg := "disable command not found"
		ext.ExtensionLogger.Error(msg)
		return msg, fmt.Errorf(msg)
	}
	ext.ExtensionLogger.Info("disable called")

	if ext.exec.supportsDisable {
		ext.ExtensionLogger.Info("Disabling extension")
		if isDisabled(ext) {
			ext.ExtensionLogger.Info("Extension is already disabled")
		} else {
			err := setDisabled(ext, true)
			if err != nil {
				reportStatus(ext, status.StatusError, disableCmd, "disable failed " + err.Error())
				return "", err
			}
		}
	} else {
		ext.ExtensionLogger.Info("VMExtension supportsDisable is set to false. No action to be taken")
	}

	// Call the callback if we have one
	if ext.exec.disableCallback != nil {
		err := ext.exec.disableCallback(ext)
		if err != nil {
			ext.ExtensionLogger.Error("Disable failed: %v", err)
			reportStatus(ext, status.StatusError, disableCmd, "disable failed " + err.Error())
			return "", err
		}
	}

	reportStatus(ext, status.StatusSuccess, disableCmd, "")
	return "", nil
}

func isDisabled(ext *VMExtension) bool {
	if ext.exec.supportsDisable == false {
		ext.ExtensionLogger.Info("supportsDisable was false, skipping check for disableFile")
		return false
	}
	// We are disabled if the disabled file exists in the config folder
	disabledFile := path.Join(ext.HandlerEnv.ConfigFolder, disabledFileName)
	exists, err := doesFileExistDisableDependency(disabledFile)
	if err != nil {
		ext.ExtensionLogger.Error("doesFileExit error detected: %v", err.Error())
	}
	return exists
}

func setDisabled(ext *VMExtension, disabled bool) error {
	disabledFile := path.Join(ext.HandlerEnv.ConfigFolder, disabledFileName)
	exists, err := doesFileExistDisableDependency(disabledFile)
	if err != nil {
		ext.ExtensionLogger.Error("doesFileExit error detected: %v", err.Error())
	}
	if exists != disabled {
		if disabled {
			// Create the file
			ext.ExtensionLogger.Info("Disabling extension")
			b := []byte("1")
			err := disableDependency.writeFile(disabledFile, b, 0644)
			if err != nil {
				ext.ExtensionLogger.Error("Could not disable the extension: %v", err)
				return err
			}

			ext.ExtensionLogger.Info("Disabled extension")
		} else {
			// Remove the file
			ext.ExtensionLogger.Info("Un-disabling extension")
			err := disableDependency.remove(disabledFile)
			if err == nil {
				ext.ExtensionLogger.Info("Re-enabled extension")
				return nil
			}

			// despite the check above, sometimes the disable file doesn't exist due to concurrent issue
			// catch errors that may arise from trying to disable a non existent file
			pathError, isPathError := err.(*os.PathError)
			if isPathError {
				if pathError.Err == syscall.ENOENT {
					ext.ExtensionLogger.Warn("Disable file was not present ignoring error")
					return nil
				}
			}

			ext.ExtensionLogger.Error("Could not re-enable the extension: %v", err)
			return err
		}
	}

	return nil
}
