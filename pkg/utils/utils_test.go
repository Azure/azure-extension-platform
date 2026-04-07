// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package utils

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// --- ComputeFileHash tests ---

func TestComputeFileHash_Success(t *testing.T) {
	content := []byte("hello world")
	tmpFile, err := os.CreateTemp(t.TempDir(), "hash_test_*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	if _, err := tmpFile.Write(content); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	hashAlg := sha256.New()
	got, err := ComputeFileHash(tmpFile.Name(), hashAlg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("expected non-empty hash result")
	}

	// Verify hash matches expected
	expected := sha256.Sum256(content)
	if hex.EncodeToString(got) != hex.EncodeToString(expected[:]) {
		t.Errorf("hash mismatch: got %x, want %x", got, expected)
	}
}

func TestComputeFileHash_EmptyFilePath(t *testing.T) {
	hashAlg := sha256.New()
	_, err := ComputeFileHash("", hashAlg)
	if err == nil {
		t.Fatal("expected error for empty file path, got nil")
	}
	if err.Error() != "file path cannot be empty" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestComputeFileHash_FileDoesNotExist(t *testing.T) {
	hashAlg := sha256.New()
	nonExistentPath := filepath.Join(t.TempDir(), "nonexistent_file.txt")
	_, err := ComputeFileHash(nonExistentPath, hashAlg)
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}
}

func TestComputeFileHash_FileNotReadable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission-based test not reliable on Windows")
	}

	tmpFile, err := os.CreateTemp(t.TempDir(), "no_read_*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.WriteString("some content")
	tmpFile.Close()

	// Remove read permission
	if err := os.Chmod(tmpFile.Name(), 0000); err != nil {
		t.Fatalf("failed to change file permissions: %v", err)
	}
	t.Cleanup(func() {
		os.Chmod(tmpFile.Name(), 0644) // restore for cleanup
	})

	hashAlg := sha256.New()
	_, err = ComputeFileHash(tmpFile.Name(), hashAlg)
	if err == nil {
		t.Fatal("expected error for unreadable file, got nil")
	}
}

func TestComputeFileHash_EmptyFile(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "empty_*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.Close()

	hashAlg := sha256.New()
	got, err := ComputeFileHash(tmpFile.Name(), hashAlg)
	if err != nil {
		t.Fatalf("unexpected error for empty file: %v", err)
	}

	expected := sha256.Sum256([]byte{})
	if hex.EncodeToString(got) != hex.EncodeToString(expected[:]) {
		t.Errorf("hash mismatch for empty file: got %x, want %x", got, expected)
	}
}

func TestComputeFileHash_DifferentAlgorithm(t *testing.T) {
	content := []byte("test content")
	tmpFile, err := os.CreateTemp(t.TempDir(), "hash_md5_*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.Write(content)
	tmpFile.Close()

	hashAlg := md5.New()
	got, err := ComputeFileHash(tmpFile.Name(), hashAlg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := md5.Sum(content)
	if hex.EncodeToString(got) != hex.EncodeToString(expected[:]) {
		t.Errorf("md5 hash mismatch: got %x, want %x", got, expected)
	}
}

// --- ComputeHash tests ---

func TestComputeHash_Success(t *testing.T) {
	input := "hello world"
	hashAlg := sha256.New()
	got := ComputeHash(input, hashAlg)
	if got == "" {
		t.Fatal("expected non-empty hash string")
	}

	expected := sha256.Sum256([]byte(input))
	if got != hex.EncodeToString(expected[:]) {
		t.Errorf("hash mismatch: got %s, want %s", got, hex.EncodeToString(expected[:]))
	}
}

func TestComputeHash_EmptyString(t *testing.T) {
	hashAlg := sha256.New()
	got := ComputeHash("", hashAlg)
	if got == "" {
		t.Fatal("expected non-empty hash string even for empty input")
	}

	expected := sha256.Sum256([]byte(""))
	if got != hex.EncodeToString(expected[:]) {
		t.Errorf("hash mismatch for empty string: got %s, want %s", got, hex.EncodeToString(expected[:]))
	}
}

func TestComputeHash_DifferentInputsDifferentHashes(t *testing.T) {
	hash1 := ComputeHash("input1", sha256.New())
	hash2 := ComputeHash("input2", sha256.New())
	if hash1 == hash2 {
		t.Error("expected different hashes for different inputs")
	}
}

func TestComputeHash_SameInputSameHash(t *testing.T) {
	input := "consistent input"
	hash1 := ComputeHash(input, sha256.New())
	hash2 := ComputeHash(input, sha256.New())
	if hash1 != hash2 {
		t.Errorf("expected same hash for same input, got %s and %s", hash1, hash2)
	}
}

func TestComputeHash_DifferentAlgorithm(t *testing.T) {
	input := "test"
	sha256Hash := ComputeHash(input, sha256.New())
	md5Hash := ComputeHash(input, md5.New())

	if sha256Hash == md5Hash {
		t.Error("expected different hashes for different algorithms")
	}
	if len(sha256Hash) != 64 {
		t.Errorf("expected SHA-256 hex string length 64, got %d", len(sha256Hash))
	}
	if len(md5Hash) != 32 {
		t.Errorf("expected MD5 hex string length 32, got %d", len(md5Hash))
	}
}

// --- GetCurrentProcessWorkingDir tests ---

func TestGetCurrentProcessWorkingDir_ReturnsNonEmptyPath(t *testing.T) {
	dir, err := GetCurrentProcessWorkingDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dir == "" {
		t.Fatal("expected non-empty working directory")
	}
}

func TestGetCurrentProcessWorkingDir_ReturnsAbsolutePath(t *testing.T) {
	dir, err := GetCurrentProcessWorkingDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !filepath.IsAbs(dir) {
		t.Errorf("expected absolute path, got: %s", dir)
	}
}
