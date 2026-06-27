package agent

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/libp2p/go-libp2p/core/peer"
	"golang.org/x/time/rate"
)

type apiError struct {
	Error string `json:"error"`
	Code  int    `json:"code"`
}

func writeJSONError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(apiError{Error: message, Code: code})
}

func (rt *Runtime) authorizeAdminRequest(ctx context.Context, r *http.Request, input map[string]interface{}) (bool, error) {
	if rt.adminAPIKey != "" {
		if token := parseBearerToken(r.Header.Get("Authorization")); token != "" && token == rt.adminAPIKey {
			input["user"] = "admin-api-key"
			input["token_authenticated"] = true
			input["peer_is_admin"] = true
		}
	}

	if rt.adminPolicyEngine == nil {
		rt.logger.Warn("Admin policy engine not configured; allowing request by default", "path", r.URL.Path)
		return true, nil
	}

	allowed, err := rt.adminPolicyEngine.Authorize(ctx, input)
	if err != nil {
		rt.logger.Error("Admin API authorization error", "path", r.URL.Path, "error", err)
		return false, err
	}

	if !allowed {
		rt.logger.Warn("Admin API unauthorized access", "path", r.URL.Path, "input", input)
		return false, nil
	}

	return true, nil
}

func (rt *Runtime) buildAdminTLSConfig() (*tls.Config, bool) {
	if rt.adminTLSCertPath == "" || rt.adminTLSKeyPath == "" {
		return nil, false
	}

	tlsConfig := &tls.Config{}

	if rt.adminTLSClientCA != "" {
		caBytes, err := os.ReadFile(rt.adminTLSClientCA)
		if err != nil {
			rt.logger.Error("Failed to read admin client CA", "error", err)
			return nil, false
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caBytes) {
			rt.logger.Error("Failed to append admin client CA certs")
			return nil, false
		}
		tlsConfig.ClientCAs = pool
		if rt.adminTLSRequireClientCert {
			tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		}
	} else if rt.adminTLSRequireClientCert {
		tlsConfig.ClientAuth = tls.RequireAnyClientCert
	}

	return tlsConfig, true
}

func parseBearerToken(header string) string {
	if header == "" {
		return ""
	}
	const prefix = "Bearer "
	if len(header) > len(prefix) && header[:len(prefix)] == prefix {
		return header[len(prefix):]
	}
	return ""
}

func (rt *Runtime) checkRateLimit(w http.ResponseWriter, r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	key := host
	const rps = 5.0
	const burst = 10

	rt.rateMu.Lock()
	lim, ok := rt.rateLimiters[key]
	if !ok {
		lim = rate.NewLimiter(rate.Limit(rps), burst)
		rt.rateLimiters[key] = lim
	}
	allowed := lim.Allow()
	rt.rateMu.Unlock()

	if !allowed {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return false
	}
	return true
}

func (rt *Runtime) createAuthzInput(r *http.Request) map[string]interface{} {
	input := map[string]interface{}{
		"method": r.Method,
		"path":   r.URL.Path,
		"user":   "anonymous",
	}

	if r.TLS != nil {
		input["transport"] = "https"
	}

	if rt.adminAPIKey != "" {
		if token := parseBearerToken(r.Header.Get("Authorization")); token != "" {
			if token == rt.adminAPIKey {
				input["user"] = "admin-api-key"
				input["token_authenticated"] = true
				input["peer_is_admin"] = true
			}
		}
	}

	agentPeerID := ""
	if rt.identity != nil {
		agentPeerID = rt.identity.PeerID.String()
	}
	input["agent_peer_id"] = agentPeerID

	reputation := 50.0
	if rt.p2p != nil && rt.identity != nil {
		reputation = rt.p2p.GetReputationScore(rt.identity.PeerID)
	}
	input["agent_reputation_score"] = reputation

	if rt.adminPeerManager != nil && agentPeerID != "" {
		isAdmin := rt.adminPeerManager.IsAuthorizedAdmin(agentPeerID, reputation, true)
		input["agent_is_admin"] = isAdmin
	}

	peerID := r.Header.Get("X-Peer-ID")
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	if parsedIP := net.ParseIP(host); peerID == "" && parsedIP != nil && parsedIP.IsLoopback() && agentPeerID != "" {
		peerID = agentPeerID
		input["peer_connected"] = true
		input["peer_is_admin"] = true
	}

	if peerID != "" {
		input["peer_id"] = peerID

		reputationScore := 50.0
		isConnected := false

		if rt.p2p != nil {
			if parsedPeerID, err := peer.Decode(peerID); err == nil {
				reputationScore = rt.p2p.GetReputationScore(parsedPeerID)
				input["peer_reputation_score"] = reputationScore

				if rt.p2p.IsPeerConnected(parsedPeerID) {
					isConnected = true
					input["peer_connected"] = true
				}
			}
		}

		if rt.adminPeerManager != nil {
			isAdmin := rt.adminPeerManager.IsAuthorizedAdmin(peerID, reputationScore, isConnected)
			input["peer_is_admin"] = isAdmin
		}
	}

	return input
}

func (rt *Runtime) appendAudit(action string, input map[string]interface{}) {
	if rt.auditLogger == nil {
		return
	}

	entry := AuditEntry{
		Actor:  fmt.Sprint(input["user"]),
		Action: action,
		Path:   fmt.Sprint(input["path"]),
		Method: fmt.Sprint(input["method"]),
		Details: map[string]interface{}{
			"authz_input": input,
		},
	}

	if peerID, ok := input["peer_id"].(string); ok {
		entry.PeerID = peerID
	}

	if err := rt.auditLogger.Append(entry); err != nil {
		rt.logger.Error("Failed to append audit entry", "action", action, "error", err)
	}
}
