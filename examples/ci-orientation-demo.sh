#!/usr/bin/env bash
set -euo pipefail

# CI-shaped demo: orientation arrive path — init → onboard (offline, no Podman required).

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORK="${PODBAY_DEMO_ORIENT_DIR:-$(mktemp -d "${TMPDIR:-/tmp}/podbay-orient.XXXXXX")}"
PODBAY="${PODBAY_BIN:-podbay}"
CONTRACT="$WORK/podbay.yaml"

cleanup() {
  if [[ -z "${PODBAY_DEMO_ORIENT_DIR:-}" ]]; then
    rm -rf "$WORK"
  fi
}
trap cleanup EXIT

rm -f "$CONTRACT"

"$PODBAY" init -f "$CONTRACT" | tee "$WORK/init.out" | grep -q 'podbay onboard'
test -f "$CONTRACT"

"$PODBAY" onboard -f "$CONTRACT" --json | tee "$WORK/onboard.json" | jq -e '
  .kind == "orientation"
  and .format_version == 1
  and (.project | length) > 0
  and (.active_services | length) > 0
  and (.graph | length) > 0
  and (.next_actions | length) >= 3
  and (.note | test("not automatic remediation"))
  and (.runtime == null or .runtime.available == false or .runtime.available == true)
' >/dev/null

# Idle next steps should include validate/deploy gates.
jq -e '
  (.next_actions | map(tostring) | join("\n")) as $a
  | ($a | test("validate"))
  and ($a | test("deploy"))
' "$WORK/onboard.json" >/dev/null

echo "ci-orientation-demo: ok (work=$WORK)"
