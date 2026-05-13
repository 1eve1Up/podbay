#!/usr/bin/env python3
"""
Repair Dev Containers configs for Podman on macOS/Linux.

Podman + Dev Containers + non-root remoteUser can leave HOME=/root while exec runs
as e.g. vscode, so Cursor/VS Code tries to mkdir /root/.cursor-server and fails.
See: https://github.com/microsoft/vscode-remote-release/issues/7657

Usage:
  python3 scripts/devcontainer-podman-home-fix.py [--dry-run] [DIR ...]

Default DIRs if none given: ~/Code ~/Documents/Code ~/Developer

Only touches JSON files that parse strictly; skips JSONC with comments.
"""
from __future__ import annotations

import argparse
import json
import os
import sys
from pathlib import Path

HOME_FOR_REMOTE_USER: dict[str, str] = {
    "vscode": "/home/vscode",
    "node": "/home/node",
    "ubuntu": "/home/ubuntu",
}


def skip_dir_parts(parts: tuple[str, ...]) -> bool:
    skip = {".git", "node_modules", "vendor", ".venv", "dist", "build", "__pycache__"}
    return any(p in skip for p in parts)


def find_devcontainer_files(roots: list[Path]) -> list[Path]:
    """Walk trees (including hidden `.devcontainer/` dirs — pathlib globs skip those)."""
    out: list[Path] = []
    skip_dirnames = {".git", "node_modules", "vendor", ".venv", "dist", "build", "__pycache__"}

    for root in roots:
        if not root.is_dir():
            continue
        root = root.resolve()
        for dirpath, dirnames, filenames in os.walk(root, topdown=True, followlinks=False):
            dirnames[:] = sorted(d for d in dirnames if d not in skip_dirnames)
            p = Path(dirpath)
            if skip_dir_parts(p.parts):
                continue
            if p.name == ".devcontainer" and "devcontainer.json" in filenames:
                out.append(p / "devcontainer.json")
            if ".devcontainer.json" in filenames:
                out.append(p / ".devcontainer.json")
    return sorted(set(out))


def patch_file(path: Path, dry_run: bool) -> str | None:
    raw = path.read_text(encoding="utf-8")
    try:
        data = json.loads(raw)
    except json.JSONDecodeError:
        return "skip-not-json"

    ru = data.get("remoteUser")
    if not isinstance(ru, str) or ru not in HOME_FOR_REMOTE_USER:
        return "skip-remoteUser"

    want_home = HOME_FOR_REMOTE_USER[ru]
    ce = data.get("containerEnv")
    if ce is None:
        data["containerEnv"] = {"HOME": want_home}
    elif not isinstance(ce, dict):
        return "skip-bad-containerEnv"
    elif ce.get("HOME") == want_home:
        return "unchanged"
    else:
        ce = dict(ce)
        ce["HOME"] = want_home
        data["containerEnv"] = ce

    new_raw = json.dumps(data, indent=2, ensure_ascii=False) + "\n"
    if dry_run:
        return f"would-update remoteUser={ru!r}"
    path.write_text(new_raw, encoding="utf-8")
    return f"updated remoteUser={ru!r} HOME={want_home!r}"


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    ap.add_argument("--dry-run", action="store_true", help="print actions only")
    ap.add_argument(
        "dirs",
        nargs="*",
        type=Path,
        help="roots to scan (default: ~/Code ~/Documents/Code ~/Developer)",
    )
    args = ap.parse_args()
    roots = list(args.dirs) if args.dirs else []
    if not roots:
        home = Path.home()
        roots = [home / "Code", home / "Documents/Code", home / "Developer"]

    files = find_devcontainer_files(roots)
    if not files:
        print("No devcontainer.json files found under:", ", ".join(str(r) for r in roots))
        return 1

    counts: dict[str, int] = {}
    for f in files:
        res = patch_file(f, args.dry_run)
        counts[res or "unknown"] = counts.get(res or "unknown", 0) + 1
        if res not in ("skip-not-json", "skip-remoteUser", "skip-bad-containerEnv", "unchanged"):
            print(f"{f}: {res}")
        elif args.dry_run and res == "unchanged":
            print(f"{f}: ok ({res})")

    print("Summary:", dict(sorted(counts.items())))
    return 0


if __name__ == "__main__":
    sys.exit(main())
