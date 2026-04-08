package catalog

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	// Update these paths to your real local test assets.
	testSignedCatalog   = "./testutils/catalog/signed.cat"
	testUnsignedCatalog = "./testutils/catalog/unsigned.cat"
	testInvalidCatalog  = "./testutils/catalog/invalid.cat"

	testMemberFile  = "./testutils/catalog/test.cmd"
	testBlockedFile = "./testutils/catalog/blocked.cmd"

	testHashAlgorithm = "sha256"
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
		require.Contains(t, ewc.Error(), "file path cannot be empty")
	})

	t.Run("failure_file_does_not_exist", func(t *testing.T) {
		status, ewc := VerifyFileSignature("./testutils/catalog/does-not-exist.cat")
		t.Logf("VerifyFileSignature(missing) status=%d", status)

		require.NotEqual(t, uint32(0), status)
		require.NotNil(t, ewc)
		require.Equal(t, 3, ewc.ErrorCode)
		require.Contains(t, ewc.Error(), "file path cannot be empty")
	})

}
