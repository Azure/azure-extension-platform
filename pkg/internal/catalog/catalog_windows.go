package catalog

import (
	"encoding/hex"
	"strings"
	"syscall"
	"unsafe"

	"github.com/Azure/azure-extension-platform/pkg/hashutils"
	"github.com/Azure/azure-extension-platform/vmextension"
	"github.com/pkg/errors"
)

const (
	WINTRUST_ACTION_GENERIC_VERIFY_V2_GUID = "00AAC56B-CD44-11d0-8CC2-00C04FC295EE"
)

var (
	Modwintrust        = syscall.NewLazyDLL("wintrust.dll")
	procWinVerifyTrust = Modwintrust.NewProc("WinVerifyTrust")
)

type WTD_UI uint32

const (
	WTD_UI_ALL    WTD_UI = 1
	WTD_UI_NONE   WTD_UI = 2
	WTD_UI_NOBAD  WTD_UI = 3
	WTD_UI_NOGOOD WTD_UI = 4
)

type WTD_REVOKE_FLAGS uint32

const (
	WTD_REVOKE_NONE       WTD_REVOKE_FLAGS = 0x00000000
	WTD_REVOKE_WHOLECHAIN WTD_REVOKE_FLAGS = 0x00000001
)

type WTD_CHOICE uint32

const (
	WTD_CHOICE_FILE    WTD_CHOICE = 1
	WTD_CHOICE_CATALOG WTD_CHOICE = 2
	WTD_CHOICE_BLOB    WTD_CHOICE = 3
	WTD_CHOICE_SIGNER  WTD_CHOICE = 4
	WTD_CHOICE_CERT    WTD_CHOICE = 5
)

type WTD_STATE_ACTION uint32

const (
	WTD_STATEACTION_IGNORE           WTD_STATE_ACTION = 0x00000000
	WTD_STATEACTION_VERIFY           WTD_STATE_ACTION = 0x00000001
	WTD_STATEACTION_CLOSE            WTD_STATE_ACTION = 0x00000002
	WTD_STATEACTION_AUTO_CACHE       WTD_STATE_ACTION = 0x00000003
	WTD_STATEACTION_AUTO_CACHE_FLUSH WTD_STATE_ACTION = 0x00000004
)

type WTD_PROVIDER_FLAGS uint32

const (
	WTD_PROV_FLAGS_MASK                                        = 0x0000FFFF
	WTD_USE_IE4_TRUST_FLAG                  WTD_PROVIDER_FLAGS = 0x1
	WTD_NO_IE4_CHAIN_FLAG                   WTD_PROVIDER_FLAGS = 0x2
	WTD_NO_POLICY_USAGE_FLAG                WTD_PROVIDER_FLAGS = 0x4
	WTD_REVOCATION_CHECK_NONE               WTD_PROVIDER_FLAGS = 0x10
	WTD_REVOCATION_CHECK_END_CERT           WTD_PROVIDER_FLAGS = 0x20
	WTD_REVOCATION_CHECK_CHAIN              WTD_PROVIDER_FLAGS = 0x40
	WTD_REVOCATION_CHECK_CHAIN_EXCLUDE_ROOT WTD_PROVIDER_FLAGS = 0x80
	WTD_SAFER_FLAG                          WTD_PROVIDER_FLAGS = 0x100
	WTD_HASH_ONLY_FLAG                      WTD_PROVIDER_FLAGS = 0x200
	WTD_USE_DEFAULT_OSVER_CHECK             WTD_PROVIDER_FLAGS = 0x400
	WTD_LIFETIME_SIGNING_FLAG               WTD_PROVIDER_FLAGS = 0x800
)

type WTD_UICONTEXT uint32

const (
	WTD_UICONTEXT_EXECUTE WTD_UICONTEXT = 0
	WTD_UICONTEXT_INSTALL WTD_UICONTEXT = 1
)

