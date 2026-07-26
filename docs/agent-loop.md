# Podbay agent loop

How agents and CI use Podbay as an operational contract: preflight, deploy, drift, evidence, and cleanup. JSON details: [cli-json.md](cli-json.md). Contract and partial selection: [contract.md](contract.md). Terminology: [glossary.md](glossary.md). Package semantics: [architecture.md](architecture.md).

---

## The agent angle

A code agent does not need another prose README telling it “run the app somehow.” It needs a clear operational contract it can validate, deploy, inspect, diff, and hand back as evidence.

Podbay gives agents:

- **A target**: edit the app and keep `podbay.yaml` true.
- **A preflight gate**: `podbay validate --json` before touching runtime.
- **A deploy gate**: `podbay deploy --json --receipt ...` with structured success/failure.
- **A drift gate**: `podbay diff --json` to prove runtime matches the contract.
- **A log gate**: `podbay logs --json` captures container logs for one or many resolved services in one document (`log_entries[]`; not combinable with `--follow`).
- **A durable receipt**: a deploy artifact that can be compared later without asking the agent to narrate from memory.
- **A shared language**: builder, reviewer, test, deploy, security, and ops agents can all reason over the same file and JSON envelopes.

That is the core difference from ordinary Compose usage: **Podbay treats runtime intent as an artifact agents can be held accountable to.**

Unified helpers keep commands aligned: **`spec.ObservabilityActiveServices`** for partial selection and **`expand.ExpandService`** for host env expansion (see [architecture.md](architecture.md)).

---

## When to use Podbay

Use Podbay when:

- You have a 1–10 service app that should run cleanly on Podman.
- You want Docker Compose-like ergonomics without relying on Docker Desktop in production-ish environments.
- You are shipping a production appliance, demo stack, AI tool, internal service, edge workload, or VM-hosted MVP.
- Multiple agents or developers are touching services and you need one runtime contract.
- You want CI to validate, deploy, diff, and collect receipts from CLI-based JSON outputs instead of scraping logs.
- Kubernetes would be premature, but shell scripts plus `podman run` are already getting sloppy.

Do not use Podbay when:

- You need cluster scheduling, autoscaling, service mesh, CRDs, or multi-node orchestration. Use Kubernetes or Red Hat® OpenShift®.
- You only need to run one container. Use `podman run`.
- Your team is already happy with Docker Compose and does not care about Podman, receipts, drift, or agent-readable JSON.

---

## Partial-deploy agent loop

For multi-service stacks, agents often deploy **one root** plus **`--dependents`**, then prove drift and collect evidence on the **same roots**—without re-parsing stderr when deploy fails at a health gate.

| Step | Command | On failure |
| --- | --- | --- |
| Preflight | `podbay validate -f <contract> [roots...] --json` | Fix contract; do not deploy |
| Deploy | `podbay deploy … --dependents --json --receipt …` | Parse `issues[].code` (see health table in [cli-json.md](cli-json.md)) |
| Drift | `podbay diff … --json` (same roots **and same `--profile` set**) | `drift == true` → inspect or redeploy |
| Evidence | `podbay logs … --json` (same roots **and same `--profile` set**) | `log_entries[]` per resolved service |
| Diagnose | `podbay explain … --json` (same roots **and same `--profile` set**) | Factual runtime/health context (not root cause) |
| Cleanup | `podbay down … --json` (same roots or full project) | — |

