#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

WORKFLOW_DIR=".github/workflows"

info() { printf "\033[1;34m[info]\033[0m %s\n" "$*"; }
warn() { printf "\033[1;33m[warn]\033[0m %s\n" "$*"; }
err()  { printf "\033[1;31m[err ]\033[0m %s\n" "$*"; }

have() { command -v "$1" >/dev/null 2>&1; }

install_actionlint_if_possible() {
  if have actionlint; then
    return 0
  fi

  if ! have go; then
    warn "actionlint not found and Go is unavailable; skipping workflow linting"
    return 1
  fi

  info "Installing actionlint with go install"
  GOBIN="${GOBIN:-$ROOT_DIR/.tools/bin}"
  mkdir -p "$GOBIN"
  GOBIN="$GOBIN" go install github.com/rhysd/actionlint/cmd/actionlint@latest
  export PATH="$GOBIN:$PATH"
}

run_actionlint() {
  install_actionlint_if_possible || return 0
  info "Running actionlint"

  mapfile -t workflow_files < <(find "$WORKFLOW_DIR" -maxdepth 1 -type f \( -name '*.yml' -o -name '*.yaml' \) | sort)
  if [[ ${#workflow_files[@]} -eq 0 ]]; then
    warn "No workflow files found under $WORKFLOW_DIR"
    return 0
  fi

  actionlint -color "${workflow_files[@]}"
}

runtime_available() {
  if have docker; then
    return 0
  fi
  if have podman; then
    return 0
  fi
  return 1
}

run_act_workflow() {
  local event="$1"
  local workflow="$2"
  shift 2

  info "Running act: event=$event workflow=$workflow"
  act "$event" -W "$workflow" "$@"
}

run_act_suite() {
  if ! have act; then
    warn "act is not installed; skipping workflow execution"
    return 0
  fi

  if ! runtime_available; then
    warn "Neither docker nor podman is available; cannot execute act jobs"
    return 0
  fi

  local release_event
  release_event="$(mktemp)"
  trap 'rm -f "$release_event"' EXIT

  cat >"$release_event" <<'JSON'
{
  "inputs": {
    "tag": "v0.0.0-local"
  }
}
JSON

  # CI workflows
  run_act_workflow push "$WORKFLOW_DIR/ci.yml"
  run_act_workflow push "$WORKFLOW_DIR/code-quality.yml"
  run_act_workflow push "$WORKFLOW_DIR/documentation.yml"

  # Release workflow (manual trigger simulation)
  run_act_workflow workflow_dispatch "$WORKFLOW_DIR/release.yml" \
    -e "$release_event" \
    -s GITHUB_TOKEN="local-dummy-token" || {
      warn "Release workflow simulation reported failures (often expected locally for publish steps)."
      return 1
    }
}

main() {
  info "Validating GitHub workflow files under $WORKFLOW_DIR"

  if [[ ! -d "$WORKFLOW_DIR" ]]; then
    err "Workflow directory not found: $WORKFLOW_DIR"
    exit 1
  fi

  run_actionlint
  run_act_suite

  info "Workflow validation completed"
}

main "$@"
