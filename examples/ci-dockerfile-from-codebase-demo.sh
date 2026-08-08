#!/usr/bin/env bash
set -euo pipefail

# CI-shaped demo: brownfield arrive — Dockerfile-only tree → init --from-codebase → onboard → validate.
# Offline-friendly: no Podman required for this path.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORK="${PODBAY_DEMO_DOCKERFILE_FROM_CODEBASE_DIR:-$(mktemp -d "${TMPDIR:-/tmp}/podbay-dockerfile-from-codebase.XXXXXX")}"
PODBAY="${PODBAY_BIN:-podbay}"
CONTRACT="$WORK/podbay.yaml"

cleanup() {
  if [[ -z "${PODBAY_DEMO_DOCKERFILE_FROM_CODEBASE_DIR:-}" ]]; then
    rm -rf "$WORK"
  fi
}
trap cleanup EXIT

rm -rf "$WORK"
mkdir -p "$WORK"
cat >"$WORK/Dockerfile" <<'EOF'
FROM docker.io/library/alpine:3.20
CMD ["sleep", "infinity"]
EOF

"$PODBAY" init --from-codebase "$WORK" -f "$CONTRACT" --json | tee "$WORK/init.json" | jq -e '
  .kind == "init"
  and .format_version == 1
  and .status == "ok"
  and (.contract_path | length) > 0
  and .source_kind == "dockerfile"
  and (.dockerfile_source | test("Dockerfile$"))
  and (.compose_source | not)
  and (.service_count | type == "number")
  and (.next_actions | map(tostring) | join("\n") | test("onboard"))
  and (.next_actions | map(tostring) | join("\n") | test("validate"))
' >/dev/null

test -f "$CONTRACT"
grep -q 'dockerfile: Dockerfile' "$CONTRACT"

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

# Compose preference: when both exist, Compose wins.
BOTH="$WORK/both"
mkdir -p "$BOTH"
cp "$WORK/Dockerfile" "$BOTH/"
cat >"$BOTH/compose.yaml" <<'EOF'
services:
  web:
    image: docker.io/library/nginx:alpine
EOF
BOTH_CONTRACT="$BOTH/podbay.yaml"
"$PODBAY" init --from-codebase "$BOTH" -f "$BOTH_CONTRACT" --json | tee "$BOTH/init.json" | jq -e '
  .kind == "init"
  and .status == "ok"
  and .source_kind == "compose"
  and (.compose_source | test("compose\\.yaml$"))
' >/dev/null
grep -q 'web:' "$BOTH_CONTRACT"

# Neither found.
EMPTY="$WORK/empty"
mkdir -p "$EMPTY"
if "$PODBAY" init --from-codebase "$EMPTY" -f "$EMPTY/podbay.yaml" --json >"$EMPTY/init.json" 2>/dev/null; then
  echo "ci-dockerfile-from-codebase-demo: expected neither-found failure" >&2
  exit 1
fi
jq -e '
  .kind == "init"
  and .status == "failed"
  and (.issues[0].code == "codebase_discovery_not_found")
' "$EMPTY/init.json" >/dev/null

# Overwrite must fail closed with a stable code.
if "$PODBAY" init --from-codebase "$WORK" -f "$CONTRACT" --json >"$WORK/overwrite.json" 2>/dev/null; then
  echo "ci-dockerfile-from-codebase-demo: expected overwrite failure" >&2
  exit 1
fi
jq -e '
  .kind == "init"
  and .status == "failed"
  and (.issues[0].code == "init_target_exists")
' "$WORK/overwrite.json" >/dev/null

echo "ci-dockerfile-from-codebase-demo: ok (work=$WORK)"
