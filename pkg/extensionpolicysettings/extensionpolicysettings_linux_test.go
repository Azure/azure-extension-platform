package extensionpolicysettings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const successfulSigningScenarioDir = "./testutils/successfulsigningscenario"
const expiredCertSigningScenarioDir = "./testutils/expiredcertsigningscenario"

// Note: These tests rely on a self-signed certificate created manually using openssl command,
// and script files manually signed using that certificate. The certificate is set to expire March 12, 2026.
// Additionally, the test for expired certificate does not require renewal (since the cert must be expired for testing purposes),
// but it does require the system date-time to be accurate.
// TO-DO: Automate the generation of certificates + signature files in the future.

func TestValidateFileSignature_ValidSignature(t *testing.T) {
	certPath := successfulSigningScenarioDir + "/anasanc1.cert"
	sigFile := successfulSigningScenarioDir + "/sig_file"
	scriptPath := successfulSigningScenarioDir + "/signedscript1.sh"

	isValid, err := ValidateFileSignature(scriptPath, sigFile, certPath, true)

	if !isValid {
		t.Errorf("Validation should be true, but is %v", isValid)
	}
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
}

func TestValidateFileSignature_ValidSignature_NoCheckforExpiration(t *testing.T) {
	certPath := successfulSigningScenarioDir + "/anasanc1.cert"
	sigFile := successfulSigningScenarioDir + "/sig_file"
	scriptPath := successfulSigningScenarioDir + "/signedscript1.sh"

	isValid, err := ValidateFileSignature(scriptPath, sigFile, certPath, false)

	if !isValid {
		t.Errorf("Validation should be true, but is %v", isValid)
	}
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
}

