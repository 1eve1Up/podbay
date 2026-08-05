#!/usr/bin/env bash
set -euo pipefail

# CI-shaped demo: deploy fails at a health gate, writes an attempt receipt, and lists it.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONTRACT="${PODBAY_DEMO_CONTRACT:-${ROOT}/examples/unhealthy-health}"
RECEIPT_DIR="${PODBAY_DEMO_RECEIPT_DIR:-${ROOT}/.podbay/receipts/ci-health-fail}"
PODBAY="${PODBAY_BIN:-podbay}"

if ! command -v podman >/dev/null 2>&1; then
  echo "skip: podman not on PATH" >&2
  exit 0
fi

mkdir -p "$RECEIPT_DIR"

set +e
out="$("$PODBAY" deploy -f "$CONTRACT" --json --health-timeout 15s --receipt "${RECEIPT_DIR}/" 2>/dev/null)"
code=$?
set -e

if [[ "$code" -eq 0 ]]; then
  echo "expected deploy to fail at health gate; got success" >&2
  exit 1
fi

echo "$out" | jq -e '.kind == "deploy" and .status == "failed" and (.issues[] | select(.code == "deploy_health_probe_failed" or .code == "deploy_health_timeout"))'

RECEIPT_PATH="$(echo "$out" | jq -r '.receipt_path // empty')"
if [[ -z "$RECEIPT_PATH" || ! -f "$RECEIPT_PATH" ]]; then
  echo "expected attempt receipt_path on failed deploy --json; got: ${RECEIPT_PATH:-<empty>}" >&2
  echo "$out" >&2
  exit 1
fi

jq -e '
  .deploy_id != null and .deploy_id != ""
  and (.contract_digest | startswith("sha256:"))
  and .status == "failed"
  and .failure != null
  and (.failure.code == "deploy_health_probe_failed" or .failure.code == "deploy_health_timeout")
' "$RECEIPT_PATH" >/dev/null

"$PODBAY" receipt list "$RECEIPT_DIR" --status failed --json | jq -e '
  .kind == "receipt_list"
  and (.receipts | length) >= 1
  and ([.receipts[].status] | all(. == "failed"))
' >/dev/null

echo "Podbay deploy health-failure attempt-receipt demo passed; receipt at ${RECEIPT_PATH}"
