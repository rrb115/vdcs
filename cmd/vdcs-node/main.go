package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/rrb115/vdcs/internal/node"
	"github.com/rrb115/vdcs/internal/server"
	"github.com/rrb115/vdcs/internal/storage"
)

func main() {
	var (
		port = flag.Int("port", 9090, "gRPC server port")
		// Deprecated: dataDir is legacy for file storage or base for sqlite?
		// User said: "vdcs-node --storage=postgres or vdcs-node --storage=scylladb".
		// Also: "Default: SQLite (Embedded...)"
		dataDir     = flag.String("data", "./data", "Data directory (for log.bin or vdcs.db)")
		trustedKeys = flag.String("trusted-keys", "", "Comma-separated list of trusted public keys (hex)")
		storageType = flag.String("storage", "sqlite", "Storage type: sqlite, file")
	)
	flag.Parse()

	// 1. Parse Trusted Keys
	keys := make(map[string][]byte)
	if *trustedKeys != "" {
		parts := strings.Split(*trustedKeys, ",")
		for i, part := range parts {
			part = strings.TrimSpace(part)
			keyBytes, err := hex.DecodeString(part)
			if err != nil {
				log.Fatalf("invalid key format for key %d: %v", i, err)
			}
			id := fmt.Sprintf("admin-%d", i)
			if i == 0 {
				id = "admin"
			}
			keys[id] = keyBytes
		}
	}

	// 2. Init Storage
	var store storage.Store
	var err error

	if err := os.MkdirAll(*dataDir, 0755); err != nil {
		log.Fatalf("failed to create data dir: %v", err)
	}

	switch *storageType {
	case "sqlite":
		dbPath := filepath.Join(*dataDir, "vdcs.db")
		store, err = storage.NewSQLiteStore(dbPath)
	case "file":
		logPath := filepath.Join(*dataDir, "log.bin")
		store, err = storage.NewFileStore(logPath)
	default:
		log.Fatalf("unknown storage type: %s", *storageType)
	}

	if err != nil {
		log.Fatalf("failed to init storage: %v", err)
	}

	// 3. Init Node
	cfg := node.Config{
		Store:       store,
		TrustedKeys: keys,
	}
	n, err := node.NewNode(cfg)
	if err != nil {
		store.Close() // Clean up if node init fails
		log.Fatalf("failed to init node: %v", err)
	}
	defer n.Close()
	// Note: n.Close() will close the store.

	// 4. Start Server
	srv := server.NewServer(n)
	log.Printf("Starting VDCS node on port %d...", *port)
	if err := srv.Start(*port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
