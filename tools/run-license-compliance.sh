#!/usr/bin/env bash
# Run inbound license checks for this monorepo (ScanCode + allowlist, optional pip metadata).
#
# Prerequisites:
#   pip install -r tools/licensing/requirements.txt
#   Debian/Ubuntu: sudo apt-get install -y libarchive13  (SONAME lib for ScanCode extractcode)
#
# Environment:
#   SCANCODE_REPORT_PATH   — JSON output path (default: ./scancode-license-report.json)
#   SCANCODE_TARGETS       — space-separated override paths (see tools/licensing/scancode_scan.sh)
#
# Flags:
#   --skip-scancode   — only run policy self-tests and optional --pip-report
#   --pip-report      — print pip list license summaries (stdlib + pip; no ScanCode deps required)
#   -h, --help

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LICENSE_DIR="${ROOT}/tools/licensing"

usage() {
  sed -n '1,20p' "$0" | tail -n +2
}

SKIP_SCANCODE=0
PIP_REPORT=0
while [[ $# -gt 0 ]]; do
  case "$1" in
    --skip-scancode) SKIP_SCANCODE=1 ;;
    --pip-report) PIP_REPORT=1 ;;
    -h | --help)
      usage
      exit 0
      ;;
    *)
      echo "unknown option: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
  shift
done

cd "${ROOT}"

echo "== License policy unit tests (unittest; requires license-expression) =="
python3 -m unittest discover -s "${LICENSE_DIR}" -p 'test_*.py' -v

if [[ "${SKIP_SCANCODE}" -eq 0 ]]; then
  echo "== ScanCode license scan =="
  bash "${LICENSE_DIR}/scancode_scan.sh"
  REPORT="${SCANCODE_REPORT_PATH:-${ROOT}/scancode-license-report.json}"
  echo "== Validate report vs allowlist =="
  python3 "${LICENSE_DIR}/validate_report.py" --report "${REPORT}"
else
  echo "(ScanCode scan skipped)"
fi

if [[ "${PIP_REPORT}" -eq 1 ]]; then
  echo "== pip package license metadata =="
  python3 "${LICENSE_DIR}/pip_licenses.py"
fi

echo "License compliance run finished OK."
