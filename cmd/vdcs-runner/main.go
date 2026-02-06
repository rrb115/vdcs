package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

func main() {
	// 1. Build Binaries
	binDir := "bin"
	if err := os.MkdirAll(binDir, 0755); err != nil {
		log.Fatalf("Failed to create bin dir: %v", err)
	}

	targets := []string{"vdcs-node", "vdcs-cli", "key-gen"}
	for _, target := range targets {
		fmt.Printf("Building %s...\n", target)
		outputPath := filepath.Join(binDir, target)
		if runtime.GOOS == "windows" {
			outputPath += ".exe"
		}
		cmd := exec.Command("go", "build", "-o", outputPath, "./cmd/"+target)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Fatalf("Failed to build %s: %v", target, err)
		}
	}

	// 2. Manage Keys
	keyFile := ".keys"
	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		fmt.Println("Generating new keys...")
		keyGenPath := filepath.Join(binDir, "key-gen")
		if runtime.GOOS == "windows" {
			keyGenPath += ".exe"
		}
		// Use absolute path for execution to avoid path issues
		absKeyGenPath, _ := filepath.Abs(keyGenPath)
		cmd := exec.Command(absKeyGenPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Fatalf("Failed to generate keys: %v", err)
		}
		if err := os.WriteFile(keyFile, output, 0600); err != nil {
			log.Fatalf("Failed to save keys: %v", err)
		}
	}

	// 3. Read Keys
	content, err := os.ReadFile(keyFile)
	if err != nil {
		log.Fatalf("Failed to read keys: %v", err)
	}
	lines := strings.Split(string(content), "\n")
	var pubKey, privKey string
	for _, line := range lines {
		if strings.Contains(line, "Public Key (Hex):") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				pubKey = parts[3]
			}
		}
		if strings.Contains(line, "Private Key (Hex):") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				privKey = parts[3]
			}
		}
	}

	if pubKey == "" {
		log.Fatal("Could not find Public Key in .keys file. Please delete .keys and try again.")
	}

	fmt.Printf("\n--- Configuration ---\n")
	fmt.Printf("Public Key: %s\n", pubKey)
	fmt.Printf("Private Key: %s\n", privKey)
	fmt.Printf("---------------------\n\n")

	// 4. Start Node
	nodeBin := "vdcs-node"
	if runtime.GOOS == "windows" {
		nodeBin += ".exe"
	}
	nodePath := filepath.Join(binDir, nodeBin)
	absNodePath, _ := filepath.Abs(nodePath)

	fmt.Println("Starting VDCS Node...")
	// Pass stdout/stderr to see logs
	nodeCmd := exec.Command(absNodePath, "-trusted-keys", pubKey)
	nodeCmd.Stdout = os.Stdout
	nodeCmd.Stderr = os.Stderr

	if err := nodeCmd.Start(); err != nil {
		log.Fatalf("Failed to start node: %v", err)
	}

	// Handle shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\nStopping VDCS Node...")
		// On Windows, Kill is often necessary as Interrupt isn't always propagated gracefully to child
		// But in Go exec, Process.Signal(os.Interrupt) works if supported.
		if err := nodeCmd.Process.Signal(os.Interrupt); err != nil {
			// Force kill if interrupt fails
			nodeCmd.Process.Kill()
		}
	}()

	if err := nodeCmd.Wait(); err != nil {
		// If the process was killed by a signal, it might return an error, which is expected.
		if exitErr, ok := err.(*exec.ExitError); ok {
			// This verifies if the error is indeed an exit error
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				// Check for signal termination (e.g., SIGINT/SIGTERM)
				if status.Signaled() {
					return // Exited via signal, normal shutdown
				}
			}
		}
		// If we're here, it might be a real crash or just non-zero exit code
		// Just log it if it wasn't our intentional kill
		// (Simple logic: if we are shutting down, we ignore. If proper crash, we see logs above)
	}
}
