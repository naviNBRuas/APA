#!/usr/bin/env bash
set -euo pipefail

# lowercase image name
IMAGE_NAME=apa:local
POD_NAME=apa-pod

# build image
podman build -t $IMAGE_NAME -f Containerfile .

# create pod if it doesn't exist
if ! podman pod exists $POD_NAME >/dev/null 2>&1; then
  podman pod create --name $POD_NAME -p 8080:8080 -p 9090:9090
fi

# remove old container if exists
if podman ps -a --format '{{.Names}}' | grep -q '^apa-agent$'; then
  podman rm -f apa-agent || true
fi

# run new container
podman run -d --pod $POD_NAME --name apa-agent \
  -v "$(pwd)/modules:/app/modules:Z" \
  -v "$(pwd)/configs:/app/configs:Z" \
  --restart=on-failure:3 \
  $IMAGE_NAME

echo "Pod ${POD_NAME} running, container apa-agent started"

