#!/bin/bash

set -e

# --- Configuration ---
AGENT_IMAGE="apa:two-agents-test"
AGENT1_NAME="apa-agent-1"
AGENT2_NAME="apa-agent-2"
AGENT1_PORT="8081"
AGENT2_PORT="8082"

# --- Build the APA image ---
echo "Building APA image: ${AGENT_IMAGE}"
podman build -t ${AGENT_IMAGE} -f Containerfile .

# --- Setup Agent 1 ---
echo "Setting up Agent 1"
AGENT1_DIR="$(mktemp -d -t apa-agent1-XXXXXX)"
AGENT1_CONFIG_PATH="${AGENT1_DIR}/agent-config.yaml"
AGENT1_IDENTITY_PATH="${AGENT1_DIR}/agent-identity.json"
AGENT1_POLICY_PATH="${AGENT1_DIR}/policy.yaml"

# Generate identity for Agent 1
go run cmd/agentd/main.go -config "${AGENT1_CONFIG_PATH}" # This will create agent-identity.json in the current dir
mv agent-identity.json "${AGENT1_IDENTITY_PATH}"

# Create policy.yaml for Agent 1
cat <<EOF > "${AGENT1_POLICY_PATH}"
trusted_authors:
  - "Navi"
EOF

# Create agent-config.yaml for Agent 1
cat <<EOF > "${AGENT1_CONFIG_PATH}"
admin_listen_address: "0.0.0.0:${AGENT1_PORT}"
log_level: "debug"
module_path: "/tmp/modules"
identity_file_path: "${AGENT1_IDENTITY_PATH}"
policy_path: "${AGENT1_POLICY_PATH}"
p2p:
  listen_addresses:
    - "/ip4/0.0.0.0/tcp/0"
  bootstrap_peers:
    - "/ip4/127.0.0.1/tcp/${AGENT2_PORT}/p2p/$(cat ${AGENT2_IDENTITY_PATH} | jq -r .PeerID)" # Placeholder for Agent 2's PeerID
  heartbeat_interval: "10s"
  service_tag: "apa-test"
update:
  server_url: "http://127.0.0.1:8000/release.json"
  check_interval: "1m"
  public_key: "75b351bef5d97a7658610fa29ebcb6a09585fdf195faac66e9041cbf2e3e09e1"
EOF

# --- Setup Agent 2 ---
echo "Setting up Agent 2"
AGENT2_DIR="$(mktemp -d -t apa-agent2-XXXXXX)"
AGENT2_CONFIG_PATH="${AGENT2_DIR}/agent-config.yaml"
AGENT2_IDENTITY_PATH="${AGENT2_DIR}/agent-identity.json"
AGENT2_POLICY_PATH="${AGENT2_DIR}/policy.yaml"

# Generate identity for Agent 2
go run cmd/agentd/main.go -config "${AGENT2_CONFIG_PATH}" # This will create agent-identity.json in the current dir
mv agent-identity.json "${AGENT2_IDENTITY_PATH}"

# Create policy.yaml for Agent 2
cat <<EOF > "${AGENT2_POLICY_PATH}"
trusted_authors:
  - "Navi"
EOF

# Create agent-config.yaml for Agent 2
cat <<EOF > "${AGENT2_CONFIG_PATH}"
admin_listen_address: "0.0.0.0:${AGENT2_PORT}"
log_level: "debug"
module_path: "/tmp/modules"
identity_file_path: "${AGENT2_IDENTITY_PATH}"
policy_path: "${AGENT2_POLICY_PATH}"
p2p:
  listen_addresses:
    - "/ip4/0.0.0.0/tcp/0"
  bootstrap_peers:
    - "/ip4/127.0.0.1/tcp/${AGENT1_PORT}/p2p/$(cat ${AGENT1_IDENTITY_PATH} | jq -r .PeerID)" # Placeholder for Agent 1's PeerID
  heartbeat_interval: "10s"
  service_tag: "apa-test"
update:
  server_url: "http://127.0.0.1:8000/release.json"
  check_interval: "1m"
  public_key: "75b351bef5d97a7658610fa29ebcb6a09585fdf195faac66e9041cbf2e3e09e1"
EOF

# --- Update bootstrap peers with actual PeerIDs ---
AGENT1_PEER_ID=$(jq -r .PeerID "${AGENT1_IDENTITY_PATH}")
AGENT2_PEER_ID=$(jq -r .PeerID "${AGENT2_IDENTITY_PATH}")

sed -i "s|# Placeholder for Agent 2's PeerID|/ip4/127.0.0.1/tcp/${AGENT2_PORT}/p2p/${AGENT2_PEER_ID}|" "${AGENT1_CONFIG_PATH}"
sed -i "s|# Placeholder for Agent 1's PeerID|/ip4/127.0.0.1/tcp/${AGENT1_PORT}/p2p/${AGENT1_PEER_ID}|" "${AGENT2_CONFIG_PATH}"

# --- Run Agent 1 ---
echo "Running Agent 1 in Podman"
podman run -d --rm \
  -p ${AGENT1_PORT}:${AGENT1_PORT} \
  -v "${AGENT1_DIR}":/app/config \
  --name ${AGENT1_NAME} \
  ${AGENT_IMAGE} \
  /app/agentd -config /app/config/agent-config.yaml

# --- Run Agent 2 ---
echo "Running Agent 2 in Podman"
podman run -d --rm \
  -p ${AGENT2_PORT}:${AGENT2_PORT} \
  -v "${AGENT2_DIR}":/app/config \
  --name ${AGENT2_NAME} \
  ${AGENT_IMAGE} \
  /app/agentd -config /app/config/agent-config.yaml

echo ""
echo "--- Agents Started ---"
echo "Agent 1 running on port ${AGENT1_PORT}, PeerID: ${AGENT1_PEER_ID}"
echo "Agent 2 running on port ${AGENT2_PORT}, PeerID: ${AGENT2_PEER_ID}"
echo ""
echo "To check Agent 1 status: curl http://localhost:${AGENT1_PORT}/admin/status"
echo "To check Agent 2 status: curl http://localhost:${AGENT2_PORT}/admin/status"
echo ""
echo "To view logs for Agent 1: podman logs ${AGENT1_NAME}"
echo "To view logs for Agent 2: podman logs ${AGENT2_NAME}"
echo ""
echo "To stop agents: podman stop ${AGENT1_NAME} ${AGENT2_NAME}"
echo "To clean up temporary directories: rm -rf ${AGENT1_DIR} ${AGENT2_DIR}"
