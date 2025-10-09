package module

// Module represents a single WASM module managed by the agent.
type Module interface {
	// Name returns the name of the module.
	Name() string
	// Start executes the module's main function or begins its operation.
	Start() error
	// Stop gracefully terminates the module's execution.
	Stop() error
}
