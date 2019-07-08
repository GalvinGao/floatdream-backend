package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"io/ioutil"
	"log"
)

type Decryptor struct {
	SecretKey *rsa.PrivateKey
}

func NewDecryptor(fileName string) Decryptor {
	file, err := ioutil.ReadFile(fileName)
	if err != nil {
		log.Panicln(err)
	}
	block, _ := pem.Decode([]byte(file))
	if block == nil {
		log.Panicln("`block` is nil")
	}
	private, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		log.Panicln(err)
	}
	return Decryptor{SecretKey: private}
}

func (d Decryptor) Decrypt(form EncryptedForm) ([]byte, error) {
	original, err := base64.StdEncoding.DecodeString(form.Payload)
	if err != nil {
		return []byte{}, nil
	}

	decrypted, err := rsa.DecryptPKCS1v15(rand.Reader, d.SecretKey, []byte(original))
	return decrypted, err
}

//func (d Decryptor) Decrypt(form EncryptedForm) (string, error) {
//	var original []byte
//	_, err := base64.StdEncoding.Decode(original, []byte(form.Payload))
//	if err != nil {
//		return "", nil
//	}
//
//	decrypted, err := rsa.DecryptPKCS1v15(rand.Reader, d.SecretKey, original)
//	return string(decrypted), err
//}