When the deploy used `--profile`, pass those flags again on `diff` / `ps` / `explain` / `logs`. Profile selection is not stored on containers; omitting `--profile` shrinks the expected set and can flag still-running profile-gated containers as unexpected. See [Profiles](contract.md#profiles).

### Failure playbook

After containers start but health never passes:

1. Read **`deploy --json`** `issues[]` for `deploy_health_timeout`, `deploy_health_probe_failed`, or `deploy_external_dep_unhealthy`; use **`service`** on the issue.
2. Run **`logs --json`** and **`explain --json`** with that service (and **`--dependents`** when downstream services are in the partial set). **`explain`** runs **single-shot** health probes capped at **5 seconds per probe** (proximate-network diagnostic budget)—not deploy’s `--health-timeout` retry window or full contract `health.timeout`.
3. Tear down with **`down` / `teardown --json`** so the next attempt starts clean.

### Runnable demo

```bash
go build -o ./podbay ./cmd/podbay
PODBAY_BIN=./podbay ./examples/ci-partial-agent-loop-demo.sh happy
PODBAY_BIN=./podbay ./examples/ci-partial-agent-loop-demo.sh fail
```

Older focused demos: `ci-receipt-demo.sh`, `ci-partial-logs-demo.sh`, `ci-deploy-health-fail-demo.sh`.

---

## Partial validate and deploy

```bash
podbay validate -f examples/nginx web
podbay deploy   -f examples/nginx web
podbay validate ./examples/nginx/podbay.yaml web
podbay deploy web
```

When `./podbay.yaml` exists in the current directory, a **single** argument that is not an existing contract path or `…/podbay.yaml` on disk is treated as a **service name** (same as Compose-style `up <service>` from the project root). If a subdirectory or file with that name is a valid contract location, that wins over the service interpretation.

- You name **explicit targets** (after profile filtering). By default the effective validate/deploy set is **only** those targets (still ordered by **`depends_on`** within that subgraph). Pass **`--dependents`** to add every profile-active service that **transitively depends on** (downstream of) any name already in the set. Every **`depends_on`** edge **child → parent** must be mirrored on **parent** **`dependents:`** (and only real dependents may appear there); **`podbay validate`** enforces both directions. **`depends_on`** still defines **startup order** among services in the set and, for edges pointing **outside** the set, **`podbay deploy`** **pre-waits** on existing containers (started/healthy) **without** redeploying them. If such a container is missing, partial deploy fails with a clear error.
- **`--dependents` from a deep prerequisite:** e.g. `podbay deploy postgres --dependents` walks **downstream** along `depends_on`: first **`api`** (depends on postgres), then **`web`** and **`worker`** (both depend on api)—so most of the app can enter the deploy set. Use **`podbay deploy postgres`** without the flag when you want **only** Postgres recreated; check the log line **“explicit targets only”** vs **“dependents expansion”** to confirm which mode ran.
- **Uptime in Podman Desktop vs Podbay’s partial set:** Podbay only **`podman rm` / `podman run`** services in the log’s list. If you redeploy **postgres** alone, other containers are **not** removed by Podbay—but a neighbor like **worker** may **exit** when the DB disappears and Podman’s **`restart: unless-stopped`** (from your contract’s `restart:` field) starts a **fresh** process. That can show the same “age” as postgres in the UI even though the partial deploy line said one service. Services that stay healthy through the blip (often **api** / **web**) keep their older uptime.
- **`podbay diff`**, **`podbay ps`**, **`podbay explain`**, **`podbay teardown`**, and **`podbay down`** accept the same optional service roots, **`--dependents`**, and **`--profile`** flags as **`validate`** / **`deploy`** (including `-f` with trailing names or `path svc [svc…]`). With **no** extra service arguments, **`diff`**, **`ps`**, and **`explain`** use the **full** profile-active set for the **current** `--profile` args (same default as before for **`diff`**). Re-pass the deploy-time `--profile` set so profile-gated services stay in that set; otherwise their containers can appear as unexpected. **`teardown` / `down`** with no extra names perform a **full** project teardown (all labelled containers). **Receipt pair** diff (`podbay diff receipt-a.json receipt-b.json`) still compares two decoded receipts only.
- With **`--json`**, `validate` / `deploy` / **`diff`** responses may include an additive **`deploy_services`** field listing explicit roots you passed on the CLI, and **`dependents_expand`** when partial roots are combined with **`--dependents`** (same shape on **`diff`**, **`ps`**, **`explain`**, and **`teardown` / `down`** as on validate/deploy). **`ps`** and **`explain`** JSON include the same fields when partial roots apply.

---

## Related docs

- [architecture.md](architecture.md) — agent loop diagram and unified semantics
- [cli-json.md](cli-json.md) — JSON envelopes and exit codes
- [contract.md](contract.md) — `podbay.yaml` reference
- [glossary.md](glossary.md) — partial selection terminology
