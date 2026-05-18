#!/usr/bin/env bash
set -euo pipefail

# Partial-deploy agent evidence demo:
# 1. validate the two-service contract
# 2. partial deploy web only
# 3. batch logs --json for explicit root web (one service in JSON; shape check)
# 4. full logs --json for profile-active set when stack is up

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONTRACT="${PODBAY_DEMO_CONTRACT:-${ROOT}/examples/two-service}"
PODBAY="${PODBAY_BIN:-podbay}"

if ! command -v podman >/dev/null 2>&1; then
  echo "skip: podman not on PATH" >&2
  exit 0
fi

"$PODBAY" validate -f "$CONTRACT" --json
"$PODBAY" deploy -f "$CONTRACT" web --json --receipt "${ROOT}/.podbay/receipts/partial-logs-demo.json"

out="$("$PODBAY" logs -f "$CONTRACT" web --json)"
echo "$out" | jq -e '.kind == "logs" and .status == "ok" and .service == "web" and (.log_entries | length) >= 1'

full="$("$PODBAY" logs -f "$CONTRACT" --json)"
echo "$full" | jq -e '.kind == "logs" and .status == "ok" and (.log_entries | length) == 2'

echo "Podbay partial logs demo passed"
