"""Inbound ScanCode license validation (stdlib + license_expression only).

validate_report.py loads YAML policy and delegates here.
"""

from __future__ import annotations

from pathlib import PurePosixPath

from license_expression import ExpressionParseError, Licensing


def path_matches_any_glob(rel_path: str, globs: list[str]) -> bool:
    normalized = rel_path.replace("\\", "/").lstrip("/")
    pp = PurePosixPath(normalized)
    for raw in globs:
        pattern = raw.replace("\\", "/").lstrip("/")
        if pp.match(pattern):
            return True
    return False


def symbols_from_expression(expr: str) -> list[str]:
    licensing = Licensing()
    parsed = licensing.parse(expr)
    return [str(sym) for sym in parsed.symbols]


def validate_scan_code_json(data: dict, policy: dict) -> list[tuple[str, str, str]]:
    """Return list of (path, expression, reason) violations."""
    approved = {str(x) for x in policy["approved_spdx_license_identifiers"]}
    first_party_globs = list(policy["first_party_path_globs"])
    ref_prefixes = tuple(policy.get("allowed_license_ref_prefixes", []))
    options = policy.get("options") or {}
    allow_ref_on_fp = bool(
        options.get("allow_unknown_license_ref_tokens_on_first_party_paths", True)
    )
    allow_ref_outside_fp = bool(
        options.get("allow_license_ref_tokens_outside_first_party", False)
    )

    violations: list[tuple[str, str, str]] = []
    files = data.get("files") or []
    for entry in files:
        if entry.get("type") != "file":
            continue
        path = entry.get("path") or ""
        expr_spdx = entry.get("detected_license_expression_spdx")
        expr_raw = entry.get("detected_license_expression")
        expr = (expr_spdx or expr_raw or "").strip()
        if not expr:
            continue

        first_party = path_matches_any_glob(path, first_party_globs)

        try:
            symbols = symbols_from_expression(expr)
        except ExpressionParseError:
            violations.append((path, expr, "unparseable license expression"))
            continue

        for sym in symbols:
            if sym in approved:
                continue
            if any(sym.startswith(prefix) for prefix in ref_prefixes):
                if first_party and allow_ref_on_fp:
                    continue
                if allow_ref_outside_fp:
                    continue
                violations.append(
                    (
                        path,
                        expr,
                        f"disallowed ScanCode license ref token {sym!r} "
                        f"(first_party={first_party})",
                    )
                )
                continue
            violations.append(
                (
                    path,
                    expr,
                    f"license symbol {sym!r} not in approved_spdx_license_identifiers",
                )
            )

    return violations
