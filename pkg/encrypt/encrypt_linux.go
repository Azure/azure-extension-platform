package encrypt

import (
	"bytes"
	"fmt"
	"github.com/Azure/azure-extension-platform/pkg/internal/crypto"
	"os/exec"
	"path"
	"path/filepath"
)

type LinuxCertificateHandler struct {
	CertOperations crypto.CertificateOperations
	certLcoation   string
}

func (ch *LinuxCertificateHandler) GetThumbprint() (certThumbprint string, err error) {
	return ch.CertOperations.GetCertificateThumbprint(), nil
}

func (ch *LinuxCertificateHandler) Encrypt(bytesToEncrypt []byte) (encryptedBytes []byte, err error) {
	thumbprint, err := ch.GetThumbprint()
	if err != nil {
		return nil, err
	}
	crt := filepath.Join(ch.certLcoation, fmt.Sprintf("%s.crt", thumbprint))

	// we use os/exec instead of azure-docker-extension/pkg/executil here as
	// other extension handlers depend on this package for parsing handler
	// settings.
	cmd := exec.Command("openssl", "smime", "-outform", "DER", "-encrypt", crt)
	var bOut, bErr bytes.Buffer
	cmd.Stdin = bytes.NewReader(bytesToEncrypt)
	cmd.Stdout = &bOut
	cmd.Stderr = &bErr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("encryption failed: error=%v stderr=%s", err, string(bErr.Bytes()))
	}
	return bOut.Bytes(), nil
}

func newCertHandler(certLocation string) (ICertHandler, error) {
	cert, err := crypto.NewSelfSignedx509Certificate()
	if err != nil {
		return nil, err
	}
	thumbprint := cert.GetCertificateThumbprint()

	certFilePath := path.Join(certLocation, fmt.Sprintf("%s.crt", thumbprint))
	keyFilePath := path.Join(certLocation, fmt.Sprintf("%s.prv", thumbprint))
	err = cert.WriteCertificateToDisk(certFilePath)
	if err != nil {
		return nil, err
	}
	err = cert.WriteKeyToDisk(keyFilePath)
	if err != nil {
		return nil, err
	}
	return &LinuxCertificateHandler{CertOperations: cert, certLcoation: certLocation}, nil
}
