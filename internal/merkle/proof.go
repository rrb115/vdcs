package merkle

import (
	"bytes"
	"errors"

	"github.com/rrb115/vdcs/internal/crypto"
)

// Proof represents a Merkle inclusion proof.
type Proof struct {
	Key       string
	ValueHash []byte
	Siblings  [][]byte
	// IsLeft indicates if the Sibling at the same index is on the Left side.
	// If true: H(Sibling || Current)
	// If false: H(Current || Sibling)
	IsLeft []bool
}

// GenerateProof creates a Merkle inclusion proof for the given key.
func (t *Tree) GenerateProof(key string) (*Proof, error) {
	// Find if key exists first? No, generateRecursiveProof will check.
	// But generateRecursiveProof needs the whole list.
	return t.generateRecursiveProof(t.keys, key)
}

// generateRecursiveProof recursively traverses the virtual tree to build the proof.
func (t *Tree) generateRecursiveProof(nodes []string, targetKey string) (*Proof, error) {
	if len(nodes) == 0 {
		return nil, errors.New("key not found (empty subtree)")
	}
	// Use the original value hash, not the computed leaf hash
	vHash, ok := t.valueHashes[targetKey]
	if !ok {
		// Should verify logic if key exists in nodes but not map?
		// Unlikely if tree constructed correctly.
		return nil, errors.New("value hash not found")
	}

	if len(nodes) == 1 {
		if nodes[0] == targetKey {
			return &Proof{
				Key:       targetKey,
				ValueHash: vHash,
				Siblings:  [][]byte{},
				IsLeft:    []bool{},
			}, nil
		}
		return nil, errors.New("key not found")
	}

	mid := len(nodes) / 2
	leftNodes := nodes[:mid]
	rightNodes := nodes[mid:]

	// Determine side
	inLeft := false
	if len(rightNodes) > 0 {
		// Since sorted, if targetKey < first key of right, it MUST be in left.
		// If targetKey >= rightNodes[0], it MUST be in right.
		if targetKey < rightNodes[0] {
			inLeft = true
		} else {
			inLeft = false
		}
	} else {
		inLeft = true
	}

	var proof *Proof
	var err error
	var siblingHash []byte
	var isLeftSibling bool

	if inLeft {
		proof, err = t.generateRecursiveProof(leftNodes, targetKey)
		if err != nil {
			return nil, err
		}
		// Sibling is the Right subtree
		siblingHash = t.computeRoot(rightNodes)
		isLeftSibling = false // Sibling is on the Right
	} else {
		proof, err = t.generateRecursiveProof(rightNodes, targetKey)
		if err != nil {
			return nil, err
		}
		// Sibling is the Left subtree
		siblingHash = t.computeRoot(leftNodes)
		isLeftSibling = true // Sibling is on the Left
	}

	// Append sibling to the proof.
	proof.Siblings = append(proof.Siblings, siblingHash)
	proof.IsLeft = append(proof.IsLeft, isLeftSibling)
	return proof, nil
}

// Verify checks if the proof is valid for the given root.
func (p *Proof) Verify(root []byte) bool {
	// 1. Start with the leaf hash.
	input := append([]byte(p.Key), p.ValueHash...)
	currentHash := crypto.Hash(input)
	currentHashBytes := currentHash[:]

	// 2. Apply siblings up to the root.
	if len(p.Siblings) != len(p.IsLeft) {
		return false
	}

	for i := 0; i < len(p.Siblings); i++ {
		sibling := p.Siblings[i]
		isLeft := p.IsLeft[i]

		var nextInput []byte
		if isLeft {
			// Sibling is Left, Current is Right
			// Hash(Sibling || Current)
			nextInput = append(sibling, currentHashBytes...)
		} else {
			// Sibling is Right, Current is Left
			// Hash(Current || Sibling)
			nextInput = append(currentHashBytes, sibling...)
		}

		newHash := crypto.Hash(nextInput)
		currentHashBytes = newHash[:]
	}

	return bytes.Equal(currentHashBytes, root)
}
