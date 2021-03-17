package extensionlauncher

import (
	"flag"
	"fmt"
	"github.com/Azure/azure-extension-platform/pkg/exithelper"
	"github.com/Azure/azure-extension-platform/pkg/handlerenv"
	"github.com/Azure/azure-extension-platform/pkg/logging"
	"github.com/Azure/azure-extension-platform/pkg/seqno"
	"github.com/Azure/azure-extension-platform/pkg/status"
	"github.com/Azure/azure-extension-platform/pkg/utils"
	"github.com/Azure/azure-extension-platform/vmextension"
	"github.com/pkg/errors"
	"os"
	"path"
)

var eh = exithelper.Exiter

func Run(handlerEnv *handlerenv.HandlerEnvironment, el *logging.ExtensionLogger, extensionName, extensionVersion, exeName, operation string) {
	writeTransitioningStatusAndStartExtensionAsASeparateProcess(extensionName, extensionVersion, exeName, operation, handlerEnv, el)
}

func ParseArgs() (extensionName, extensionVersion, exeName, operation string, err error) {
	flag.StringVar(&extensionName, "extensionname", "", "name of the extension")
	flag.StringVar(&extensionVersion, "extensionversion", "", "version of the extension")
	flag.StringVar(&exeName, "exename", "", "the name of the extension executable file")
	flag.StringVar(&operation, "operation", "", "the operation to perform on the extension")
	flag.Parse()
	if extensionName == "" {
		err = fmt.Errorf("could not parse extension name")
	}
	if extensionVersion == "" {
		err = errors.Wrap(err, "could not parse extension version")
	}
	if exeName == "" {
		err = errors.Wrap(err, "could not parse extension executable name")
	}
	if operation == "" {
		err = errors.Wrap(err, "could not parse operation")
	}
	return
}

func writeTransitioningStatusAndStartExtensionAsASeparateProcess(extensionName, extensionVersion, exeName, operation string, handlerEnv *handlerenv.HandlerEnvironment, el *logging.ExtensionLogger) {
	writeTransitioningStatus(extensionName, extensionVersion, operation, handlerEnv, el)
	workingDir, err := utils.GetCurrentProcessWorkingDir()
	if err != nil {
		el.Error("could not get current working directory %s", err.Error())
		eh.Exit(exithelper.EnvironmentError)
	}
	runExecutableAsIndependentProcess(exeName, operation, workingDir, handlerEnv.LogFolder, el)
}

func writeTransitioningStatus(extensionName, extensionVersion, operation string, handlerEnv *handlerenv.HandlerEnvironment, el *logging.ExtensionLogger) {
	if operation == vmextension.EnableOperation.ToString() {
		// we write transitioning status only for Enable command
		currentSequenceNumber, err := seqno.FindSeqNum(el, handlerEnv.ConfigFolder)
		if err != nil {
			el.Error("could not retrieve the current sequence number %s", err.Error())
			eh.Exit(exithelper.EnvironmentError)
		}
		// if status file exists, no need to overwrite it
		statusFilePath := path.Join(handlerEnv.StatusFolder, fmt.Sprintf("%d.status", currentSequenceNumber))

		fileInfo, statErr := os.Stat(statusFilePath)
		if os.IsNotExist(statErr) {
			statusReport := status.New(status.StatusTransitioning, vmextension.EnableOperation.ToStatusName(), fmt.Sprintf("extension %s version %s started execution", extensionName, extensionVersion))
			err := statusReport.Save(handlerEnv.StatusFolder, currentSequenceNumber)
			if err != nil {
				// don't exit
				el.Warn("could not write transitioning status for extension %s version %s", extensionName, extensionVersion)
			}
		} else if fileInfo != nil {
			el.Info("%d.status file already exists, will not create new status file with transitioning status", currentSequenceNumber)
		} else if err != nil {
			el.Warn("could not determine the existence or absence of status file, will continue without writing placeholder status file")
		}
	}
}
