#!/usr/bin/env python3
"""Print declared license metadata for each package shown by `pip list`.

Reads installed distribution metadata (License, License-Expression, License :: classifiers).
This reflects what maintainers put in package metadata, not a full legal audit.
"""

from __future__ import annotations

import argparse
import json
import subprocess
import sys
from importlib import metadata


def pip_list_json() -> list[dict]:
    proc = subprocess.run(
        [sys.executable, "-m", "pip", "list", "--format=json"],
        check=True,
        capture_output=True,
        text=True,
    )
    return json.loads(proc.stdout)


def license_fields(
    name: str,
) -> tuple[str | None, str | None, list[str], bool]:
    try:
        meta = metadata.metadata(name)
    except metadata.PackageNotFoundError:
        return None, None, [], False
    raw = meta.get("License")
    expr = meta.get("License-Expression")
    classifiers = [
        c for c in (meta.get_all("Classifier") or []) if c.startswith("License ::")
    ]
    return raw, expr, classifiers, True


def format_license_summary(
    raw: str | None, expr: str | None, classifiers: list[str]
) -> str:
    parts: list[str] = []
    if expr and expr.strip():
        parts.append(f"SPDX: {expr.strip()}")
    if raw and raw.strip():
        collapsed = " ".join(raw.split())
        if collapsed.upper() != "UNKNOWN":
            parts.append(collapsed)
    if classifiers:
        parts.append("; ".join(classifiers))
    if not parts:
        return "(no license metadata)"
    return " | ".join(parts)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--json",
        action="store_true",
        help="Print one JSON object per line (package, version, license_summary, ...)",
    )
    args = parser.parse_args()

    packages = pip_list_json()
    rows: list[dict] = []
    for pkg in sorted(packages, key=lambda p: p["name"].lower()):
        name = pkg["name"]
        version = pkg.get("version", "")
        raw, expr, classifiers, found = license_fields(name)
        summary = format_license_summary(raw, expr, classifiers)
        rows.append(
            {
                "package": name,
                "version": version,
                "license_summary": summary,
                "license_expression": expr,
                "license_text_field": raw,
                "license_classifiers": classifiers,
                "metadata_found": found,
            }
        )

    if args.json:
        for row in rows:
            print(json.dumps(row, ensure_ascii=False))
        return 0

    width = max(len(r["package"]) for r in rows) if rows else 0
    width = max(width, len("package"))
    for row in rows:
        pkg = row["package"]
        ver = row["version"]
        summ = row["license_summary"]
        flag = "" if row["metadata_found"] else " [metadata missing]"
        print(f"{pkg:{width}}  {ver:12}  {summ}{flag}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
