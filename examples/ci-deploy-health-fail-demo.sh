#!/usr/bin/env bash
set -euo pipefail

# CI-shaped demo: deploy fails at a health gate with structured deploy --json issues.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONTRACT="${PODBAY_DEMO_CONTRACT:-${ROOT}/examples/unhealthy-health}"
PODBAY="${PODBAY_BIN:-podbay}"

if ! command -v podman >/dev/null 2>&1; then
  echo "skip: podman not on PATH" >&2
  exit 0
fi

set +e
out="$("$PODBAY" deploy -f "$CONTRACT" --json --health-timeout 15s 2>/dev/null)"
code=$?
set -e

if [[ "$code" -eq 0 ]]; then
  echo "expected deploy to fail at health gate; got success" >&2
  exit 1
fi

echo "$out" | jq -e '.kind == "deploy" and .status == "failed" and (.issues[] | select(.code == "deploy_health_probe_failed" or .code == "deploy_health_timeout"))'

echo "Podbay deploy health-failure demo passed"
