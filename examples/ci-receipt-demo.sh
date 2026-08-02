#!/usr/bin/env bash
set -euo pipefail

# CI-shaped Podbay demo:
# 1. validate the contract with machine-readable output
# 2. deploy and write a receipt into a project-local history directory
# 3. fail closed if runtime drift is detected

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONTRACT="${PODBAY_DEMO_CONTRACT:-${ROOT}/examples/nginx}"
# Directory mode: deploy writes <dir>/<UTC>-<deploy_id>.json
RECEIPT_DIR="${PODBAY_DEMO_RECEIPT_DIR:-${ROOT}/.podbay/receipts/ci-demo}"
PODBAY="${PODBAY_BIN:-podbay}"

mkdir -p "$RECEIPT_DIR"

"$PODBAY" validate -f "$CONTRACT" --json
DEPLOY_JSON="$("$PODBAY" deploy -f "$CONTRACT" --receipt "${RECEIPT_DIR}/" --json)"
echo "$DEPLOY_JSON" | jq -e '.status == "ok"' >/dev/null
RECEIPT_PATH="$(echo "$DEPLOY_JSON" | jq -r '.receipt_path')"
test -n "$RECEIPT_PATH" && test -f "$RECEIPT_PATH"
echo "$DEPLOY_JSON" | jq -e --arg p "$RECEIPT_PATH" '
  .receipt_path == $p
  and (.receipt_path | test("ci-demo"))
' >/dev/null

# Evidence fields on the written receipt
jq -e '
  .deploy_id != null and .deploy_id != ""
  and (.contract_digest | startswith("sha256:"))
  and .status == "ok"
' "$RECEIPT_PATH" >/dev/null

"$PODBAY" receipt list "$RECEIPT_DIR" --json | jq -e '.kind == "receipt_list" and (.receipts | length) >= 1' >/dev/null
"$PODBAY" diff -f "$CONTRACT" --json | jq -e '.drift == false'

echo "Podbay CI demo passed; receipt written to ${RECEIPT_PATH}"
