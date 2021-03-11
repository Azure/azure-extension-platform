package extensionlauncher

func RunExecutableAsIndependentProcess(exeName, args, workingDir, logDir string, el *logging.ExtensionLogger){
	commandToExecute := fmt.Sprintf("%s %s &", workingDir, exeName, args)
	commandHandlerToUse.Execute(commandToExecute, workingDir, logDir, false, el)
}
