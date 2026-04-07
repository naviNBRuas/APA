#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IMAGE_TAG=${IMAGE_TAG:-apa:smoke}
CONTAINER_NAME=${CONTAINER_NAME:-apa-smoke}
HOST_PORT=${HOST_PORT:-18080}
STATE_DIR="${STATE_DIR:-"$(mktemp -d "${TMPDIR:-/tmp}/apa-state-XXXXXX")"}"

RUNTIME=${CONTAINER_RUNTIME:-}
if [[ -z "${RUNTIME}" ]]; then
  if command -v podman >/dev/null 2>&1; then
    RUNTIME="podman"
  elif command -v docker >/dev/null 2>&1; then
    RUNTIME="docker"
  else
    echo "[ERROR] Neither podman nor docker is available. Install one or set CONTAINER_RUNTIME." >&2
    exit 1
  fi
fi

echo "[INFO] Using container runtime: ${RUNTIME}"

# Ensure the state directory is writable by the container user
chmod 0777 "${STATE_DIR}"

echo "[INFO] Building image ${IMAGE_TAG} from Containerfile"
"${RUNTIME}" build -t "${IMAGE_TAG}" -f "${ROOT_DIR}/Containerfile" "${ROOT_DIR}"

cleanup() {
  echo "[INFO] Cleaning up"
  "${RUNTIME}" rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
  rm -rf "${STATE_DIR}"
}
trap cleanup EXIT

# Run the agent container
"${RUNTIME}" run -d --rm \
  --name "${CONTAINER_NAME}" \
  -p "${HOST_PORT}:8080" \
  -v "${STATE_DIR}:/app/state" \
  "${IMAGE_TAG}"

echo "[INFO] Waiting for admin health endpoint on :${HOST_PORT}"
for _ in {1..30}; do
  if curl -fsS "http://localhost:${HOST_PORT}/admin/health" >/dev/null; then
    HEALTHY=1
    break
  fi
  sleep 1
done

if [[ -z "${HEALTHY:-}" ]]; then
  echo "[ERROR] Health check failed after 30s" >&2
  exit 1
fi

echo "[INFO] Health check passed"

echo "[INFO] Fetching status snapshot"
curl -fsS "http://localhost:${HOST_PORT}/admin/status" | sed 's/.*/[STATUS] &/'

echo "[INFO] Containerized smoke test completed successfully"
