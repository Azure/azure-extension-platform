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

type cmdFunc func(ext *VMExtension) (msg string, err error)

var errNotFound error = errors.New("NotFound")

// cmd is an internal structure that specifies how an operation should run
type cmd struct {
	f                  cmdFunc // associated function
	name               string  // human readable string
	shouldReportStatus bool    // determines if running this should log to a .status file
	failExitCode       int     // exitCode to use when commands fail
}

// executionInfo contains internal information necessary for the extension to execute
type executionInfo struct {
	cmds                map[string]cmd                                       // Execution commands keyed by operation
	requiresSeqNoChange bool                                                 // True if Enable will only execute if the sequence number changes
	supportsDisable     bool                                                 // Whether to run extension agnostic disable code
	enableCallback      EnableCallbackFunc                                   // A method provided by the extension for Enable
	updateCallback      CallbackFunc                                         // A method provided by the extension for Update
	disableCallback     CallbackFunc                                         // A method provided by the extension for disable
	manager             environmentmanager.IGetVMExtensionEnvironmentManager // Used by tests to mock the environment
}

// VMExtension is an abstraction for standard extension operations in an OS agnostic manner
type VMExtension struct {
	Name                    string                                 // The name of the extension. This will contain 'Windows' or 'Linux'
	Version                 string                                 // The version of the extension
	RequestedSequenceNumber uint                                   // The requested sequence number to run
	CurrentSequenceNumber   *uint                                  // The last run sequence number, null means no existing sequence number was found
	HandlerEnv              *handlerenv.HandlerEnvironment         // Contains information about the folders necessary for the extension
	Settings                *settings.HandlerSettings              // Contains settings passed to the extension
	ExtensionEvents         *extensionevents.ExtensionEventManager // Allows extensions to raise events
	ExtensionLogger         *logging.ExtensionLogger               // Automatically logs to the log directory
	exec                    *executionInfo                         // Internal information necessary for the extension to run
}

type prodGetVMExtensionEnvironmentManager struct {
}

func (*prodGetVMExtensionEnvironmentManager) GetHandlerEnvironment(name string, version string) (*handlerenv.HandlerEnvironment, error) {
	return getHandlerEnvironment(name, version)
}

func (*prodGetVMExtensionEnvironmentManager) FindSeqNum(el *logging.ExtensionLogger, configFolder string) (uint, error) {
	return seqno.FindSeqNum(el, configFolder)
}

func (*prodGetVMExtensionEnvironmentManager) GetCurrentSequenceNumber(el *logging.ExtensionLogger, retriever seqno.ISequenceNumberRetriever, name, version string) (uint, error) {
	return seqno.GetCurrentSequenceNumber(el, retriever, name, version)
}

func (*prodGetVMExtensionEnvironmentManager) GetHandlerSettings(el *logging.ExtensionLogger, he *handlerenv.HandlerEnvironment, seqNo uint) (*settings.HandlerSettings, error) {
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

	// Create our event manager. This will be disabled if no eventsFolder exists
	extensionLogger := logging.New(handlerEnv)
	extensionEvents := extensionevents.New(extensionLogger, handlerEnv)

	// Determine the sequence number requested
	newSeqNo, err := manager.FindSeqNum(extensionLogger, handlerEnv.ConfigFolder)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to find sequence number")
	}

	// Determine the current sequence number
	retriever := seqno.ProcSequenceNumberRetriever{}
	var currentSeqNo = new(uint)
	retrievedSequenceNumber, err := manager.GetCurrentSequenceNumber(extensionLogger, &retriever, initInfo.Name, initInfo.Version)
	if err != nil {
		if err == extensionerrors.ErrNoSettingsFiles {
			// current sequence number could not be found, this is a special error
			currentSeqNo = nil
		} else {
			return nil, fmt.Errorf("Failed to read the current sequence number due to '%v'", err)
		}
	} else {
		*currentSeqNo = retrievedSequenceNumber
	}

	cmdInstall := cmd{install, "Install", false, initInfo.InstallExitCode}
	cmdEnable := cmd{enable, "Enable", true, initInfo.OtherExitCode}
	cmdUninstall := cmd{uninstall, "Uninstall", false, initInfo.OtherExitCode}

	// Only support Update and Disable if we need to
	var cmdDisable cmd
	var cmdUpdate cmd
	if initInfo.UpdateCallback != nil {
		cmdUpdate = cmd{update, "Update", true, 3}
	} else {
		cmdUpdate = cmd{noop, "Update", true, 3}
	}

	if initInfo.SupportsDisable || initInfo.DisableCallback != nil {
		cmdDisable = cmd{disable, "Disable", true, 3}
	} else {
		cmdDisable = cmd{noop, "Disable", true, 3}
	}

	settings, err := manager.GetHandlerSettings(extensionLogger, handlerEnv, newSeqNo)
	if err != nil {
		return nil, err
	}

	ext = &VMExtension{
		Name:                    initInfo.Name + getOSName(),
		Version:                 initInfo.Version,
		RequestedSequenceNumber: newSeqNo,
		CurrentSequenceNumber:   currentSeqNo,
		HandlerEnv:              handlerEnv,
		Settings:                settings,
		ExtensionEvents:         extensionEvents,
		ExtensionLogger:         extensionLogger,
		exec: &executionInfo{
			manager:             manager,
			requiresSeqNoChange: initInfo.RequiresSeqNoChange,
			supportsDisable:     initInfo.SupportsDisable,
			enableCallback:      initInfo.EnableCallback,
			disableCallback:     initInfo.DisableCallback,
			updateCallback:      initInfo.UpdateCallback,
			cmds: map[string]cmd{
				"install":   cmdInstall,
				"uninstall": cmdUninstall,
				"enable":    cmdEnable,
				"update":    cmdUpdate,
				"disable":   cmdDisable,
			},
		},
	}

	return ext, nil
}

