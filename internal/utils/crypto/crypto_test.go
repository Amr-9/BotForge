package crypto_test

import (
	"strings"
	"testing"

	"github.com/Amr-9/botforge/internal/utils/crypto"
)

const validKey = "12345678901234567890123456789012" // 32 bytes

// ==================== EncryptDeterministic Tests ====================

func TestEncryptDeterministic_ValidInput(t *testing.T) {
	plaintext := "Hello, World!"

	ciphertext, err := crypto.EncryptDeterministic(plaintext, validKey)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if ciphertext == "" {
		t.Error("Expected non-empty ciphertext")
	}

	if ciphertext == plaintext {
		t.Error("Ciphertext should not equal plaintext")
	}
}

func TestEncryptDeterministic_EmptyString(t *testing.T) {
	ciphertext, err := crypto.EncryptDeterministic("", validKey)
	if err != nil {
		t.Fatalf("Expected no error for empty string, got: %v", err)
	}

	if ciphertext == "" {
		t.Error("Expected non-empty ciphertext even for empty plaintext")
	}
}

func TestEncryptDeterministic_UnicodeText(t *testing.T) {
	// Arabic text
	plaintext := "مرحباً بالعالم"

	ciphertext, err := crypto.EncryptDeterministic(plaintext, validKey)
	if err != nil {
		t.Fatalf("Expected no error for unicode text, got: %v", err)
	}

	if ciphertext == "" {
		t.Error("Expected non-empty ciphertext for unicode text")
	}
}

func TestEncryptDeterministic_LongText(t *testing.T) {
	// Generate a long text (10KB)
	plaintext := strings.Repeat("A", 10*1024)

	ciphertext, err := crypto.EncryptDeterministic(plaintext, validKey)
	if err != nil {
		t.Fatalf("Expected no error for long text, got: %v", err)
	}

	if ciphertext == "" {
		t.Error("Expected non-empty ciphertext for long text")
	}
}

func TestEncryptDeterministic_Deterministic(t *testing.T) {
	plaintext := "Test deterministic encryption"

	ciphertext1, err := crypto.EncryptDeterministic(plaintext, validKey)
	if err != nil {
		t.Fatalf("First encryption failed: %v", err)
	}

	ciphertext2, err := crypto.EncryptDeterministic(plaintext, validKey)
	if err != nil {
		t.Fatalf("Second encryption failed: %v", err)
	}

	if ciphertext1 != ciphertext2 {
		t.Error("Same plaintext with same key should produce same ciphertext")
	}
}

func TestEncryptDeterministic_DifferentPlaintextsDifferentCiphertexts(t *testing.T) {
	ciphertext1, _ := crypto.EncryptDeterministic("Hello", validKey)
	ciphertext2, _ := crypto.EncryptDeterministic("World", validKey)

	if ciphertext1 == ciphertext2 {
		t.Error("Different plaintexts should produce different ciphertexts")
	}
}

func TestEncryptDeterministic_InvalidKeyLength_Short(t *testing.T) {
	_, err := crypto.EncryptDeterministic("test", "short_key")
	if err == nil {
		t.Error("Expected error for key shorter than 32 bytes")
	}

	if !strings.Contains(err.Error(), "32 bytes") {
		t.Errorf("Error should mention 32 bytes requirement, got: %v", err)
	}
}

func TestEncryptDeterministic_InvalidKeyLength_Long(t *testing.T) {
	longKey := strings.Repeat("x", 64)
	_, err := crypto.EncryptDeterministic("test", longKey)
	if err == nil {
		t.Error("Expected error for key longer than 32 bytes")
	}
}

func TestEncryptDeterministic_EmptyKey(t *testing.T) {
	_, err := crypto.EncryptDeterministic("test", "")
	if err == nil {
		t.Error("Expected error for empty key")
	}
}

// ==================== DecryptDeterministic Tests ====================

