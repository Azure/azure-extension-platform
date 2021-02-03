package vmextension

import (
	"encoding/json"
	"fmt"
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
	"io/ioutil"
	"os"
	"path/filepath"
)

// HandlerEnvFileName is the file name of the Handler Environment as placed by the
// Azure Guest Agent.
const handlerEnvFileName = "HandlerEnvironment.json"

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

// HandlerEnvironment describes the handler environment configuration presented
// to the extension handler by the Azure Guest Agent.
type handlerEnvironmentInternal struct {
	Version            float64 `json:"version"`
	Name               string  `json:"name"`
	HandlerEnvironment struct {
		HeartbeatFile       string `json:"heartbeatFile"`
		StatusFolder        string `json:"statusFolder"`
		ConfigFolder        string `json:"configFolder"`
		LogFolder           string `json:"logFolder"`
		EventsFolder        string `json:"eventsFolder"`
		EventsFolderPreview string `json:"eventsFolder_preview"`
		DeploymentID        string `json:"deploymentid"`
		RoleName            string `json:"rolename"`
		Instance            string `json:"instance"`
		HostResolverAddress string `json:"hostResolverAddress"`
	}
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

	// Create our event manager. This will be disabled if no eventsFolder exists
	extensionLogger := logging.New(handlerEnv)
	extensionEvents := extensionevents.New(extensionLogger, handlerEnv)

	// Determine the sequence number requested
	newSeqNo := func() (uint, error) { return manager.FindSeqNum(extensionLogger, handlerEnv.ConfigFolder) }

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
		cmdUpdate = cmd{update, "Update", false, 3}
	} else {
		cmdUpdate = cmd{noop, "Update", false, 3}
	}

	if initInfo.SupportsDisable || initInfo.DisableCallback != nil {
		cmdDisable = cmd{disable, "Disable", true, 3}
	} else {
		cmdDisable = cmd{noop, "Disable", true, 3}
	}

	settings := func() (*settings.HandlerSettings, error) {
		return manager.GetHandlerSettings(extensionLogger, handlerEnv)
	}

	ext = &VMExtension{
		Name:                       initInfo.Name + getOSName(),
		Version:                    initInfo.Version,
		GetRequestedSequenceNumber: newSeqNo,
		CurrentSequenceNumber:      currentSeqNo,
		HandlerEnv:                 handlerEnv,
		GetSettings:                settings,
		ExtensionEvents:            extensionEvents,
		ExtensionLogger:            extensionLogger,
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

// GetHandlerEnv locates the HandlerEnvironment.json file by assuming it lives
// next to or one level above the extension handler (read: this) executable,
// reads, parses and returns it.
func getHandlerEnvironment(name string, version string) (he *handlerenv.HandlerEnvironment, _ error) {
	contents, _, err := findAndReadFile(handlerEnvFileName)
	if err != nil {
		return nil, err
	}

	handlerEnvInternal, err := parseHandlerEnv(contents)
	if err != nil {
		return nil, err
	}

	// The data directory is a subdirectory of waagent, with the extension name
	dataFolder := getDataFolder(name, version)

	// TODO: before this API goes public, remove the eventsfolder_preview
	// This is only used for private preview of the events
	eventsFolder := handlerEnvInternal.HandlerEnvironment.EventsFolder
	if eventsFolder == "" {
		eventsFolder = handlerEnvInternal.HandlerEnvironment.EventsFolderPreview
	}

	return &handlerenv.HandlerEnvironment{
		HeartbeatFile:       handlerEnvInternal.HandlerEnvironment.HeartbeatFile,
		StatusFolder:        handlerEnvInternal.HandlerEnvironment.StatusFolder,
		ConfigFolder:        handlerEnvInternal.HandlerEnvironment.ConfigFolder,
		LogFolder:           handlerEnvInternal.HandlerEnvironment.LogFolder,
		DataFolder:          dataFolder,
		EventsFolder:        eventsFolder,
		DeploymentID:        handlerEnvInternal.HandlerEnvironment.DeploymentID,
		RoleName:            handlerEnvInternal.HandlerEnvironment.RoleName,
		Instance:            handlerEnvInternal.HandlerEnvironment.Instance,
		HostResolverAddress: handlerEnvInternal.HandlerEnvironment.HostResolverAddress,
	}, nil
}

// ParseHandlerEnv parses the HandlerEnvironment.json format.
func parseHandlerEnv(b []byte) (*handlerEnvironmentInternal, error) {
	var hf []handlerEnvironmentInternal

	if err := json.Unmarshal(b, &hf); err != nil {
		return nil, fmt.Errorf("vmextension: failed to parse handler env: %v", err)
	}
	if len(hf) != 1 {
		return nil, fmt.Errorf("vmextension: expected 1 config in parsed HandlerEnvironment, found: %v", len(hf))
	}
	return &hf[0], nil
}

// findAndReadFile locates the specified file on disk relative to our currently
// executing process and attempts to read the file
func findAndReadFile(fileName string) (b []byte, fileLoc string, _ error) {
	dir, err := scriptDir()
	if err != nil {
		return nil, "", fmt.Errorf("vmextension: cannot find base directory of the running process: %v", err)
	}

	paths := []string{
		filepath.Join(dir, fileName),       // this level (i.e. executable is in [EXT_NAME]/.)
		filepath.Join(dir, "..", fileName), // one up (i.e. executable is in [EXT_NAME]/bin/.)
	}

	for _, p := range paths {
		o, err := ioutil.ReadFile(p)
		if err != nil && !os.IsNotExist(err) {
			return nil, "", fmt.Errorf("vmextension: error examining '%s' at '%s': %v", fileName, p, err)
		} else if err == nil {
			fileLoc = p
			b = o
			break
		}
	}

	if b == nil {
		return nil, "", errNotFound
	}

	return b, fileLoc, nil
}

// scriptDir returns the absolute path of the running process.
func scriptDir() (string, error) {
	p, err := filepath.Abs(os.Args[0])
	if err != nil {
		return "", err
	}
	return filepath.Dir(p), nil
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

	s := status.New(t, c.name, status.StatusMsg(c.name, t, msg))
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
	cmd, ok := ve.exec.cmds[op]
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
