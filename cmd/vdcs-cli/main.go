package main

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/rrb115/vdcs/internal/crypto"
	verlog "github.com/rrb115/vdcs/internal/log"
	"github.com/rrb115/vdcs/internal/merkle"
	vdcspb "github.com/rrb115/vdcs/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: vdcs-cli <command> [args]")
		fmt.Println("Commands: set, get, audit")
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "set":
		runSet(args)
	case "get":
		runGet(args)
	case "audit":
		runAudit(args)
	default:
		log.Fatalf("unknown command: %s", cmd)
	}
}

func connect() vdcspb.VDCSClient {
	conn, err := grpc.NewClient("localhost:9090", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	return vdcspb.NewVDCSClient(conn)
}

func runSet(args []string) {
	setCmd := flag.NewFlagSet("set", flag.ExitOnError)
	key := setCmd.String("key", "", "Key to set")
	value := setCmd.String("value", "", "Value string")
	authorID := setCmd.String("author", "admin", "Author ID")
	privKeyHex := setCmd.String("priv-key", "", "Private key (hex)")

	if err := setCmd.Parse(args); err != nil {
		log.Fatal(err)
	}

	if *key == "" || *privKeyHex == "" {
		log.Fatal("missing required flags: -key, -priv-key")
	}

	pkBytes, err := hex.DecodeString(*privKeyHex)
	if err != nil {
		log.Fatal(err)
	}
	if len(pkBytes) != ed25519.PrivateKeySize {
		log.Fatalf("invalid private key size: %d", len(pkBytes))
	}
	privKey := ed25519.PrivateKey(pkBytes)

	// 1. Prepare Entry
	// Wait, we need the next index?
	// The current simple Node design doesn't assign index for us, we must provide it.
	// We need to query HEAD first to get index.
	client := connect()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	state, err := client.GetLatestRoot(ctx, &vdcspb.Empty{})
	if err != nil {
		log.Fatalf("failed to get state: %v", err)
	}

	// Index = Version?
	// Version is "Last Applied Index". So if version=0 (genesis), next is 1?
	// Or logic depending on 0-based.
	// If Log size is N, next index is N.
	// Version is usually N.
	index := state.Version

	fmt.Printf("Proposing at Index %d...\n", index)

	valHash := crypto.Hash([]byte(*value))

	entry := &vdcspb.ConfigEntry{
		Index:     index,
		Timestamp: time.Now().UnixNano(),
		AuthorId:  *authorID,
		Key:       *key,
		ValueHash: valHash[:],
		Operation: vdcspb.Operation_OPERATION_SET,
		PrevHash:  state.LastEntryHash,
		Value:     []byte(*value),
	}

	entryHash, err := verlog.ComputeEntryHash(entry)
	if err != nil {
		log.Fatal(err)
	}
	entry.EntryHash = entryHash
	entry.Signature = crypto.Sign(privKey, entryHash)

	_, err = client.ProposeEntry(ctx, entry)
	if err != nil {
		log.Fatalf("Propose failed: %v", err)
	}
	fmt.Printf("Successfully proposed entry %d\n", index)
}

func runGet(args []string) {
	getCmd := flag.NewFlagSet("get", flag.ExitOnError)
	key := getCmd.String("key", "", "Key to get")

	if err := getCmd.Parse(args); err != nil {
		log.Fatal(err)
	}

	client := connect()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. Get Trusted Root
	// In a real system, we get this from a trusted source or via gossip.
	// Here we query the node (trust-on-first-use or assume node is honest about root for current view)
	// But effectively verification checks consistency between Root and Proof.
	state, err := client.GetLatestRoot(ctx, &vdcspb.Empty{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Trusted Root (Version %d): %x\n", state.Version, state.StateRoot)

	// 2. Get Proof
	resp, err := client.GetProof(ctx, &vdcspb.GetProofRequest{Key: *key})
	if err != nil {
		log.Fatal(err)
	}

	proof := &merkle.Proof{
		Key:       resp.Key,
		ValueHash: resp.ValueHash,
		Siblings:  resp.Siblings,
		IsLeft:    resp.IsLeft,
	}

	if proof.Verify(state.StateRoot) {
		fmt.Printf("Verified Value Hash: %x\n", proof.ValueHash)
	} else {
		log.Fatal("PROOF VERIFICATION FAILED!")
	}
}

func runAudit(args []string) {
	fmt.Println("Audit not implemented yet")
}
