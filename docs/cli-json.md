# Podbay CLI JSON and exit behavior

Reference for `--json` envelopes, receipts, and automation exit codes. Contract fields: [contract.md](contract.md). Automation patterns: [agent-loop.md](agent-loop.md). Terminology: [glossary.md](glossary.md).

---

## JSON output and receipts

Podbay’s JSON output is designed for tools, agents, and CI. Versioned documents include:

- `format_version`
- `kind`, such as `validate`, `deploy`, `diff`, `receipt_read`, `receipt_list`, `orientation`, `teardown`, `import_compose`, or `logs`
- `status`, usually `ok` or `failed`
- `issues[]`, with stable-ish codes, levels, messages, and optional service names
- optional `deploy_services` on **`validate`** / **`deploy`** / **`diff`** / **`ps`** / **`explain`** / **`teardown` / `down`** / **`onboard`** JSON when you pass explicit service roots on the CLI; optional **`dependents_expand`** when partial roots are combined with **`--dependents`**

### Orientation / onboard

`podbay onboard --json` emits a versioned **`kind: orientation`** document shared with the additive **`orientation`** object on **`explain --json`**:

- identity: `project`, `contract_path`, optional `profiles` / `deploy_services` / `dependents_expand`
- `active_services`, `graph` (depends_on skim plus requirements: `ports`, `expose`, `health` as `http`/`exec`, `source` as `build`/`image`)
- optional `runtime` (live summary when Podman is available; omitted or `available: false` offline)
- `next_actions` — ordered agent-loop CLI hints (rule-based)
- `note` — always states structured context/next-steps only (not automatic remediation or root-cause diagnosis)

Load failures with `--json` use `status: failed` and issue code **`orientation_load_error`**.

```bash
podbay init -f /tmp/demo/podbay.yaml
podbay onboard -f /tmp/demo/podbay.yaml --json
podbay explain -f examples/nginx --json   # includes orientation when Podman is up
PODBAY_BIN=./podbay ./examples/ci-orientation-demo.sh
```

Orientation is **arrive** packaging. Failure intelligence remains **`receipt handoff`**. Explain remains factual runtime/health; orientation does not add causal diagnosis.

### Deploy health-gate failures

**`deploy --json` health-gate failures** (runtime, after containers start) emit structured `issues[]` entries with a **`service`** field and stable codes:

| Code | When |
| --- | --- |
| `deploy_health_timeout` | Health probe deadline exceeded for a service in the deploy set |
| `deploy_health_probe_failed` | HTTP/exec probe failed before timeout |
| `deploy_external_dep_unhealthy` | Partial deploy waited on an external dependency’s health and it failed |
| `deploy_error` | Non-health failures (build, start, volume, unexpected errors) |

Success **`deploy --json`** includes `status: ok`, optional `receipt_path` (absolute path of the written receipt file), and partial-selection fields. Health-gate **failures** with `--receipt` also include `receipt_path` when an attempt receipt was written. Preflight validate failures before deploy still surface validate-style issues, not health-gate codes, and do not write a receipt.

### Deploy receipts (evidence)

`--receipt PATH` writes a versioned receipt JSON (`format_version` **1**):

- **File mode** — `PATH` is a file; that file is written atomically.
- **Directory mode** — `PATH` is an existing directory or ends with `/`; writes `<dir>/<UTC>-<deploy_id>.json`. `deploy --json` `receipt_path` is always the **file** written.

**Success** receipts include evidence fields: `deploy_id`, `contract_digest` (`sha256:` of contract file bytes), `status: ok`, and when partial roots apply, `deploy_services` / `dependents_expand` (same semantics as deploy `--json`).

**Attempt** receipts (health-gate failures after deploy has started, when `--receipt` is set) use the same identity fields with `status: failed` and a `failure` object (`service`, `code`, `class`, `probe_kind`, `message`, optional external-dep context) aligned with `deploy_health_*` issue codes. Service snapshots are best-effort. No receipt is written on pure preflight validate failure or when `--receipt` is unset.

Older receipts without evidence/failure fields still decode.

