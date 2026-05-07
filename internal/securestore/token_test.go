package securestore

import "testing"

func TestEncryptDecryptToken(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	encrypted, err := EncryptToken("test-token")
	if err != nil {
		t.Fatalf("EncryptToken: %v", err)
	}
	if encrypted == "test-token" {
		t.Fatalf("encrypted token should not equal plaintext")
	}

	decrypted, err := DecryptToken(encrypted)
	if err != nil {
		t.Fatalf("DecryptToken: %v", err)
	}
	if decrypted != "test-token" {
		t.Fatalf("decrypted token = %q, want %q", decrypted, "test-token")
	}
}
