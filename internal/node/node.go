package node

import (
	"fmt"
	"sync"

	"github.com/rrb115/vdcs/internal/log"
	"github.com/rrb115/vdcs/internal/merkle"
	"github.com/rrb115/vdcs/internal/state"
	"github.com/rrb115/vdcs/internal/storage"
	vdcspb "github.com/rrb115/vdcs/proto"
)

// Node represents a running VDCS node.
type Node struct {
	mu          sync.RWMutex
	log         *log.ConfigLog
	state       *state.StateMachine
	store       storage.Store
	trustedKeys map[string][]byte
}

// Config holds node configuration.
type Config struct {
	StoragePath string
	TrustedKeys map[string][]byte // AuthorID -> PubKey
}

// NewNode initializes a new node.
func NewNode(cfg Config) (*Node, error) {
	// 1. Init components
	l := log.NewConfigLog()
	for id, key := range cfg.TrustedKeys {
		l.AddTrustedAuthor(id, key)
	}

	sm := state.NewStateMachine()

	st, err := storage.NewFileStore(cfg.StoragePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open storage: %w", err)
	}

	n := &Node{
		log:         l,
		state:       sm,
		store:       st,
		trustedKeys: cfg.TrustedKeys,
	}

	// 2. Replay log
	if err := n.replay(); err != nil {
		st.Close()
		return nil, fmt.Errorf("failed to replay log: %w", err)
	}

	return n, nil
}

// replay loads all entries from disk and applies them.
func (n *Node) replay() error {
	entries, err := n.store.LoadAll()
	if err != nil {
		return err
	}

	for _, entry := range entries {
		// We trust the disk content implies it was validated when written?
		// Or should we re-validate?
		// Ideally re-validate to detect corruption.
		if err := n.log.Append(entry); err != nil {
			return fmt.Errorf("replay validation failed at index %d: %w", entry.Index, err)
		}
		n.state.Apply(entry)
	}
	return nil
}

// ProposeEntry adds a new configuration entry.
// For now, this is a direct operation. In Raft, this would Propose to the cluster.
func (n *Node) ProposeEntry(entry *vdcspb.ConfigEntry) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	// 1. Assign Index if missing?
	// The AUTHOR should sign the index.
	// If the Author doesn't know the index, they can't sign it.
	// This implies the client must query HEAD, get index+1, sign, and submit.
	// Optimistic concurrency.

	// 2. Validate against Log
	if err := n.log.Append(entry); err != nil {
		return err
	}

	// 3. Persist
	if err := n.store.Append(entry); err != nil {
		// If persist fails, we are in inconsistent state (Log has it, Disk doesn't).
		// Panic or rollback?
		// Rollback is hard. Panic is safer.
		// Or careful Log structure.
		// For v1, let's just return error and assume the node might need restart if critical.
		return fmt.Errorf("failed to persist: %w", err)
	}

	// 4. Apply to State
	n.state.Apply(entry)

	return nil
}

// GetLatestRoot returns the current Merkle root, version, and head entry hash.
func (n *Node) GetLatestRoot() (uint64, []byte, []byte) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	// Get version from State
	ver := n.state.Version()
	root := n.state.Root()

	// Get Head Hash from Log
	// Log stores entries. Head is at index = ver.
	// Wait, ver is "Last Applied Index".
	// If Log has entries [0, 1], Version is 1. Head is entry 1.
	// We need EntryHash of entry[Version].
	// Log Get is 0-indexed.

	// If log is empty (Version 0 implies genesis? Or empty?)
	// My Log logic: "nextIndex" starts at 0.
	// If Log size is 0, no head.
	size := n.log.Size()

	if size == 0 {
		return 0, root, nil // Genesis state
	}

	// Head is at size-1
	headEntry, err := n.log.Get(size - 1)
	if err != nil {
		// Should not happen if size > 0
		return ver, root, nil
	}

	return ver, root, headEntry.EntryHash
}

// GetProof returns a proof for a key.
func (n *Node) GetProof(key string) (*merkle.Proof, error) {
	return n.state.Prove(key)
}

// Close shuts down the node.
func (n *Node) Close() error {
	return n.store.Close()
}
