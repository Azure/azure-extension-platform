// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package vmextension

import (
	"fmt"
	"os"
	"strings"

	"github.com/Azure/azure-extension-platform/pkg/environmentmanager"
	"github.com/Azure/azure-extension-platform/pkg/exithelper"
	"github.com/Azure/azure-extension-platform/pkg/extensionerrors"
	"github.com/Azure/azure-extension-platform/pkg/extensionevents"
	"github.com/Azure/azure-extension-platform/pkg/handlerenv"
	"github.com/Azure/azure-extension-platform/pkg/logging"
	"github.com/Azure/azure-extension-platform/pkg/seqno"
	"github.com/Azure/azure-extension-platform/pkg/settings"
	"github.com/Azure/azure-extension-platform/pkg/status"
	"github.com/pkg/errors"
)

// HandlerEnvFileName is the file name of the Handler Environment as placed by the
// Azure Guest Agent.
const handlerEnvFileName = "HandlerEnvironment.json"

type OperationName string

const (
	InstallOperation    OperationName = "install"
	UninstallOperation  OperationName = "uninstall"
	EnableOperation     OperationName = "enable"
	UpdateOperation     OperationName = "update"
	DisableOperation    OperationName = "disable"
	ResetStateOperation OperationName = "resetstate"
	invalid             OperationName = "invalid"
)

func (operationName OperationName) ToString() string {
	return string(operationName)
}

func (operationName OperationName) ToStatusName() string {
	return strings.Title(string(operationName))
}

type cmdFunc func(ext *VMExtension) (msg string, err error)

func OperationNameFromString(operation string) (OperationName, error) {
	switch operation {
	case InstallOperation.ToString():
		return InstallOperation, nil
	case UninstallOperation.ToString():
		return UninstallOperation, nil
	case EnableOperation.ToString():
		return EnableOperation, nil
	case UpdateOperation.ToString():
		return UpdateOperation, nil
	case DisableOperation.ToString():
		return DisableOperation, nil
	case ResetStateOperation.ToString():
		return ResetStateOperation, nil
	default:
		return invalid, extensionerrors.ErrInvalidOperationName
	}
}

// cmd is an internal structure that specifies how an operation should run
type cmd struct {
	f                  cmdFunc       // associated function
	operation          OperationName // human readable string
	shouldReportStatus bool          // determines if running this should log to a .status file
	failExitCode       int           // exitCode to use when commands fail
}

// executionInfo contains internal information necessary for the extension to execute
type executionInfo struct {
	cmds                map[OperationName]cmd                                // Execution commands keyed by operation
	requiresSeqNoChange bool                                                 // True if Enable will only execute if the sequence number changes
	supportsDisable     bool                                                 // Whether to run extension agnostic disable code
	supportsResetState  bool                                                 // Whether to run the extension agnostic ResetState code
	enableCallback      EnableCallbackFunc                                   // A method provided by the extension for Enable
	updateCallback      CallbackFunc                                         // A method provided by the extension for Update
	disableCallback     CallbackFunc                                         // A method provided by the extension for Disable
	resetStateCallBack  CallbackFunc                                         // A method provided by the extension for ResetState
	installCallback     CallbackFunc                                         // A method provided by the extension for Update
	uninstallCallback   CallbackFunc                                         // A method provided by the extension for Uninstall
	manager             environmentmanager.IGetVMExtensionEnvironmentManager // Used by tests to mock the environment
}

// VMExtension is an abstraction for standard extension operations in an OS agnostic manner
type VMExtension struct {
	Name                       string                                    // The name of the extension. This will contain 'Windows' or 'Linux'
	Version                    string                                    // The version of the extension
	GetRequestedSequenceNumber func() (uint, error)                      // Function to get the requested sequence number to run
	CurrentSequenceNumber      *uint                                     // The last run sequence number, null means no existing sequence number was found
	HandlerEnv                 *handlerenv.HandlerEnvironment            // Contains information about the folders necessary for the extension
	GetSettings                func() (*settings.HandlerSettings, error) // Function to get settings passed to the extension
	ExtensionEvents            *extensionevents.ExtensionEventManager    // Allows extensions to raise events
	ExtensionLogger            *logging.ExtensionLogger                  // Automatically logs to the log directory
	exec                       *executionInfo                            // Internal information necessary for the extension to run
	statusFormatter            status.StatusMessageFormatter             // Custom status message formatter from initialization info
}

type prodGetVMExtensionEnvironmentManager struct {
}

