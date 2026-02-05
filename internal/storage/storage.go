package storage

import (
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"sync"

	vdcspb "github.com/rrb115/vdcs/proto"
	"google.golang.org/protobuf/proto"
)

// Store defines the interface for persisting log entries.
type Store interface {
	Append(entry *vdcspb.ConfigEntry) error
	// LoadAll returns all entries in order.
	LoadAll() ([]*vdcspb.ConfigEntry, error)
	Close() error
}

// FileStore implements a simple append-only file storage.
type FileStore struct {
	mu   sync.Mutex
	file *os.File
	path string
}

// NewFileStore opens or creates a file at the given path.
func NewFileStore(path string) (*FileStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}

	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	return &FileStore{
		file: f,
		path: path,
	}, nil
}

// Append writes an entry to the file.
// Format: [Length (8 bytes)][Protobuf Data]
func (fs *FileStore) Append(entry *vdcspb.ConfigEntry) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	data, err := proto.Marshal(entry)
	if err != nil {
		return err
	}

	// Write length prefix
	lenBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(lenBuf, uint64(len(data)))

	if _, err := fs.file.Write(lenBuf); err != nil {
		return err
	}
	if _, err := fs.file.Write(data); err != nil {
		return err
	}

	// Ensure durability
	return fs.file.Sync()
}

// LoadAll reads all entries from the file.
func (fs *FileStore) LoadAll() ([]*vdcspb.ConfigEntry, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Seek to beginning
	if _, err := fs.file.Seek(0, 0); err != nil {
		return nil, err
	}

	var entries []*vdcspb.ConfigEntry
	lenBuf := make([]byte, 8)

	for {
		// Read Length
		_, err := io.ReadFull(fs.file, lenBuf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		length := binary.BigEndian.Uint64(lenBuf)
		data := make([]byte, length)

		// Read Data
		_, err = io.ReadFull(fs.file, data)
		if err != nil {
			return nil, err
		}

		entry := &vdcspb.ConfigEntry{}
		if err := proto.Unmarshal(data, entry); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

func (fs *FileStore) Close() error {
	return fs.file.Close()
}