```bash
podbay receipt /path/to/receipt.json          # human summary (shows evidence / failure fields when present)
podbay receipt /path/to/receipt.json --json   # kind: receipt_read
podbay receipt list .podbay/receipts/myproj --json                 # kind: receipt_list, newest first
podbay receipt list .podbay/receipts/myproj --status failed --json # attempts only (ok also matches legacy empty status)
podbay receipt last-ok .podbay/receipts/myproj --json              # kind: receipt_last_ok (or receipt_no_last_ok)
podbay diff --vs-last-ok .podbay/receipts/myproj /path/to/attempt.json --json
podbay receipt handoff /path/to/attempt.json --store .podbay/receipts/myproj --json  # kind: receipt_handoff
```

Receipts are evidence/audit artifacts (not crypto, SBOM, or rollback). **Structured handoff summaries** (`receipt handoff`) ship ordered next-action hints aligned with the agent-loop playbook; they are **not** automatic remediation or root-cause diagnosis.

### Import compose JSON

See [contract.md#import-compose---json-ci-and-agents](contract.md) for **`import_compose`** success/failure shapes and stable **`issues[].code`** values.

### Init JSON (`init` / `init --from-codebase`)

Pass **`--json`** to **`podbay init`** for a **`kind: init`** document (`format_version` **1**).

**Success (`status: ok`):** `contract_path` is the written `podbay.yaml`. For **`--from-codebase`**, also `source_kind` (`compose` or `dockerfile`), `compose_source` or `dockerfile_source`, `service_count`, and ordered **`next_actions`** (`onboard` / `validate`). Dockerfile success also lists **`extracted`** / **`gaps`** (`expose`, `health`, `published_ports`) and may append a hand-tighten hint when published ports or health are still missing. Greenfield success includes `project` and the same next-action dialect.

**Failure (`status: failed`, exit 1):** `issues[]` with stable codes, including:

| `code` | When |
| --- | --- |
| **`codebase_discovery_not_found`** | Neither a well-known Compose file nor Dockerfile under the directory (automatic fallback path). |
| **`compose_discovery_not_found`** | Explicit `--compose` path unusable / Compose discovery miss when not falling through. |
| **`dockerfile_discovery_not_found`** | Explicit `--dockerfile` path unusable / Dockerfile discovery miss. |
| **`init_target_exists`** | Target `podbay.yaml` already exists (refuse overwrite). |
| **`import_*`** | Same import/load codes as `import compose` when discovery finds a Compose file but load/translate fails. |
| **`init_error`** | Other init failures. |

### Demos

```bash
PODBAY_BIN=./podbay ./examples/ci-orientation-demo.sh
PODBAY_BIN=./podbay ./examples/ci-from-codebase-demo.sh
PODBAY_BIN=./podbay ./examples/ci-dockerfile-from-codebase-demo.sh
PODBAY_BIN=./podbay ./examples/ci-partial-agent-loop-demo.sh happy
PODBAY_BIN=./podbay ./examples/ci-partial-agent-loop-demo.sh fail
PODBAY_BIN=./podbay ./examples/ci-deploy-health-fail-demo.sh
PODBAY_BIN=./podbay ./examples/ci-receipt-intelligence-demo.sh
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
podbay diff --vs-last-ok .podbay/receipts/myproj /tmp/attempt.json --json
```

Deploy receipts use **`format_version`** in the JSON file. Receipt pair diff compares two decoded receipt files only (not contract vs runtime). When both sides have `contract_digest`, a mismatch is drift; when only one side has a digest, pair-diff emits an incomparable **warn** (legacy-compatible). `deploy_id` is not required to match across pair diffs.

`--vs-last-ok <dir>` resolves the newest ok receipt in the store (legacy empty status counts as ok) and runs the same pair-diff against the given current receipt. When no prior ok exists, the document fails with issue code **`receipt_no_last_ok`** and does **not** report false drift.

Receipts are useful as deployment evidence, release artifacts, drift gates, and agent handoff objects (`receipt handoff --json`).

### Logs JSON

**`logs --json`** returns one versioned document per invocation (`kind: logs`). Success includes **`log_entries[]`** for all resolved services; with exactly one resolved service, top-level **`service`** and **`log_body`** are also set. **`--json`** cannot be combined with **`--follow`**.

---

## Exit behavior

Podbay is meant to fail closed in automation:

- `validate` exits non-zero on fail-level validation issues.
- `deploy` exits non-zero on validation or runtime failure. It does not write a receipt on preflight validate failure or when `--receipt` is unset; health-gate failures with `--receipt` write an intentional attempt receipt (`status: failed`).
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
