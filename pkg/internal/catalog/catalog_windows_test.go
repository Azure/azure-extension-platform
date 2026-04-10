package catalog

import (
	"os"
	"testing"

	"github.com/Azure/azure-extension-platform/pkg/hashutils"
	"github.com/stretchr/testify/require"
)

const (
	// Update these paths to your real local test assets.
	testSignedCatalog   = "./testutils/signed.cat"
	testUnsignedCatalog = "./testutils/unsigned.cat"
	testInvalidCatalog  = "./testutils/invalid.cat"

	testMemberFile  = "./testutils/test.cmd"
	testBlockedFile = "./testutils/blocked.cmd"

	testHashAlgorithm = hashutils.HashTypeSHA256
)

func requireFileExistsOrSkip(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Skipf("test asset not found: %s (update hard-coded test path)", path)
	}
}

func TestVerifyFileSignature(t *testing.T) {

	requireFileExistsOrSkip(t, testMemberFile)
	requireFileExistsOrSkip(t, testSignedCatalog)
	requireFileExistsOrSkip(t, testUnsignedCatalog)
	requireFileExistsOrSkip(t, testInvalidCatalog)

	t.Run("success_signed_catalog", func(t *testing.T) {
		status, err := VerifyFileSignature(testSignedCatalog) // update the return value here, then update the return type of the function.
		require.Equal(t, uint32(0), status)
		require.Nil(t, err)
	})

	t.Run("failure_unsigned_catalog", func(t *testing.T) {
		status, ewc := VerifyFileSignature(testUnsignedCatalog)
		t.Logf("VerifyFileSignature(unsigned) status=%d", status)

		require.NotEqual(t, uint32(0), status)
		require.NotNil(t, ewc)
		require.Equal(t, 1, ewc.ErrorCode)
	})

	t.Run("failure_empty_file_path", func(t *testing.T) {
		status, ewc := VerifyFileSignature("")
		t.Logf("VerifyFileSignature(empty) status=%d", status)

		require.NotEqual(t, uint32(0), status)
		require.NotNil(t, ewc)
		require.Equal(t, 1, ewc.ErrorCode)
		require.Contains(t, ewc.Error(), "file cannot be empty")
	})

	t.Run("failure_file_does_not_exist", func(t *testing.T) {
		status, ewc := VerifyFileSignature("./testutils/does-not-exist.cat")
		t.Logf("VerifyFileSignature(missing) status=%d", status)

		require.NotEqual(t, uint32(0), status)
		require.NotNil(t, ewc)
		require.Equal(t, 3, ewc.ErrorCode)
		require.Contains(t, ewc.Error(), "file does not exist at path")
	})
}

func TestValidateFileAgainstCatalog(t *testing.T) {
	requireFileExistsOrSkip(t, testSignedCatalog)
	requireFileExistsOrSkip(t, testInvalidCatalog)
	requireFileExistsOrSkip(t, testMemberFile)
	requireFileExistsOrSkip(t, testBlockedFile)

	t.Run("success_signed_catalog_and_member_file", func(t *testing.T) {
		status, err := ValidateFileAgainstCatalog(testSignedCatalog, testMemberFile, testHashAlgorithm)
		require.Equal(t, uint32(0), status)
		require.Nil(t, err)
	})

	t.Run("failure_blocked_file_not_in_catalog", func(t *testing.T) {
		status, err := ValidateFileAgainstCatalog(testSignedCatalog, testBlockedFile, testHashAlgorithm)
		t.Logf("ValidateFileAgainstCatalog(blocked) status=%d", status)

		require.NotEqual(t, uint32(0), status)
		require.NotNil(t, err)
		require.Contains(t, err.Error(), "file is not signed by the catalog file provided")
		require.Equal(t, err.ErrorCode, 1)
	})

	t.Run("failure_invalid_catalog_format", func(t *testing.T) {
		status, err := ValidateFileAgainstCatalog(testInvalidCatalog, testMemberFile, testHashAlgorithm)
		t.Logf("ValidateFileAgainstCatalog(invalid format) status=%d", status)

		require.NotEqual(t, uint32(0), status)
		require.NotNil(t, err)
		require.Contains(t, err.Error(), "Invalid catalog file format")
		require.Equal(t, err.ErrorCode, 1)
	})

	t.Run("failure_empty_catalog_file_path", func(t *testing.T) {
		status, err := ValidateFileAgainstCatalog("", testMemberFile, testHashAlgorithm)
		t.Logf("ValidateFileAgainstCatalog(empty catalog path) status=%d", status)

		require.NotEqual(t, uint32(0), status)
		require.NotNil(t, err)
		require.Contains(t, err.Error(), "catalog file path cannot be empty")
		require.Equal(t, err.ErrorCode, 1)
	})

	t.Run("failure_empty_member_file_path", func(t *testing.T) {
		status, err := ValidateFileAgainstCatalog(testSignedCatalog, "", testHashAlgorithm)
		t.Logf("ValidateFileAgainstCatalog(empty member file path) status=%d", status)

		require.NotEqual(t, uint32(0), status)
		require.NotNil(t, err)
		require.Contains(t, err.Error(), "member file path cannot be empty")
		require.Equal(t, err.ErrorCode, 1)
	})

	t.Run("failure_catalog_file_does_not_exist", func(t *testing.T) {
		status, err := ValidateFileAgainstCatalog("./testutils/catalog/does-not-exist.cat", testMemberFile, testHashAlgorithm)
		t.Logf("ValidateFileAgainstCatalog(missing catalog) status=%d", status)

		require.NotEqual(t, uint32(0), status)
		require.NotNil(t, err)
		require.Contains(t, err.Error(), "catalog file does not exist at path")
		require.Equal(t, err.ErrorCode, 1)
	})

	t.Run("failure_file_to_verify_does_not_exist", func(t *testing.T) {
		status, err := ValidateFileAgainstCatalog(testSignedCatalog, "./testutils/catalog/missing.cmd", testHashAlgorithm)
		t.Logf("ValidateFileAgainstCatalog(missing member) status=%d", status)

		require.NotEqual(t, uint32(0), status)
		require.NotNil(t, err)
		require.Contains(t, err.Error(), "memberfile does not exist at path")
		require.Equal(t, err.ErrorCode, 1)
	})
}
