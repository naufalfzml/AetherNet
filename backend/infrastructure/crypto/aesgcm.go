package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

type EncryptedBlob struct {
	Nonce      []byte `json:"nonce"`
	Ciphertext []byte `json:"ciphertext"`
}

func GenerateAES256Key() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

func EncryptAESGCM(key, plaintext, additionalData []byte) (EncryptedBlob, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return EncryptedBlob{}, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return EncryptedBlob{}, err
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return EncryptedBlob{}, err
	}

	return EncryptedBlob{
		Nonce:      nonce,
		Ciphertext: aead.Seal(nil, nonce, plaintext, additionalData),
	}, nil
}

func DecryptAESGCM(key []byte, blob EncryptedBlob, additionalData []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(blob.Nonce) != aead.NonceSize() {
		return nil, fmt.Errorf("invalid nonce size: %d", len(blob.Nonce))
	}
	return aead.Open(nil, blob.Nonce, blob.Ciphertext, additionalData)
}
