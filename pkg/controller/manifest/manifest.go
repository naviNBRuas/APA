package manifest

// Manifest defines the metadata for a controller module.
type Manifest struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Path    string `json:"path"` // Path to the controller binary/script
	Hash    string `json:"hash"` // SHA-256 hash of the controller binary/script
	// Add other metadata like author, capabilities, policy, etc.
}
