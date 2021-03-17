package main

import (
	"github.com/Azure/azure-extension-platform/extensionlauncher"
	"github.com/Azure/azure-extension-platform/pkg/exithelper"
	"github.com/Azure/azure-extension-platform/pkg/handlerenv"
	"github.com/Azure/azure-extension-platform/pkg/logging"
)

var el = logging.New(nil)
var eh = exithelper.Exiter

func main (){

	extName, extVersion, exeName, operation, err := extensionlauncher.ParseArgs()
	if err != nil {
		el.Error("error parsing arguments %s", err.Error())
		eh.Exit(exithelper.ArgumentError)
	}
	handlerEnv, err := handlerenv.GetHandlerEnvironment(extName, extVersion)
	if err != nil {
		el.Error("could not retrieve handler environment %s", err.Error())
		eh.Exit(exithelper.EnvironmentError)
	}
	el = logging.New(handlerEnv)
	extensionlauncher.Run(handlerEnv, el, extName, extVersion, exeName, operation)
	eh.Exit(0)
}
