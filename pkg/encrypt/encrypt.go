package encrypt

type ICertHandler interface{
	GetThumbprint()(certThumbprint string, err error)
    Encrypt(bytesToEncrypt []byte)( encryptedBytes []byte, err error)
}

func New()(ICertHandler, error){
	return newCertHandler()
}