package catalog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const catalogWindowsTestHashAlgorithm = "sha256"

type catalogWindowsTestAssets struct {
	signedCatalog   string
	unsignedCatalog string
	invalidCatalog  string
	memberFile      string
	blockedFile     string
}

func loadCatalogWindowsTestAssets(t *testing.T) catalogWindowsTestAssets {
	t.Helper()

	return catalogWindowsTestAssets{
		signedCatalog: catalogWindowsResolveTestAsset(
			t,
			"AEP_TEST_SIGNED_CATALOG"),
		unsignedCatalog: catalogWindowsResolveTestAsset(
			t,
			"AEP_TEST_UNSIGNED_CATALOG"),
		invalidCatalog: catalogWindowsResolveTestAsset(
			t,
			"AEP_TEST_INVALID_CATALOG"),
		memberFile: catalogWindowsResolveTestAsset(
			t,
			"AEP_TEST_MEMBER_FILE"),
		blockedFile: catalogWindowsResolveTestAsset(
			t,
			"AEP_TEST_BLOCKED_FILE"),
	}
}

func catalogWindowsResolveTestAsset(t *testing.T, envKey string) string {
	t.Helper()

	if fromEnv := os.Getenv(envKey); fromEnv != "" {
		if _, err := os.Stat(fromEnv); err == nil {
			return fromEnv
		}
		t.Fatalf("environment variable %s is set but file does not exist: %s", envKey, fromEnv)
	}
	t.Skipf("missing required test asset for %s; set %s or place a fixture in testdata", envKey, envKey)
	return ""
}

func catalogWindowsExtractClarification(err error) (code string, message string, ok bool) {
	for current := err; current != nil; current = errors.Unwrap(current) {
		rv := reflect.ValueOf(current)
		if !rv.IsValid() {
			continue
		}
		if rv.Kind() == reflect.Ptr {
			if rv.IsNil() {
				continue
			}
			rv = rv.Elem()
		}
		if rv.Kind() != reflect.Struct {
			continue
		}

		errorCodeField := rv.FieldByName("ErrorCode")
		errField := rv.FieldByName("Err")
		if !errorCodeField.IsValid() || !errField.IsValid() {
			continue
		}

		code = fmt.Sprint(errorCodeField.Interface())

		if errField.CanInterface() {
			if underlying, ok := errField.Interface().(error); ok && underlying != nil {
				message = underlying.Error()
			}
		}

		return code, message, true
	}

	return "", "", false
}

func catalogWindowsRequireNoClarification(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("expected nil clarification, got %T: %v", err, err)
	}
}

func catalogWindowsRequireClarificationPresent(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("expected non-nil clarification, got nil")
	}

	code, message, ok := catalogWindowsExtractClarification(err)
	if !ok {
		t.Fatalf("expected ErrorWithClarification-shaped error, got %T: %v", err, err)
	}
	if strings.TrimSpace(code) == "" {
		t.Fatalf("expected non-empty clarification ErrorCode, got %q", code)
	}
	if strings.TrimSpace(message) == "" {
		t.Fatalf("expected non-empty clarification Err message, got %q", message)
	}
}

func catalogWindowsRequireClarificationMatch(t *testing.T, err error, wantCode string, wantMessage string) {
	t.Helper()

	if err == nil {
		t.Fatal("expected non-nil clarification, got nil")
	}

	gotCode, gotMessage, ok := catalogWindowsExtractClarification(err)
	if !ok {
		t.Fatalf("expected ErrorWithClarification-shaped error, got %T: %v", err, err)
	}

	if gotCode != wantCode {
		t.Fatalf("unexpected ErrorCode: got %q, want %q", gotCode, wantCode)
	}
	if gotMessage != wantMessage {
		t.Fatalf("unexpected Err message: got %q, want %q", gotMessage, wantMessage)
	}
}

