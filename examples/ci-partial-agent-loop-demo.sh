#!/usr/bin/env bash
set -euo pipefail

# Partial-deploy agent-loop contract (unified CI demo)
#
# Fixtures:
#   happy — examples/two-service (partial root web + --dependents → web, worker)
#   fail  — examples/unhealthy-health (full deploy; health gate on sick; client depends_on sick:service_healthy)
#
# Env: PODBAY_BIN (default podbay), PODBAY_DEMO_CONTRACT (overrides fixture per mode)
#
# Happy path (same service roots on every gate):
#   validate --json → deploy --json --receipt → diff --json (drift == false)
#   → logs --json → down --json
#
# Failure path (branch on deploy --json issues[], not stderr):
#   deploy --json (expect failed + deploy_health_probe_failed|deploy_health_timeout)
#   → logs --json on failing service + --dependents → explain --json (same roots) → down --json
#
# Usage: ci-partial-agent-loop-demo.sh [happy|fail]

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PODBAY="${PODBAY_BIN:-podbay}"
RECEIPT_DIR="${ROOT}/.podbay/receipts"
MODE="${1:-happy}"

require_podman() {
  if ! command -v podman >/dev/null 2>&1; then
    echo "skip: podman not on PATH" >&2
    exit 0
  fi
}

usage() {
  echo "usage: $0 [happy|fail]" >&2
  exit 2
}

run_happy() {
  local contract="${PODBAY_DEMO_CONTRACT:-${ROOT}/examples/two-service}"
  local receipt="${RECEIPT_DIR}/partial-agent-loop-happy.json"
  mkdir -p "$RECEIPT_DIR"

  "$PODBAY" validate -f "$contract" --json | jq -e '.kind == "validate" and .status == "ok"'

  "$PODBAY" deploy -f "$contract" web --dependents --json --receipt "$receipt" | jq -e \
    '.kind == "deploy" and .status == "ok"'

  "$PODBAY" diff -f "$contract" web --dependents --json | jq -e \
    '.kind == "diff" and .drift == false'

  "$PODBAY" logs -f "$contract" web --dependents --json | jq -e \
    '.kind == "logs" and .status == "ok" and (.log_entries | length) >= 1'

  "$PODBAY" down -f "$contract" web --dependents --json | jq -e \
    '.kind == "teardown" and .status == "ok"'
}

run_fail() {
  local contract="${PODBAY_DEMO_CONTRACT:-${ROOT}/examples/unhealthy-health}"
  local deploy_out deploy_code fail_svc

  set +e
  deploy_out="$("$PODBAY" deploy -f "$contract" --json --health-timeout 15s 2>/dev/null)"
  deploy_code=$?
  set -e

  if [[ "$deploy_code" -eq 0 ]]; then
    echo "expected deploy to fail at health gate; got success" >&2
    exit 1
  fi

  echo "$deploy_out" | jq -e \
    '.kind == "deploy" and .status == "failed" and (.issues[] | select(.code == "deploy_health_probe_failed" or .code == "deploy_health_timeout"))'

  fail_svc="$(echo "$deploy_out" | jq -r '
    [.issues[] | select(.code == "deploy_health_probe_failed" or .code == "deploy_health_timeout") | .service]
    | map(select(. != null and . != "")) | first // "sick"
  ')"

  "$PODBAY" logs -f "$contract" "$fail_svc" --dependents --json | jq -e \
    '.kind == "logs" and .status == "ok"'

  "$PODBAY" explain -f "$contract" "$fail_svc" --dependents --json | jq -e \
    '.kind == "explain" and .status == "ok"'

  "$PODBAY" down -f "$contract" --json | jq -e \
    '.kind == "teardown" and .status == "ok"'
}

case "$MODE" in
  happy)
    require_podman
    run_happy
    echo "Podbay partial agent-loop demo (happy) passed"
    ;;
  fail)
    require_podman
    run_fail
    echo "Podbay partial agent-loop demo (fail) passed"
    ;;
  *)
    usage
    ;;
esac
