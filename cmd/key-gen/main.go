package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
)

func main() {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Private Key (Hex): %s\n", hex.EncodeToString(priv))
	fmt.Printf("Public Key (Hex):  %s\n", hex.EncodeToString(pub))
	fmt.Println("\nUse the Public Key to start the node (--trusted-keys).")
	fmt.Println("Use the Private Key to sign entries with the CLI (--priv-key).")
}
