package infra

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

// TestEncrypt_Success tests successful encryption
func TestEncrypt_Success(t *testing.T) {
	plaintext := "my-secret-password"
	key := make([]byte, 32) // Valid 32-byte key

	ciphertext, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt() failed: %v", err)
	}

	if ciphertext == "" {
		t.Error("Encrypt() returned empty ciphertext")
	}

	if ciphertext == plaintext {
		t.Error("Encrypt() returned plaintext instead of ciphertext")
	}
}

// TestDecrypt_Success tests successful decryption
func TestDecrypt_Success(t *testing.T) {
	plaintext := "my-secret-password"
	key := make([]byte, 32)

	// First encrypt
	ciphertext, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt() failed: %v", err)
	}

	// Then decrypt
	decrypted, err := Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("Decrypt() failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Decrypt() returned %q, want %q", decrypted, plaintext)
	}
}

// TestEncrypt_Decrypt_Roundtrip tests complete encrypt-decrypt cycle
func TestEncrypt_Decrypt_Roundtrip(t *testing.T) {
	testCases := []string{
		"short",
		"medium-length-password",
		"a-much-longer-password-with-special-chars-!@#$%^&*()",
		"password-with-unicode-你好世界",
		"",
	}

	key := make([]byte, 32)

	for _, plaintext := range testCases {
		t.Run(plaintext, func(t *testing.T) {
			ciphertext, err := Encrypt(plaintext, key)
			if err != nil {
				t.Fatalf("Encrypt() failed: %v", err)
			}

			decrypted, err := Decrypt(ciphertext, key)
			if err != nil {
				t.Fatalf("Decrypt() failed: %v", err)
			}

			if decrypted != plaintext {
				t.Errorf("roundtrip failed: got %q, want %q", decrypted, plaintext)
			}
		})
	}
}

// TestDecrypt_TamperedCiphertext_Fails tests that tampered ciphertext fails decryption
func TestDecrypt_TamperedCiphertext_Fails(t *testing.T) {
	plaintext := "my-secret-password"
	key := make([]byte, 32)

	ciphertext, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt() failed: %v", err)
	}

	// Decode the base64 ciphertext to tamper with bytes
	decoded, err := base64ToBytes(ciphertext)
	if err != nil {
		t.Fatalf("Failed to decode ciphertext: %v", err)
	}

	// Tamper with the ciphertext bytes (not the nonce)
	if len(decoded) > 13 { // Ensure we have room to tamper
		decoded[13] = decoded[13] ^ 0xFF // Flip bits in the first byte of actual ciphertext
	}

	// Re-encode to base64
	tampered := bytesToBase64(decoded)

	_, err = Decrypt(tampered, key)
	if err == nil {
		t.Error("Decrypt() with tampered ciphertext should fail but succeeded")
	}

	if !errors.Is(err, ErrDecryptionFailed) && !strings.Contains(err.Error(), "decryption failed") {
		t.Errorf("expected ErrDecryptionFailed, got: %v", err)
	}
}

// TestDecrypt_WrongKey_Fails tests that wrong key fails decryption
func TestDecrypt_WrongKey_Fails(t *testing.T) {
	plaintext := "my-secret-password"
	rightKey := make([]byte, 32)
	wrongKey := make([]byte, 32)
	wrongKey[0] = 0xFF // Make it different

	ciphertext, err := Encrypt(plaintext, rightKey)
	if err != nil {
		t.Fatalf("Encrypt() failed: %v", err)
	}

	_, err = Decrypt(ciphertext, wrongKey)
	if err == nil {
		t.Error("Decrypt() with wrong key should fail but succeeded")
	}

	if !errors.Is(err, ErrDecryptionFailed) && !strings.Contains(err.Error(), "decryption failed") {
		t.Errorf("expected ErrDecryptionFailed, got: %v", err)
	}
}

// TestEncrypt_InvalidKeyLength_Fails tests that invalid key lengths fail
func TestEncrypt_InvalidKeyLength_Fails(t *testing.T) {
	plaintext := "my-secret-password"

	testCases := [][]byte{
		make([]byte, 16), // Too short
		make([]byte, 24), // Still too short
		make([]byte, 64), // Too long
		nil,              // Empty
	}

	for _, key := range testCases {
		t.Run("", func(t *testing.T) {
			_, err := Encrypt(plaintext, key)
			if err == nil {
				t.Error("Encrypt() with invalid key length should fail but succeeded")
			}

			if !errors.Is(err, ErrInvalidKeyLength) {
				t.Errorf("expected ErrInvalidKeyLength, got: %v", err)
			}
		})
	}
}

// TestDecrypt_InvalidKeyLength_Fails tests that invalid key lengths fail decryption
func TestDecrypt_InvalidKeyLength_Fails(t *testing.T) {
	ciphertext := "invalid-base64-but-key-length-test-comes-first"
	key := make([]byte, 16) // Wrong length

	_, err := Decrypt(ciphertext, key)
	if err == nil {
		t.Error("Decrypt() with invalid key length should fail but succeeded")
	}

	if !errors.Is(err, ErrInvalidKeyLength) {
		t.Errorf("expected ErrInvalidKeyLength, got: %v", err)
	}
}