func TestDecryptDeterministic_ValidInput(t *testing.T) {
	original := "Hello, World!"

	ciphertext, err := crypto.EncryptDeterministic(original, validKey)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	decrypted, err := crypto.DecryptDeterministic(ciphertext, validKey)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	if decrypted != original {
		t.Errorf("Expected '%s', got '%s'", original, decrypted)
	}
}

func TestDecryptDeterministic_RoundTrip_Unicode(t *testing.T) {
	testCases := []string{
		"مرحباً بالعالم", // Arabic
		"こんにちは世界",        // Japanese
		"🎉🚀💻",            // Emoji
		"Mixed العربية English 日本語",
	}

	for _, original := range testCases {
		ciphertext, err := crypto.EncryptDeterministic(original, validKey)
		if err != nil {
			t.Fatalf("Encryption failed for '%s': %v", original, err)
		}

		decrypted, err := crypto.DecryptDeterministic(ciphertext, validKey)
		if err != nil {
			t.Fatalf("Decryption failed for '%s': %v", original, err)
		}

		if decrypted != original {
			t.Errorf("Round trip failed: expected '%s', got '%s'", original, decrypted)
		}
	}
}

func TestDecryptDeterministic_RoundTrip_EmptyString(t *testing.T) {
	ciphertext, err := crypto.EncryptDeterministic("", validKey)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	decrypted, err := crypto.DecryptDeterministic(ciphertext, validKey)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	if decrypted != "" {
		t.Errorf("Expected empty string, got '%s'", decrypted)
	}
}

func TestDecryptDeterministic_InvalidKeyLength(t *testing.T) {
	ciphertext, _ := crypto.EncryptDeterministic("test", validKey)

	_, err := crypto.DecryptDeterministic(ciphertext, "short_key")
	if err == nil {
		t.Error("Expected error for invalid key length")
	}
}

func TestDecryptDeterministic_WrongKey(t *testing.T) {
	ciphertext, _ := crypto.EncryptDeterministic("test", validKey)

	wrongKey := "98765432109876543210987654321098" // Different 32-byte key
	_, err := crypto.DecryptDeterministic(ciphertext, wrongKey)
	if err == nil {
		t.Error("Expected error when decrypting with wrong key")
	}
}

func TestDecryptDeterministic_CorruptedData(t *testing.T) {
	ciphertext, _ := crypto.EncryptDeterministic("test", validKey)

	// Corrupt the ciphertext by modifying a character
	corrupted := ciphertext[:len(ciphertext)-1] + "X"

	_, err := crypto.DecryptDeterministic(corrupted, validKey)
	if err == nil {
		t.Error("Expected error for corrupted ciphertext")
	}
}

func TestDecryptDeterministic_InvalidBase64(t *testing.T) {
	_, err := crypto.DecryptDeterministic("not-valid-base64!!!", validKey)
	if err == nil {
		t.Error("Expected error for invalid base64")
	}
}

func TestDecryptDeterministic_TooShortData(t *testing.T) {
	// Base64 of less than 12 bytes
	shortData := "AQIDBAUG" // Only 6 bytes when decoded

	_, err := crypto.DecryptDeterministic(shortData, validKey)
	if err == nil {
		t.Error("Expected error for data shorter than nonce size")
	}
}

func TestDecryptDeterministic_EmptyCiphertext(t *testing.T) {
	_, err := crypto.DecryptDeterministic("", validKey)
	if err == nil {
		t.Error("Expected error for empty ciphertext")
	}
}

// ==================== Security Tests ====================

func TestDifferentKeys_DifferentCiphertexts(t *testing.T) {
	plaintext := "secret message"
	key1 := "12345678901234567890123456789012"
	key2 := "abcdefghijklmnopqrstuvwxyz123456"

	ciphertext1, _ := crypto.EncryptDeterministic(plaintext, key1)
	ciphertext2, _ := crypto.EncryptDeterministic(plaintext, key2)

	if ciphertext1 == ciphertext2 {
		t.Error("Same plaintext with different keys should produce different ciphertexts")
	}
}
