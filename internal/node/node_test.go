package node

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rrb115/vdcs/internal/crypto"
	"github.com/rrb115/vdcs/internal/log"
	"github.com/rrb115/vdcs/internal/storage"
	vdcspb "github.com/rrb115/vdcs/proto"
)

func TestNodeLifecycle(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "vdcs-node-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	storagePath := filepath.Join(tmpDir, "log.bin")

	// Keys
	pub, priv, _ := crypto.GenerateKey()
	authorID := "admin"
	trustedKeys := map[string][]byte{
		authorID: pub,
	}

	// 1. Start Node 1
	st, err := storage.NewFileStore(storagePath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	cfg := Config{
		Store:       st,
		TrustedKeys: trustedKeys,
	}
	node, err := NewNode(cfg)
	if err != nil {
		t.Fatalf("failed to start node: %v", err)
	}

	// 2. Propose Entry
	entry := &vdcspb.ConfigEntry{
		Index:     0,
		Timestamp: time.Now().UnixNano(),
		AuthorId:  authorID,
		Key:       "db_host",
		ValueHash: func() []byte { h := crypto.Hash([]byte("localhost")); return h[:] }(),
		Operation: vdcspb.Operation_OPERATION_SET,
	}
	// Sign
	hash, _ := log.ComputeEntryHash(entry)
	entry.EntryHash = hash
	entry.Signature = crypto.Sign(priv, hash)

	if err := node.ProposeEntry(entry); err != nil {
		t.Fatalf("propose failed: %v", err)
	}

	// Check State
	_, root, _ := node.GetLatestRoot()
	proof, err := node.GetProof("db_host")
	if err != nil {
		t.Fatal(err)
	}
	if !proof.Verify(root) {
		t.Error("proof failed")
	}

	// 3. Restart Node
	if err := node.Close(); err != nil {
		// Note: node.Close() closes the store.
		t.Fatal(err)
	}

	st2, err := storage.NewFileStore(storagePath)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	cfg.Store = st2

	node2, err := NewNode(cfg)
	if err != nil {
		t.Fatalf("failed to restart node: %v", err)
	}
	defer node2.Close()

	// 4. Verify State recovered
	_, root2, _ := node2.GetLatestRoot()
	if !bytes.Equal(root, root2) {
		t.Error("root mismatch after restart")
	}

	proof2, err := node2.GetProof("db_host")
	if err != nil {
		t.Fatal(err)
	}
	if !proof2.Verify(root2) {
		t.Error("proof failed after restart")
	}
}
