package log

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rrb115/vdcs/internal/crypto"
	vdcspb "github.com/rrb115/vdcs/proto"
	"google.golang.org/protobuf/proto"
)

var (
	ErrInvalidIndex     = errors.New("invalid index")
	ErrInvalidPrevHash  = errors.New("invalid previous hash")
	ErrInvalidSignature = errors.New("invalid signature")
	ErrInvalidHash      = errors.New("invalid entry hash")
)

// ConfigLog represents the append-only log of configuration changes.
type ConfigLog struct {
	mu           sync.RWMutex
	entries      []*vdcspb.ConfigEntry
	trustedKeys  map[string]struct{}     // Set of trusted AuthorIDs
	authorConfig map[string]AuthorConfig // Map AuthorID -> Public Key
}

type AuthorConfig struct {
	PublicKey []byte
}

// NewConfigLog creates a new empty log.
func NewConfigLog() *ConfigLog {
	return &ConfigLog{
		entries:      make([]*vdcspb.ConfigEntry, 0),
		trustedKeys:  make(map[string]struct{}),
		authorConfig: make(map[string]AuthorConfig),
	}
}

// AddTrustedAuthor adds a trusted author to the log.
func (l *ConfigLog) AddTrustedAuthor(authorID string, pubKey []byte) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.trustedKeys[authorID] = struct{}{}
	l.authorConfig[authorID] = AuthorConfig{PublicKey: pubKey}
}

// Append validates and adds a new entry to the log.
func (l *ConfigLog) Append(entry *vdcspb.ConfigEntry) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 1. Validate Basic Fields
	nextIndex := uint64(len(l.entries)) // 0-indexed log storage, but entries usually 1-indexed?
	// Let's assume 0-indexed for simplicity or match spec.
	// Spec says "Index uint64".
	// If first entry is 0:
	if entry.Index != nextIndex {
		return fmt.Errorf("%w: expected %d, got %d", ErrInvalidIndex, nextIndex, entry.Index)
	}

	// 2. Validate PrevHash
	if nextIndex == 0 {
		// Genesis entry
		if len(entry.PrevHash) != 0 {
			return fmt.Errorf("%w: genesis prevHash must be empty", ErrInvalidPrevHash)
		}
	} else {
		lastEntry := l.entries[nextIndex-1]
		// We need the hash of the last entry.
		// ideally entry.PrevHash == Hash(lastEntry without signature? or with?)
		// Spec says: "EntryHash [32]byte ... Signature covers EntryHash".
		// So PrevHash should point to the previous entry's EntryHash.
		if !bytes.Equal(entry.PrevHash, lastEntry.EntryHash) {
			return fmt.Errorf("%w: mismatch", ErrInvalidPrevHash)
		}
	}

	// 3. Validate Timestamp (strictly monotonic? or just plausible?)
	// For now, let's just ignore strict timestamp checks unless required.
	_ = time.Unix(0, entry.Timestamp)

	// 4. Validate EntryHash
	// Recompute hash.
	computedHash, err := ComputeEntryHash(entry)
	if err != nil {
		return err
	}
	if !bytes.Equal(computedHash, entry.EntryHash) {
		return fmt.Errorf("%w: computed %x != provided %x", ErrInvalidHash, computedHash, entry.EntryHash)
	}

	// 5. Validate Signature
	if _, ok := l.trustedKeys[entry.AuthorId]; !ok {
		return fmt.Errorf("author %s not trusted", entry.AuthorId)
	}
	pubKey := l.authorConfig[entry.AuthorId].PublicKey
	if !crypto.Verify(pubKey, entry.EntryHash, entry.Signature) {
		return ErrInvalidSignature
	}

	// 6. Commit
	l.entries = append(l.entries, entry)
	return nil
}

// Get returns the entry at the given index.
func (l *ConfigLog) Get(index uint64) (*vdcspb.ConfigEntry, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if index >= uint64(len(l.entries)) {
		return nil, ErrInvalidIndex
	}
	return l.entries[index], nil
}

// Size returns the current size of the log.
func (l *ConfigLog) Size() uint64 {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return uint64(len(l.entries))
}

// ComputeEntryHash calculates the SHA-256 hash of the entry fields (excluding signature).
// To be deterministic, we should serialize the fields or a subset of them.
// We used protobuf.
// Strategy: Create a copy, clear Signature, Marshal, Hash.
func ComputeEntryHash(entry *vdcspb.ConfigEntry) ([]byte, error) {
	// Shallow copy to avoid mutating the original
	c := *entry
	// Clear fields not part of the hash
	c.Signature = nil
	// Valid question: Is EntryHash part of the hash?
	// User spec: "EntryHash [32]byte ... Signature covers *all fields*".
	// "EntryHash" is the hash of the entry. It can't contain itself.
	// So EntryHash field in the struct is likely just a container for the result.
	// We MUST clear EntryHash before hashing.
	c.EntryHash = nil

	// Serialize
	data, err := proto.Marshal(&c)
	if err != nil {
		return nil, err
	}
	h := crypto.Hash(data)
	return h[:], nil
}