type winTrustData struct {
	cbStruct            uint32
	pPolicyCallbackData uintptr
	pSIPClientData      uintptr
	dwUIChoice          WTD_UI
	fdWRevocationChecks WTD_REVOKE_FLAGS
	dwUnionChoice       WTD_CHOICE
	union               [8]byte // This is a placeholder for the actual union data, which can be one of several types depending on dwUnionChoice
	dwStateAction       WTD_STATE_ACTION
	hWVTStateData       uintptr
	pwszURLReference    uintptr
	dwProvFlags         WTD_PROVIDER_FLAGS
	dwUIContext         WTD_UICONTEXT
}

type winTrustFileInfo struct {
	cbStruct       uint32
	pcwszFile      uintptr
	hFile          syscall.Handle
	pgKnownSubject uintptr
}

type winTrustCatalogInfo struct {
	cbStruct             uint32
	dwCatalogVersion     uint32
	pcwszCatalogFilePath uintptr
	pcwszMemberTag       uintptr
	pcwszMemberFilePath  uintptr
	hMemberFile          syscall.Handle
	pbCalculatedFileHash uintptr
	cbCalculatedFileHash uint32
	pcCatalogContext     uintptr
}

type winTrustSignatureSettings struct {
	cbStruct           uint32
	dwIndex            uint32
	dwFlags            uint32
	dwVerifiedSigIndex uint32
}

func verifySignatureWinVerifyTrust(hWnd syscall.Handle, actionId *syscall.GUID, data *winTrustData) (uint32, error) {
	r1, _, err := syscall.SyscallN(procWinVerifyTrust.Addr(), uintptr(hWnd), uintptr(unsafe.Pointer(actionId)), uintptr(unsafe.Pointer(data)))
	if r1 != 0 {
		return uint32(r1), err
	}
	return 0, nil
}

// VerifyFileSignature verifies Authenticode signature for a file using WinVerifyTrust.
// Returns WinVerifyTrust status code (0 == valid).
func VerifyFileSignature(filePath string) (uint32, *vmextension.ErrorWithClarification) {
	filePathPtr, err := syscall.UTF16PtrFromString(filePath) // WinVerifyTrust takes in a LPCWSTR type, which is a ptr to a const null-terminated UTF-16 string
	if err != nil {
		return 1, vmextension.NewErrorWithClarificationPtr(1, errors.Wrap(err, "failed to convert file path to UTF16"))
	}

	fileInfo := winTrustFileInfo{
		cbStruct:       uint32(unsafe.Sizeof(winTrustFileInfo{})),
		pcwszFile:      uintptr(unsafe.Pointer(filePathPtr)),
		hFile:          0,
		pgKnownSubject: 0,
	}

	trustData := winTrustData{
		cbStruct:            uint32(unsafe.Sizeof(winTrustData{})),
		pPolicyCallbackData: 0,
		pSIPClientData:      0,
		dwUIChoice:          WTD_UI_NONE,
		fdWRevocationChecks: WTD_REVOKE_NONE,
		dwUnionChoice:       WTD_CHOICE_FILE,
		dwStateAction:       WTD_STATEACTION_VERIFY,
		hWVTStateData:       0,
		pwszURLReference:    0,
		dwProvFlags:         WTD_REVOCATION_CHECK_NONE,
		dwUIContext:         WTD_UICONTEXT_EXECUTE,
	}

	*(*uintptr)(unsafe.Pointer(&trustData.union[0])) = uintptr(unsafe.Pointer(&fileInfo))

	actionID := syscall.GUID{
		Data1: 0x00AAC56B,
		Data2: 0xCD44,
		Data3: 0x11D0,
		Data4: [8]byte{0x8C, 0xC2, 0x00, 0xC0, 0x4F, 0xC2, 0x95, 0xEE},
	}

	status, verifyErr := verifySignatureWinVerifyTrust(0, &actionID, &trustData)

	// Always close provider state.
	trustData.dwStateAction = WTD_STATEACTION_CLOSE
	_, _ = verifySignatureWinVerifyTrust(0, &actionID, &trustData)

	if status != 0 {
		return status, vmextension.NewErrorWithClarificationPtr(int(status), errors.Wrap(verifyErr, "failed to verify file signature"))
	} else {
		return 0, nil
	}
}

