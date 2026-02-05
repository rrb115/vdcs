package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "github.com/mattn/go-sqlite3"
	vdcspb "github.com/rrb115/vdcs/proto"
	"google.golang.org/protobuf/proto"
)

// SQLiteStore implements the Store interface using SQLite.
type SQLiteStore struct {
	mu   sync.Mutex
	db   *sql.DB
	path string
}

// NewSQLiteStore opens or creates a SQLite database at the given path.
func NewSQLiteStore(path string) (*SQLiteStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite db: %w", err)
	}

	// Create table
	query := `
	CREATE TABLE IF NOT EXISTS entries (
		idx INTEGER PRIMARY KEY,
		data BLOB
	);`
	if _, err := db.Exec(query); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	return &SQLiteStore{
		db:   db,
		path: path,
	}, nil
}

// Append writes an entry to the database.
func (s *SQLiteStore) Append(entry *vdcspb.ConfigEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := proto.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	// We use the entry's Index as the primary key.
	// This ensures we don't have gaps or duplicates if we enforce it.
	_, err = s.db.Exec("INSERT INTO entries (idx, data) VALUES (?, ?)", entry.Index, data)
	if err != nil {
		return fmt.Errorf("failed to insert entry: %w", err)
	}

	return nil
}

// LoadAll reads all entries from the database in order.
func (s *SQLiteStore) LoadAll() ([]*vdcspb.ConfigEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.Query("SELECT data FROM entries ORDER BY idx ASC")
	if err != nil {
		return nil, fmt.Errorf("failed to query entries: %w", err)
	}
	defer rows.Close()

	var entries []*vdcspb.ConfigEntry
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		entry := &vdcspb.ConfigEntry{}
		if err := proto.Unmarshal(data, entry); err != nil {
			return nil, fmt.Errorf("failed to unmarshal entry: %w", err)
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}

// Close closes the database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
