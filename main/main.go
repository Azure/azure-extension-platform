// Sample code to for how to use azure-extension-helper with your extension

package main

import (
	"github.com/D1v38om83r/azure-extension-platform/pkg/vmextension"
	"github.com/go-kit/kit/log"
	"os"
)

const (
	extensionName    = "TestExtension"
	extensionVersion = "0.0.0.1"
)

var enableCallbackFunc vmextension.EnableCallbackFunc = func(ctx log.Logger, ext *vmextension.VMExtension) (string, error) {
	// put your extension specific code here
	// on enable, the extension will call this code
	return "put your extension code here", nil
}

var updateCallbackFunc vmextension.CallbackFunc = func(ctx log.Logger, ext *vmextension.VMExtension) error {
	// optional
	// on update, the extension will call this code
	return nil
}

var disableCallbackFunc vmextension.CallbackFunc = func(ctx log.Logger, ext *vmextension.VMExtension) error {
	// optional
	// on disable, the extension will call this code
	return nil
}

var logger = log.NewSyncLogger(log.NewLogfmtLogger(os.Stdout))

func main() {
	err := getExtensionAndRun()
	if err != nil {
		os.Exit(2)
	}
}

func getExtensionAndRun() (error) {
	initilizationInfo, err := vmextension.GetInitializationInfo(extensionName, extensionVersion, true, enableCallbackFunc)
	if err != nil {
		return err
	}

	initilizationInfo.DisableCallback = disableCallbackFunc
	initilizationInfo.UpdateCallback = updateCallbackFunc
	ctx := log.With(log.With(logger, "time", log.DefaultTimestampUTC), "version", extensionVersion)
	vmExt , err := vmextension.GetVMExtension(ctx, initilizationInfo)
	if err != nil {
		return err
	}
	vmExt.Do(ctx)
	return nil
}
