package apa.authz

# Default deny - only explicitly allowed peers can access admin APIs
default allow = false

# Define admin peer IDs (these would be replaced with actual peer IDs in production)
admin_peers = {
    "QmAdminPeer1",
    "QmAdminPeer2",
    "QmAdminPeer3"
}

# Define minimum reputation threshold for admin access
admin_reputation_threshold = 90.0

# Allow access if the peer is explicitly authorized as an admin peer
allow {
    input.peer_id != ""
    input.peer_is_admin == true
}

# Allow access if the peer is in the hardcoded admin peers list
allow {
    input.peer_id != ""
    input.peer_id in admin_peers
}

# Allow access if the peer has high reputation and is connected
allow {
    input.peer_id != ""
    input.peer_connected == true
    input.peer_reputation_score >= admin_reputation_threshold
}

# Allow access if the peer is the agent itself (self-access)
allow {
    input.peer_id != ""
    input.peer_id == input.agent_peer_id
    input.agent_is_admin == true
}

# Allow specific actions for specific resources
# Example: allow health checks for all connected peers
allow {
    input.path == "/admin/health"
    input.method == "GET"
    input.peer_connected == true
}

# Allow access based on agent's own reputation
allow {
    input.agent_reputation_score >= 95.0
    input.peer_connected == true
}