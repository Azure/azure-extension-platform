// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package crypto

import (
	"crypto/tls"
	"fmt"
	"os"
	"testing"
)

var TestInit = func() { os.Mkdir("./testoutput", 0770) }

func TestCreateSelfSignedCertificate_GeneratesValidCertKeyPair(t *testing.T) {
	TestInit()
	certPath := "./testoutput/Cert.crt"
	keyPath := "./testoutput/private.key"

	cert, err := NewSelfSignedx509Certificate()
	if err != nil {
		t.Fatal(err.Error())
	}
	if err = cert.WriteCertificateToDisk(certPath); err != nil {
		t.Fatal(err.Error())
	}
	if err = cert.WriteKeyToDisk(keyPath); err != nil {
		t.Fatal(err.Error())
	}

	if _, err := tls.LoadX509KeyPair(certPath, keyPath); err != nil {
		t.Fatal(err.Error())
	}

	t.Logf("Certificate thumbprint was %s", cert.GetCertificateThumbprint())
}

func TestCreateSelfSignedCertificate_CertificatesAndKeysAreExclusive(t *testing.T) {
	TestInit()
	certPath1 := "./testoutput/certificate1.crt"
	keyPath1 := "./testoutput/private1.key"
	cert, err := NewSelfSignedx509Certificate()
	if err != nil {
		t.Fatal(err.Error())
	}
	if err = cert.WriteCertificateToDisk(certPath1); err != nil {
		t.Fatal(err.Error())
	}
	if err = cert.WriteKeyToDisk(keyPath1); err != nil {
		t.Fatal(err.Error())
	}

	certPath2 := "./testoutput/certificate2.crt"
	keyPath2 := "./testoutput/private2.key"
	cert2, err := NewSelfSignedx509Certificate()
	if err != nil {
		t.Fatal(err.Error())
	}
	if err = cert2.WriteCertificateToDisk(certPath2); err != nil {
		t.Fatal(err.Error())
	}
	if err = cert2.WriteKeyToDisk(keyPath2); err != nil {
		t.Fatal(err.Error())
	}

	if _, err := tls.LoadX509KeyPair(certPath1, keyPath2); err == nil {
		t.Fatal("Certificate was verified with wrong key")
	} else {
		fmt.Printf("Mismatched Cert and key didn't match as expected: %s\n", err.Error())
	}

	if _, err := tls.LoadX509KeyPair(certPath2, keyPath1); err == nil {
		t.Fatal("Certificate was verified with wrong key")
	} else {
		fmt.Printf("Mismatched Cert and key didn't match as expected: %s\n", err.Error())
	}
}
