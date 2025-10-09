package module

// Manifest defines the metadata and security properties of a WASM module.
type Manifest struct {
	Name         string   `json:"name"`
	Version      string   `json:v"ersion"`
	Arch         string   `json:"arch"`
	OS           string   `json:"os"`
	WasmFile     string   `json:"wasm_file,omitempty"` // Path to the wasm file, relative to the manifest
	WasmURL      string   `json:"wasm_url,omitempty"`  // URL to the wasm file
	Hash         string   `json:"hash"`               // SHA-256 hash of the wasm file
	Signatures   []string `json:"signatures"`
	Entry        string   `json:"entry"` // The exported function to run
	Capabilities []string `json:"capabilities"`
	Policy       string   `json:"policy"` // Path to a Rego policy file
}
