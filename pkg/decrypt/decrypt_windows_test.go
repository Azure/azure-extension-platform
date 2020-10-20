package decrypt

import (
	"encoding/hex"
	"github.com/Azure/VMApplication-Extension/VmExtensionHelper/extensionerrors"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

const (
	certNCryptKeySpec    = 0xFFFFFFFF
	szOID_RSA_RC4        = "1.2.840.113549.3.4"
)

var (
	procCryptAcquireCertificatePrivateKey = modcrypt32.NewProc("CryptAcquireCertificatePrivateKey")
	procNCryptFreeObject                  = modcrypt32.NewProc("NCryptFreeObject")
)

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
type testCertInfo struct {
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

type testCertContext struct {
	EncodingType uint32
	EncodedCert  *byte
	Length       uint32
	CertInfo     *testCertInfo
	Store        syscall.Handle
}

func Test_getCertificateThumbprint(t *testing.T) {
	_, thumbprint, err := getCertAndThumbprint(t)
	require.NoError(t, err, "getCertificateThumbprint failed")
	require.True(t, len(thumbprint) == 40)
}

func Test_decryptSettingsCertNotFound(t *testing.T) {
	invalidCert := "123456790abcdefedcba09876543210123456789"
	decoded := make([]byte, 5) // We'll never process this because the cert is wrong

	_, err := DecryptProtectedSettings("", invalidCert, decoded)
	require.Error(t, err, extensionerrors.ErrCertWithThumbprintNotFound)
}

func Test_decryptSettingsMisencoded(t *testing.T) {
	serialized := getTestData()
	cert, thumbprint, err := getCertAndThumbprint(t)
	require.NoError(t, err, "getCertificateThumbprint failed")
	encrypted, err := encryptTestData(t, serialized, cert)
	require.NoError(t, err, "encryptTestData failed")

	// Mess with the encrypted data
	encrypted[0] = 5
	encrypted[1] = 3

	_, err = DecryptProtectedSettings("", thumbprint, encrypted)
	require.Error(t, err, extensionerrors.ErrInvalidProtectedSettingsData)
}

func Test_decryptProtectedSettings(t *testing.T) {
	serialized := getTestData()
	cert, thumbprint, err := getCertAndThumbprint(t)
	require.NoError(t, err, "getCertificateThumbprint failed")
	encrypted, err := encryptTestData(t, serialized, cert)
	require.NoError(t, err, "encryptTestData failed")

	v, err := DecryptProtectedSettings("", thumbprint, encrypted)
	require.NoError(t, err, "decryptProtectedSettings failed")
	landMammal, ok := v["AfricanLandMammal"].(string)
	require.True(t, ok, "African land mammal is not OK")
	chipmunk, ok := v["ChipmunkType"].(string)
	require.True(t, ok, "Chipmunk is not OK")
	number, ok := v["InterestingNumber"].(string)
	require.True(t, ok, "Number is not OK")
	require.Equal(t, "cheetah", landMammal)
	require.Equal(t, "Townsends", chipmunk)
	require.Equal(t, "42", number)
}

func getCertAndThumbprint(t *testing.T) (*syscall.CertContext, string, error) {
	handle, err := syscall.CertOpenStore(windows.CERT_STORE_PROV_SYSTEM, 0, 0, windows.CERT_SYSTEM_STORE_LOCAL_MACHINE, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("MY"))))
	require.False(t, int(handle) == 0, "This test must run as admin")
	require.NoError(t, err, "CertOpenStore failed")
	defer syscall.CertCloseStore(handle, 0)

	// Due to the trickyness of creating our own cert, we'll just pick a cert and then check
	// if what we get back looks like a thumbprint
	cert, err := getAUsableCert(t, handle)
	require.NoError(t, err, "getUsableCert failed")

	thumbprint, err := getCertificateThumbprint(cert)
	require.NoError(t, err, "getCertificateThumbprint failed")
	encodedThumbprint := hex.EncodeToString(thumbprint)

	return cert, encodedThumbprint, err
}