func ValidateFileAgainstCatalog(catalogFilePath, fileToVerifyPath string, hashAlgorithm hashutils.HashType) (uint32, *vmextension.ErrorWithClarification) {
	hashfunction, err := hashutils.GetHashAlgorithm(hashAlgorithm)
	if err != nil {
		return 1, vmextension.NewErrorWithClarificationPtr(1, errors.Wrap(err, "failed to get hash algorithm"))
	}

	fileHash, err := hashutils.ComputeFileHash(fileToVerifyPath, hashfunction)
	if err != nil {
		return 1, vmextension.NewErrorWithClarificationPtr(1, errors.Wrap(err, "failed to compute file hash"))
	}
	if len(fileHash) == 0 {
		return 1, vmextension.NewErrorWithClarificationPtr(1, errors.New("calculated file hash is empty"))
	}

	catalogPathPtr, err := syscall.UTF16PtrFromString(catalogFilePath)
	if err != nil {
		return 1, vmextension.NewErrorWithClarificationPtr(1, errors.Wrap(err, "failed to convert catalog file path to UTF16"))
	}
	memberFilePathPtr, err := syscall.UTF16PtrFromString(fileToVerifyPath)
	if err != nil {
		return 1, vmextension.NewErrorWithClarificationPtr(1, errors.Wrap(err, "failed to convert member file path to UTF16"))
	}

	// Catalog member tag is the file hash as uppercase hex.
	memberTag := strings.ToUpper(hex.EncodeToString([]byte(fileHash)))
	memberTagPtr, err := syscall.UTF16PtrFromString(memberTag)
	if err != nil {
		return 1, vmextension.NewErrorWithClarificationPtr(1, errors.Wrap(err, "failed to convert member tag to UTF16"))
	}

	fileHashPtr, err := syscall.UTF16PtrFromString(fileHash)
	if err != nil {
		return 1, vmextension.NewErrorWithClarificationPtr(1, errors.Wrap(err, "failed to convert file hash to UTF16"))
	}

	catalogInfo := winTrustCatalogInfo{
		cbStruct:             uint32(unsafe.Sizeof(winTrustCatalogInfo{})),
		dwCatalogVersion:     0,
		pcwszCatalogFilePath: uintptr(unsafe.Pointer(catalogPathPtr)),
		pcwszMemberTag:       uintptr(unsafe.Pointer(memberTagPtr)),
		pcwszMemberFilePath:  uintptr(unsafe.Pointer(memberFilePathPtr)),
		hMemberFile:          0,
		pbCalculatedFileHash: uintptr(unsafe.Pointer(fileHashPtr)),
		cbCalculatedFileHash: uint32(len(fileHash)),
		pcCatalogContext:     0,
	}

	trustData := winTrustData{
		cbStruct:            uint32(unsafe.Sizeof(winTrustData{})),
		pPolicyCallbackData: 0,
		pSIPClientData:      0,
		dwUIChoice:          WTD_UI_NONE,
		fdWRevocationChecks: WTD_REVOKE_NONE,
		dwUnionChoice:       WTD_CHOICE_CATALOG,
		dwStateAction:       WTD_STATEACTION_VERIFY,
		hWVTStateData:       0,
		pwszURLReference:    0,
		dwProvFlags:         WTD_REVOCATION_CHECK_NONE,
		dwUIContext:         WTD_UICONTEXT_EXECUTE,
	}
	*(*uintptr)(unsafe.Pointer(&trustData.union[0])) = uintptr(unsafe.Pointer(&catalogInfo))

	actionID := syscall.GUID{
		Data1: 0x00AAC56B,
		Data2: 0xCD44,
		Data3: 0x11D0,
		Data4: [8]byte{0x8C, 0xC2, 0x00, 0xC0, 0x4F, 0xC2, 0x95, 0xEE},
	}

	status, verifyErr := verifySignatureWinVerifyTrust(0, &actionID, &trustData)

	// Always close provider state.
	trustData.dwStateAction = WTD_STATEACTION_CLOSE
	_, _ = verifySignatureWinVerifyTrust(0, &actionID, &trustData)

	if status != 0 {
		return status, vmextension.NewErrorWithClarificationPtr(int(status), errors.Wrap(verifyErr, "failed to verify file signature"))
	} else {
		return 0, nil
	}
}
