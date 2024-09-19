// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package decrypt

import (
	"bytes"
	"fmt"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
)

var getCertificateDir = func(configFolder string) (certificateFolder string) {
	return path.Join(configFolder, "..", "..")
}

// decryptProtectedSettings decrypts the read protected settigns using certificates
func DecryptProtectedSettings(configFolder string, thumbprint string, decoded []byte) (string, error) {
	// go two levels up where certs are placed (/var/lib/waagent)
	crt := filepath.Join(getCertificateDir(configFolder), fmt.Sprintf("%s.crt", thumbprint))
	prv := filepath.Join(getCertificateDir(configFolder), fmt.Sprintf("%s.prv", thumbprint))

	// we use os/exec instead of azure-docker-extension/pkg/executil here as
	// other extension handlers depend on this package for parsing handler
	// settings.

	//using cms command to support for FIPS 140-3
	cmd := exec.Command("openssl", "cms", "-inform", "DER", "-decrypt", "-recip", crt, "-inkey", prv)
	var bOut, bErr bytes.Buffer
	var errMsg error
	cmd.Stdin = bytes.NewReader(decoded)
	cmd.Stdout = &bOut
	cmd.Stderr = &bErr

	//back up smime command in case cms fails
	if err := cmd.Run(); err != nil {
		errMsg = fmt.Errorf("decrypting protected settings with cms command failed: error=%v stderr=%s \n now decrypting with smime command", err, bErr.String())
		cmd = exec.Command("openssl", "smime", "-inform", "DER", "-decrypt", "-recip", crt, "-inkey", prv)
		cmd.Stdin = bytes.NewReader(decoded)
		bOut.Reset()
		bErr.Reset()
		cmd.Stdout = &bOut
		cmd.Stderr = &bErr
		if err := cmd.Run(); err != nil {
			return "", errors.Wrapf(errMsg, "decrypting protected settings with smime command failed: error=%v stderr=%s", err, bErr.String())
		}
	}

	v := bOut.String()
	return v, nil
}
