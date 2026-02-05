package crypto

import (
	"crypto/ed25519"
	"encoding/hex"
	"testing"
)

func TestHash(t *testing.T) {
	data := []byte("hello world")
	hash := Hash(data)
	if len(hash) != 32 {
		t.Errorf("expected hash length 32, got %d", len(hash))
	}
	// SHA-256 of "hello world"
	expected := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	if hex.EncodeToString(hash[:]) != expected {
		t.Errorf("expected %s, got %s", expected, hex.EncodeToString(hash[:]))
	}
}

func TestSignAndVerify(t *testing.T) {
	pub, priv, err := GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	msg := []byte("verifiable configuration")
	sig := Sign(priv, msg)

	if len(sig) != ed25519.SignatureSize {
		t.Errorf("expected signature length %d, got %d", ed25519.SignatureSize, len(sig))
	}

	if !Verify(pub, msg, sig) {
		t.Error("verification failed for valid signature")
	}

	if Verify(pub, []byte("tampered data"), sig) {
		t.Error("verification succeeded for tampered data")
	}
}
