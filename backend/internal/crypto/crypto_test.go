package crypto_test

import (
	"testing"

	"github.com/redora/redora/backend/internal/crypto"
)

func TestEncryptDecrypt(t *testing.T) {
	secretKey := "my-secret-app-key-12345"
	enc := crypto.NewEncryptor(secretKey)

	originalPassword := "SuperSecretRedisPassword123!"
	encrypted, err := enc.Encrypt(originalPassword)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	if encrypted == originalPassword {
		t.Fatal("Encrypted string matches plaintext!")
	}

	decrypted, err := enc.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	if decrypted != originalPassword {
		t.Fatalf("Expected '%s', got '%s'", originalPassword, decrypted)
	}
}

func TestEmptyPassword(t *testing.T) {
	enc := crypto.NewEncryptor("key")
	encrypted, err := enc.Encrypt("")
	if err != nil {
		t.Fatalf("Encryption of empty string failed: %v", err)
	}
	if encrypted != "" {
		t.Fatalf("Expected empty encrypted string, got '%s'", encrypted)
	}

	decrypted, err := enc.Decrypt("")
	if err != nil {
		t.Fatalf("Decryption of empty string failed: %v", err)
	}
	if decrypted != "" {
		t.Fatalf("Expected empty decrypted string, got '%s'", decrypted)
	}
}
