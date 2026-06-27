package selfhealing

// RestartProcessStrategy is a healing strategy that restarts failed processes
type RestartProcessStrategy struct {
	name        string
	description string
	priority    int
	config      map[string]interface{}
}

// RebuildModuleStrategy is a healing strategy that rebuilds corrupted modules
type RebuildModuleStrategy struct {
	name        string
	description string
	priority    int
	config      map[string]interface{}
}

// NetworkReconnectStrategy is a healing strategy that reconnects network connections
type NetworkReconnectStrategy struct {
	name        string
	description string
	priority    int
	config      map[string]interface{}
}

// MemoryOptimizationStrategy is a healing strategy that optimizes memory usage
type MemoryOptimizationStrategy struct {
	name        string
	description string
	priority    int
	config      map[string]interface{}
}

// QuarantineNodeStrategy is a healing strategy that quarantines compromised nodes
type QuarantineNodeStrategy struct {
	name        string
	description string
	priority    int
	config      map[string]interface{}
}
