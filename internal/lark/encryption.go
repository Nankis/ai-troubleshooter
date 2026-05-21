package lark

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
)

type encryptedEventEnvelope struct {
	Encrypt string `json:"encrypt"`
}

func decodeEncryptedEventBody(body []byte, encryptKey string) ([]byte, error) {
	var envelope encryptedEventEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}
	if envelope.Encrypt == "" {
		if encryptKey != "" {
			return nil, fmt.Errorf("lark encrypt key is configured but event body is not encrypted")
		}
		return body, nil
	}
	if encryptKey == "" {
		return nil, fmt.Errorf("lark encrypted event received but encrypt key is not configured")
	}
	return decryptEvent(encryptKey, envelope.Encrypt)
}

func decryptEvent(encryptKey string, encrypted string) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return nil, fmt.Errorf("decode lark encrypted event: %w", err)
	}
	if len(ciphertext) == 0 || len(ciphertext)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("invalid lark encrypted event block size")
	}
	key := sha256.Sum256([]byte(encryptKey))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	plaintext := make([]byte, len(ciphertext))
	mode := cipher.NewCBCDecrypter(block, key[:aes.BlockSize])
	mode.CryptBlocks(plaintext, ciphertext)
	plaintext, err = pkcs7Unpad(plaintext, aes.BlockSize)
	if err != nil {
		return nil, fmt.Errorf("decrypt lark encrypted event: %w", err)
	}
	return plaintext, nil
}

func pkcs7Unpad(value []byte, blockSize int) ([]byte, error) {
	if len(value) == 0 || len(value)%blockSize != 0 {
		return nil, fmt.Errorf("invalid padded data length")
	}
	padding := int(value[len(value)-1])
	if padding == 0 || padding > blockSize || padding > len(value) {
		return nil, fmt.Errorf("invalid padding")
	}
	for _, b := range value[len(value)-padding:] {
		if int(b) != padding {
			return nil, fmt.Errorf("invalid padding")
		}
	}
	return value[:len(value)-padding], nil
}
