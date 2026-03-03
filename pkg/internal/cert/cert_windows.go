package cert

import (
	"syscall"
)

const (
	WINTRUST_ACTION_GENERIC_VERIFY_V2_GUID = "00AAC56B-CD44-11d0-8CC2-00C04FC295EE"
)

var (
	Modwintrust        = syscall.NewLazyDLL("wintrust.dll")
	procWinVerifyTrust = Modwintrust.NewProc("WinVerifyTrust")
)

type WTD_UI uint

const (
	WTD_UI_ALL    WTD_UI = 1
	WTD_UI_NONE   WTD_UI = 2
	WTD_UI_NOBAD  WTD_UI = 3
	WTD_UI_NOGOOD WTD_UI = 4
)

type WTD_REVOKE_FLAGS uint

const (
	WTD_REVOKE_NONE       WTD_REVOKE_FLAGS = 0x00000000
	WTD_REVOKE_WHOLECHAIN WTD_REVOKE_FLAGS = 0x00000001
)

type WTD_CHOICE uint

const (
	WTD_CHOICE_FILE    WTD_CHOICE = 1
	WTD_CHOICE_CATALOG WTD_CHOICE = 2
	WTD_CHOICE_BLOB    WTD_CHOICE = 3
	WTD_CHOICE_SIGNER  WTD_CHOICE = 4
	WTD_CHOICE_CERT    WTD_CHOICE = 5
)

type WTD_STATE_ACTION uint

const (
	WTD_STATEACTION_IGNORE           WTD_STATE_ACTION = 0x00000000
	WTD_STATEACTION_VERIFY           WTD_STATE_ACTION = 0x00000001
	WTD_STATEACTION_CLOSE            WTD_STATE_ACTION = 0x00000002
	WTD_STATEACTION_AUTO_CACHE       WTD_STATE_ACTION = 0x00000003
	WTD_STATEACTION_AUTO_CACHE_FLUSH WTD_STATE_ACTION = 0x00000004
)

type WTD_PROVIDER_FLAGS uint

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

type WTD_UICONTEXT uint

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
	cbStruct         uint32
	pcwszCatalogFile uintptr
	pcwszMemberTag   uintptr
	hMemberFile      syscall.Handle
	pgKnownSubject   uintptr
}