// Do is the main worker method of the extension and determines which operation
// to run, if necessary
func (ve *VMExtension) Do() {
	// parse command line arguments
	cmd := ve.parseCmd(os.Args)
	ve.ExtensionLogger.Info("Running operation %v for seqNo %v", strings.ToLower(cmd.name), ve.RequestedSequenceNumber)

	// remember the squence number
	err := ve.exec.manager.SetSequenceNumberInternal(ve.Name, ve.Version, ve.RequestedSequenceNumber)
	if err != nil {
		ve.ExtensionLogger.Error("failed to write the new sequence number: %v", err)
	}

	// execute the command
	reportStatus(ve, status.StatusTransitioning, cmd, "")
	msg, err := cmd.f(ve)
	if err != nil {
		ve.ExtensionLogger.Error("failed to handle: %v", err)
		reportStatus(ve, status.StatusError, cmd, err.Error()+msg)
		exithelper.Exiter.Exit(cmd.failExitCode)
	}

	reportStatus(ve, status.StatusSuccess, cmd, msg)
}

// reportStatus saves operation status to the status file for the extension
// handler with the optional given message, if the given cmd requires reporting
// status.
//
// If an error occurs reporting the status, it will be logged and returned.
func reportStatus(ve *VMExtension, t status.StatusType, c cmd, msg string) error {
	if !c.shouldReportStatus {
		ve.ExtensionLogger.Info("status ot reported for operation (by design)")
		return nil
	}

	s := status.New(t, c.name, status.StatusMsg(c.name, t, msg))
	if err := s.Save(ve.HandlerEnv.StatusFolder, ve.RequestedSequenceNumber); err != nil {
		ve.ExtensionLogger.Error("Failed to save handler status: %v", err)
		return errors.Wrap(err, "failed to save handler status")
	}
	return nil
}

// parseCmd looks at os.Args and parses the subcommand. If it is invalid,
// it prints the usage string and an error message and exits with code 0.
func (ve *VMExtension) parseCmd(args []string) cmd {
	if len(args) != 2 {
		ve.printUsage(args)
		fmt.Println("Incorrect usage.")
		exithelper.Exiter.Exit(2)
	}

	op := args[1]
	cmd, ok := ve.exec.cmds[op]
	if !ok {
		ve.printUsage(args)
		fmt.Printf("Incorrect command: %q\n", op)
		exithelper.Exiter.Exit(2)
	}
	return cmd
}

// printUsage prints the help string and version of the program to stdout with a
// trailing new line.
func (ve *VMExtension) printUsage(args []string) {
	fmt.Printf("Usage: %s ", os.Args[0])
	i := 0
	for k := range ve.exec.cmds {
		fmt.Printf(k)
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
