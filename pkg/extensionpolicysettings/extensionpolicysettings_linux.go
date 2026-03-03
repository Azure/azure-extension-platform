package extensionpolicysettings

import (
	"bytes"
	"fmt"
	"os/exec"
)

// declare function (obviously)
// write the passed-in signature to a file
// get the cert location (assuming cert gonna be passed in )

//create base command for open ssl

// check and see if c groups are enabled; if so , scope the command using systemd just in case
// if it fails, then a) disable c groups and b)just run the comman directly.

func ValidateFileSignature(filePath string, signature []byte, certPath string) (isValid bool, err error) {
	// os.eexec
	cmd := exec.Command("openssl", "cms", "-verify", "-content", filePath, "-certfile", certPath, "-signature", "-")

	cmd.Stdin = bytes.NewReader(signature)
	var bOut, bErr bytes.Buffer
	cmd.Stdout = &bOut
	cmd.Stderr = &bErr

	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("signature validation failed: error=%v stderr=%s", err, string(bErr.Bytes()))
	}
	return true, nil
}
