#!/usr/bin/env bash
set -euo pipefail

# CI-shaped demo: brownfield arrive — Compose tree → init --from-codebase → onboard → validate.
# Offline-friendly: no Podman required for this path.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORK="${PODBAY_DEMO_FROM_CODEBASE_DIR:-$(mktemp -d "${TMPDIR:-/tmp}/podbay-from-codebase.XXXXXX")}"
PODBAY="${PODBAY_BIN:-podbay}"
CONTRACT="$WORK/podbay.yaml"
COMPOSE_SRC="$ROOT/examples/compose-include"

cleanup() {
  if [[ -z "${PODBAY_DEMO_FROM_CODEBASE_DIR:-}" ]]; then
    rm -rf "$WORK"
  fi
}
trap cleanup EXIT

rm -rf "$WORK"
mkdir -p "$WORK"
cp "$COMPOSE_SRC/docker-compose.yml" "$COMPOSE_SRC/fragment.yml" "$WORK/"

"$PODBAY" init --from-codebase "$WORK" -f "$CONTRACT" --json | tee "$WORK/init.json" | jq -e '
  .kind == "init"
  and .format_version == 1
  and .status == "ok"
  and (.contract_path | length) > 0
  and (.compose_source | test("docker-compose\\.yml$"))
  and (.service_count | type == "number")
  and (.next_actions | map(tostring) | join("\n") | test("onboard"))
  and (.next_actions | map(tostring) | join("\n") | test("validate"))
' >/dev/null

test -f "$CONTRACT"

"$PODBAY" onboard -f "$CONTRACT" --json | tee "$WORK/onboard.json" | jq -e '
  .kind == "orientation"
  and .format_version == 1
  and (.active_services | length) > 0
  and (.next_actions | length) >= 1
' >/dev/null

"$PODBAY" validate -f "$CONTRACT" --json | tee "$WORK/validate.json" | jq -e '
  .kind == "validate"
  and .status == "ok"
' >/dev/null

# Overwrite must fail closed with a stable code.
if "$PODBAY" init --from-codebase "$WORK" -f "$CONTRACT" --json >"$WORK/overwrite.json" 2>/dev/null; then
  echo "ci-from-codebase-demo: expected overwrite failure" >&2
  exit 1
fi
jq -e '
  .kind == "init"
  and .status == "failed"
  and (.issues[0].code == "init_target_exists")
' "$WORK/overwrite.json" >/dev/null

echo "ci-from-codebase-demo: ok (work=$WORK)"
