package main

import (
	"encoding/json"
	"log"
	"net/http"
)

// APIData represents the data structure for our API responses
type APIData struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// DashboardData represents the data for the dashboard
type DashboardData struct {
	AgentStatus        string `json:"agent_status"`
	AgentUptime        string `json:"agent_uptime"`
	ConnectedPeers     int    `json:"connected_peers"`
	NetworkStatus      string `json:"network_status"`
	ActiveModules      int    `json:"active_modules"`
	ModuleErrors       int    `json:"module_errors"`
	ActiveControllers  int    `json:"active_controllers"`
	ControllerErrors   int    `json:"controller_errors"`
}

// ModuleData represents module information
type ModuleData struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Status  string `json:"status"`
}

// ControllerData represents controller information
type ControllerData struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Status  string `json:"status"`
}

// PolicyData represents policy information
type PolicyData struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

// PeerData represents peer information
type PeerData struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	LastSeen string `json:"last_seen"`
}

func main() {
	// Create a simple HTTP server
	http.HandleFunc("/api/dashboard", dashboardHandler)
	http.HandleFunc("/api/modules", modulesHandler)
	http.HandleFunc("/api/controllers", controllersHandler)
	http.HandleFunc("/api/policies", policiesHandler)
	http.HandleFunc("/api/peers", peersHandler)
	http.HandleFunc("/api/settings", settingsHandler)

	// Serve static files from the web/ui directory
	fs := http.FileServer(http.Dir("web/ui/"))
	http.Handle("/", fs)

	// Start the server
	log.Println("Starting web UI server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

// dashboardHandler returns dashboard data
func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	// Create sample dashboard data
	data := DashboardData{
		AgentStatus:        "Running",
		AgentUptime:        "2 days, 4 hours",
		ConnectedPeers:     12,
		NetworkStatus:      "Healthy",
		ActiveModules:      8,
		ModuleErrors:       0,
		ActiveControllers:  5,
		ControllerErrors:   0,
	}

	// Create API response
	response := APIData{
		Status:  "success",
		Message: "Dashboard data retrieved successfully",
		Data:    data,
	}

	// Send JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// modulesHandler returns module data
func modulesHandler(w http.ResponseWriter, r *http.Request) {
	// Create sample module data
	modules := []ModuleData{
		{Name: "simple-adder", Version: "v1.0.0", Status: "Active"},
		{Name: "system-info", Version: "v1.2.1", Status: "Active"},
		{Name: "data-logger", Version: "v1.0.5", Status: "Active"},
		{Name: "net-monitor", Version: "v2.0.0", Status: "Active"},
		{Name: "crypto-hasher", Version: "v1.1.0", Status: "Error"},
	}

	// Create API response
	response := APIData{
		Status:  "success",
		Message: "Modules data retrieved successfully",
		Data:    modules,
	}

	// Send JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// controllersHandler returns controller data
func controllersHandler(w http.ResponseWriter, r *http.Request) {
	// Create sample controller data
	controllers := []ControllerData{
		{Name: "task-orchestrator", Version: "v1.0.0", Status: "Active"},
		{Name: "health-controller", Version: "v1.2.1", Status: "Active"},
		{Name: "recovery-controller", Version: "v1.0.5", Status: "Active"},
		{Name: "example-controller", Version: "v1.0.0", Status: "Active"},
		{Name: "p2p-router", Version: "v1.0.0", Status: "Active"},
	}

	// Create API response
	response := APIData{
		Status:  "success",
		Message: "Controllers data retrieved successfully",
		Data:    controllers,
	}

	// Send JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// policiesHandler returns policy data
func policiesHandler(w http.ResponseWriter, r *http.Request) {
	// Create sample policy data
	policies := []PolicyData{
		{Name: "module-policy", Type: "OPA/Rego", Status: "Active"},
		{Name: "controller-policy", Type: "OPA/Rego", Status: "Active"},
		{Name: "network-policy", Type: "OPA/Rego", Status: "Active"},
		{Name: "security-policy", Type: "OPA/Rego", Status: "Active"},
	}

	// Create API response
	response := APIData{
		Status:  "success",
		Message: "Policies data retrieved successfully",
		Data:    policies,
	}

	// Send JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// peersHandler returns peer data
func peersHandler(w http.ResponseWriter, r *http.Request) {
	// Create sample peer data
	peers := []PeerData{
		{ID: "QmPeer1", Status: "Connected", LastSeen: "2 minutes ago"},
		{ID: "QmPeer2", Status: "Connected", LastSeen: "5 minutes ago"},
		{ID: "QmPeer3", Status: "Connected", LastSeen: "1 minute ago"},
		{ID: "QmPeer4", Status: "Disconnected", LastSeen: "1 hour ago"},
		{ID: "QmPeer5", Status: "Connected", LastSeen: "30 seconds ago"},
	}

	// Create API response
	response := APIData{
		Status:  "success",
		Message: "Peers data retrieved successfully",
		Data:    peers,
	}

	// Send JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// settingsHandler handles settings requests
func settingsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Return current settings
		settings := map[string]interface{}{
			"agent_name":          "APA-Node-001",
			"log_level":           "info",
			"heartbeat_interval":  30,
			"controller_dir":      "/controllers",
			"module_dir":          "/modules",
			"policy_file":         "/configs/policy.yaml",
			"bootstrap_peers":     []string{"/ip4/127.0.0.1/tcp/9090/p2p/QmPeer1"},
			"listen_addresses":    []string{"/ip4/0.0.0.0/tcp/9090"},
			"service_tag":         "apa-agent",
			"admin_api_port":      8080,
			"admin_api_host":      "0.0.0.0",
			"audit_log_enabled":   true,
			"audit_log_file":      "/logs/audit.log",
			"max_module_size":     "100MB",
			"max_controller_size": "50MB",
		}

		// Create API response
		response := APIData{
			Status:  "success",
			Message: "Settings retrieved successfully",
			Data:    settings,
		}

		// Send JSON response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

	case http.MethodPost:
		// Update settings
		var settings map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// In a real implementation, we would validate and save the settings
		// For now, we'll just return a success response
		response := APIData{
			Status:  "success",
			Message: "Settings updated successfully",
		}

		// Send JSON response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}