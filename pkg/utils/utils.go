// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package utils

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
)

// GetCurrentProcessWorkingDir returns the absolute path of the running process.
func GetCurrentProcessWorkingDir() (string, error) {
	p, err := filepath.Abs(os.Args[0])
	if err != nil {
		return "", err
	}
	return filepath.Dir(p), nil
}

func ComputeFileHash(filePath string, hashAlg hash.Hash) ([]byte, error) {
	// make sure filepath is not empty and file exists
	if filePath == "" {
		return nil, fmt.Errorf("file path cannot be empty")
	}
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist at path: %s", filePath)
	}

	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file for hashing: %w", err)
	}
	defer f.Close()
	// We can stream the file contents to the hasher which is more efficient for large files.
	if _, err := io.Copy(hashAlg, f); err != nil {
		return nil, fmt.Errorf("failed to read file for hashing: %w", err)
	}
	return hashAlg.Sum(nil), nil
}

// ComputeHash computes the hash of a string using the provided hash algorithm.
func ComputeHash(contents string, hashAlg hash.Hash) string {
	var hashStr string
	hashAlg.Write([]byte(contents))
	hash := hashAlg.Sum(nil)
	hashStr = hex.EncodeToString(hash[:])
	return hashStr
}

// map hash algorithm string inputs to actual hasher
func GetHashAlgorithm(hashOpt string) (hash.Hash, error) {
	switch hashOpt {
	case "sha1":
		return sha1.New(), nil
	case "sha256":
		return sha256.New(), nil
	default:
		return nil, fmt.Errorf("unsupported hash type option: %v", hashOpt)
	}
}
