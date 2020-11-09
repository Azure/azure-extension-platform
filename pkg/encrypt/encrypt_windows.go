package encrypt

import (
	"encoding/hex"
	"fmt"
	"github.com/Azure/azure-extension-platform/pkg/internal/crypto"
	"golang.org/x/sys/windows"
	"syscall"
	"time"
	"unsafe"
)

const (
	certNCryptKeySpec = 0xFFFFFFFF
	szOID_RSA_RC4     = "1.2.840.113549.3.4"
)

var (
	procCryptAcquireCertificatePrivateKey = crypto.Modcrypt32.NewProc("CryptAcquireCertificatePrivateKey")
	procNCryptFreeObject                  = crypto.Modcrypt32.NewProc("NCryptFreeObject")
)

type certHandler struct {
	certContext *syscall.CertContext
}

func (cHandler *certHandler) GetThumbprint()(certThumbprint string, err error){
	thumbprintHex, err := crypto.GetCertificateThumbprint(cHandler.certContext)
	if err != nil {
		return "", err
	}
	certThumbprint = hex.EncodeToString(thumbprintHex)
	return
}

func (cHandler *certHandler)  Encrypt(bytesToEncrypt []byte)( encryptedBytes []byte, err error){
	alg := szOID_RSA_RC4
	buffer := []byte(alg)
	procCryptEncryptMessage := crypto.Modcrypt32.NewProc("CryptEncryptMessage")
	cai := cryptAlgorithmIdentifier{
		pszObjID: uintptr(unsafe.Pointer(&buffer[0])),
		parameters: cryptObjectIDBlob{
			cbData: uint32(0),
			pbData: uintptr(0),
		},
	}

	cemp := cryptEncryptMessagePara{
		cbSize:                     uint32(0),
		dwMsgEncodingType:          uint32(windows.X509_ASN_ENCODING | windows.PKCS_7_ASN_ENCODING),
		hCryptProv:                 uint32(0),
		ContentEncryptionAlgorithm: cai,
		pvEncryptionAuxInfo:        uintptr(0),
		dwFlags:                    uint32(0),
		dwInnerContentType:         uint32(0),
	}
	cemp.cbSize = uint32(unsafe.Sizeof(cemp))

	// Call the first time to get the size
	var pbToBeEncrypted *byte
	var cbEncryptedBlob uint32
	pbToBeEncrypted = &bytesToEncrypt[0]
	ret, _, err := syscall.Syscall9(
		procCryptEncryptMessage.Addr(),
		7,
		uintptr(unsafe.Pointer(&cemp)),                 //pEncryptPara,
		uintptr(1),                                     // cRecipientCert,
		uintptr(unsafe.Pointer(&cHandler.certContext)), // rgpRecipientCert,
		uintptr(unsafe.Pointer(pbToBeEncrypted)),       // *pbToBeEncrypted,
		uintptr(len(bytesToEncrypt)),                   // cbToBeEncrypted,
		uintptr(0),                                     // *pbEncryptedBlob,
		uintptr(unsafe.Pointer(&cbEncryptedBlob)),      // *pcbEncryptedBlob,
		0,
		0)

	if ret == 0 {
		return nil, fmt.Errorf("CryptEncryptMessage failed due to '%v'", err)
	}

	// Build the buffer
	if  cbEncryptedBlob <= 0{
		return nil, fmt.Errorf("the count of encrypted bytes was 0")
	}
	encryptedBytes = make([]byte, cbEncryptedBlob)
	var pencryptedBytes *byte
	pencryptedBytes = &encryptedBytes[0]

	// Perform the encryption
	ret, _, err = syscall.Syscall9(
		procCryptEncryptMessage.Addr(),
		7,
		uintptr(unsafe.Pointer(&cemp)),                 // pEncryptPara,
		uintptr(1),                                     // cRecipientCert,
		uintptr(unsafe.Pointer(&cHandler.certContext)), // rgpRecipientCert,
		uintptr(unsafe.Pointer(pbToBeEncrypted)),       // *pbToBeEncrypted,
		uintptr(len(bytesToEncrypt)),                   // cbToBeEncrypted,
		uintptr(unsafe.Pointer(pencryptedBytes)),       // *pbEncryptedBlob,
		uintptr(unsafe.Pointer(&cbEncryptedBlob)),      // *pcbEncryptedBlob,
		0,
		0)

	if ret == 0 {
		return nil, fmt.Errorf("CryptEncryptMessage failed due to '%v'", err)
	}

	return encryptedBytes, nil
}

func newCertHandler()(ICertHandler, error){
	handle, err := syscall.CertOpenStore(windows.CERT_STORE_PROV_SYSTEM, 0, 0, windows.CERT_SYSTEM_STORE_LOCAL_MACHINE, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("MY"))))
	defer syscall.CertCloseStore(handle, 0)

	// Due to the trickyness of creating our own cert, we'll just pick a cert and then check
	// if what we get back looks like a thumbprint
	cert, err := getAUsableCert(handle)
	return &certHandler{certContext: cert}, err
}

