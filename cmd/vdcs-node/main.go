package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/rrb115/vdcs/internal/node"
	"github.com/rrb115/vdcs/internal/server"
)

func main() {
	var (
		port        = flag.Int("port", 9090, "gRPC server port")
		dataDir     = flag.String("data", "./data", "Data directory")
		trustedKeys = flag.String("trusted-keys", "", "Comma-separated list of trusted public keys (hex)")
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
			// Use simple ID like "admin" if not provided?
			// For now, let's assume one key or map ID=Key logic needs clearer flag format.
			// Let's assume input format "ID:Key" or just use key as ID if simplified.
			// Re-reading spec: "AuthorID string".
			// Let's use "admin" for the first key for simplicity in v1.
			id := fmt.Sprintf("admin-%d", i)
			if i == 0 {
				id = "admin"
			}
			keys[id] = keyBytes
		}
	}

	// 2. Ensure Data Dir
	logPath := filepath.Join(*dataDir, "log.bin")

	// 3. Init Node
	cfg := node.Config{
		StoragePath: logPath,
		TrustedKeys: keys,
	}
	n, err := node.NewNode(cfg)
	if err != nil {
		log.Fatalf("failed to init node: %v", err)
	}
	defer n.Close()

	// 4. Start Server
	srv := server.NewServer(n)
	log.Printf("Starting VDCS node on port %d...", *port)
	if err := srv.Start(*port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
