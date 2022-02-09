// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package encrypt

// this package is meant for creating protected settings for testing extensions
// not intended for use in production code

type ICertHandler interface {
	GetThumbprint() (certThumbprint string, err error)
	Encrypt(bytesToEncrypt []byte) (encryptedBytes []byte, err error)
}

// certLocation is ignored for windows
func New(certLocation string) (ICertHandler, error) {
	return newCertHandler(certLocation)
}

