package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// EncryptDeterministic encrypts text using AES-GCM with a deterministic nonce derived from the plaintext.
// This allows the same plaintext to always result in the same ciphertext (if key is same).
// Format: Base64(Nonce + Ciphertext + Tag)
func EncryptDeterministic(plaintext, key string) (string, error) {
	if len(key) != 32 {
		return "", fmt.Errorf("key must be exactly 32 bytes (got %d)", len(key))
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Derive deterministic Nonce using HMAC-SHA256(key, plaintext)
	// We use the first 12 bytes of the HMAC output as the nonce.
	nonce := deriveNonce(plaintext, key)

	// Encrypt
	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)

	// Combine Nonce + Ciphertext (GCM tag is already appended to ciphertext by Seal)
	finalPayload := append(nonce, ciphertext...)

	return base64.StdEncoding.EncodeToString(finalPayload), nil
}

// DecryptDeterministic decrypts a base64 encoded string encrypted with EncryptDeterministic
func DecryptDeterministic(cryptoText, key string) (string, error) {
	if len(key) != 32 {
		return "", fmt.Errorf("key must be exactly 32 bytes")
	}

	data, err := base64.StdEncoding.DecodeString(cryptoText)
	if err != nil {
		return "", err
	}

	if len(data) < 12 {
		return "", fmt.Errorf("invalid ciphertext length")
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := data[:12]
	ciphertext := data[12:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// deriveNonce generates a deterministic 12-byte nonce
func deriveNonce(plaintext, key string) []byte {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(plaintext))
	sum := h.Sum(nil)
	return sum[:12] // GCM standard nonce size
}
