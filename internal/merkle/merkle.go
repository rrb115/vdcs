package merkle

import (
	"sort"

	"github.com/rrb115/vdcs/internal/crypto"
)

// Tree represents a Merkle tree computed from a KV state.
type Tree struct {
	root        []byte
	leaves      map[string][]byte // Map key -> computed leaf hash
	valueHashes map[string][]byte // Map key -> original value hash
	keys        []string          // Sorted keys for deterministic tree construction
}

// NewTree builds a Merkle tree from the given map of key -> valueHash.
// The keys are sorted to ensure determinism.
func NewTree(kv map[string][]byte) *Tree {
	keys := make([]string, 0, len(kv))
	leaves := make(map[string][]byte)
	valueHashes := make(map[string][]byte)

	for k, vHash := range kv {
		keys = append(keys, k)
		// Leaf = Hash(Key || ValueHash)
		input := append([]byte(k), vHash...)
		hash := crypto.Hash(input)
		leaves[k] = hash[:]
		valueHashes[k] = vHash
	}

	sort.Strings(keys)

	t := &Tree{
		leaves:      leaves,
		valueHashes: valueHashes,
		keys:        keys,
	}
	t.root = t.computeRoot(keys)
	return t
}

// Root returns the Merkle root of the tree.
func (t *Tree) Root() []byte {
	return t.root
}

// computeRoot recursively builds the tree.
func (t *Tree) computeRoot(nodes []string) []byte {
	if len(nodes) == 0 {
		h := crypto.Hash([]byte("empty"))
		return h[:] // Hash of empty for empty tree
	}
	if len(nodes) == 1 {
		return t.leaves[nodes[0]]
	}

	mid := len(nodes) / 2
	leftRoot := t.computeRoot(nodes[:mid])
	rightRoot := t.computeRoot(nodes[mid:])

	h := crypto.Hash(append(leftRoot, rightRoot...))
	return h[:]
}