// TestDecrypt_InvalidCiphertext_Fails tests that invalid ciphertext fails decryption
func TestDecrypt_InvalidCiphertext_Fails(t *testing.T) {
	key := make([]byte, 32)

	testCases := []string{
		"",                  // Empty
		"invalid-base64!!!", // Invalid base64
		"dG9vLXNob3J0",      // Valid base64 but too short
	}

	for _, ciphertext := range testCases {
		t.Run(ciphertext, func(t *testing.T) {
			_, err := Decrypt(ciphertext, key)
			if err == nil {
				t.Error("Decrypt() with invalid ciphertext should fail but succeeded")
			}

			if !errors.Is(err, ErrInvalidCiphertext) && !errors.Is(err, ErrDecryptionFailed) {
				t.Errorf("expected ErrInvalidCiphertext or ErrDecryptionFailed, got: %v", err)
			}
		})
	}
}

// TestEncrypt_NonceUniqueness tests that each encryption generates unique nonce
func TestEncrypt_NonceUniqueness(t *testing.T) {
	plaintext := "my-secret-password"
	key := make([]byte, 32)

	ciphertext1, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("First Encrypt() failed: %v", err)
	}

	ciphertext2, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Second Encrypt() failed: %v", err)
	}

	if ciphertext1 == ciphertext2 {
		t.Error("Encrypt() should generate unique ciphertext for same plaintext (nonce uniqueness)")
	}
}

// TestEncrypt_InvalidCipherCreation tests error handling when cipher creation fails
func TestEncrypt_InvalidCipherCreation(t *testing.T) {
	plaintext := "test-password"

	// Test with invalid key length (should fail at cipher creation)
	invalidKey := make([]byte, 16)

	_, err := Encrypt(plaintext, invalidKey)
	if err == nil {
		t.Error("Encrypt() with invalid key should fail")
	}

	if !errors.Is(err, ErrInvalidKeyLength) {
		t.Errorf("Expected ErrInvalidKeyLength, got: %v", err)
	}
}

// TestDecrypt_InvalidCipherCreation tests error handling when cipher creation fails during decryption
func TestDecrypt_InvalidCipherCreationDuringDecryption(t *testing.T) {
	key := make([]byte, 32)

	// Create valid ciphertext first
	plaintext := "test-password"
	ciphertext, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Failed to create test ciphertext: %v", err)
	}

	// Try to decrypt with invalid key
	invalidKey := make([]byte, 16)

	_, err = Decrypt(ciphertext, invalidKey)
	if err == nil {
		t.Error("Decrypt() with invalid key should fail")
	}

	if !errors.Is(err, ErrInvalidKeyLength) {
		t.Errorf("Expected ErrInvalidKeyLength, got: %v", err)
	}
}

// TestDecrypt_TruncatedCiphertext tests decryption with truncated ciphertext
func TestDecrypt_TruncatedCiphertext(t *testing.T) {
	key := make([]byte, 32)

	// Create valid ciphertext first
	plaintext := "test-password"
	ciphertext, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Failed to create test ciphertext: %v", err)
	}

	// Decode and truncate the ciphertext
	decoded, err := base64ToBytes(ciphertext)
	if err != nil {
		t.Fatalf("Failed to decode ciphertext: %v", err)
	}

	// Truncate to less than nonce size (should be 12 bytes for GCM)
	if len(decoded) > 10 {
		truncated := decoded[:10]
		truncatedCiphertext := bytesToBase64(truncated)

		_, err = Decrypt(truncatedCiphertext, key)
		if err == nil {
			t.Error("Decrypt() with truncated ciphertext should fail")
		}

		if !errors.Is(err, ErrInvalidCiphertext) {
			t.Errorf("Expected ErrInvalidCiphertext, got: %v", err)
		}
	}
}

// TestDecrypt_OnlyNonce tests decryption with ciphertext containing only nonce
func TestDecrypt_OnlyNonce(t *testing.T) {
	key := make([]byte, 32)

	// Create valid ciphertext first
	plaintext := "test-password"
	ciphertext, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Failed to create test ciphertext: %v", err)
	}

	// Decode and keep only the nonce part (first 12 bytes for GCM)
	decoded, err := base64ToBytes(ciphertext)
	if err != nil {
		t.Fatalf("Failed to decode ciphertext: %v", err)
	}

	// Keep only nonce (12 bytes for GCM)
	if len(decoded) > 12 {
		onlyNonce := decoded[:12]
		onlyNonceCiphertext := bytesToBase64(onlyNonce)

		_, err = Decrypt(onlyNonceCiphertext, key)
		if err == nil {
			t.Error("Decrypt() with only nonce should fail")
		}

		if !errors.Is(err, ErrDecryptionFailed) && !errors.Is(err, ErrInvalidCiphertext) {
			t.Errorf("Expected ErrDecryptionFailed or ErrInvalidCiphertext, got: %v", err)
		}
	}
}

// Helper function to tamper with ciphertext
func tamperWithCiphertext(ciphertext string) string {
	if len(ciphertext) == 0 {
		return "tampered"
	}

	// Change the last character
	runes := []rune(ciphertext)
	runes[len(runes)-1]++
	if runes[len(runes)-1] > 126 {
		runes[len(runes)-1] = 33 // Wrap around
	}

	return string(runes)
}

// Helper functions for base64 encoding/decoding in tests
func base64ToBytes(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

func bytesToBase64(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}