func TestValidateFileSignature_InvalidFilePath(t *testing.T) {
	invalidFilePath := "/nonexistent/path/to/file.txt"
	certPath := successfulSigningScenarioDir + "/anasanc1.cert"
	sigFile := successfulSigningScenarioDir + "/sig_file"

	isValid, err := ValidateFileSignature(invalidFilePath, sigFile, certPath, true)

	if err == nil {
		t.Errorf("expected error for invalid file path, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "file does not exist at path: "+invalidFilePath) {
		t.Errorf("unexpected error message: %v", err)
	}
	if isValid {
		t.Errorf("expected isValid=false for invalid file, got true")
	}
}

func TestValidateFileSignature_InvalidCertPath(t *testing.T) {
	invalidCertPath := successfulSigningScenarioDir + "/nonexistent.cert"
	scriptPath := successfulSigningScenarioDir + "/signedscript1.sh"
	sigFile := successfulSigningScenarioDir + "/sig_file"

	// Execute: Call validation with nonexistent cert
	isValid, err := ValidateFileSignature(scriptPath, sigFile, invalidCertPath, true)

	if err == nil {
		t.Errorf("expected error for invalid cert path, got nil")
	}
	if isValid {
		t.Errorf("expected isValid=false, got true")
	}
	if err != nil && !strings.Contains(err.Error(), "cert file does not exist at path: "+invalidCertPath) {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestValidateFileSignature_EmptyPaths(t *testing.T) {
	certPath := successfulSigningScenarioDir + "/anasanc1.cert"
	sigFile := successfulSigningScenarioDir + "/sig_file"
	scriptPath := successfulSigningScenarioDir + "/signedscript1.sh"

	tests := []struct {
		name      string
		file      string
		sig       string
		cert      string
		wantError string
	}{
		{"empty file path", "", sigFile, certPath, "file path cannot be empty"},
		{"empty signature path", scriptPath, "", certPath, "signature file path cannot be empty"},
		{"empty cert path", scriptPath, sigFile, "", "cert file path cannot be empty"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			isValid, err := ValidateFileSignature(tc.file, tc.sig, tc.cert, true)
			if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Errorf("expected error containing %q, got: %v", tc.wantError, err)
			}
			if isValid {
				t.Errorf("expected isValid=false")
			}
		})
	}
}

func TestValidateFileSignature_EmptyScriptAndCert(t *testing.T) {
	dir := t.TempDir()
	emptyCert := filepath.Join(dir, "empty.cert")
	emptySig := filepath.Join(dir, "empty.sig")
	sigFile := successfulSigningScenarioDir + "/sig_file"

	if err := os.WriteFile(emptyCert, []byte{}, 0600); err != nil {
		t.Error(err)
	}
	if err := os.WriteFile(emptySig, []byte{}, 0600); err != nil {
		t.Error(err)
	}

	t.Run("empty cert file", func(t *testing.T) {
		isValid, err := ValidateFileSignature(successfulSigningScenarioDir+"/signedscript1.sh", sigFile, emptyCert, true)
		if err == nil || !strings.Contains(err.Error(), "cert file at "+emptyCert+" is empty") {
			t.Errorf("unexpected error for empty cert file: %v", err)
		}
		if isValid {
			t.Errorf("expected isValid=false")
		}
	})

	t.Run("empty signature file", func(t *testing.T) {
		isValid, err := ValidateFileSignature(successfulSigningScenarioDir+"/signedscript1.sh", emptySig, successfulSigningScenarioDir+"/anasanc1.cert", true)
		if err == nil || !strings.Contains(err.Error(), "signature file at "+emptySig+" is empty") {
			t.Errorf("unexpected error for empty signature file: %v", err)
		}
		if isValid {
			t.Errorf("expected isValid=false")
		}
	})
}

// Ignore test for now, until certificate expires tomorrow April 1st 2026.
// func TestValidateFileSignature_CertIsExpired(t *testing.T) {
// 	// Setup: Create test file with expired certificate
// 	expiredCertSignedFile := expiredCertSigningScenarioDir + "/expiredcertscript.sh"
// 	expiredCertPath := expiredCertSigningScenarioDir + "/expired.cert"
// 	sigFile := expiredCertSigningScenarioDir + "/expiredcert_sig_file"

// 	// Execute: Call validation with expired certificate
// 	isValid, err := ValidateFileSignature(expiredCertSignedFile, sigFile, expiredCertPath, true)

// 	// Assert: Expect validation to fail
// 	if err == nil {
// 		t.Errorf("expected error for expired certificate, got nil")
// 	}
// 	if err != nil && !strings.Contains(err.Error(), "signature validation failed") {
// 		t.Errorf("unexpected error message: %v", err)
// 	}
// 	if isValid {
// 		t.Errorf("expected isValid=false for expired certificate, got true")
// 	}
// }

func TestValidateFileSignature_ExpiredCert_NoCheckforExpiration(t *testing.T) {
	expiredCertSignedFile := expiredCertSigningScenarioDir + "/expiredcertscript.sh"
	expiredCertPath := expiredCertSigningScenarioDir + "/expired.cert"
	sigFile := expiredCertSigningScenarioDir + "/expiredcert_sig_file"

	isValid, err := ValidateFileSignature(expiredCertSignedFile, sigFile, expiredCertPath, false)
	if !isValid {
		t.Errorf("expected validation to succeed without expiration check, got false with error: %v", err)
	}
}

func TestValidateFileSignature_SignatureDoesNotMatchContent(t *testing.T) {
	certPath := successfulSigningScenarioDir + "/anasanc1.cert"
	sigFile := successfulSigningScenarioDir + "/sig_file"
	otherScript := expiredCertSigningScenarioDir + "/expiredcertscript.sh" // mismatched content

	isValid, err := ValidateFileSignature(otherScript, sigFile, certPath, true)
	if err == nil || !strings.Contains(err.Error(), "signature validation failed") {
		t.Errorf("expected error to be signature validation failure, got: %v", err)
	}
	if isValid {
		t.Errorf("expected isValid=false")
	}
}
