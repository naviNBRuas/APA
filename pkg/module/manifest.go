package module

// Manifest defines the metadata and security properties of a WASM module.
type Manifest struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Arch         string   `json:"arch"`
	OS           string   `json:"os"`
	WasmFile     string   `json:"wasm_file"` // Path to the wasm file, relative to the manifest
	Hash         string   `json:"hash"`      // SHA-256 hash of the wasm file
	Signatures   []string `json:"signatures"`
	Entry        string   `json:"entry"` // The exported function to run
	Capabilities []string `json:"capabilities"`
	Policy       string   `json:"policy"` // Path to a Rego policy file
}
