package storage

import (
	"os"
	"path/filepath"
	"testing"

	vdcspb "github.com/rrb115/vdcs/proto"
)

func TestFileStore(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "vdcs-storage-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	path := filepath.Join(tmpDir, "log.bin")

	// 1. Open new store
	store, err := NewFileStore(path)
	if err != nil {
		t.Fatal(err)
	}

	entries := []*vdcspb.ConfigEntry{
		{Index: 0, Key: "k1", ValueHash: []byte("h1")},
		{Index: 1, Key: "k2", ValueHash: []byte("h2")},
	}

	// 2. Append entries
	for _, e := range entries {
		if err := store.Append(e); err != nil {
			t.Fatalf("failed to append: %v", err)
		}
	}

	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	// 3. Re-open and Load
	store2, err := NewFileStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store2.Close()

	loaded, err := store2.LoadAll()
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}

	if len(loaded) != len(entries) {
		t.Errorf("expected %d entries, got %d", len(entries), len(loaded))
	}

	for i, e := range loaded {
		if e.Key != entries[i].Key {
			t.Errorf("entry %d key mismatch: got %s, want %s", i, e.Key, entries[i].Key)
		}
	}
}
