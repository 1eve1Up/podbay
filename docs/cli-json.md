# Podbay CLI JSON and exit behavior

Reference for `--json` envelopes, receipts, and automation exit codes. Contract fields: [contract.md](contract.md). Automation patterns: [agent-loop.md](agent-loop.md). Terminology: [glossary.md](glossary.md).

---

## JSON output and receipts

Podbay’s JSON output is designed for tools, agents, and CI. Versioned documents include:

- `format_version`
- `kind`, such as `validate`, `deploy`, `diff`, `receipt_read`, `receipt_list`, `teardown`, `import_compose`, or `logs`
- `status`, usually `ok` or `failed`
- `issues[]`, with stable-ish codes, levels, messages, and optional service names
- optional `deploy_services` on **`validate`** / **`deploy`** / **`diff`** / **`ps`** / **`explain`** / **`teardown` / `down`** JSON when you pass explicit service roots on the CLI; optional **`dependents_expand`** when partial roots are combined with **`--dependents`**

### Deploy health-gate failures

**`deploy --json` health-gate failures** (runtime, after containers start) emit structured `issues[]` entries with a **`service`** field and stable codes:

| Code | When |
| --- | --- |
| `deploy_health_timeout` | Health probe deadline exceeded for a service in the deploy set |
| `deploy_health_probe_failed` | HTTP/exec probe failed before timeout |
| `deploy_external_dep_unhealthy` | Partial deploy waited on an external dependency’s health and it failed |
| `deploy_error` | Non-health failures (build, start, volume, unexpected errors) |

Success **`deploy --json`** includes `status: ok`, optional `receipt_path` (absolute path of the written receipt file), and partial-selection fields. Preflight validate failures before deploy still surface validate-style issues, not health-gate codes.

### Deploy receipts (evidence)

After a **fully successful** deploy, `--receipt PATH` writes a versioned receipt JSON (`format_version` **1**):

- **File mode** — `PATH` is a file; that file is written atomically.
- **Directory mode** — `PATH` is an existing directory or ends with `/`; writes `<dir>/<UTC>-<deploy_id>.json`. `deploy --json` `receipt_path` is always the **file** written.

New success receipts include evidence fields: `deploy_id`, `contract_digest` (`sha256:` of contract file bytes), `status: ok`, and when partial roots apply, `deploy_services` / `dependents_expand` (same semantics as deploy `--json`). Older receipts without these fields still decode.

```bash
podbay receipt /path/to/receipt.json          # human summary (shows evidence fields when present)
podbay receipt /path/to/receipt.json --json   # kind: receipt_read
podbay receipt list .podbay/receipts/myproj --json   # kind: receipt_list, newest first
```

Receipts remain **success-only** evidence/audit artifacts (not crypto, SBOM, rollback, or failure-attempt records).

### Import compose JSON

See [contract.md#import-compose---json-ci-and-agents](contract.md) for **`import_compose`** success/failure shapes and stable **`issues[].code`** values.

### Demos

```bash
PODBAY_BIN=./podbay ./examples/ci-partial-agent-loop-demo.sh happy
PODBAY_BIN=./podbay ./examples/ci-partial-agent-loop-demo.sh fail
PODBAY_BIN=./podbay ./examples/ci-deploy-health-fail-demo.sh
```

### Example CI gate

```bash
set -euo pipefail

podbay validate -f examples/nginx --json
podbay deploy   -f examples/nginx --receipt .podbay/receipts/nginx/ --json
podbay receipt list .podbay/receipts/nginx --json
podbay diff     -f examples/nginx --json | jq -e '.drift == false'
```


The same flow is available as a runnable demo:

```bash
go build -o ./podbay ./cmd/podbay
PODBAY_BIN=./podbay ./examples/ci-receipt-demo.sh
```

In CI, `validate --json` is the preflight gate, `deploy --receipt` writes the evidence artifact, and `diff --json | jq -e '.drift == false'` fails closed when the live Podman runtime no longer matches the contract. If an operator introduces drift by starting or changing containers outside Podbay, the `jq` gate exits non-zero instead of asking the caller to scrape logs.

### Receipt pair diff

Receipt comparison does not need a live Podman runtime:

```bash
podbay diff /tmp/receipt-before.json /tmp/receipt-after.json --json
```

Deploy receipts use **`format_version`** in the JSON file. Receipt pair diff compares two decoded receipt files only (not contract vs runtime). When both sides have `contract_digest`, a mismatch is drift; when only one side has a digest, pair-diff emits an incomparable **warn** (legacy-compatible). `deploy_id` is not required to match across pair diffs.

Receipts are useful as deployment evidence, release artifacts, drift gates, and agent handoff objects.

### Logs JSON

**`logs --json`** returns one versioned document per invocation (`kind: logs`). Success includes **`log_entries[]`** for all resolved services; with exactly one resolved service, top-level **`service`** and **`log_body`** are also set. **`--json`** cannot be combined with **`--follow`**.

---

## Exit behavior

Podbay is meant to fail closed in automation:

- `validate` exits non-zero on fail-level validation issues.
- `deploy` exits non-zero on validation or runtime failure and does not write a partial receipt.
- `diff` exits non-zero when drift is detected or comparison cannot complete.
- `teardown` / `down` remove what they can and report structured issues in JSON; network removal warnings are non-fatal.
- `logs --json` exits **0** on success (including empty per-entry `log_body` values) and **1** on contract load/resolution errors, Podman unavailability, `podman logs` failure, **`--json` with `--follow`**, or **`--follow`** with multiple resolved services. Success may include **`log_entries[]`**, **`deploy_services`**, and **`dependents_expand`** when partial roots apply.
- `import compose --json` exits **0** on success and **1** on failure (JSON on stdout in both cases).

Use `--json` when the caller is a script, CI job, or code agent.

---

## Related docs

- [contract.md](contract.md) — `podbay.yaml` and import JSON codes
- [agent-loop.md](agent-loop.md) — validate → deploy → diff gate sequence
- [glossary.md](glossary.md) — JSON field vocabulary
- [architecture.md](architecture.md) — agent loop semantics
