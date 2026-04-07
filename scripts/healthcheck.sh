#!/usr/bin/env bash
# simple healthcheck: ensure modules dir exists and at least one module present
MODULE_DIR="/app/modules"
if [ ! -d "$MODULE_DIR" ]; then
  echo "modules dir missing"
  exit 1
fi
if [ -z "$(ls -A "$MODULE_DIR" 2>/dev/null)" ]; then
  echo "no modules found"
  exit 1
fi
echo "ok"
exit 0