func TestVerifyFileSignature(t *testing.T) {
	assets := loadCatalogWindowsTestAssets(t)

	t.Run("success_signed_catalog", func(t *testing.T) {
		status, err := VerifyFileSignature(assets.signedCatalog)
		if status != 0 {
			t.Fatalf("expected success status 0, got %d", status)
		}
		catalogWindowsRequireNoClarification(t, err)
	})

	t.Run("failure_unsigned_catalog", func(t *testing.T) {
		status, err := VerifyFileSignature(assets.unsignedCatalog)
		t.Logf("VerifyFileSignature(%q) status=%d", assets.unsignedCatalog, status)

		if status == 0 {
			t.Fatal("expected non-zero status for unsigned catalog")
		}

		catalogWindowsRequireClarificationMatch(
			t,
			err,
			"UnabletoVerifyFileSignature",
			"unable to verify file signature because catalog file is unsigned",
		)
	})

	t.Run("failure_invalid_catalog_format", func(t *testing.T) {
		status, err := VerifyFileSignature(assets.invalidCatalog)
		t.Logf("VerifyFileSignature(%q) status=%d", assets.invalidCatalog, status)

		if status == 0 {
			t.Fatal("expected non-zero status for invalid catalog format")
		}

		catalogWindowsRequireClarificationMatch(
			t,
			err,
			"UnabletoVerifyFileSignature",
			"catalog file is in an improper format",
		)
	})

	t.Run("failure_empty_file_path", func(t *testing.T) {
		status, err := VerifyFileSignature("")
		t.Logf("VerifyFileSignature(empty) status=%d", status)

		if status == 0 {
			t.Fatal("expected non-zero status for empty file path")
		}

		catalogWindowsRequireClarificationPresent(t, err)
	})

	t.Run("failure_file_does_not_exist", func(t *testing.T) {
		missingPath := filepath.Join(t.TempDir(), "does-not-exist.cat")

		status, err := VerifyFileSignature(missingPath)
		t.Logf("VerifyFileSignature(%q) status=%d", missingPath, status)

		if status == 0 {
			t.Fatal("expected non-zero status for missing file path")
		}

		catalogWindowsRequireClarificationPresent(t, err)
	})

	t.Run("failure_file_path_contains_nul", func(t *testing.T) {
		status, err := VerifyFileSignature("bad\x00path.cat")
		t.Logf("VerifyFileSignature(path-with-nul) status=%d", status)

		if status == 0 {
			t.Fatal("expected non-zero status for path containing NUL")
		}

		catalogWindowsRequireClarificationPresent(t, err)
	})
}

