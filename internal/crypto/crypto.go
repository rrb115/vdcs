package crypto

import (
	"crypto/ed25519"
	"crypto/sha256"
)

// Hash computes the SHA-256 hash of the input data.
func Hash(data []byte) [32]byte {
	return sha256.Sum256(data)
}

// GenerateKey generates a new Ed25519 key pair.
func GenerateKey() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	return ed25519.GenerateKey(nil)
}

// Sign signs the message with the private key using Ed25519.
func Sign(privateKey ed25519.PrivateKey, message []byte) []byte {
	return ed25519.Sign(privateKey, message)
}

// Verify verifies the signature against the message and public key.
func Verify(publicKey ed25519.PublicKey, message, signature []byte) bool {
	return ed25519.Verify(publicKey, message, signature)
}
