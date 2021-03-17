package extensionlauncher

import (
	"fmt"
	"github.com/Azure/azure-extension-platform/pkg/commandhandler"
	"github.com/Azure/azure-extension-platform/pkg/logging"
)

var commandHandlerToUse = commandhandler.New()


func runExecutableAsIndependentProcess(exeName, args, workingDir, logDir string, el *logging.ExtensionLogger){
	commandToExecute := fmt.Sprintf("start /d %s /b %s %s", workingDir, exeName, args)
	commandHandlerToUse.Execute(commandToExecute, workingDir, logDir, false, el)
}
