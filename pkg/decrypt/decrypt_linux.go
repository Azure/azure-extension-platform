package decrypt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
)

// decryptProtectedSettings decrypts the read protected settigns using certificates
func DecryptProtectedSettings(configFolder string, thumbprint string, decoded []byte) (map[string]interface{}, error) {
	// go two levels up where certs are placed (/var/lib/waagent)
	crt := filepath.Join(configFolder, "..", "..", fmt.Sprintf("%s.crt", thumbprint))
	prv := filepath.Join(configFolder, "..", "..", fmt.Sprintf("%s.prv", thumbprint))

	// we use os/exec instead of azure-docker-extension/pkg/executil here as
	// other extension handlers depend on this package for parsing handler
	// settings.
	cmd := exec.Command("openssl", "smime", "-inform", "DER", "-decrypt", "-recip", crt, "-inkey", prv)
	var bOut, bErr bytes.Buffer
	cmd.Stdin = bytes.NewReader(decoded)
	cmd.Stdout = &bOut
	cmd.Stderr = &bErr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("decrypting protected settings failed: error=%v stderr=%s", err, string(bErr.Bytes()))
	}

	// decrypted: json object for protected settings
	var v map[string]interface{}
	if err := json.Unmarshal(bOut.Bytes(), &v); err != nil {
		return nil, fmt.Errorf("failed to unmarshal decrypted settings json: %v", err)
	}

	return v, nil
}