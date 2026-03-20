package crypto_test

import (
	"strings"
	"testing"

	"github.com/yourname/gatelink-engine/internal/crypto"
)

const testKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func newTestKeystore(t *testing.T) *crypto.Keystore {
	t.Helper()
	ks, err := crypto.NewWithKey(testKey)
	if err != nil {
		t.Fatalf("create keystore: %v", err)
	}
	return ks
}

func TestEncryptDecryptRoundtrip(t *testing.T) {
	ks := newTestKeystore(t)

	testKeys := []string{
		"api-key-test-12345678",
		"proj-openai-test-key",
		"",
		strings.Repeat("a", 200),
	}

	for _, original := range testKeys {
		encrypted, err := ks.Encrypt(original)
		if err != nil {
			t.Errorf("encrypt %q: %v", original, err)
			continue
		}

		if encrypted == original && original != "" {
			t.Errorf("ciphertext should not equal plaintext")
		}

		decrypted, err := ks.Decrypt(encrypted)
		if err != nil {
			t.Errorf("decrypt: %v", err)
			continue
		}

		if decrypted != original {
			t.Errorf("roundtrip failed: got %q, want %q", decrypted, original)
		}
	}
}

func TestEncryptProducesRandomCiphertext(t *testing.T) {
	ks := newTestKeystore(t)

	plaintext := "same-test-key-value"
	encrypted1, _ := ks.Encrypt(plaintext)
	encrypted2, _ := ks.Encrypt(plaintext)

	if encrypted1 == encrypted2 {
		t.Error("same plaintext should produce different ciphertext (nonce must be random)")
	}
}

func TestDecryptWithWrongCiphertext(t *testing.T) {
	ks := newTestKeystore(t)

	encrypted, _ := ks.Encrypt("test-key")

	_, err := ks.Decrypt(encrypted + "tampered")
	if err == nil {
		t.Error("should fail to decrypt with tampered ciphertext")
	}
}

func TestDecryptInvalidInput(t *testing.T) {
	ks := newTestKeystore(t)

	cases := []string{
		"not-hex-at-all",
		"",
		"abc",
	}

	for _, c := range cases {
		_, err := ks.Decrypt(c)
		if err == nil {
			t.Errorf("should fail for invalid input: %q", c)
		}
	}
}

func TestHint(t *testing.T) {
	cases := []struct {
		input  string
		expect string
	}{
		{"sk-ant-api03-longkey123456789", "sk-ant-a...6789"},
		{"short", "***"},
	}

	for _, c := range cases {
		got := crypto.Hint(c.input)
		if got != c.expect {
			t.Errorf("Hint(%q) = %q, want %q", c.input, got, c.expect)
		}
	}
}
