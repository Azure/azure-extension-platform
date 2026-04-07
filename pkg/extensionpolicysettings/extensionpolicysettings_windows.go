package extensionpolicysettings

import (
	"fmt"
	"os"

	"github.com/Azure/azure-extension-platform/pkg/internal/catalog"
)

func ValidateFileSignature(filePath string) (isValid bool, err error) {

	if filePath == "" {
		return false, fmt.Errorf("file path cannot be empty")
	}
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			return false, fmt.Errorf("file does not exist at path: %s", filePath)
		}
		return false, fmt.Errorf("error accessing file: %v", err)
	}

	status, verifyErr := catalog.VerifyFileSignature(filePath)
	if status != 0 {
		if verifyErr != nil {
			return false, fmt.Errorf("signature verification failed for %s (status: 0x%08X): %v", filePath, status, verifyErr)
		}
		return false, fmt.Errorf("signature verification failed for %s (status: 0x%08X)", filePath, status)
	}

	return true, nil

}
