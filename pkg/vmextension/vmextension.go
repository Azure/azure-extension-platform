package vmextension

import (
	"fmt"
	"github.com/D1v38om83r/azure-extension-platform/pkg/seqno"
	"github.com/D1v38om83r/azure-extension-platform/pkg/settings"
	"github.com/D1v38om83r/azure-extension-platform/pkg/extensionerrors"
	"github.com/D1v38om83r/azure-extension-platform/pkg/handlerenv"
	"github.com/D1v38om83r/azure-extension-platform/pkg/status"
	"os"
	"strings"
	"github.com/go-kit/kit/log"
	"github.com/pkg/errors"

)

type cmdFunc func(ctx log.Logger, ext *VMExtension) (msg string, err error)

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
	cmds                map[string]cmd                   // Execution commands keyed by operation
	requiresSeqNoChange bool                             // True if Enable will only execute if the sequence number changes
	supportsDisable     bool                             // Whether to run extension agnostic disable code
	enableCallback      EnableCallbackFunc               // A method provided by the extension for Enable
	updateCallback      CallbackFunc                     // A method provided by the extension for Update
	disableCallback     CallbackFunc                     // A method provided by the extension for disable
	manager             getVMExtensionEnvironmentManager // Used by tests to mock the environment
}

// VMExtension is an abstraction for standard extension operations in an OS agnostic manner
type VMExtension struct {
	Name                    string              // The name of the extension. This will contain 'Windows' or 'Linux'
	Version                 string              // The version of the extension
	RequestedSequenceNumber uint                // The requested sequence number to run
	CurrentSequenceNumber   uint                // The last run sequence number
	HandlerEnv              *handlerenv.HandlerEnvironment // Contains information about the folders necessary for the extension
	Settings                *settings.HandlerSettings    // Contains settings passed to the extension
	exec                    *executionInfo      // Internal information necessary for the extension to run
}



// Allows for mocking all environment operations when running tests against VM extensions
type getVMExtensionEnvironmentManager interface {
	getHandlerEnvironment(name string, version string) (*handlerenv.HandlerEnvironment, error)
	findSeqNum(ctx log.Logger, configFolder string) (uint, error)
	getCurrentSequenceNumber(ctx log.Logger, retriever seqno.ISequenceNumberRetriever, name, version string) (uint, error)
	getHandlerSettings(ctx log.Logger, he *handlerenv.HandlerEnvironment, seqNo uint) (*settings.HandlerSettings, error)
	setSequenceNumberInternal(ve *VMExtension, seqNo uint) error
}

type prodGetVMExtensionEnvironmentManager struct {
}

func (*prodGetVMExtensionEnvironmentManager) getHandlerEnvironment(name string, version string) (*handlerenv.HandlerEnvironment, error) {
	return getHandlerEnvironment(name, version)
}

func (*prodGetVMExtensionEnvironmentManager) findSeqNum(ctx log.Logger, configFolder string) (uint, error) {
	return seqno.FindSeqNum(ctx, configFolder)
}

func (*prodGetVMExtensionEnvironmentManager) getCurrentSequenceNumber(ctx log.Logger, retriever seqno.ISequenceNumberRetriever, name, version string) (uint, error) {
	return seqno.GetCurrentSequenceNumber(ctx, retriever, name, version)
}

func (*prodGetVMExtensionEnvironmentManager) getHandlerSettings(ctx log.Logger, he *handlerenv.HandlerEnvironment, seqNo uint) (*settings.HandlerSettings, error) {
	return settings.GetHandlerSettings(ctx, he, seqNo)
}

func (*prodGetVMExtensionEnvironmentManager) setSequenceNumberInternal(ve *VMExtension, seqNo uint) error {
	return seqno.SetSequenceNumber(ve.Name, ve.Version, seqNo)
}

// GetVMExtension returns a new VMExtension object
func GetVMExtension(ctx log.Logger, initInfo *InitializationInfo) (ext *VMExtension, _ error) {
	return getVMExtensionInternal(ctx, initInfo, &prodGetVMExtensionEnvironmentManager{})
}

// Internal method that allows mocking for unit tests
func getVMExtensionInternal(ctx log.Logger, initInfo *InitializationInfo, manager getVMExtensionEnvironmentManager) (ext *VMExtension, _ error) {
	if initInfo == nil {
		return nil, extensionerrors.ErrArgCannotBeNull
	}

	if len(initInfo.Name) < 1 || len(initInfo.Version) < 1 {
		return nil, extensionerrors.ErrArgCannotBeNullOrEmpty
	}

	if initInfo.EnableCallback == nil {
		return nil, extensionerrors.ErrArgCannotBeNull
	}

	handlerEnv, err := manager.getHandlerEnvironment(initInfo.Name, initInfo.Version)
	if err != nil {
		return nil, err
	}

	// Determine the sequence number requested
	newSeqNo, err := manager.findSeqNum(ctx, handlerEnv.ConfigFolder)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to find sequence number")
	}

	// Determine the current sequence number
	retriever := seqno.ProcSequenceNumberRetriever{}
	currentSeqNo, err := manager.getCurrentSequenceNumber(ctx, &retriever, initInfo.Name, initInfo.Version)
	if err != nil {
		return nil, fmt.Errorf("Failed to read the current sequence number due to '%v'", err)
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

	settings, err := manager.getHandlerSettings(ctx, handlerEnv, newSeqNo)
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
func (ve *VMExtension) Do(ctx log.Logger) {
	// parse command line arguments
	cmd := ve.parseCmd(os.Args)
	ctx = log.With(ctx, "operation", strings.ToLower(cmd.name))

	ctx = log.With(ctx, "seq", ve.RequestedSequenceNumber)

	// remember the squence number
	err := ve.exec.manager.setSequenceNumberInternal(ve, ve.RequestedSequenceNumber)
	if err != nil {
		ctx.Log("message", "failed to write the new sequence number", "error", err)
	}

	// execute the command
	reportStatus(ctx, ve, status.StatusTransitioning, cmd, "")
	msg, err := cmd.f(ctx, ve)
	if err != nil {
		ctx.Log("event", "failed to handle", "error", err)
		reportStatus(ctx, ve, status.StatusError, cmd, err.Error()+msg)
		os.Exit(cmd.failExitCode)
	}

	reportStatus(ctx, ve, status.StatusSuccess, cmd, msg)
}

// reportStatus saves operation status to the status file for the extension
// handler with the optional given message, if the given cmd requires reporting
// status.
//
// If an error occurs reporting the status, it will be logged and returned.
func reportStatus(ctx log.Logger, ve *VMExtension, t status.StatusType, c cmd, msg string) error {
	if !c.shouldReportStatus {
		ctx.Log("status", "not reported for operation (by design)")
		return nil
	}

	s := status.New(t, c.name, status.StatusMsg(c.name, t, msg))
	if err := s.Save(ve.HandlerEnv.StatusFolder, ve.RequestedSequenceNumber); err != nil {
		ctx.Log("event", "failed to save handler status", "error", err)
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
		os.Exit(2)
	}

	op := args[1]
	cmd, ok := ve.exec.cmds[op]
	if !ok {
		ve.printUsage(args)
		fmt.Printf("Incorrect command: %q\n", op)
		os.Exit(2)
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

func noop(ctx log.Logger, ext *VMExtension) (string, error) {
	ctx.Log("event", "noop")
	return "", nil
}


