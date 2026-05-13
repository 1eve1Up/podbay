#!/usr/bin/env bash
# Run ScanCode for inbound license detection on Podbay first-party paths.
# Requires: pip install -r tools/licensing/requirements.txt
#
# libarchive: extractcode uses the `extractcode-libarchive-system-provided` plugin, which expects
# the SONAME library (e.g. /usr/lib/<arch>-linux-gnu/libarchive.so.13 on Debian/Ubuntu). Install
# `libarchive13` (apt) before running. Do not point EXTRACTCODE_LIBARCHIVE_PATH at the unversioned
# libarchive.so linker stub—ctypes can load the wrong object and fail with undefined symbols.
#
# Optional: SCANCODE_TARGETS="cmd internal README.md" overrides default scan list (space-separated).

set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${HERE}/../.." && pwd)"
cd "${ROOT}"

REPORT_PATH="${SCANCODE_REPORT_PATH:-${ROOT}/scancode-license-report.json}"

IGNORE_ARGS=(
  --ignore "*.pyc"
  --ignore "__pycache__/*"
  --ignore "**/__pycache__/*"
  --ignore ".venv/*"
  --ignore "**/.venv/**"
  --ignore ".pytest_cache/*"
  --ignore "**/.pytest_cache/**"
  --ignore "build/*"
  --ignore "dist/*"
  --ignore "*.egg-info/*"
)

# Default: Go tree, docs, and license tooling (not full tools/ to avoid scanning transient venvs).
if [[ -n "${SCANCODE_TARGETS:-}" ]]; then
  read -r -a TARGETS <<<"${SCANCODE_TARGETS}"
else
  TARGETS=(
    cmd
    internal
    examples
    README.md
    LICENSE
    CONTRIBUTING.md
    SECURITY.md
    RELEASES.md
    go.mod
    go.sum
    tools/licensing
  )
fi

# Drop missing paths so scancode does not fail on absent optional files.
EXISTING=()
for p in "${TARGETS[@]}"; do
  if [[ -e "${ROOT}/${p}" ]]; then
    EXISTING+=("${p}")
  fi
done

if [[ ${#EXISTING[@]} -eq 0 ]]; then
  echo "scancode_scan: no existing paths to scan (cwd=${ROOT})" >&2
  exit 1
fi

scancode \
  -l \
  --license-score 80 \
  -n 2 \
  --timeout 120.0 \
  "${IGNORE_ARGS[@]}" \
  --json-pp "${REPORT_PATH}" \
  "${EXISTING[@]}"

echo "ScanCode report: ${REPORT_PATH}"
