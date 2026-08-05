#!/usr/bin/env bash
set -euo pipefail

# CI-shaped demo: receipt store intelligence — last-ok, vs-last-ok, handoff.
# Uses synthetic receipts (no Podman required) so agents can prove the fail→list→compare→handoff loop offline.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STORE="${PODBAY_DEMO_RECEIPT_DIR:-${ROOT}/.podbay/receipts/ci-intelligence}"
PODBAY="${PODBAY_BIN:-podbay}"

mkdir -p "$STORE"
rm -f "$STORE"/*.json

OK_PATH="$STORE/ok.json"
FAIL_PATH="$STORE/fail.json"

cat >"$OK_PATH" <<'EOF'
{
  "format_version": 1,
  "generated_at": "2026-08-01T12:00:00Z",
  "contract_path": "/demo/podbay.yaml",
  "project": "demo",
  "deploy_id": "ok-deploy-id",
  "contract_digest": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  "status": "ok",
  "services": [
    {"service": "web", "container_name": "demo-web", "image": "web:1"}
  ]
}
EOF

cat >"$FAIL_PATH" <<'EOF'
{
  "format_version": 1,
  "generated_at": "2026-08-02T12:00:00Z",
  "contract_path": "/demo/podbay.yaml",
  "project": "demo",
  "deploy_id": "fail-deploy-id",
  "contract_digest": "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
  "status": "failed",
  "failure": {
    "service": "web",
    "code": "deploy_health_timeout",
    "class": "timeout",
    "message": "health gate timed out"
  },
  "services": [
    {"service": "web", "container_name": "demo-web", "image": "web:2"}
  ]
}
EOF

"$PODBAY" receipt list "$STORE" --status failed --json | jq -e '
  .kind == "receipt_list"
  and (.receipts | length) == 1
  and .receipts[0].status == "failed"
' >/dev/null

LAST_OK="$("$PODBAY" receipt last-ok "$STORE")"
test -f "$LAST_OK"
test "$(basename "$LAST_OK")" = "ok.json"

"$PODBAY" receipt last-ok "$STORE" --json | jq -e '
  .kind == "receipt_last_ok"
  and .status == "ok"
  and (.receipt_path | endswith("ok.json"))
' >/dev/null

# vs-last-ok must match explicit two-path diff (drift true on digest/image).
set +e
vs_out="$("$PODBAY" diff --vs-last-ok "$STORE" "$FAIL_PATH" --json 2>/dev/null)"
vs_code=$?
two_out="$("$PODBAY" diff "$LAST_OK" "$FAIL_PATH" --json 2>/dev/null)"
two_code=$?
set -e

test "$vs_code" -eq 1
test "$two_code" -eq 1
echo "$vs_out" | jq -e '.kind == "diff" and .status == "failed" and .drift == true and .receipt_pair != null' >/dev/null
echo "$two_out" | jq -e '.kind == "diff" and .status == "failed" and .drift == true and .receipt_pair != null' >/dev/null

"$PODBAY" receipt handoff "$FAIL_PATH" --store "$STORE" --json | jq -e '
  .kind == "receipt_handoff"
  and .status == "ok"
  and .handoff.deploy_id == "fail-deploy-id"
  and .handoff.failure.code == "deploy_health_timeout"
  and (.handoff.last_ok_path | endswith("ok.json"))
  and .handoff.drift == true
  and (.handoff.next_actions | length) >= 3
  and (.handoff.next_actions[0] | contains("logs"))
  and (.handoff.note | test("not automatic remediation"; "i"))
' >/dev/null

# no-prior-ok: empty store must not invent drift
EMPTY="$STORE/empty"
mkdir -p "$EMPTY"
set +e
empty_out="$("$PODBAY" diff --vs-last-ok "$EMPTY" "$FAIL_PATH" --json 2>/dev/null)"
empty_code=$?
set -e
test "$empty_code" -eq 1
echo "$empty_out" | jq -e '
  .kind == "diff"
  and .status == "failed"
  and (has("drift") | not)
  and ([.issues[].code] | index("receipt_no_last_ok") != null)
' >/dev/null

echo "Podbay receipt intelligence demo passed (list → last-ok → vs-last-ok → handoff)"
