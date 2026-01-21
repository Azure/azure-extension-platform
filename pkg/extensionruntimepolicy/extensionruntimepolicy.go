package extensionruntimepolicy

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/Azure/azure-extension-platform/pkg/extensionerrors"
	"github.com/Azure/azure-extension-platform/pkg/logging"
	"github.com/Azure/azure-extension-platform/pkg/status"
)

const extensionRuntimePolicyFileName = "ExtensionRuntimePolicy.json" // Lourdes come back to this.

// ScriptType is consistent with the ScriptType defined in CRP.
type ScriptType string

const (
	Inline ScriptType = "Inline"
	Downloaded ScriptType = "Downloaded"
	Gallery ScriptType = "Gallery"
	Diagnostic ScriptType = "Diagnostic"
	CommandId ScriptType = "CommandId"
	None ScriptType = "None"
)

// FileType here refers to files types that can be signed. 
type FileType string

const (
	All FileType = "All"
	None FileType = "None"
	Script FileType = "Script"
)

type AllowedScriptTypes struct {
	AllowedCommandId bool
	Gallery bool
	Diagnostic bool
	Inline bool
	AllowedDownloaded bool
	AllowAll bool
}

// ExtensionRuntimePolicy is internal structure used to deserialize the extension runtime policy file
type extensionRuntimePolicy struct {
	RequireSigning FileType
	FileRootCert string
	DownloadedScriptsAllowList []string
	CommandIdAllowList []string
	RunAsUser string
	LimitScripts AllowedScriptTypes
	DisableOutputBlobs bool
	DisallowDomainPasswordChange bool
	ApplicationAllowList []string
}

type extensionRuntimePolicyFile struct {
	RuntimePolicy []extensionRuntimePolicyContainer `json:"runtimeSettings"`
}

//func GetExtensionRunTimePolicy(el logging.ILogger, statusFolder string, seqNo uint) (erp *extensionRuntimePolicy, _ error) {