package hashutils

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
)

type HashType int

const (
	HashTypeNone   HashType = 0
	HashTypeSHA1   HashType = 1
	HashTypeSHA256 HashType = 2
)

func GetHashAlgorithm(hashOpt HashType) (hash.Hash, error) {
	switch hashOpt {
	case HashTypeSHA1:
		return sha1.New(), nil
	case HashTypeSHA256:
		return sha256.New(), nil
	default:
		return nil, fmt.Errorf("unsupported hash type option: %v", hashOpt)
	}
}

// This is a separate function from ComputeHash because streaming the file contents into the hasher is
// more efficient than reading the entire file into memory at once, especially for larger files.
func ComputeFileHash(filePath string, hashAlg hash.Hash) (string, error) {
	// make sure filepath is not empty and file exists
	if filePath == "" {
		return "", fmt.Errorf("file path cannot be empty")
	}
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("file does not exist at path: %s", filePath)
	}

	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file for hashing: %w", err)
	}
	defer f.Close()
	// We can stream the file contents to the hasher which is more efficient for large files.
	if _, err := io.Copy(hashAlg, f); err != nil {
		return "", fmt.Errorf("failed to read file for hashing: %w", err)
	}

	hash := hashAlg.Sum(nil)
	hashStr := hex.EncodeToString(hash[:])

	return hashStr, nil
}

func ComputeHash(contents string, hashAlg hash.Hash) string {
	var hashStr string
	hashAlg.Write([]byte(contents))
	hash := hashAlg.Sum(nil)
	hashStr = hex.EncodeToString(hash[:])
	return hashStr
}