// We look for a cert with the following
// - Not expired
// - Has a private key
// Note that the dev code uses syscall.CertContext. However that doesn't have the CERT_INFO
// structure, so we need to find the cert manually, then convert it to the syscall structure
func getAUsableCert(t *testing.T, handle syscall.Handle) (cert *syscall.CertContext, _ error) {
	var testCert *testCertContext
	var prevCert *testCertContext
	procCertEnumCertificatesInStore := modcrypt32.NewProc("CertEnumCertificatesInStore")

	for {
		ret, _, _ := syscall.Syscall(
			procCertEnumCertificatesInStore.Addr(),
			2,
			uintptr(handle),
			uintptr(unsafe.Pointer(prevCert)),
			0)

		// Not that we don't handle ENotFound, since that's an error case for us (we couldn't find a cert)
		require.False(t, ret == 0, "CertEnumCertificatesInStore failed")
		testCert = (*testCertContext)(unsafe.Pointer(ret))
		usable := isAUsableCert(t, testCert)
		if usable {
			// We need a syscall.CertContext
			syscallContext := (*syscall.CertContext)(unsafe.Pointer(ret))
			return syscallContext, nil
		}

		prevCert = testCert
	}
}

func isAUsableCert(t *testing.T, cert *testCertContext) (usable bool) {
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

func getOIDFromPtr(pszPtr uintptr) string {
	// First get the length of our string
	length := 0
	for {
		c := *(*byte)(unsafe.Pointer(uintptr(pszPtr) + uintptr(length)))
		if c == 0 {
			break
		}

		length++
	}

	// Now build our buffer and populate it
	buffer := make([]byte, length)
	for i := range buffer {
		buffer[i] = *(*byte)(unsafe.Pointer(uintptr(pszPtr) + uintptr(i)))
	}

	return string(buffer)
}

func hasPrivateKey(cert *testCertContext) bool {
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

func encryptTestData(t *testing.T, testData string, cert *syscall.CertContext) ([]byte, error) {
	bytesToEncrypt := []byte(testData)

	alg := szOID_RSA_RC4
	buffer := []byte(alg)
	procCryptEncryptMessage := modcrypt32.NewProc("CryptEncryptMessage")
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
		uintptr(unsafe.Pointer(&cemp)),           //pEncryptPara,
		uintptr(1),                               // cRecipientCert,
		uintptr(unsafe.Pointer(&cert)),           // rgpRecipientCert,
		uintptr(unsafe.Pointer(pbToBeEncrypted)), // *pbToBeEncrypted,
		uintptr(len(bytesToEncrypt)),             // cbToBeEncrypted,
		uintptr(0),                               // *pbEncryptedBlob,
		uintptr(unsafe.Pointer(&cbEncryptedBlob)), // *pcbEncryptedBlob,
		0,
		0)
	require.False(t, ret == 0, "CryptEncryptMessage failed due to '%v'", err)

	// Build the buffer
	require.True(t, cbEncryptedBlob > 0)
	var encryptedBytes = make([]byte, cbEncryptedBlob)
	var pencryptedBytes *byte
	pencryptedBytes = &encryptedBytes[0]

	// Perform the encryption
	ret, _, err = syscall.Syscall9(
		procCryptEncryptMessage.Addr(),
		7,
		uintptr(unsafe.Pointer(&cemp)),            //pEncryptPara,
		uintptr(1),                                // cRecipientCert,
		uintptr(unsafe.Pointer(&cert)),            // rgpRecipientCert,
		uintptr(unsafe.Pointer(pbToBeEncrypted)),  // *pbToBeEncrypted,
		uintptr(len(bytesToEncrypt)),              // cbToBeEncrypted,
		uintptr(unsafe.Pointer(pencryptedBytes)),  // *pbEncryptedBlob,
		uintptr(unsafe.Pointer(&cbEncryptedBlob)), // *pcbEncryptedBlob,
		0,
		0)
	require.False(t, ret == 0, "CryptEncryptMessage failed due to '%v'", err)

	return encryptedBytes, nil
}

// To avoid serialization hassles, since Go adds annoying escapes when it serializes json
// we just manually deserialize here, since we're testing the dev code - not our test encryption code
func getTestData() string {
	testData := "{\"AfricanLandMammal\":\"cheetah\",\"InterestingNumber\":\"42\",\"ChipmunkType\":\"Townsends\"}"
	return testData
}
