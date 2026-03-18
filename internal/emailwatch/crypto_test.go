package emailwatch

import (
	"encoding/hex"
	"testing"
)

func testKey() string {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	return hex.EncodeToString(key)
}

func TestEncryptDecrypt(t *testing.T) {
	key := testKey()
	plaintext := "my_secret_password_123"

	encrypted, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	if string(encrypted) == plaintext {
		t.Error("encrypted text should not equal plaintext")
	}

	decrypted, err := Decrypt(encrypted, key)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Decrypt = %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptDifferentNonces(t *testing.T) {
	key := testKey()
	plaintext := "same_password"

	enc1, _ := Encrypt(plaintext, key)
	enc2, _ := Encrypt(plaintext, key)

	if string(enc1) == string(enc2) {
		t.Error("two encryptions of same plaintext should produce different ciphertexts")
	}

	dec1, _ := Decrypt(enc1, key)
	dec2, _ := Decrypt(enc2, key)
	if dec1 != dec2 {
		t.Error("both should decrypt to same plaintext")
	}
}

func TestDecryptWrongKey(t *testing.T) {
	key1 := testKey()
	key2 := hex.EncodeToString(make([]byte, 32)) // all zeros

	encrypted, _ := Encrypt("secret", key1)
	_, err := Decrypt(encrypted, key2)
	if err == nil {
		t.Error("expected error when decrypting with wrong key")
	}
}

func TestEncryptInvalidKey(t *testing.T) {
	_, err := Encrypt("test", "short")
	if err == nil {
		t.Error("expected error for short key")
	}

	_, err = Encrypt("test", "not-hex")
	if err == nil {
		t.Error("expected error for non-hex key")
	}
}

func TestDecryptTooShort(t *testing.T) {
	key := testKey()
	_, err := Decrypt([]byte("short"), key)
	if err == nil {
		t.Error("expected error for too-short ciphertext")
	}
}

func TestEncryptEmptyPlaintext(t *testing.T) {
	key := testKey()
	encrypted, err := Encrypt("", key)
	if err != nil {
		t.Fatalf("Encrypt empty: %v", err)
	}
	decrypted, err := Decrypt(encrypted, key)
	if err != nil {
		t.Fatalf("Decrypt empty: %v", err)
	}
	if decrypted != "" {
		t.Errorf("expected empty, got %q", decrypted)
	}
}
