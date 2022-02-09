// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package decrypt

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"github.com/Azure/azure-extension-platform/pkg/constants"
	"github.com/Azure/azure-extension-platform/pkg/encrypt"
	"github.com/stretchr/testify/assert"
	"os"
	"path"
	"testing"
)

var testdir = path.Join(".", "testdir")

func initTest(t *testing.T) {
	err := os.MkdirAll(testdir, constants.FilePermissions_UserOnly_ReadWriteExecute)
	if err != nil {
		t.Fatalf("could not create testdir")
	}

	getCertificateDir = func(configFolder string) (certificateFolder string) {
		return testdir
	}
}

func cleanupTest() {
	os.RemoveAll(testdir)
}

// base 64 string can be longer than input param
func generateRandomProtectedSettings(len int) (string, error) {
	buff := make([]byte, len)
	_, err := rand.Read(buff)
	if err != nil {
		return "", err
	}
	base64string := base64.StdEncoding.EncodeToString(buff)
	protSettings := ProtectedSettings{Key1: "value1", Key2: base64string}
	bytes, err := json.Marshal(&protSettings)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

type ProtectedSettings struct {
	Key1 string
	Key2 string
}

func TestCanEncryptAndDecrypt(t *testing.T) {
	initTest(t)
	defer cleanupTest()
	certHandler, err := encrypt.New(testdir)
	assert.NoError(t, err, "certificate creation should succeed")
	thumbprint, err := certHandler.GetThumbprint()
	assert.NoError(t, err, "getting thumbprint should succeed")
	stringToEncrypt, err := generateRandomProtectedSettings(30)
	bytesToEncrypt := []byte(stringToEncrypt)
	assert.NoError(t, err, "error creating random string")
	encryptedBytes, err := certHandler.Encrypt(bytesToEncrypt)
	assert.NoError(t, err, "encryption should succeed")
	assert.NotEqualValues(t, stringToEncrypt, encryptedBytes, "encrypted bytes should be different from original")
	decrypted, err := DecryptProtectedSettings(testdir, thumbprint, encryptedBytes)
	protSettings := ProtectedSettings{}
	assert.NoError(t, json.Unmarshal([]byte(decrypted), &protSettings))
	assert.Equal(t, "value1", protSettings.Key1, "values associated with key1 should be the same")
	assert.Equal(t, stringToEncrypt, decrypted, "the decrypted message should be the same as the original")
}
