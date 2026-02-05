package log

import (
	"testing"
	"time"

	"github.com/rrb115/vdcs/internal/crypto"
	vdcspb "github.com/rrb115/vdcs/proto"
)

func TestLogAppend(t *testing.T) {
	// 1. Setup keys
	pub, priv, _ := crypto.GenerateKey()
	authorID := "author1"

	// 2. Setup Log
	l := NewConfigLog()
	l.AddTrustedAuthor(authorID, pub)

	// 3. Create Genesis Entry
	entry := &vdcspb.ConfigEntry{
		Index:     0,
		Timestamp: time.Now().UnixNano(),
		AuthorId:  authorID,
		Key:       "key1",
		ValueHash: func() []byte { h := crypto.Hash([]byte("value1")); return h[:] }(),
		Operation: vdcspb.Operation_OPERATION_SET,
		PrevHash:  nil,
	}

	// Compute EntryHash
	hash, err := ComputeEntryHash(entry)
	if err != nil {
		t.Fatal(err)
	}
	entry.EntryHash = hash
	entry.Signature = crypto.Sign(priv, hash)

	// 4. Append
	if err := l.Append(entry); err != nil {
		t.Fatalf("failed to append genesis entry: %v", err)
	}

	if l.Size() != 1 {
		t.Errorf("expected size 1, got %d", l.Size())
	}

	// 5. Append Second Entry
	entry2 := &vdcspb.ConfigEntry{
		Index:     1,
		Timestamp: time.Now().UnixNano(),
		AuthorId:  authorID,
		Key:       "key2",
		ValueHash: func() []byte { h := crypto.Hash([]byte("value2")); return h[:] }(),
		Operation: vdcspb.Operation_OPERATION_SET,
		PrevHash:  entry.EntryHash, // Must match previous EntryHash
	}
	hash2, err := ComputeEntryHash(entry2)
	if err != nil {
		t.Fatal(err)
	}
	entry2.EntryHash = hash2
	entry2.Signature = crypto.Sign(priv, hash2)

	if err := l.Append(entry2); err != nil {
		t.Fatalf("failed to append second entry: %v", err)
	}
}

func TestLogValidation(t *testing.T) {
	pub, priv, _ := crypto.GenerateKey()
	authorID := "author1"
	l := NewConfigLog()
	l.AddTrustedAuthor(authorID, pub)

	baseEntry := func() *vdcspb.ConfigEntry {
		return &vdcspb.ConfigEntry{
			Index:     0,
			Timestamp: 100,
			AuthorId:  authorID,
			Key:       "k",
			ValueHash: []byte{1},
			Operation: vdcspb.Operation_OPERATION_SET,
		}
	}

	// 1. Invalid Index
	e1 := baseEntry()
	e1.Index = 1
	if l.Append(e1) == nil {
		t.Error("expected error for invalid index 1 (expected 0)")
	}

	// 2. Invalid Signature
	e2 := baseEntry()
	h, _ := ComputeEntryHash(e2)
	e2.EntryHash = h
	e2.Signature = []byte("bad sig")
	if l.Append(e2) == nil {
		t.Error("expected error for invalid signature")
	}

	// 3. Computed Hash Mismatch
	e3 := baseEntry()
	h3, _ := ComputeEntryHash(e3)
	e3.EntryHash = h3
	e3.Signature = crypto.Sign(priv, h3)
	e3.Timestamp++ // Tamper with field AFTER signing/hashing
	if l.Append(e3) == nil {
		t.Error("expected error for hash mismatch")
	}
}
