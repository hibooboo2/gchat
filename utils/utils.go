package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

func Hash(val string) string {
	h := md5.New()
	h.Write([]byte(val))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func Encrypt(val string, passphrase string) string {
	data := []byte(val)
	block, err := aes.NewCipher([]byte(Hash(passphrase)))
	if err != nil {
		panic(err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		panic(err)
	}
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return base64.StdEncoding.EncodeToString(ciphertext)
}

func Decrypt(val string, passphrase string) string {
	data, err := base64.StdEncoding.DecodeString(val)
	if err != nil {
		panic(err)
	}
	key := []byte(Hash(passphrase))
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err)
	}
	nonceSize := gcm.NonceSize()
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		panic(err)
	}
	return string(plaintext)
}
