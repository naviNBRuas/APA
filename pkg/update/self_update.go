package update

import (
	"log"
	"os"

	"github.com/inconshreveable/go-update"
)

const newBinaryName = "agentd.new"

// ApplyPendingUpdate checks for a new binary and applies it.
// This should be called at the very start of the main function.
func ApplyPendingUpdate() {
	// Check if a new binary exists
	_, err := os.Stat(newBinaryName)
	if os.IsNotExist(err) {
		return // No update pending
	}
	if err != nil {
		log.Printf("[ERROR] Failed to stat new binary: %v", err)
		return
	}

	log.Println("[INFO] New binary found, applying update...")

	// Open the new binary file for reading.
	file, err := os.Open(newBinaryName)
	if err != nil {
		log.Printf("[ERROR] Failed to open new binary: %v", err)
		return
	}
	defer func() { _ = file.Close() }()

	// Use a library to handle the cross-platform complexities of replacing
	// the currently running executable.
	err = update.Apply(file, update.Options{})
	if err != nil {
		log.Printf("[ERROR] Failed to apply update: %v", err)
		_ = os.Remove(newBinaryName)
		// Restore from rollback backup if available
		if _, statErr := os.Stat("agentd.rollback"); statErr == nil {
			log.Println("[INFO] Restoring from rollback backup")
			rb, readErr := os.ReadFile("agentd.rollback")
			if readErr == nil {
				if writeErr := os.WriteFile(newBinaryName, rb, 0755); writeErr == nil {
					log.Println("[INFO] Rollback binary prepared. Restart to apply.")
				}
			}
		}
	}
}
