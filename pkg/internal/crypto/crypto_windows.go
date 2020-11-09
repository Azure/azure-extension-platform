package crypto

import (
	"fmt"
	"syscall"
	"unsafe"
)

const (
	CertHashPropID    = 3
	CrypteEAsn1BadTag = 2148086027
)

var (
	Modcrypt32                            = syscall.NewLazyDLL("crypt32.dll")
	ProcCertGetCertificateContextProperty = Modcrypt32.NewProc("CertGetCertificateContextProperty")
	ProcCryptDecryptMessage               = Modcrypt32.NewProc("CryptDecryptMessage")
)


func GetCertificateThumbprint(cert *syscall.CertContext) ([]byte, error) {
	// Call it once to retrieve the thumbprint size
	var cbComputedHash uint32
	ret, _, err := syscall.Syscall6(
		ProcCertGetCertificateContextProperty.Addr(),
		4,
		uintptr(unsafe.Pointer(cert)),            // pCertContext
		uintptr(CertHashPropID),                  // dwPropId
		uintptr(0),                               // pvData)
		uintptr(unsafe.Pointer(&cbComputedHash)), // pcbData
		0,
		0,
	)

	if ret == 0 {
		return nil, fmt.Errorf("VmExtension: Could not hash certificate due to '%d'", syscall.Errno(err))
	}

	// Create our buffer
	if cbComputedHash == 0 {
		return nil, nil
	}

	var computedHashBuffer = make([]byte, cbComputedHash)
	var pComputedHash *byte
	pComputedHash = &computedHashBuffer[0]
	ret, _, err = syscall.Syscall6(
		ProcCertGetCertificateContextProperty.Addr(),
		4,
		uintptr(unsafe.Pointer(cert)),            // pCertContext
		uintptr(CertHashPropID),                  // dwPropId
		uintptr(unsafe.Pointer(pComputedHash)),   // pvData)
		uintptr(unsafe.Pointer(&cbComputedHash)), // pcbData
		0,
		0,
	)
	if ret == 0 {
		return nil, fmt.Errorf("VmExtension: Could not hash certificate due to '%d'", syscall.Errno(err))
	}

	return computedHashBuffer, nil
}
