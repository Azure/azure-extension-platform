// Sample code to for how to use azure-extension-helper with your extension

package main

import (
	"os"

	"github.com/Azure/azure-extension-platform/vmextension"
	"github.com/go-kit/kit/log"
)

const (
	extensionName    = "TestExtension"
	extensionVersion = "0.0.0.1"
)

var enableCallbackFunc vmextension.EnableCallbackFunc = func(ext *vmextension.VMExtension) (string, error) {
	// put your extension specific code here
	// on enable, the extension will call this code
	return "put your extension code here", nil
}

var updateCallbackFunc vmextension.CallbackFunc = func(ext *vmextension.VMExtension) error {
	// optional
	// on update, the extension will call this code
	return nil
}

var disableCallbackFunc vmextension.CallbackFunc = func(ext *vmextension.VMExtension) error {
	// optional
	// on disable, the extension will call this code
	return nil
}

var getVMExtensionFuncToCall = vmextension.GetVMExtension
var getInitializationInfoFuncToCall = vmextension.GetInitializationInfo

var logger = log.NewSyncLogger(log.NewLogfmtLogger(os.Stdout))

func main() {
	err := getExtensionAndRun()
	if err != nil {
		os.Exit(2)
	}
}

func getExtensionAndRun() error {
	initilizationInfo, err := getInitializationInfoFuncToCall(extensionName, extensionVersion, true, enableCallbackFunc)
	if err != nil {
		return err
	}

	initilizationInfo.DisableCallback = disableCallbackFunc
	initilizationInfo.UpdateCallback = updateCallbackFunc
	vmExt, err := getVMExtensionFuncToCall(initilizationInfo)
	if err != nil {
		return err
	}
	vmExt.Do()
	return nil
}
