// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package decrypt

import (
	"encoding/json"
	"testing"

	"github.com/Azure/azure-extension-platform/pkg/encrypt"
	"github.com/Azure/azure-extension-platform/pkg/extensionerrors"
	"github.com/stretchr/testify/require"
)

func Test_getCertificateThumbprint(t *testing.T) {
	encryptHandler, err := encrypt.New("")
	require.NoError(t, err, "could not get a certificate")
	thumbprint, err := encryptHandler.GetThumbprint()
	require.NoError(t, err, "getCertificateThumbprint failed")
	require.True(t, len(thumbprint) == 40)
}

func Test_decryptSettingsCertNotFound(t *testing.T) {
	invalidCert := "123456790abcdefedcba09876543210123456789"
	decoded := make([]byte, 5) // We'll never process this because the cert is wrong

	_, err := DecryptProtectedSettings("", invalidCert, decoded)
	require.Error(t, err, extensionerrors.ErrCertWithThumbprintNotFound)
}

func Test_decryptSettingsMisencoded(t *testing.T) {
	serialized := getTestData()
	encryptHandler, err := encrypt.New("")
	require.NoError(t, err, "could not get a certificate")
	thumbprint, err := encryptHandler.GetThumbprint()
	require.NoError(t, err, "getCertificateThumbprint failed")
	encrypted, err := encryptHandler.Encrypt([]byte(serialized))
	require.NoError(t, err, "encryptTestData failed")

	// Mess with the encrypted data
	encrypted[0] = 5
	encrypted[1] = 3

	_, err = DecryptProtectedSettings("", thumbprint, encrypted)
	require.Error(t, err, extensionerrors.ErrInvalidProtectedSettingsData)
}

func Test_decryptProtectedSettings(t *testing.T) {
	serialized := getTestData()
	encryptHandler, err := encrypt.New("")
	require.NoError(t, err, "could not get a certificate")
	thumbprint, err := encryptHandler.GetThumbprint()
	require.NoError(t, err, "getCertificateThumbprint failed")
	encrypted, err := encryptHandler.Encrypt([]byte(serialized))
	require.NoError(t, err, "encryptTestData failed")

	s, err := DecryptProtectedSettings("", thumbprint, encrypted)
	require.NoError(t, err, "decryptProtectedSettings failed")
	v := make(map[string]interface{})
	err = json.Unmarshal([]byte(s), &v)
	require.NoError(t, err, "json unmarshal failed")
	landMammal, ok := v["AfricanLandMammal"].(string)
	require.True(t, ok, "African land mammal is not OK")
	chipmunk, ok := v["ChipmunkType"].(string)
	require.True(t, ok, "Chipmunk is not OK")
	number, ok := v["InterestingNumber"].(string)
	require.True(t, ok, "Number is not OK")
	require.Equal(t, "cheetah", landMammal)
	require.Equal(t, "Townsends", chipmunk)
	require.Equal(t, "42", number)
}

// To avoid serialization hassles, since Go adds annoying escapes when it serializes json
// we just manually deserialize here, since we're testing the dev code - not our test encryption code
func getTestData() string {
	testData := "{\"AfricanLandMammal\":\"cheetah\",\"InterestingNumber\":\"42\",\"ChipmunkType\":\"Townsends\"}"
	return testData
}

