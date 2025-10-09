package main

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"log"
)

func main() {
	pubKey, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		log.Fatalf("Failed to generate key pair: %v", err)
	}

	fmt.Printf("Public Key (hex): %s\n", hex.EncodeToString(pubKey))
}

