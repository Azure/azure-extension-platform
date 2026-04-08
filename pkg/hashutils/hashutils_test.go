package hashutils

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

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
	defer os.Remove(tmpFile.Name())

	hashAlg := sha256.New()
	got, err := ComputeFileHash(tmpFile.Name(), hashAlg)

	require.Nil(t, err, "expected no error from ComputeFileHash, got: %v", err)
	require.NotEmpty(t, got, "expected non-empty hash result")

	// Verify hash matches expected
	expected := sha256.Sum256(content)
	require.Equal(t, hex.EncodeToString(expected[:]), got, "hash mismatch")
}

func TestComputeFileHash_EmptyFilePath(t *testing.T) {
	hashAlg := sha256.New()
	_, err := ComputeFileHash("", hashAlg)
	require.NotNil(t, err, "expected error for empty file path")
	require.Equal(t, "file path cannot be empty", err.Error(), "unexpected error message")
}

func TestComputeFileHash_FileDoesNotExist(t *testing.T) {
	hashAlg := sha256.New()
	nonExistentPath := "./nonexistent_file.txt"
	_, err := ComputeFileHash(nonExistentPath, hashAlg)
	require.NotNil(t, err, "expected error for non-existent file path")
	require.Contains(t, err.Error(), "file does not exist at path", "unexpected error message")
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
	defer os.Remove(tmpFile.Name())

	// Remove read permission
	if err := os.Chmod(tmpFile.Name(), 0000); err != nil {
		t.Fatalf("failed to change file permissions: %v", err)
	}
	t.Cleanup(func() {
		os.Chmod(tmpFile.Name(), 0644) // restore for cleanup
	})

	hashAlg := sha256.New()
	_, err = ComputeFileHash(tmpFile.Name(), hashAlg)
	require.NotNil(t, err, "expected error for unreadable file")
	require.Contains(t, err.Error(), "failed to open file for hashing", "unexpected error message")
}

func TestComputeFileHash_EmptyFile(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "empty_*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	hashAlg := sha256.New()
	got, err := ComputeFileHash(tmpFile.Name(), hashAlg)
	if err != nil {
		t.Fatalf("unexpected error for empty file: %v", err)
	}

	expected := sha256.Sum256([]byte{})
	require.Equal(t, hex.EncodeToString(expected[:]), got, "hash mismatch for empty file")
}

func TestComputeHash_Success(t *testing.T) {
	input := "hello world"
	hashAlg := sha256.New()
	got := ComputeHash(input, hashAlg)
	require.NotEmpty(t, got, "expected non-empty hash result")

	expected := sha256.Sum256([]byte(input))
	require.Equal(t, hex.EncodeToString(expected[:]), got, "hash mismatch")
}

func TestComputeHash_EmptyString(t *testing.T) {
	hashAlg := sha256.New()
	got := ComputeHash("", hashAlg)
	require.NotEmpty(t, got, "expected non-empty hash result for empty string")

	expected := sha256.Sum256([]byte{})
	require.Equal(t, hex.EncodeToString(expected[:]), got, "hash mismatch for empty string")
}

func TestComputeHash_DifferentInputsDifferentHashes(t *testing.T) {
	hash1 := ComputeHash("input1", sha256.New())
	hash2 := ComputeHash("input2", sha256.New())
	require.NotEqual(t, hash1, hash2, "expected different hashes for different inputs")
}

func TestComputeHash_SameInputSameHash(t *testing.T) {
	input := "consistent input"
	hash1 := ComputeHash(input, sha256.New())
	hash2 := ComputeHash(input, sha256.New())
	require.Equal(t, hash1, hash2, "expected same hash for same input")
}

func TestComputeHash_DifferentAlgorithm(t *testing.T) {
	input := "test"
	sha256Hash := ComputeHash(input, sha256.New())
	sha1Hash := ComputeHash(input, sha1.New())

	require.NotEqual(t, sha256Hash, sha1Hash, "expected different hashes for different algorithms")
	require.Equal(t, 64, len(sha256Hash), "expected SHA-256 hex string length 64")
	require.Equal(t, 40, len(sha1Hash), "expected SHA-1 hex string length 40")
}
