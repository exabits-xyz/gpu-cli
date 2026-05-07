package securestore

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	keyFileName       = "token.key"
	encryptedPrefix   = "v1:"
	encryptionKeySize = 32
)

// EncryptToken encrypts a token for local storage in ~/.exabits/config.yaml.
func EncryptToken(token string) (string, error) {
	key, err := loadOrCreateKey()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("cannot create token cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("cannot create token cipher mode: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("cannot generate token nonce: %w", err)
	}

	sealed := gcm.Seal(nonce, nonce, []byte(token), nil)
	return encryptedPrefix + base64.StdEncoding.EncodeToString(sealed), nil
}

// DecryptToken decrypts a value previously returned by EncryptToken.
func DecryptToken(encrypted string) (string, error) {
	if !strings.HasPrefix(encrypted, encryptedPrefix) {
		return "", fmt.Errorf("unsupported encrypted token format")
	}

	payload, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(encrypted, encryptedPrefix))
	if err != nil {
		return "", fmt.Errorf("cannot decode encrypted token: %w", err)
	}

	key, err := loadOrCreateKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("cannot create token cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("cannot create token cipher mode: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(payload) <= nonceSize {
		return "", fmt.Errorf("encrypted token is too short")
	}

	nonce, ciphertext := payload[:nonceSize], payload[nonceSize:]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("cannot decrypt token: %w", err)
	}
	return string(plain), nil
}

func loadOrCreateKey() ([]byte, error) {
	dir, err := configDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("cannot create config directory: %w", err)
	}

	path := filepath.Join(dir, keyFileName)
	key, err := os.ReadFile(path)
	if err == nil {
		if len(key) != encryptionKeySize {
			return nil, fmt.Errorf("invalid token key length in %s", path)
		}
		return key, nil
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("cannot read token key: %w", err)
	}

	key = make([]byte, encryptionKeySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("cannot generate token key: %w", err)
	}
	if err := os.WriteFile(path, key, 0600); err != nil {
		return nil, fmt.Errorf("cannot write token key: %w", err)
	}
	return key, nil
}

func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".exabits"), nil
}
