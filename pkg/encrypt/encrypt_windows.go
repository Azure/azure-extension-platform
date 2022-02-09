// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package encrypt

import (
	"encoding/hex"
	"fmt"
	"github.com/Azure/azure-extension-platform/pkg/internal/crypto"
	"golang.org/x/sys/windows"
	"syscall"
	"unsafe"
)

const (
	szOID_RSA_RC4 = "1.2.840.113549.3.4"
)

type certHandler struct {
	certContext *syscall.CertContext
}

func (cHandler *certHandler) GetThumbprint() (certThumbprint string, err error) {
	thumbprintHex, err := crypto.GetCertificateThumbprint(cHandler.certContext)
	if err != nil {
		return "", err
	}
	certThumbprint = hex.EncodeToString(thumbprintHex)
	return
}

func (cHandler *certHandler) Encrypt(bytesToEncrypt []byte) (encryptedBytes []byte, err error) {
	alg := szOID_RSA_RC4
	buffer := []byte(alg)
	procCryptEncryptMessage := crypto.Modcrypt32.NewProc("CryptEncryptMessage")
	cai := crypto.CryptAlgorithmIdentifier{
		PszObjID: uintptr(unsafe.Pointer(&buffer[0])),
		Parameters: crypto.CryptObjectIDBlob{
			CbData: uint32(0),
			PbData: uintptr(0),
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
		uintptr(unsafe.Pointer(&cemp)), //pEncryptPara,
		uintptr(1),                     // cRecipientCert,
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
	if cbEncryptedBlob <= 0 {
		return nil, fmt.Errorf("the count of encrypted bytes was 0")
	}
	encryptedBytes = make([]byte, cbEncryptedBlob)
	var pencryptedBytes *byte
	pencryptedBytes = &encryptedBytes[0]

	// Perform the encryption
	ret, _, err = syscall.Syscall9(
		procCryptEncryptMessage.Addr(),
		7,
		uintptr(unsafe.Pointer(&cemp)), // pEncryptPara,
		uintptr(1),                     // cRecipientCert,
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

func newCertHandler(certLocation string) (ICertHandler, error) {
	handle, err := syscall.CertOpenStore(windows.CERT_STORE_PROV_SYSTEM, 0, 0, windows.CERT_SYSTEM_STORE_LOCAL_MACHINE, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("MY"))))
	defer syscall.CertCloseStore(handle, 0)

	// Due to the trickyness of creating our own cert, we'll just pick a cert and then check
	// if what we get back looks like a thumbprint
	cert, err := crypto.GetAUsableCert(handle)
	return &certHandler{certContext: cert}, err
}

type cryptEncryptMessagePara struct {
	cbSize                     uint32
	dwMsgEncodingType          uint32
	hCryptProv                 uint32
	ContentEncryptionAlgorithm crypto.CryptAlgorithmIdentifier
	pvEncryptionAuxInfo        uintptr
	dwFlags                    uint32
	dwInnerContentType         uint32
}
