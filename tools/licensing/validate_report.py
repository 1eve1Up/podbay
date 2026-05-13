#!/usr/bin/env python3
"""Validate a ScanCode JSON report against tools/licensing/license_policy.allowlist.yaml."""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

import yaml

from license_engine import validate_scan_code_json


def load_policy(path: Path) -> dict:
    data = yaml.safe_load(path.read_text(encoding="utf-8"))
    if not isinstance(data, dict):
        raise ValueError(f"policy root must be a mapping: {path}")
    missing = [
        k
        for k in ("approved_spdx_license_identifiers", "first_party_path_globs")
        if k not in data
    ]
    if missing:
        raise ValueError(f"policy missing keys {missing}: {path}")
    return data


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--report",
        type=Path,
        required=True,
        help="ScanCode JSON report (--json-pp output)",
    )
    parser.add_argument(
        "--policy",
        type=Path,
        default=Path(__file__).resolve().parent / "license_policy.allowlist.yaml",
        help="YAML inbound license policy",
    )
    args = parser.parse_args(argv)

    policy = load_policy(args.policy)
    report = json.loads(args.report.read_text(encoding="utf-8"))
    violations = validate_scan_code_json(report, policy)

    if violations:
        print("License policy violations:", file=sys.stderr)
        for path, expr, reason in violations:
            print(f"  {path}: {reason}", file=sys.stderr)
            print(f"    expression: {expr}", file=sys.stderr)
        return 1
    print("License policy check passed (no disallowed inbound license symbols).")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
