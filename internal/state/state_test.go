package state

import (
	"bytes"
	"testing"

	"github.com/rrb115/vdcs/internal/crypto"
	vdcspb "github.com/rrb115/vdcs/proto"
)

func TestStateApply(t *testing.T) {
	sm := NewStateMachine()

	key := "key1"
	valHash := crypto.Hash([]byte("value1"))
	valHashBytes := valHash[:]

	// 1. Apple SET Operation
	entry := &vdcspb.ConfigEntry{
		Index:     0,
		Key:       key,
		ValueHash: valHashBytes,
		Operation: vdcspb.Operation_OPERATION_SET,
	}
	sm.Apply(entry)

	got, ok := sm.Get(key)
	if !ok {
		t.Error("key not found after SET")
	}
	if !bytes.Equal(got, valHashBytes) {
		t.Error("value hash mismatch")
	}

	// 2. Check Root
	root := sm.Root()
	if len(root) != 32 {
		t.Error("invalid root length")
	}

	// 3. Verify Proof
	proof, err := sm.Prove(key)
	if err != nil {
		t.Fatal(err)
	}
	if !proof.Verify(root) {
		t.Error("proof verification failed")
	}

	// 4. Apply DELETE Operation
	entry2 := &vdcspb.ConfigEntry{
		Index:     1,
		Key:       key,
		Operation: vdcspb.Operation_OPERATION_DELETE,
	}
	sm.Apply(entry2)

	_, ok = sm.Get(key)
	if ok {
		t.Error("key found after DELETE")
	}
}
