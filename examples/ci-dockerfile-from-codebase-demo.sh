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
  and (.next_actions | map(tostring) | join("\n") | test("hand-tighten"))
  and ((.extracted | not) or (.extracted | length) == 0)
  and (.gaps | index("published_ports") != null)
' >/dev/null

test -f "$CONTRACT"
grep -q 'dockerfile: Dockerfile' "$CONTRACT"
if grep -q '^[[:space:]]*ports:' "$CONTRACT"; then
  echo "ci-dockerfile-from-codebase-demo: bare Dockerfile invented ports" >&2
  exit 1
fi

"$PODBAY" onboard -f "$CONTRACT" --json | tee "$WORK/onboard.json" | jq -e '
  .kind == "orientation"
  and .format_version == 1
  and (.active_services | length) > 0
  and (.next_actions | length) >= 1
  and (.next_actions | map(tostring) | join("\n") | test("hand-tighten"))
  and ([.graph[] | select(.source == "build")] | length) > 0
' >/dev/null

"$PODBAY" validate -f "$CONTRACT" --json | tee "$WORK/validate.json" | jq -e '
  .kind == "validate"
  and .status == "ok"
' >/dev/null

# Declared EXPOSE + HEALTHCHECK are copied; published ports stay hand-tighten.
RICH="$WORK/rich"
mkdir -p "$RICH"
cat >"$RICH/Dockerfile" <<'EOF'
FROM docker.io/library/alpine:3.20
EXPOSE 8080
HEALTHCHECK CMD wget -q -O- http://127.0.0.1/
CMD ["sleep", "infinity"]
EOF
RICH_CONTRACT="$RICH/podbay.yaml"
"$PODBAY" init --from-codebase "$RICH" -f "$RICH_CONTRACT" --json | tee "$RICH/init.json" | jq -e '
  .kind == "init"
  and .status == "ok"
  and .source_kind == "dockerfile"
  and (.extracted | index("expose") != null)
  and (.extracted | index("health") != null)
  and (.gaps | index("published_ports") != null)
' >/dev/null
grep -q 'expose:' "$RICH_CONTRACT"
grep -q 'health:' "$RICH_CONTRACT"
if grep -q '^[[:space:]]*ports:' "$RICH_CONTRACT"; then
  echo "ci-dockerfile-from-codebase-demo: EXPOSE invented published ports" >&2
  exit 1
fi
"$PODBAY" onboard -f "$RICH_CONTRACT" --json | tee "$RICH/onboard.json" | jq -e '
  .kind == "orientation"
  and (.graph[0].source == "build")
  and (.graph[0].health == "exec")
  and (.graph[0].expose | index("8080") != null)
  and ((.graph[0].ports | not) or (.graph[0].ports | length) == 0)
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
