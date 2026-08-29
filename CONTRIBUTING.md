# Contributing to Podbay

Thanks for helping improve Podbay.

Podbay is a public-preview runtime contract layer for narrow Podman stacks. Please keep contributions scoped, testable, and honest about what is shipped versus planned.

## Prerequisites

- Go 1.22+
- Podman on `PATH` for deploy/runtime tests
- `curl` on `PATH` for HTTP health checks and `podbay explain`
- `jq` for JSON examples and CI-style demos

## Local Verification

Public Podbay checkouts run the root Go checks through this script (`gofmt -l .`, then `go vet ./...`, then `go test ./...`, matching `.github/workflows/go.yml`). 

For a quick Go-only check while iterating:

```bash
gofmt -l .
go vet ./...
go test ./...
```

Some integration tests require Podman and may skip when Podman is unavailable.

## License compliance (optional)

Inbound license checks (ScanCode plus the SPDX allowlist in `tools/licensing/license_policy.allowlist.yaml`) run in CI when pull requests touch Go sources, docs, examples, or `tools/licensing/`. To reproduce locally:

```bash
# Debian/Ubuntu: ScanCode’s extractcode stack needs the real libarchive SONAME (e.g. libarchive.so.13).
sudo apt-get update && sudo apt-get install -y libarchive13

pip install -r tools/licensing/requirements.txt
./tools/run-license-compliance.sh
```

Use `./tools/run-license-compliance.sh --skip-scancode` to run only the policy unit tests (no ScanCode install needed beyond `license-expression`). Append `--pip-report` to print `pip list` license metadata for the active environment.

`requirements.txt` includes `extractcode-libarchive-system-provided`, which locates `libarchive.so.13` under `/usr/lib/<triplet>-linux-gnu/`. If you override paths, set `EXTRACTCODE_LIBARCHIVE_PATH` to that SONAME file—not the unversioned `libarchive.so` stub—or ScanCode may fail with undefined `archive_read_new`.

## Development Workflow

1. Open or choose an issue with a clear scope.
2. Keep changes small and focused.
3. Add or update tests when behavior changes.
4. Update docs when user-facing behavior, CLI flags, contract fields, or receipt fields change.

## Architecture

Package boundaries and the import pipeline are documented in [docs/architecture.md](docs/architecture.md). Use [docs/contract-change-checklist.md](docs/contract-change-checklist.md) when changing contract fields or import behavior.

| Doc | When to read |
| --- | --- |
| [docs/glossary.md](docs/glossary.md) | Terminology across CLI flags, JSON fields, and Go packages |
| [docs/contract.md](docs/contract.md) | `podbay.yaml` schema, networks, profiles, Compose import |
| [docs/cli-json.md](docs/cli-json.md) | `--json` envelopes, receipts, exit behavior |
| [docs/agent-loop.md](docs/agent-loop.md) | Validate → deploy → diff automation and partial-deploy demos |

## Stability Expectations

Podbay uses calendar-based release versions. `v2026.8.0` is the current public preview, not a 1.0 compatibility commitment.

Until a future `v2026.x-stable` or `v1.0` commitment:

- `podbay.yaml` may evolve.
- JSON receipts are versioned but may gain or change fields.
- CLI behavior is intended to remain scriptable, especially with `--json`, but may still change.
- Release notes should call out breaking changes and migration guidance.

## Security

Do not open public issues for suspected vulnerabilities. Follow `SECURITY.md`.

Podbay executes real Podman operations as the invoking user. Treat untrusted `podbay.yaml` files like untrusted deployment scripts.
