package main

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"os"
)

func main() {
	// Generate a new Ed25519 key pair
	pubKey, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		fmt.Printf("Failed to generate key pair: %v\n", err)
		os.Exit(1)
	}

	// Encode the private key as hex
	privKeyHex := hex.EncodeToString(privKey)
	pubKeyHex := hex.EncodeToString(pubKey)

	// Write the private key to a file
	err = os.WriteFile("configs/signing_private.key", []byte(privKeyHex), 0600)
	if err != nil {
		fmt.Printf("Failed to write private key to file: %v\n", err)
		os.Exit(1)
	}

	// Print the public key (to be added to the config)
	fmt.Println("Keys generated successfully!")
	fmt.Printf("Private key written to configs/signing_private.key\n")
	fmt.Printf("Public key (add this to configs/agent-config.yaml):\n%s\n", pubKeyHex)
}