func TestValidateFileAgainstCatalog(t *testing.T) {
	assets := loadCatalogWindowsTestAssets(t)

	t.Run("success_signed_catalog_and_member_file", func(t *testing.T) {
		status, err := ValidateFileAgainstCatalog(assets.signedCatalog, assets.memberFile, catalogWindowsTestHashAlgorithm)
		if status != 0 {
			t.Fatalf("expected success status 0, got %d", status)
		}
		catalogWindowsRequireNoClarification(t, err)
	})

	t.Run("failure_blocked_file_not_in_signed_catalog", func(t *testing.T) {
		status, err := ValidateFileAgainstCatalog(assets.signedCatalog, assets.blockedFile, catalogWindowsTestHashAlgorithm)
		t.Logf("ValidateFileAgainstCatalog(%q, %q) status=%d", assets.signedCatalog, assets.blockedFile, status)

		if status == 0 {
			t.Fatal("expected non-zero status for blocked file")
		}

		catalogWindowsRequireClarificationMatch(
			t,
			err,
			"UnableToVerifyFileAgainstCatalog",
			"the file provided is not signed by the catalog file provided",
		)
	})

	t.Run("failure_invalid_catalog_format", func(t *testing.T) {
		status, err := ValidateFileAgainstCatalog(assets.invalidCatalog, assets.memberFile, catalogWindowsTestHashAlgorithm)
		t.Logf("ValidateFileAgainstCatalog(%q, %q) status=%d", assets.invalidCatalog, assets.memberFile, status)

		if status == 0 {
			t.Fatal("expected non-zero status for invalid catalog format")
		}

		catalogWindowsRequireClarificationMatch(
			t,
			err,
			"UnableToVerifyFileAgainstCatalog",
			"The catalog file provided is an improper format",
		)
	})

	t.Run("failure_empty_catalog_file_path", func(t *testing.T) {
		status, err := ValidateFileAgainstCatalog("", assets.memberFile, catalogWindowsTestHashAlgorithm)
		t.Logf("ValidateFileAgainstCatalog(empty-catalog, %q) status=%d", assets.memberFile, status)

		if status == 0 {
			t.Fatal("expected non-zero status for empty catalog path")
		}

		catalogWindowsRequireClarificationPresent(t, err)
	})

	t.Run("failure_empty_file_to_verify_path", func(t *testing.T) {
		status, err := ValidateFileAgainstCatalog(assets.signedCatalog, "", catalogWindowsTestHashAlgorithm)
		t.Logf("ValidateFileAgainstCatalog(%q, empty-file) status=%d", assets.signedCatalog, status)

		if status == 0 {
			t.Fatal("expected non-zero status for empty file-to-verify path")
		}

		catalogWindowsRequireClarificationPresent(t, err)
	})

	t.Run("failure_catalog_file_does_not_exist", func(t *testing.T) {
		missingCatalog := filepath.Join(t.TempDir(), "missing.cat")

		status, err := ValidateFileAgainstCatalog(missingCatalog, assets.memberFile, catalogWindowsTestHashAlgorithm)
		t.Logf("ValidateFileAgainstCatalog(%q, %q) status=%d", missingCatalog, assets.memberFile, status)

		if status == 0 {
			t.Fatal("expected non-zero status for missing catalog file")
		}

		catalogWindowsRequireClarificationPresent(t, err)
	})

	t.Run("failure_file_to_verify_does_not_exist", func(t *testing.T) {
		missingFile := filepath.Join(t.TempDir(), "missing.cmd")

		status, err := ValidateFileAgainstCatalog(assets.signedCatalog, missingFile, catalogWindowsTestHashAlgorithm)
		t.Logf("ValidateFileAgainstCatalog(%q, %q) status=%d", assets.signedCatalog, missingFile, status)

		if status == 0 {
			t.Fatal("expected non-zero status for missing file-to-verify")
		}

		catalogWindowsRequireClarificationPresent(t, err)
	})

	t.Run("failure_invalid_hash_algorithm", func(t *testing.T) {
		status, err := ValidateFileAgainstCatalog(assets.signedCatalog, assets.memberFile, "not-a-real-hash")
		t.Logf("ValidateFileAgainstCatalog(%q, %q, invalid-hash) status=%d", assets.signedCatalog, assets.memberFile, status)

		if status == 0 {
			t.Fatal("expected non-zero status for invalid hash algorithm")
		}

		catalogWindowsRequireClarificationPresent(t, err)
	})

	t.Run("failure_catalog_file_path_contains_nul", func(t *testing.T) {
		status, err := ValidateFileAgainstCatalog("bad\x00catalog.cat", assets.memberFile, catalogWindowsTestHashAlgorithm)
		t.Logf("ValidateFileAgainstCatalog(path-with-nul, %q) status=%d", assets.memberFile, status)

		if status == 0 {
			t.Fatal("expected non-zero status for catalog path containing NUL")
		}

		catalogWindowsRequireClarificationPresent(t, err)
	})

	t.Run("failure_file_to_verify_path_contains_nul", func(t *testing.T) {
		status, err := ValidateFileAgainstCatalog(assets.signedCatalog, "bad\x00file.cmd", catalogWindowsTestHashAlgorithm)
		t.Logf("ValidateFileAgainstCatalog(%q, path-with-nul) status=%d", assets.signedCatalog, status)

		if status == 0 {
			t.Fatal("expected non-zero status for file-to-verify path containing NUL")
		}

		catalogWindowsRequireClarificationPresent(t, err)
	})
}
