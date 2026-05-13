#!/usr/bin/env bash
set -euo pipefail

# CI-shaped Podbay demo:
# 1. validate the contract with machine-readable output
# 2. deploy and write a receipt artifact
# 3. fail closed if runtime drift is detected

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONTRACT="${PODBAY_DEMO_CONTRACT:-${ROOT}/examples/nginx}"
RECEIPT="${PODBAY_DEMO_RECEIPT:-${ROOT}/.podbay/receipts/ci-demo.json}"
PODBAY="${PODBAY_BIN:-podbay}"

mkdir -p "$(dirname "$RECEIPT")"

"$PODBAY" validate -f "$CONTRACT" --json
"$PODBAY" deploy -f "$CONTRACT" --receipt "$RECEIPT" --json
"$PODBAY" diff -f "$CONTRACT" --json | jq -e '.drift == false'

echo "Podbay CI demo passed; receipt written to ${RECEIPT}"