func (*prodGetVMExtensionEnvironmentManager) GetHandlerEnvironment(name string, version string) (*handlerenv.HandlerEnvironment, error) {
	return handlerenv.GetHandlerEnvironment(name, version)
}

func (*prodGetVMExtensionEnvironmentManager) FindSeqNum(el *logging.ExtensionLogger, configFolder string) (uint, error) {
	return seqno.FindSeqNum(el, configFolder)
}

func (*prodGetVMExtensionEnvironmentManager) GetCurrentSequenceNumber(el *logging.ExtensionLogger, retriever seqno.ISequenceNumberRetriever, name, version string) (uint, error) {
	return seqno.GetCurrentSequenceNumber(el, retriever, name, version)
}

func (em *prodGetVMExtensionEnvironmentManager) GetHandlerSettings(el *logging.ExtensionLogger, he *handlerenv.HandlerEnvironment) (*settings.HandlerSettings, error) {
	seqNo, err := em.FindSeqNum(el, he.ConfigFolder)
	if err != nil {
		return nil, err
	}
	return settings.GetHandlerSettings(el, he, seqNo)
}

func (*prodGetVMExtensionEnvironmentManager) SetSequenceNumberInternal(extensionName, extensionVersion string, seqNo uint) error {
	return seqno.SetSequenceNumber(extensionName, extensionVersion, seqNo)
}

// GetVMExtension returns a new VMExtension object
func GetVMExtension(initInfo *InitializationInfo) (ext *VMExtension, _ error) {
	return getVMExtensionInternal(initInfo, &prodGetVMExtensionEnvironmentManager{})
}

// GetVMExtensionForTesting mocks out the environment part of the VM extension for use with your extension
func GetVMExtensionForTesting(initInfo *InitializationInfo, manager environmentmanager.IGetVMExtensionEnvironmentManager) (ext *VMExtension, _ error) {
	return getVMExtensionInternal(initInfo, manager)
}

// Internal method that allows mocking for unit tests
func getVMExtensionInternal(initInfo *InitializationInfo, manager environmentmanager.IGetVMExtensionEnvironmentManager) (ext *VMExtension, _ error) {
	if initInfo == nil {
		return nil, extensionerrors.ErrArgCannotBeNull
	}

	if len(initInfo.Name) < 1 || len(initInfo.Version) < 1 {
		return nil, extensionerrors.ErrArgCannotBeNullOrEmpty
	}

	if initInfo.EnableCallback == nil {
		return nil, extensionerrors.ErrArgCannotBeNull
	}

	handlerEnv, err := manager.GetHandlerEnvironment(initInfo.Name, initInfo.Version)
	if err != nil {
		return nil, err
	}

	extensionLogger := logging.NewWithName(handlerEnv, initInfo.LogFileNameFormat)

	// Create our event manager. This will be disabled if no eventsFolder exists
	extensionEvents := extensionevents.New(extensionLogger, handlerEnv)

	// Determine the sequence number requested
	newSeqNo := func() (uint, error) { return manager.FindSeqNum(extensionLogger, handlerEnv.ConfigFolder) }

	// Determine the current sequence number
	retriever := seqno.ProdSequenceNumberRetriever{}
	var currentSeqNo = new(uint)
	retrievedSequenceNumber, err := manager.GetCurrentSequenceNumber(extensionLogger, &retriever, initInfo.Name, initInfo.Version)
	if err != nil {
		if err == extensionerrors.ErrNoSettingsFiles || err == extensionerrors.ErrNoMrseqFile {
			// current sequence number could not be found, this is a special error
			currentSeqNo = nil
		} else {
			return nil, fmt.Errorf("failed to read the current sequence number due to '%v'", err)
		}
	} else {
		*currentSeqNo = retrievedSequenceNumber
	}

	cmdInstall := cmd{install, InstallOperation, false, initInfo.InstallExitCode}
	cmdEnable := cmd{enable, EnableOperation, true, initInfo.OtherExitCode}
	cmdUninstall := cmd{uninstall, UninstallOperation, false, initInfo.OtherExitCode}

	// Only support Update and Disable if we need to
	var cmdDisable cmd
	var cmdUpdate cmd
	var cmdResetState cmd
	if initInfo.UpdateCallback != nil {
		cmdUpdate = cmd{update, UpdateOperation, false, 3}
	} else {
		cmdUpdate = cmd{noop, UpdateOperation, false, 3}
	}

	if initInfo.SupportsDisable || initInfo.DisableCallback != nil {
		cmdDisable = cmd{disable, DisableOperation, true, 3}
	} else {
		cmdDisable = cmd{noop, DisableOperation, true, 3}
	}

	if initInfo.SupportsResetState || initInfo.ResetStateCallback != nil {
		cmdResetState = cmd{resetState, ResetStateOperation, false, 3}
	} else {
		cmdResetState = cmd{noop, ResetStateOperation, false, 3}
	}

	settings := func() (*settings.HandlerSettings, error) {
		return manager.GetHandlerSettings(extensionLogger, handlerEnv)
	}

	var statusFormatter status.StatusMessageFormatter
	if initInfo.CustomStatusFormatter != nil {
		statusFormatter = initInfo.CustomStatusFormatter
	} else {
		statusFormatter = status.StatusMsg
	}

	ext = &VMExtension{
		Name:                       initInfo.Name,
		Version:                    initInfo.Version,
		GetRequestedSequenceNumber: newSeqNo,
		CurrentSequenceNumber:      currentSeqNo,
		HandlerEnv:                 handlerEnv,
		GetSettings:                settings,
		ExtensionEvents:            extensionEvents,
		ExtensionLogger:            extensionLogger,
		statusFormatter:            statusFormatter,
		exec: &executionInfo{
			manager:             manager,
			requiresSeqNoChange: initInfo.RequiresSeqNoChange,
			supportsDisable:     initInfo.SupportsDisable,
			supportsResetState:  initInfo.SupportsResetState,
			enableCallback:      initInfo.EnableCallback,
			disableCallback:     initInfo.DisableCallback,
			updateCallback:      initInfo.UpdateCallback,
			resetStateCallBack:  initInfo.ResetStateCallback,
			installCallback:     initInfo.InstallCallback,
			uninstallCallback:   initInfo.UninstallCallback,
			cmds: map[OperationName]cmd{
				InstallOperation:    cmdInstall,
				UninstallOperation:  cmdUninstall,
				EnableOperation:     cmdEnable,
				UpdateOperation:     cmdUpdate,
				DisableOperation:    cmdDisable,
				ResetStateOperation: cmdResetState,
			},
		},
	}

	return ext, nil
}

