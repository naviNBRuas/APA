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
	defer file.Close()

	// Use a library to handle the cross-platform complexities of replacing
	// the currently running executable.
	err = update.Apply(file, update.Options{})
	if err != nil {
		log.Printf("[ERROR] Failed to apply update: %v", err)
		// If the update failed, we might want to try to remove the new binary
		// to avoid getting stuck in a loop.
		if removeErr := os.Remove(newBinaryName); removeErr != nil {
			log.Printf("[ERROR] Failed to remove new binary after failed apply: %v", removeErr)
		}
	}
	// If Apply succeeds, it will have already restarted the process.
	// If it fails, we log the error and continue with the old binary.
}
