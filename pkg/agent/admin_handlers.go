package agent

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"gopkg.in/yaml.v3"
)

func (rt *Runtime) auditHandler(w http.ResponseWriter, r *http.Request) {
	if !rt.checkRateLimit(w, r) {
		return
	}
	input := rt.createAuthzInput(r)
	if allowed, err := rt.authorizeAdminRequest(r.Context(), r, input); err != nil {
		writeJSONError(w, "Authorization error", http.StatusInternalServerError)
		return
	} else if !allowed {
		writeJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	defer rt.appendAudit("audit", input)

	w.Header().Set("Content-Type", "application/json")
	if rt.auditLogger == nil {
		writeJSONError(w, "Audit logging not enabled", http.StatusNotImplemented)
		return
	}
	entries, err := rt.auditLogger.ReadRecent(100)
	if err != nil {
		rt.logger.Error("Failed to read audit log", "error", err)
		writeJSONError(w, "Failed to read audit log", http.StatusInternalServerError)
		return
	}
	if err := json.NewEncoder(w).Encode(entries); err != nil {
		rt.logger.Error("Failed to encode audit log entries", "error", err)
		writeJSONError(w, "Failed to encode audit log entries", http.StatusInternalServerError)
		return
	}
}

func (rt *Runtime) healthHandler(w http.ResponseWriter, r *http.Request) {
	if !rt.checkRateLimit(w, r) {
		return
	}
	input := rt.createAuthzInput(r)
	if allowed, err := rt.authorizeAdminRequest(r.Context(), r, input); err != nil {
		writeJSONError(w, "Authorization error", http.StatusInternalServerError)
		return
	} else if !allowed {
		writeJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	defer rt.appendAudit("health", input)

	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintln(w, "OK")
}

func (rt *Runtime) statusHandler(w http.ResponseWriter, r *http.Request) {
	if !rt.checkRateLimit(w, r) {
		return
	}
	input := rt.createAuthzInput(r)
	if allowed, err := rt.authorizeAdminRequest(r.Context(), r, input); err != nil {
		writeJSONError(w, "Authorization error", http.StatusInternalServerError)
		return
	} else if !allowed {
		writeJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	defer rt.appendAudit("status", input)

	status := StatusResponse{
		Version:       rt.updateManager.CurrentVersion(),
		PeerID:        rt.identity.PeerID.String(),
		LoadedModules: rt.moduleManager.ListModules(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		rt.logger.Error("Failed to encode status response", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (rt *Runtime) metricsHandler(w http.ResponseWriter, r *http.Request) {
	if !rt.checkRateLimit(w, r) {
		return
	}

	input := rt.createAuthzInput(r)
	if allowed, err := rt.authorizeAdminRequest(r.Context(), r, input); err != nil {
		writeJSONError(w, "Authorization error", http.StatusInternalServerError)
		return
	} else if !allowed {
		writeJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	defer rt.appendAudit("metrics", input)

	metrics := map[string]interface{}{
		"uptime_seconds":     time.Since(rt.startTime).Seconds(),
		"audit_enabled":      rt.auditLogger != nil,
		"topics_joined":      map[string]bool{},
		"peer_count":         0,
		"heartbeat_interval": rt.config.P2P.HeartbeatInterval.Seconds(),
	}

	if rt.p2p != nil {
		metrics["peer_count"] = rt.p2p.PeerCount()
		metrics["admitted_peers"] = rt.p2p.AdmittedPeers
		metrics["topics_health"] = rt.p2p.GetTopicHealth()
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metrics); err != nil {
		rt.logger.Error("Failed to encode metrics", "error", err)
		writeJSONError(w, "Failed to encode metrics", http.StatusInternalServerError)
		return
	}
}

func (rt *Runtime) modulesHandler(w http.ResponseWriter, r *http.Request) {
	if !rt.checkRateLimit(w, r) {
		return
	}
	input := rt.createAuthzInput(r)
	if allowed, err := rt.authorizeAdminRequest(r.Context(), r, input); err != nil {
		writeJSONError(w, "Authorization error", http.StatusInternalServerError)
		return
	} else if !allowed {
		writeJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	defer rt.appendAudit("modules", input)

	switch r.Method {
	case http.MethodGet:
		modules := rt.moduleManager.ListModules()
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(modules); err != nil {
			rt.logger.Error("Failed to encode modules list response", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	case http.MethodPost:
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		if req.Name == "" {
			writeJSONError(w, "Missing module name", http.StatusBadRequest)
			return
		}
		if err := rt.moduleManager.LoadModule(req.Name); err != nil {
			rt.logger.Error("Failed to load module", "name", req.Name, "error", err)
			writeJSONError(w, "Failed to load module: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, "Module %s loaded successfully.\n", req.Name)
	default:
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (rt *Runtime) controllersHandler(w http.ResponseWriter, r *http.Request) {
	if !rt.checkRateLimit(w, r) {
		return
	}
	input := rt.createAuthzInput(r)
	if allowed, err := rt.authorizeAdminRequest(r.Context(), r, input); err != nil {
		writeJSONError(w, "Authorization error", http.StatusInternalServerError)
		return
	} else if !allowed {
		writeJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	defer rt.appendAudit("controllers", input)

	switch r.Method {
	case http.MethodGet:
		controllers := rt.controllerManager.ListControllers()
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(controllers); err != nil {
			rt.logger.Error("Failed to encode controllers list response", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	case http.MethodPost:
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		if req.Name == "" {
			writeJSONError(w, "Missing controller name", http.StatusBadRequest)
			return
		}
		if err := rt.controllerManager.LoadController(req.Name); err != nil {
			rt.logger.Error("Failed to load controller", "name", req.Name, "error", err)
			writeJSONError(w, "Failed to load controller: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, "Controller %s loaded successfully.\n", req.Name)
	default:
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (rt *Runtime) configHandler(w http.ResponseWriter, r *http.Request) {
	if !rt.checkRateLimit(w, r) {
		return
	}
	input := rt.createAuthzInput(r)
	if allowed, err := rt.authorizeAdminRequest(r.Context(), r, input); err != nil {
		writeJSONError(w, "Authorization error", http.StatusInternalServerError)
		return
	} else if !allowed {
		writeJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	defer rt.appendAudit("config", input)

	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(rt.sanitizedConfig()); err != nil {
			rt.logger.Error("Failed to encode config response", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	case http.MethodPost:
		var newConfig Config
		if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
			writeJSONError(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		configData, err := yaml.Marshal(newConfig)
		if err != nil {
			rt.logger.Error("Failed to marshal config", "error", err)
			writeJSONError(w, "Failed to process config: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if err := rt.ApplyConfig(configData); err != nil {
			rt.logger.Error("Failed to apply config", "error", err)
			writeJSONError(w, "Failed to apply config: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintln(w, "Config updated successfully.")
	default:
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (rt *Runtime) updateHandler(w http.ResponseWriter, r *http.Request) {
	if !rt.checkRateLimit(w, r) {
		return
	}
	input := rt.createAuthzInput(r)
	if allowed, err := rt.authorizeAdminRequest(r.Context(), r, input); err != nil {
		writeJSONError(w, "Authorization error", http.StatusInternalServerError)
		return
	} else if !allowed {
		writeJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	defer rt.appendAudit("update", input)

	switch r.Method {
	case http.MethodPost:
		go rt.updateManager.CheckForUpdate()
		w.WriteHeader(http.StatusAccepted)
		_, _ = fmt.Fprintln(w, "Update check initiated.")
	default:
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (rt *Runtime) peerCopyHandler(w http.ResponseWriter, r *http.Request) {
	if !rt.checkRateLimit(w, r) {
		return
	}
	input := rt.createAuthzInput(r)
	if allowed, err := rt.authorizeAdminRequest(r.Context(), r, input); err != nil {
		writeJSONError(w, "Authorization error", http.StatusInternalServerError)
		return
	} else if !allowed {
		writeJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	defer rt.appendAudit("peer_copy", input)

	peerID := r.URL.Query().Get("peer_id")
	moduleName := r.URL.Query().Get("module_name")
	if peerID == "" || moduleName == "" {
		writeJSONError(w, "Missing peer_id or module_name parameter", http.StatusBadRequest)
		return
	}

	if err := rt.recoveryController.RequestPeerCopy(r.Context(), peerID, moduleName); err != nil {
		rt.logger.Error("Failed to request peer copy", "peer_id", peerID, "module_name", moduleName, "error", err)
		writeJSONError(w, "Failed to request peer copy: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, "Module %s successfully copied from peer %s.\n", moduleName, peerID)
}

func (rt *Runtime) triggerRegenerationHandler(w http.ResponseWriter, r *http.Request) {
	if !rt.checkRateLimit(w, r) {
		return
	}
	input := rt.createAuthzInput(r)
	if allowed, err := rt.authorizeAdminRequest(r.Context(), r, input); err != nil {
		return
	} else if !allowed {
		return
	}
	defer rt.appendAudit("trigger_regeneration", input)

	if err := rt.regenerator.TriggerRegeneration(r.Context()); err != nil {
		rt.logger.Error("Failed to trigger regeneration", "error", err)
		writeJSONError(w, "Failed to trigger regeneration: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintln(w, "Regeneration triggered successfully.")
}

func (rt *Runtime) triggerPropagationHandler(w http.ResponseWriter, r *http.Request) {
	if !rt.checkRateLimit(w, r) {
		return
	}
	input := rt.createAuthzInput(r)
	if allowed, err := rt.authorizeAdminRequest(r.Context(), r, input); err != nil {
		return
	} else if !allowed {
		return
	}
	defer rt.appendAudit("trigger_propagation", input)

	if err := rt.propagationManager.TriggerPropagation(r.Context()); err != nil {
		rt.logger.Error("Failed to trigger propagation", "error", err)
		writeJSONError(w, "Failed to trigger propagation: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintln(w, "Propagation triggered successfully.")
}
