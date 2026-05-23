# Podbay Releases

## v2026.5.3

**Date:** May 2026  
**Stability:** public preview  
**Contract status:** evolving  

### Shipped scope (additions since `v2026.5.2`)

- **`podbay deploy --json` health-gate failures** — structured `issues[]` with stable codes (`deploy_health_timeout`, `deploy_health_probe_failed`, `deploy_external_dep_unhealthy`) and per-issue **`service`**; non-health failures still use **`deploy_error`**. Success JSON unchanged.
- **Demo:** `examples/ci-deploy-health-fail-demo.sh` and `examples/unhealthy-health/` for parseable deploy failure at a health gate.

## v2026.5.2

**Date:** May 2026  
**Stability:** public preview  
**Contract status:** evolving  

### Shipped scope (additions since `v2026.5.1`)

- **`podbay logs` partial selection** — optional service roots and **`--dependents`** use the same resolution as **`diff`**, **`ps`**, **`explain`**, and **`teardown` / `down`** (via **`spec.ObservabilityActiveServices`**). With no service names, the profile-active set is used.
- **`podbay logs --json` batch evidence** — one **`kind: logs`** document with **`log_entries[]`** (`service`, `container_name`, `log_body`) per resolved service. When exactly one service resolves, top-level **`service`** / **`log_body`** remain for backward compatibility. Additive **`deploy_services`** / **`dependents_expand`** when partial roots apply.
- **Demo:** `examples/ci-partial-logs-demo.sh` and `examples/two-service/` for partial deploy + batch logs.

## v2026.5.1

**Date:** May 2026  
**Stability:** public preview  
**Contract status:** evolving  
**Receipt format:** versioned  
**Production claim:** suitable for narrow Podman stacks, not a Kubernetes replacement.

`v2026.5.1` is the second public May 2026 release. It is usable, but the `podbay.yaml` contract is not yet 1.0-stable.

### Shipped scope (additions since `v2026.5.0`)

- **Compose `include:` (v1 subset)** — local relative paths only; merged before `extends:` in `podbay import compose`. See [README](README.md) (Import from Compose) and `examples/compose-include/`.
- **`podbay import compose --json`** — on failure, emits a versioned JSON envelope (`kind: import_compose`) with stable `issues[].code` values for compose read/parse, include graph errors, and unsupported include shapes; on success, emits one JSON document (`status: ok`) with `contract_yaml`, `service_count`, optional `project`, and optional `output_path` when `-o` is used. See [README](README.md) (`import compose --json`).
- **`podbay validate` / `podbay deploy` partial by service name** — optional service arguments after `-f <contract>` or after `<contract-path>` (multi-arg form) select explicit targets within `--profile`; by default the effective set is **only** those services. **`--dependents`** adds the transitive downstream closure within the profile-active map. **`dependents:`** must list every **`depends_on`** child (inverse validation). `validate`/`deploy` **`--json`** may include `deploy_services` and, with **`--dependents`**, `dependents_expand`. **`podbay diff`**, **`podbay ps`**, and **`podbay explain`** use the same partial roots and **`--dependents`** for contract-vs-runtime views (default: full profile-active set); **`diff --json`** / **`ps --json`** / **`explain --json`** include `deploy_services` / `dependents_expand` when applicable. **Receipt pair** diff is unchanged (two decoded receipt files). See [README](README.md).
- **Bidirectional `depends_on` / `dependents:`** — every child→parent edge appears on the parent's **`dependents`** list; validate rejects missing or stray entries. See [README](README.md).
- **`podbay teardown` / `podbay down` partial selection** — optional service roots and **`--dependents`** match **`validate`** / **`deploy`**; partial teardown skips project network removal while labelled containers remain; **`--volumes`** is rejected with partial roots; **`teardown` / `down` `--json`** may include **`deploy_services`** and **`dependents_expand`**. See [README](README.md).
- **`podbay logs --json`** — one versioned JSON document per invocation (`kind: logs`); captures non-streaming `podman logs` output in **`log_body`**. **`--json`** cannot be combined with **`--follow`**. See [README](README.md).
- **Breaking (pre-GA):** the service-level YAML key **`dependencies`** was renamed to **`dependents`**; the old key is **not** loaded. **`podbay import compose`** emits **`dependents:`**. Validate issue codes use the **`dependents_*`** prefix (**`dependents_unknown_service`**, **`dependents_invalid_dependent`**, **`dependents_missing_inverse`**) instead of **`dependencies_*`**. Migrate contracts and any CI that asserted the old codes.

Known limitations, non-goals, install/verify steps, and stability framing for `v2026.5.1` are unchanged from `v2026.5.0` unless called out above; see the `v2026.5.0` section below.

## v2026.5.0

**Date:** May 2026  
**Stability:** public preview  
**Contract status:** evolving  
**Receipt format:** versioned  
**Production claim:** suitable for narrow Podman stacks, not a Kubernetes replacement.

`v2026.5.0` is the first public May 2026 release. It is usable, but the `podbay.yaml` contract is not yet 1.0-stable.

### Shipped Scope

- `podbay.yaml` runtime contract for Podman-based stacks
- preflight validation for dependency graphs, ports, paths, commands, profiles, network rules, and health expectations
- deterministic Podman deploy behavior with project/service labels
- deploy receipts with `format_version`
- contract-vs-runtime diff and receipt-vs-receipt diff
- versioned JSON output for automation on key commands
- factual runtime inspection through `podbay explain`
- Compose import for a documented subset, including practical Podman parity behavior
- public trust baseline: Apache-2.0 license, security policy, contribution guide, issue template, and root Go CI
- CI receipt demo using `validate --json`, `deploy --receipt`, and `diff --json | jq`

### Install and Verify

Install from a clone:

```bash
go install ./cmd/podbay
```

Run the CI receipt demo:

```bash
go build -o ./podbay ./cmd/podbay
PODBAY_BIN=./podbay ./examples/ci-receipt-demo.sh
```

### Stability Statement

Podbay uses calendar-based release versions. Calendar versions identify releases; they are not compatibility promises.

Until a future `v2026.x-stable` or `v1.0` commitment:

- **Contract stability:** `podbay.yaml` may evolve between public preview releases.
- **Receipt format stability:** receipts are machine-readable and versioned with `format_version`, but fields may still evolve before a stable commitment.
- **CLI compatibility:** core commands are intended to remain scriptable, especially with `--json`, but flags and output details may still change during public preview.
- **Migration policy:** release notes will call out breaking changes and provide migration guidance when contract, receipt, or CLI behavior changes.

### Known Limitations

- Quadlet/systemd compilation is not shipped in `v2026.5.0`.
- `podbay explain` reports factual runtime state, health probes, dependencies, and unexpected containers; it does not infer root cause.
- Receipts are deploy evidence and audit artifacts, not rollback, SBOM/provenance, or cryptographic attestation.
- Podman-backed integration behavior depends on the host environment and Podman version.
- Compose import intentionally supports a documented subset rather than every Compose feature.

### Non-Goals

Podbay is not:

- a Kubernetes replacement
- a scheduler or long-running control plane
- a managed CI/CD platform
- a secret manager
- a universal Compose implementation
- a sandbox for untrusted deployment contracts

### Fast Follow

Likely next release work:

- Compose `profiles:` / deeper `include:` parity (e.g. `env_file` on include entries, remote includes)
- richer external-network/IPAM behavior
- first real Quadlet/systemd adapter slice
- narrower, testable diagnostic improvements for `podbay explain`
- release checksums, JSON Schema publication, and stronger provenance artifacts