type cryptEncryptMessagePara struct {
	cbSize                     uint32
	dwMsgEncodingType          uint32
	hCryptProv                 uint32
	ContentEncryptionAlgorithm cryptAlgorithmIdentifier
	pvEncryptionAuxInfo        uintptr
	dwFlags                    uint32
	dwInnerContentType         uint32
}

type cryptAlgorithmIdentifier struct {
	pszObjID   uintptr
	parameters cryptObjectIDBlob
}

type cryptIntegerBlob struct {
	cbData uint32
	pbData uintptr
}

type cryptObjectIDBlob struct {
	cbData uint32
	pbData uintptr
}

type cryptBitBlob struct {
	cbData      uint32
	pbData      uintptr
	cUnusedBits uint32
}

type certNameBlob struct {
	cbData uint32
	pbData uintptr
}

type certPublicKeyInfo struct {
	Algorithm cryptAlgorithmIdentifier
	PublicKey cryptBitBlob
}

// This struct is not implemented in syscall, so we need to do this ourselves
type certInfo struct {
	dwVersion            uint32
	SerialNumber         cryptIntegerBlob
	SignatureAlgorithm   cryptAlgorithmIdentifier
	Issuer               certNameBlob
	NotBefore            syscall.Filetime
	NotAfter             syscall.Filetime
	Subject              certNameBlob
	SubjectPublicKeyInfo certPublicKeyInfo
	IssuerUniqueID       cryptBitBlob
	SubjectUniqueID      cryptBitBlob
	cExtension           uint32
	rgExtension          uintptr
}


type certContext struct {
	EncodingType uint32
	EncodedCert  *byte
	Length       uint32
	CertInfo     *certInfo
	Store        syscall.Handle
}

// We look for a cert with the following
// - Not expired
// - Has a private key
// Note that the dev code uses syscall.CertContext. However that doesn't have the CERT_INFO
// structure, so we need to find the cert manually, then convert it to the syscall structure
func getAUsableCert( handle syscall.Handle) (cert *syscall.CertContext, _ error) {
	var testCert *certContext
	var prevCert *certContext
	procCertEnumCertificatesInStore := crypto.Modcrypt32.NewProc("CertEnumCertificatesInStore")

	for {
		ret, _, _ := syscall.Syscall(
			procCertEnumCertificatesInStore.Addr(),
			2,
			uintptr(handle),
			uintptr(unsafe.Pointer(prevCert)),
			0)

		// Not that we don't handle ENotFound, since that's an error case for us (we couldn't find a cert)
		testCert = (*certContext)(unsafe.Pointer(ret))
		usable := isAUsableCert(testCert)
		if usable {
			// We need a syscall.CertContext
			syscallContext := (*syscall.CertContext)(unsafe.Pointer(ret))
			return syscallContext, nil
		}

		prevCert = testCert
	}
}

func isAUsableCert(cert *certContext) (usable bool) {
	// First check if the cert has expired
	ended := time.Unix(0, cert.CertInfo.NotAfter.Nanoseconds())
	started := time.Unix(0, cert.CertInfo.NotBefore.Nanoseconds())
	now := time.Now()
	if now.After(ended) || now.Before(started) {
		return false
	}

	// Check that it has a private key
	if !hasPrivateKey(cert) {
		return false
	}

	return true
}

func hasPrivateKey(cert *certContext) bool {
	var ncryptKeyHandle uintptr
	var dwKeySpec uint32
	var fCallerFreeProvOrNCryptKey uint32
	ret, _, err := syscall.Syscall6(
		procCryptAcquireCertificatePrivateKey.Addr(),
		6,
		uintptr(unsafe.Pointer(cert)),
		uintptr(0),
		uintptr(0),
		uintptr(unsafe.Pointer(&ncryptKeyHandle)),
		uintptr(unsafe.Pointer(&dwKeySpec)),
		uintptr(unsafe.Pointer(&fCallerFreeProvOrNCryptKey)))
	if ret == 0 {
		if err > 0 {
			// If for some reason we can't retrieve the private key, move on
			return false
		}
	}

	// Figure out if we need to release the handle
	if fCallerFreeProvOrNCryptKey != 0 {
		if dwKeySpec == certNCryptKeySpec {
			// We received an CERT_NCRYPT_KEY_SPEC
			syscall.Syscall(
				procNCryptFreeObject.Addr(),
				1,
				uintptr(ncryptKeyHandle),
				0,
				0)
		} else {
			handle := syscall.Handle(ncryptKeyHandle)
			syscall.CryptReleaseContext(handle, 0)
		}
	}

	return true
}
