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

var logger = logging.New(nil)
var eh = exithelper.Exiter

var getHandlerEnvFuncToUse = handlerenv.GetHandlerEnvironment

func runExtensionLauncher(){
	extensionName, extensionVersion, exeName, operation, err := parseArgs()
	if err != nil {
		logger.Error("error parsing arguments %s", err.Error())
		eh.Exit(exithelper.ArgumentError)
	}
	writeTransitioningStatusAndStartExtensionAsASeparateProcess(extensionName, extensionVersion, exeName, operation)
	eh.Exit(0)
}

func parseArgs()(extensionName, extensionVersion, exeName, operation string, err error){
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


func writeTransitioningStatusAndStartExtensionAsASeparateProcess(extensionName, extensionVersion, exeName, operation string){
	writeTransitioningStatus(extensionName, extensionVersion, operation)
	workingDir, err := utils.GetCurrentProcessWorkingDir()
	if err != nil {
		logger.Error("could not get current working directory %s", err.Error())
		eh.Exit(exithelper.EnvironmentError)
	}
	RunExecutableAsIndependentProcess(exeName, operation, workingDir, logger)
	eh.Exit(0)
}

func writeTransitioningStatus(extensionName, extensionVersion, operation string){
	handlerEnv, err := getHandlerEnvFuncToUse(extensionName, extensionVersion)
	if err != nil {
		logger.Error("could not retrieve handler environment %s", err.Error())
		eh.Exit(exithelper.EnvironmentError)
	}

	// update logger as soon as we get HandlerEnvironment
	logger = logging.New(handlerEnv)
	if operation == vmextension.EnableOperation.ToCommandName(){
		// we write transitioning status only for Enable command
		currentSequenceNumber, err := seqno.GetCurrentSequenceNumber(logger, &seqno.ProdSequenceNumberRetriever{}, extensionName, extensionVersion)
		if err != nil {
			logger.Error("could not retrieve the current sequence number %s", err.Error())
			eh.Exit(exithelper.EnvironmentError)
		}
		// if status file exists, no need to overwrite it
		statusFilePath := path.Join(handlerEnv.StatusFolder, fmt.Sprintf("%d.status",currentSequenceNumber))

		_, statErr := os.Stat(statusFilePath)
		if os.IsNotExist(statErr) {
			statusReport := status.New(status.StatusTransitioning, vmextension.EnableOperation.ToPascalCaseName(), fmt.Sprintf("extension %s version %s started execution", extensionName, extensionVersion))
			err := statusReport.Save(handlerEnv.StatusFolder, currentSequenceNumber)
			if err != nil {
				// don't exit
				logger.Warn("could not write transitioning status for extension %s version %s", extensionName, extensionVersion)
			}

		}
	}
}
