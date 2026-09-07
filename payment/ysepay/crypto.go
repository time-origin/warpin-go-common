package ysepay

import (
	"crypto"
	"crypto/aes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

const aesKeyAlphabet = "0123456789ABCDEF"

func canonicalSigningData(fields map[string]string) string {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		if key != "sign" && key != "$jacocoData" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+fields[key])
	}
	return strings.Join(parts, "&")
}

func signFields(fields map[string]string, privateKey *rsa.PrivateKey) (string, error) {
	digest := sha256.Sum256([]byte(canonicalSigningData(fields)))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, digest[:])
	if err != nil {
		return "", fmt.Errorf("%w: request signing failed", ErrCrypto)
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

func verifyFields(fields map[string]string, publicKey *rsa.PublicKey) error {
	encoded := fields["sign"]
	if encoded == "" {
		return fmt.Errorf("%w: signature is missing", ErrInvalidSignature)
	}
	signature, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return fmt.Errorf("%w: signature encoding is invalid", ErrInvalidSignature)
	}
	digest := sha256.Sum256([]byte(canonicalSigningData(fields)))
	if err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, digest[:], signature); err != nil {
		return fmt.Errorf("%w: verification failed", ErrInvalidSignature)
	}
	return nil
}

func generateAESKey(random io.Reader) ([]byte, error) {
	key := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(random, key); err != nil {
		return nil, fmt.Errorf("%w: cannot generate AES key", ErrCrypto)
	}
	for index := range key {
		key[index] = aesKeyAlphabet[int(key[index])%len(aesKeyAlphabet)]
	}
	return key, nil
}

func encryptAES(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid AES key", ErrCrypto)
	}
	padding := aes.BlockSize - len(plaintext)%aes.BlockSize
	padded := make([]byte, len(plaintext)+padding)
	copy(padded, plaintext)
	for index := len(plaintext); index < len(padded); index++ {
		padded[index] = byte(padding)
	}
	ciphertext := make([]byte, len(padded))
	for offset := 0; offset < len(padded); offset += aes.BlockSize {
		block.Encrypt(ciphertext[offset:offset+aes.BlockSize], padded[offset:offset+aes.BlockSize])
	}
	return ciphertext, nil
}

func decryptAES(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid AES key", ErrCrypto)
	}
	if len(ciphertext) == 0 || len(ciphertext)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("%w: invalid AES ciphertext length", ErrCrypto)
	}
	plaintext := make([]byte, len(ciphertext))
	for offset := 0; offset < len(ciphertext); offset += aes.BlockSize {
		block.Decrypt(plaintext[offset:offset+aes.BlockSize], ciphertext[offset:offset+aes.BlockSize])
	}
	padding := int(plaintext[len(plaintext)-1])
	if padding == 0 || padding > aes.BlockSize || padding > len(plaintext) {
		return nil, fmt.Errorf("%w: invalid AES padding", ErrCrypto)
	}
	for _, value := range plaintext[len(plaintext)-padding:] {
		if int(value) != padding {
			return nil, fmt.Errorf("%w: invalid AES padding", ErrCrypto)
		}
	}
	return plaintext[:len(plaintext)-padding], nil
}

func encryptAESKey(key []byte, publicKey *rsa.PublicKey) (string, error) {
	ciphertext, err := rsa.EncryptPKCS1v15(rand.Reader, publicKey, key)
	if err != nil {
		return "", fmt.Errorf("%w: AES key wrapping failed", ErrCrypto)
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func decodeBase64Ciphertext(value string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, errors.New("invalid base64 ciphertext")
	}
	return decoded, nil
}
