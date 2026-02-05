package merkle

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/rrb115/vdcs/internal/crypto"
)

func TestMerkleTree(t *testing.T) {
	// 1. Setup data
	kv := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
		"key3": []byte("value3"),
	}

	// 2. Build tree
	// Note: keys are "key1", "key2", "key3". Sorted stays same.
	tree := NewTree(kv)

	// 3. Check root
	root := tree.Root()
	if len(root) != 32 {
		t.Errorf("expected root length 32, got %d", len(root))
	}
	t.Logf("Root: %s", hex.EncodeToString(root))

	// 4. Verify Proofs for each key
	for k, v := range kv {
		proof, err := tree.GenerateProof(k)
		if err != nil {
			t.Fatalf("failed to generate proof for %s: %v", k, err)
		}

		// Verify values match
		if !bytes.Equal(proof.ValueHash, v) {
			t.Errorf("proof value hash mismatch for %s", k)
		}

		// Verify proof
		if !proof.Verify(root) {
			t.Errorf("proof verification failed for %s", k)
		}
	}
}

func TestMerkleTree_Empty(t *testing.T) {
	kv := map[string][]byte{}
	tree := NewTree(kv)
	root := tree.Root()
	expected := crypto.Hash([]byte("empty"))
	if !bytes.Equal(root, expected[:]) {
		t.Errorf("expected empty hash for empty tree")
	}
}

func TestMerkleTree_Single(t *testing.T) {
	kv := map[string][]byte{"k": []byte("v")}
	tree := NewTree(kv)
	root := tree.Root()

	// Leaf = Hash(k || v)
	expected := crypto.Hash(append([]byte("k"), []byte("v")...))
	if !bytes.Equal(root, expected[:]) {
		t.Errorf("expected leaf hash for single node tree")
	}

	proof, err := tree.GenerateProof("k")
	if err != nil {
		t.Fatal(err)
	}
	if !proof.Verify(root) {
		t.Error("proof failed for single node")
	}
}

func TestMerkleTree_Tamper(t *testing.T) {
	kv := map[string][]byte{
		"a": []byte("1"),
		"b": []byte("2"),
	}
	tree := NewTree(kv)
	proof, _ := tree.GenerateProof("a")

	// Helper to flip a byte
	tamper := func(b []byte) []byte {
		c := make([]byte, len(b))
		copy(c, b)
		c[0] ^= 0xFF
		return c
	}

	// 1. Tamper ValueHash
	proof.ValueHash = tamper(proof.ValueHash)
	if proof.Verify(tree.Root()) {
		t.Error("proof verified with tampered ValueHash")
	}
	proof.ValueHash = kv["a"] // reset

	// 2. Tamper Key (conceptually separate input, but proof struct has Key)
	proof.Key = "c"
	if proof.Verify(tree.Root()) {
		t.Error("proof verified with tampered Key")
	}
	proof.Key = "a" // reset

	// 3. Tamper Sibling
	if len(proof.Siblings) > 0 {
		proof.Siblings[0] = tamper(proof.Siblings[0])
		if proof.Verify(tree.Root()) {
			t.Error("proof verified with tampered Sibling")
		}
	}
}
