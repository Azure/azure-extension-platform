package extensionpolicysettings

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

func ValidateFileSignature(filePath string, signatureFilePath string, certPath string, checkExpired bool) (isValid bool, err error) {
	// Ensure openssl is available
	if _, err := exec.LookPath("openssl"); err != nil {
		return false, fmt.Errorf("openssl is not installed or not found in PATH: %v", err)
	}

	paths := map[string]string{
		"file":           filePath,
		"cert file":      certPath,
		"signature file": signatureFilePath,
	}

	for name, path := range paths {
		if path == "" {
			return false, fmt.Errorf("%s path cannot be empty", name)
		}
		info, err := os.Stat(path)

		if err != nil {
			if os.IsNotExist(err) {
				return false, fmt.Errorf("%s does not exist at path: %s", name, path)
			}
			return false, fmt.Errorf("error accessing %s: %v", name, err)
		}
		// Allow an empty script file.
		if name != "file" && info.Size() == 0 {
			return false, fmt.Errorf("%s at %s is empty", name, path)
		}
	}

	cmd := exec.Command("openssl", "cms", "-verify", "-inform", "DER", "-in", signatureFilePath, "-content", filePath, "-CAfile", certPath, "-verify_retcode", "-no_check_time")

	// Including the -verify_retcode flag will have openssl to exit zero ONLY if validation was successful.
	if checkExpired {
		cmd = exec.Command("openssl", "cms", "-verify", "-inform", "DER", "-in", signatureFilePath, "-content", filePath, "-CAfile", certPath, "-verify_retcode")
	}

	var bOut, bErr bytes.Buffer
	cmd.Stdout = &bOut
	cmd.Stderr = &bErr

	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("signature validation failed: error=%v. stderr=%s", err, bErr.String())
	}
	return true, nil
}
