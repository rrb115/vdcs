package state

import (
	"sync"

	"github.com/rrb115/vdcs/internal/merkle"
	vdcspb "github.com/rrb115/vdcs/proto"
)

// StateMachine maintains the current in-memory state derived from the log.
type StateMachine struct {
	mu      sync.RWMutex
	kv      map[string][]byte // Current Key -> ValueHash
	values  map[string][]byte // Optional: Current Key -> Value (if we store values inline)
	version uint64            // Last applied index
	tree    *merkle.Tree      // Cached Merkle Tree
}

// NewStateMachine creates a empty state machine.
func NewStateMachine() *StateMachine {
	return &StateMachine{
		kv:      make(map[string][]byte),
		values:  make(map[string][]byte),
		version: 0,
	}
}

// Apply applies a config entry to the state.
// It assumes the entry has already been validated by the Log.
func (sm *StateMachine) Apply(entry *vdcspb.ConfigEntry) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// In a real system, we might want to ensure entry.Index == sm.version + 1.
	// But entry.Index is 0-based from log.
	// Let's track strictly.
	// If first entry (0), sm.version is 0.
	// Wait, version usually tracks *latest applied*.
	// Let's say initial version is -1 (uint64 wrap) or use a "AppliedCount".
	// Let's act idempotent or strict.

	switch entry.Operation {
	case vdcspb.Operation_OPERATION_SET:
		sm.kv[entry.Key] = entry.ValueHash
		// If entry.Value is present, store it?
		// Spec says "ValueHash [32]byte".
		// But in Section 2.2: "KeyValues map[string][]byte".
		// We'll store what we have.
		// If ValueHash is the value, then we are good.
		// If ValueHash is H(Value), we need Value to support `vdcs get`.
	case vdcspb.Operation_OPERATION_DELETE:
		delete(sm.kv, entry.Key)
	}

	sm.version = entry.Index
	// Invalidate tree
	sm.tree = nil
}

// Root returns the Merkle root of the current state.
// Rebuilds key if dirty.
func (sm *StateMachine) Root() []byte {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.tree == nil {
		sm.tree = merkle.NewTree(sm.kv)
	}
	return sm.tree.Root()
}

// Get returns the value hash for a key.
func (sm *StateMachine) Get(key string) ([]byte, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	v, ok := sm.kv[key]
	return v, ok
}

// Version returns the last applied configuration index.
func (sm *StateMachine) Version() uint64 {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.version
}

// Prove generates a proof for the key against the current root.
func (sm *StateMachine) Prove(key string) (*merkle.Proof, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.tree == nil {
		sm.tree = merkle.NewTree(sm.kv)
	}
	return sm.tree.GenerateProof(key)
}