// Do is the main worker method of the extension and determines which operation
// to run, if necessary
func (ve *VMExtension) Do() {
	// parse command line arguments
	eh := exithelper.Exiter
	cmd := ve.parseCmd(os.Args, eh)
	_, err := cmd.f(ve)
	if err != nil {
		ve.ExtensionLogger.Error("failed to handle: %v", err)
		eh.Exit(cmd.failExitCode)
	}
}

// reportStatus saves operation status to the status file for the extension
// handler with the optional given message, if the given cmd requires reporting
// status.
//
// If an error occurs reporting the status, it will be logged and returned.
func reportStatus(ve *VMExtension, t status.StatusType, c cmd, msg string) error {
	if !c.shouldReportStatus {
		ve.ExtensionLogger.Info("status not reported for operation (by design)")
		return nil
	}

	requestedSequenceNumber, err := ve.GetRequestedSequenceNumber()
	if err != nil {
		return err
	}

	s := status.New(t, c.operation.ToStatusName(), ve.statusFormatter(c.operation.ToStatusName(), t, msg))
	if err := s.Save(ve.HandlerEnv.StatusFolder, requestedSequenceNumber); err != nil {
		ve.ExtensionLogger.Error("Failed to save handler status: %v", err)
		return errors.Wrap(err, "failed to save handler status")
	}
	return nil
}

// parseCmd looks at os.Args and parses the subcommand. If it is invalid,
// it prints the usage string and an error message and exits with code 0.
func (ve *VMExtension) parseCmd(args []string, eh exithelper.IExitHelper) cmd {
	if len(args) != 2 {
		ve.printUsage(args)
		fmt.Println("Incorrect usage.")
		eh.Exit(2)
		return cmd{}
	}

	op := args[1]
	operation, _ := OperationNameFromString(op)
	cmd, ok := ve.exec.cmds[operation]
	if !ok {
		ve.printUsage(args)
		fmt.Printf("Incorrect command: %q\n", op)
		eh.Exit(2)
	}
	return cmd
}

// printUsage prints the help string and version of the program to stdout with a
// trailing new line.
func (ve *VMExtension) printUsage(args []string) {
	fmt.Printf("Usage: %s ", os.Args[0])
	i := 0
	for k := range ve.exec.cmds {
		fmt.Print(k.ToString())
		if i != len(ve.exec.cmds)-1 {
			fmt.Printf("|")
		}
		i++
	}
	fmt.Println()
	fmt.Println(ve.Version)
}

func noop(ext *VMExtension) (string, error) {
	ext.ExtensionLogger.Info("noop")
	return "", nil
}
