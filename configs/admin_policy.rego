package apa.authz

# Default deny - only explicitly allowed peers can access admin APIs
default allow = false

# Static admin peers (replace with real peer IDs in production)
admin_peers := {
    "QmAdminPeer1",
    "QmAdminPeer2",
    "QmAdminPeer3",
}

# Minimum reputation threshold for privileged access
admin_reputation_threshold := 90.0

# Allow liveness/readiness probes
allow {
    input.path == "/admin/health"
    input.method == "GET"
}

# Allow read-only status for monitoring
allow {
    input.path == "/admin/status"
    input.method == "GET"
}

# Allow explicitly authorized admin peers
allow {
    input.peer_id != ""
    input.peer_is_admin == true
}

# Allow peers in the static admin list
allow {
    input.peer_id != ""
    admin_peers[input.peer_id]
}

# Allow connected peers with sufficient reputation
allow {
    input.peer_id != ""
    input.peer_connected == true
    input.peer_reputation_score >= admin_reputation_threshold
}

# Allow the agent itself
allow {
    input.peer_id != ""
    input.peer_id == input.agent_peer_id
    input.agent_is_admin == true
}

# Allow based on the agent's own reputation when connected
allow {
    input.agent_reputation_score >= 95.0
    input.peer_connected == true
}