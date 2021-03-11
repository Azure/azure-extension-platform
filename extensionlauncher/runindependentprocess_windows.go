package extensionlauncher

import (
	"fmt"
	"github.com/Azure/azure-extension-platform/pkg/commandhandler"
	"github.com/Azure/azure-extension-platform/pkg/logging"
)

var commandHandlerToUse = commandhandler.New()


func RunExecutableAsIndependentProcess(exeName, args, workingDir string, el logging.IExtensionLogger){
	commandToExecute := fmt.Sprintf("start /d %s /b %s %s", workingDir, exeName, args)
	commandHandlerToUse.Execute(commandToExecute, workingDir, false, el)
}
