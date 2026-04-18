package encryption

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	cases := []struct {
		name      string
		plaintext string
		secret    string
	}{
		{"simple password", "secret-password", "test-key-32-chars-long-for-aes!!"},
		{"empty plaintext", "", "test-key-32-chars-long-for-aes!!"},
		{"unicode plaintext", "pässwörd-日本語", "test-key-32-chars-long-for-aes!!"},
		{"long plaintext", strings.Repeat("a", 1000), "test-key-32-chars-long-for-aes!!"},
		{"short secret", "hello", "x"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			encrypted, err := Encrypt(tc.plaintext, tc.secret)
			if err != nil {
				t.Fatalf("Encrypt failed: %v", err)
			}
			if encrypted == "" {
				t.Fatal("expected non-empty ciphertext")
			}

			decrypted, err := Decrypt(encrypted, tc.secret)
			if err != nil {
				t.Fatalf("Decrypt failed: %v", err)
			}
			if decrypted != tc.plaintext {
				t.Errorf("got %q, want %q", decrypted, tc.plaintext)
			}
		})
	}
}

func TestEncrypt_DifferentOutputEachCall(t *testing.T) {
	// GCM uses a random nonce — each call produces a different ciphertext
	enc1, _ := Encrypt("password", "secret")
	enc2, _ := Encrypt("password", "secret")
	if enc1 == enc2 {
		t.Error("expected different ciphertext each call due to random nonce")
	}
}

func TestDecrypt_WrongSecret(t *testing.T) {
	encrypted, err := Encrypt("password", "correct-secret")
	if err != nil {
		t.Fatal(err)
	}
	_, err = Decrypt(encrypted, "wrong-secret")
	if err == nil {
		t.Error("expected error when decrypting with wrong secret")
	}
}

func TestDecrypt_InvalidBase64(t *testing.T) {
	_, err := Decrypt("not-valid-base64!!!", "secret")
	if err == nil {
		t.Error("expected error for invalid base64 input")
	}
}

func TestDecrypt_TooShortCiphertext(t *testing.T) {
	// Valid base64 but shorter than nonce size (12 bytes for GCM)
	short := base64.StdEncoding.EncodeToString([]byte("tooshort"))
	_, err := Decrypt(short, "secret")
	if err == nil {
		t.Error("expected error for ciphertext too short")
	}
}
