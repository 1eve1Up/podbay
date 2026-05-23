# Podbay

Podbay is a **runtime contract layer for Podman-based application stacks**.

Podbay is intentionally shaped like Compose where that helps, but it is not trying to reinvent Docker Compose, `podman compose`, or Kubernetes. 

Podbay is for the awkward but increasingly common middle ground of the GenAI era: whether for dev integration environments, or even for production-grade, all-in-one, "appliance-like" AI agent solutions, and especially when you have a monorepo which contains a small multi-container app, with one or more code agents that are continually changing things, and a human or CI job eventually needs to run all of that on a real host or small HA cluster at the end of the pipeline... then everyone involved benefits from having a machine-readable answer to:

> What is supposed to run, what actually ran, what changed, and should I trust this deployment?

Podbay seeks to answer that with `podbay.yaml`, validation, deterministic Podman execution, health/dependency gates, JSON output, deploy receipts, drift detection, and explainable runtime state.

## Why does Podbay need to exist?

Because the existing container orchestration projects below are all amazing at what they respectively do, but were built for a different era, and are presently optimizing for different goals.

| Tool | Great at | Gap (What Podbay seeks to close) |
| --- | --- | --- |
| Docker | Building and running containers | Does not describe a whole app contract, dependency health, drift, receipts, or agent-friendly deploy truth. |
| Docker Compose | Local multi-container developer workflows | Great app shape, but Docker-centered and not designed as an auditable runtime contract for agents/CI/operators. |
| Podman | Daemonless/rootless container runtime | Powerful primitive, but raw `podman run` leaves app-level intent scattered across scripts, docs, LLM context windows, and fragile institutional memory. |
| `podman compose` | Running Compose-ish files with Podman | A partial compatibility "bridge" for those who want Docker Compose-like features in Podman, but not a first-class contract, receipt, drift, and explanation model. |
| Kubernetes | Cluster orchestration at scale | Too much machinery when the target is one host, a demo appliance, or an MVP HA stack. Yes, projects like Minikube and OpenShift Local minify K8S for dev environments, but they still inherit a number of implementation surfaces that can challenge both agents and humans alike in deploying to them consistently. |

Podbay is certainly not intended to be “a simpler Kubernetes.” Instead, it is being built as **a missing contract layer between Compose-shaped app intent and Podman-shaped runtime reality**.

## The one-sentence pitch

**Podbay gives humans, agents, and CI the same operational truth for small-to-medium container stacks, without forcing the project into choosing Kubernetes before it needs Kubernetes.**

## Release and stability

**Version:** `v2026.5.1`  
**Stability:** public preview  
**Contract status:** evolving  
**Receipt format:** versioned  
**Production claim:** suitable for narrow Podman stacks, not a Kubernetes replacement.

`v2026.5.1` is the current public May 2026 preview release (`v2026.5.0` was the first). It is usable, but the `podbay.yaml` contract is not yet 1.0-stable.

See `RELEASES.md` for release notes, known limitations, non-goals, and migration guidance.

Podbay uses calendar-based release versions. Calendar versions identify releases; they are not a substitute for compatibility promises. Until a future `v2026.x-stable` or `v1.0` compatibility commitment, assume the following:

- **Contract stability:** the `podbay.yaml` schema may evolve between public preview releases.
- **Receipt format stability:** receipts are machine-readable and versioned with `format_version`, but fields may still evolve before a stable compatibility commitment.
- **CLI compatibility:** core commands are intended to remain scriptable, especially with `--json`, but flags and output details may still change during public preview.
- **Migration policy:** release notes will call out breaking changes and provide migration guidance when contract, receipt, or CLI behavior changes.

## What Podbay does today

- Defines a stack in `podbay.yaml`: services, builds, images, ports, volumes, networks, profiles, `depends_on` (child→parent startup order and health gates), optional `dependents` (each parent lists every service that **`depends_on`** that parent—validated in both directions), optional **`--dependents`** for transitive downstream partial expansion, health checks, requirements, and Podman-specific parity settings.
- Imports many Compose files with `podbay import compose` so existing projects can migrate incrementally.
- Validates before running with `podbay validate`, including dependency graph, port checks, paths, commands, profiles, network rules, and healthy-dependency requirements.
- Deploys with Podman using deterministic names and labels: `podbay_<project>_<service>` plus `podbay.project` / `podbay.service`.
- Writes deploy receipts with `podbay deploy --receipt receipt.json` so a deployment has a durable machine-readable artifact.
- Compares contract vs runtime with `podbay diff`, or receipt vs receipt with `podbay diff before.json after.json`.
- Explains runtime state with `podbay explain`, including health probes and dependency context.
- Emits versioned JSON for agents and CI on key commands: `validate`, `deploy`, `diff`, `receipt`, `teardown`, `down`, and `logs`.
- Handles practical Podman parity issues that otherwise waste operator and agent time: named volume `:U`, Podman Machine DNS, `host-gateway`, `host.docker.internal` / `host.containers.internal`, network MTU, and health/log failure hints.

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

## Quick start

Requirements:

- Go 1.22+
- Podman on `PATH`
- `curl` on `PATH` for HTTP health checks and `explain`

Install from a clone:

```bash
cd /path/to/podbay
go install ./cmd/podbay
export PATH="$(go env GOPATH)/bin:$PATH"
```

Verify:

```bash
podbay version
podbay --help
```

Run the nginx example:

```bash
podbay validate -f examples/nginx
podbay deploy   -f examples/nginx --receipt /tmp/podbay-nginx-receipt.json
podbay diff     -f examples/nginx
podbay ps       -f examples/nginx
podbay explain  -f examples/nginx
podbay logs     -f examples/nginx web
podbay down     -f examples/nginx
```

Machine-readable form:

```bash
podbay validate -f examples/nginx --json
podbay deploy   -f examples/nginx --receipt /tmp/podbay-nginx-receipt.json --json
podbay diff     -f examples/nginx --json
podbay receipt  /tmp/podbay-nginx-receipt.json --json
podbay down     -f examples/nginx --json
podbay logs     -f examples/nginx --json web
# partial deploy evidence (same roots as deploy/diff):
# podbay logs -f examples/two-service web --json
```

## Minimal `podbay.yaml`

```yaml
version: "1"
project: myapp

requirements:
  - type: command_exists
    command: podman

services:
  web:
    image: docker.io/library/nginx:alpine
    ports:
      - "8080:80"
    health:
      http:
        url: http://127.0.0.1:8080/
        timeout: 15s
    requirements:
      - type: port_available
        port: 8080

volumes: {}
networks: {}
```

Create one with:

```bash
podbay init
```

## Import from Compose

Podbay can generate a first-pass contract from an existing Compose file:

```bash
podbay import compose docker-compose.yml -o podbay.yaml
podbay validate -f podbay.yaml
```

Import supports common Compose patterns including `services`, `image`, `build` with an image tag, ports, volumes, environment, `env_file`, `depends_on`, healthchecks, profiles, restart, labels, `extra_hosts`, `user`, DNS, command, internal bridge networks, **scoped `external:` networks** (see [Networks](#networks)), file-backed configs/secrets, Ansible Vault-backed secret materialization, Compose `extends:` for a documented subset, and **Compose `include:`** for a **narrow local-file subset** (see below).

### Compose `include:` (import v1 subset)

`podbay import compose` resolves top-level **`include:`** before `extends:` and before translation to `podbay.yaml`, using the same **compose root directory** as the primary file (the directory of the file you pass to `import compose`).

**Supported**

- **`include`** as a YAML list of **relative paths** (strings), e.g. `include: [ "./common.yml" ]`, or list of mappings **`path: ./common.yml`** only.
- **Nested includes** in included files (each path resolved relative to **that** file’s directory), up to **16** files on the include chain (same cap order of magnitude as `extends:` depth).
- **Merge order:** each included file is merged in list order; **later** list entries **overwrite** earlier ones for the same top-level map keys (`services`, `volumes`, `networks`, `configs`, `secrets`). The **primary** compose file’s top-level keys **overwrite** everything from `include:` (Compose-style: the root file wins on conflict).
- **Path safety:** included paths must stay **under** the primary compose file’s directory after `filepath.Clean` / `filepath.Join` (no `..` escape to siblings of the compose root). **Absolute** include paths are **rejected**.

**Explicitly not supported (import errors)**

- **Remote includes** (`http://`, `https://`, `git@`, …).
- **`env_file`** (or other extra keys) on **`include`** mapping entries—only **`path`** is accepted.
- **Absolute** include paths.

**Trust:** Included files are part of the deployment contract surface—review them like the primary Compose file ([`SECURITY.md`](SECURITY.md)).

Import intentionally rejects or requires manual adjustment for features Podbay does not claim to support yet, including non-bridge drivers on **internal** networks, Compose IPAM blocks, `build` without `image`, unsupported `depends_on` conditions, ephemeral published ports, Swarm-only `deploy`, Swarm secret/config drivers, and arbitrary Compose extensions beyond the documented import parity notes.

Treat import as a migration assistant, not magic. The winning loop is:

```bash
podbay import compose docker-compose.yml -o podbay.yaml
podbay validate -f podbay.yaml
# tighten the contract by hand
podbay deploy -f podbay.yaml --receipt /tmp/receipt.json
podbay diff -f podbay.yaml
```

For **Compose `include:`**, try the in-repo fixture:

```bash
podbay import compose examples/compose-include/docker-compose.yml -o /tmp/podbay-from-include.yaml
podbay validate -f /tmp/podbay-from-include.yaml
```

### `import compose --json` (CI and agents)

Pass **`--json`** to **`podbay import compose`** for a **machine-readable path** aligned with **`validate --json`** / **`diff --json`** (`format_version` **1**).

**Success:** Print **one JSON object** to **stdout** with **`kind`:** **`import_compose`**, **`status`:** **`ok`**, **`contract_path`** set to the **absolute path of the Compose file** you passed as the argument (for this kind the field names the Compose source, not a `podbay.yaml`), **`contract_yaml`** containing the full generated Podbay contract as a UTF-8 string, **`service_count`** (number of services in the contract), and **`project`** when the contract sets it. If **`-o` / `--output`** is set, the same YAML bytes are written to that file **first**, then **`output_path`** in the JSON is the absolute output path. Exit code **`0`**. No raw YAML is written to stdout when **`--json`** is set (only JSON).

**Failure:** Print **one JSON object** to **stdout** with **`kind`:** **`import_compose`**, **`status`:** **`failed`**, **`contract_path`** set to the **absolute path of the compose file** you passed as the argument (for this kind the field names the Compose source, not a `podbay.yaml`), and **`issues`** with at least one fail-level row containing stable **`code`** and **`message`**. Exit code **`1`**.

**Stable `code` values (v1):**

| `code` | When |
| --- | --- |
| **`import_compose_file_not_found`** | Primary compose path does not exist (`ENOENT`). |
| **`import_compose_read`** | Other OS read errors on the compose file or an included file. |
| **`import_compose_parse`** | YAML/Compose decode failed. |
| **`import_include_cycle`** | **`include:`** graph would cycle. |
| **`import_include_depth`** | **`include:`** chain exceeds the documented depth cap. |
| **`import_include_path_escape`** | Included path escapes the primary compose file directory. |
| **`import_include_unsupported`** | Remote **`include`**, absolute include path, or unsupported **`include`** mapping keys (e.g. **`env_file`** on the include entry). |
| **`import_contract_error`** | Compose loaded but translation or encode to **`podbay.yaml`** failed; **`message`** is actionable. |

For automation, parse **stdout** for the JSON document; a human-oriented **`Error:`** line may appear on **stderr** from the CLI wrapper.

Example (expect exit **1** and **`import_include_cycle`** when pointed at a two-file cycle):

```bash
podbay import compose /path/to/cyclic-root.yml --json
```

| Command | Purpose |
| --- | --- |
| `podbay init` | Create a starter `podbay.yaml`. |
| `podbay import compose <file>` | Convert a Compose file into a first-pass Podbay contract. Use **`--json`** for versioned JSON on stdout (**success** or **failure**; stable **`issues[].code`** on failure). |
| `podbay validate` | Load the contract and run preflight checks. Optional **service names** after the contract path (or after `-f`) select **explicit targets** within the `--profile` active set; by default the checked set is **exactly** those names (no automatic parent pull). Pass **`--dependents`** to include the **transitive closure** of profile-active services that **`depends_on`** any service already in the working set. **`dependents`** on a service **P** must list every profile-active child that **`depends_on` P**, and every **`dependents`** entry must **`depends_on`** its parent—validate fails if either direction is wrong. **`depends_on`** still defines startup order and, for edges **outside** the active set, **`podbay deploy`** **pre-waits** on existing containers without redeploying them. Use **`--json`** for CI/agents. |
| `podbay deploy` | Same selection rules and **`--dependents`** flag as **`validate`**, then build/start that subset. Receipt lists deployed services only. |
| `podbay receipt <file>` | Read and validate a deploy receipt. |
| `podbay diff` | Compare contract vs Podman runtime, or compare two deploy receipts (two args that both decode as receipts). Optional service roots + **`--dependents`** match **`validate`** / **`deploy`**. |
| `podbay ps` | Show resolved services and container state; same partial roots + **`--dependents`** as **`validate`**. |
| `podbay logs [service ...]` | Show logs for resolved service containers; optional partial roots and **`--dependents`** (same rules as **`validate`** / **`deploy`**). With no service names, uses the full profile-active set. **`--json`** returns **`log_entries[]`**; one resolved service also sets top-level **`service`** / **`log_body`**. **`--follow`** only when exactly one service resolves. |
| `podbay explain` | Summarize expected vs actual runtime; same partial roots + **`--dependents`** as **`validate`**. Dependency JSON/text context appears when partial selection narrows to **one** service. |
| `podbay teardown` / `podbay down` | Remove containers; optional **`--profile`**, optional **service roots** after the contract path (or after `-f`), and **`--dependents`** — same resolution rules as **`validate`** / **`deploy`**. With **no** service roots, removes **all** project-labeled containers (full teardown). **Partial** teardown removes only matching containers, skips project **network** removal while any labelled container remains, and **rejects `--volumes` / `-v`** (use a full teardown to remove named volumes). **`--json`** may include **`deploy_services`** and **`dependents_expand`** when partial roots apply. |
| `podbay version` | Print build metadata and host-gateway behavior. |

Most commands look for `podbay.yaml` in the current directory. Use `-f` / `--file` or pass a contract path:

```bash
podbay validate
podbay validate -f /path/to/project
podbay validate /path/to/project/podbay.yaml
```

Do **not** pass a contract path as a positional argument when you already set `-f` / `--file` for the contract (same rule as before). With `-f`, any **additional** positionals are **service names** for partial validate/deploy (see below).

### Partial validate and deploy (and matching `diff` / `ps` / `explain`)

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
- **`podbay diff`**, **`podbay ps`**, **`podbay explain`**, **`podbay teardown`**, and **`podbay down`** accept the same optional service roots and **`--dependents`** flag as **`validate`** / **`deploy`** (including `-f` with trailing names or `path svc [svc…]`). With **no** extra service arguments, **`diff`**, **`ps`**, and **`explain`** use the **full** profile-active set (same default as before for **`diff`**). **`teardown` / `down`** with no extra names perform a **full** project teardown (all labelled containers). **Receipt pair** diff (`podbay diff receipt-a.json receipt-b.json`) still compares two decoded receipts only.
- With **`--json`**, `validate` / `deploy` / **`diff`** responses may include an additive **`deploy_services`** field listing explicit roots you passed on the CLI, and **`dependents_expand`** when partial roots are combined with **`--dependents`** (same shape on **`diff`**, **`ps`**, **`explain`**, and **`teardown` / `down`** as on validate/deploy). **`ps`** and **`explain`** JSON include the same fields when partial roots apply.

## JSON output and receipts

Podbay’s JSON output is designed for tools, agents, and CI. Versioned documents include:

- `format_version`
- `kind`, such as `validate`, `deploy`, `diff`, `receipt_read`, or `teardown`
- `status`, usually `ok` or `failed`
- `issues[]`, with stable-ish codes, levels, messages, and optional service names
- optional `deploy_services` on **`validate`** / **`deploy`** / **`diff`** / **`ps`** / **`explain`** / **`teardown` / `down`** JSON when you pass explicit service roots on the CLI; optional **`dependents_expand`** when partial roots are combined with **`--dependents`**

**`deploy --json` health-gate failures** (runtime, after containers start) emit structured `issues[]` entries with a **`service`** field and stable codes:

| Code | When |
| --- | --- |
| `deploy_health_timeout` | Health probe deadline exceeded for a service in the deploy set |
| `deploy_health_probe_failed` | HTTP/exec probe failed before timeout |
| `deploy_external_dep_unhealthy` | Partial deploy waited on an external dependency’s health and it failed |
| `deploy_error` | Non-health failures (build, start, volume, unexpected errors) |

Success **`deploy --json`** shape is unchanged (`status: ok`, `receipt_path`, partial-selection fields). Preflight validate failures before deploy still surface validate-style issues, not health-gate codes.

Demo:

```bash
PODBAY_BIN=./podbay ./examples/ci-deploy-health-fail-demo.sh
```

Example CI gate:

```bash
set -euo pipefail

podbay validate -f examples/nginx --json
podbay deploy   -f examples/nginx --receipt /tmp/receipt.json --json
podbay diff     -f examples/nginx --json | jq -e '.drift == false'
podbay receipt  /tmp/receipt.json --json
```

The same flow is available as a runnable demo:

```bash
go build -o ./podbay ./cmd/podbay
PODBAY_BIN=./podbay ./examples/ci-receipt-demo.sh
```

In CI, `validate --json` is the preflight gate, `deploy --receipt` writes the evidence artifact, and `diff --json | jq -e '.drift == false'` fails closed when the live Podman runtime no longer matches the contract. If an operator introduces drift by starting or changing containers outside Podbay, the `jq` gate exits non-zero instead of asking the caller to scrape logs.

Receipt comparison does not need a live Podman runtime:

```bash
podbay diff /tmp/receipt-before.json /tmp/receipt-after.json --json
```

That makes receipts useful as deployment evidence, release artifacts, drift gates, and agent handoff objects.

## Contract shape

Top-level fields:

- `version`: schema version, currently `"1"`.
- `project`: project name used for labels, container names, volumes, and networks.
- `host_env_files`: optional files for host-side `${VAR:-default}` substitution. Defaults to `.env.example` then `.env` when present.
- `requirements`: contract-wide checks such as `command_exists`, `port_available`, and writable paths.
- `services`: service definitions.
- `volumes`: named Podman volumes.
- `networks`: optional logical networks (internal Podbay-managed bridges and/or external joins); see [Networks](#networks).
- `network`: options for created project bridge networks, currently including `mtu`.
- `podman`: Podman-specific parity settings.

Common service fields:

- `image`
- `build.context` and `build.dockerfile`
- `depends_on`, as a list or Compose-style map with `service_started` / `service_healthy` (startup order and health gates).
- `dependents` (optional): on a **parent** service, lists **dependents** — service names that **`depends_on`** this parent. Validate requires the lists to match **`depends_on`** exactly (no missing or stray entries). Partial validate/deploy uses **explicit targets only** unless you pass **`--dependents`** (transitive downstream within the profile-active map). **`podbay import compose`** derives **`dependents`** where needed so typical imports validate without hand-editing.
- `profiles`
- `ports`
- `expose`
- `volumes`
- `environment`
- `env_file`
- `extra_hosts`
- `user`
- `dns`
- `command`
- `health.http` or `health.exec`
- `requirements`
- `restart`
- `networks`

See `examples/nginx/podbay.yaml` and `examples/flowboard/podbay.yaml` for working contract shapes.

## Networks

Podbay supports two kinds of entries under top-level `networks:`:

| Kind | Contract shape | Deploy behavior | Teardown (`down` / `teardown`) |
| --- | --- | --- | --- |
| **Internal** (default) | `driver: bridge` or omit `driver` | Podbay creates `podbay_<project>_<logical_key>` if missing (same as multi-network internal bridges) | Removes that Podman network |
| **External** | `external: true` | Joins an **existing** Podman network; does **not** create it | **Does not** remove the network |

**External name resolution** (Compose-compatible for this sprint):

- If `name:` is set on the network entry, that string is the Podman network name to join.
- If `name:` is omitted, the logical YAML key (e.g. `edge` in `networks.edge`) is the Podman network name.

**Requirements:**

- Create external networks yourself before `podbay deploy` (e.g. `podman network create mynet`).
- `podbay validate` does not query Podman for external-network state, but a contract that declares `command_exists: podman` under `requirements:` (as the shipped examples do) will still fail closed when Podman is absent. `deploy` fails if an external network is missing.
- Do not set non-bridge `driver:` on external networks (Podbay does not create them; overlay/host drivers are rejected).

**Mixing internal and external** is allowed: internal networks are still project-scoped; external networks use the names above.

**Compose import** maps `external: true` and `external: { name: ... }` into `networks:` on the emitted `podbay.yaml`.

Example hand-written contract:

```yaml
version: "1"
project: myapp
networks:
  app:
  edge:
    external: true
    name: shared_edge
services:
  api:
    image: docker.io/library/alpine:latest
    command: ["sleep", "300"]
    networks:
      - app
  sidecar:
    image: docker.io/library/alpine:latest
    command: ["sleep", "300"]
    networks:
      - edge
```

## Podman parity features

Podbay exists partly because “Compose-shaped app” plus “Podman runtime” still has sharp edges. The current codebase handles several of them directly:

| Problem | Podbay behavior |
| --- | --- |
| Named volumes with non-root containers | Adds Podman `:U` to declared named volumes by default unless `podman.disable_auto_volume_u: true`. |
| Docker Desktop host aliases | If `host.docker.internal` is configured, Podbay also adds `host.containers.internal` with the same target. |
| `host-gateway` on macOS/Windows Podman Machine | Resolves a concrete IP before `podman run`; override with `PODBAY_HOST_GATEWAY_IP`. |
| Podman Machine bridge DNS surprises | Adds default bridge DNS on macOS/Windows unless disabled or overridden. |
| VPN / slirp MTU issues | Supports `network.mtu` on project bridge creation. |
| Compose service DNS expectations | Adds network aliases so service keys resolve on the Podbay network. |
| Health/dependency ordering | Supports `service_started` and `service_healthy`; health gates block only when another active service asks for healthy dependency behavior. |
| Runtime mystery | `explain`, `ps`, `logs`, `diff`, and receipts make actual state inspectable. |

## Profiles

Profiles behave like Docker Compose profiles:

- Services without `profiles:` are active by default.
- Services with `profiles:` are active only when a matching `--profile` is passed.

Example:

```bash
podbay validate -f examples/flowboard --profile observability
podbay deploy   -f examples/flowboard --profile observability
```

## Environment substitution

Host-side `${VAR:-default}` substitution applies to fields such as ports, volumes, environment values, DNS, extra hosts, `user`, Ansible Vault paths, and HTTP health URLs.

By default, Podbay reads host substitution values from:

1. process environment
2. `.env.example` under the contract directory, when present
3. `.env` under the contract directory, when present

Later files override earlier ones. Use `host_env_files` to choose a different set.

This differs from Docker Compose in one important way: Podbay deliberately lets `.env.example` participate in host-side substitution because many agent-built repos commit useful local defaults there. When a value must differ inside a container, use a dedicated variable name. The Flowboard example uses `PODBAY_OLLAMA_BASE_URL` for this reason.

## Exit behavior

Podbay is meant to fail closed in automation:

- `validate` exits non-zero on fail-level validation issues.
- `deploy` exits non-zero on validation or runtime failure and does not write a partial receipt.
- `diff` exits non-zero when drift is detected or comparison cannot complete.
- `teardown` / `down` remove what they can and report structured issues in JSON; network removal warnings are non-fatal.
- `logs --json` exits **0** on success (including empty per-entry `log_body` values) and **1** on contract load/resolution errors, Podman unavailability, `podman logs` failure, **`--json` with `--follow`**, or **`--follow`** with multiple resolved services. Success may include **`log_entries[]`**, **`deploy_services`**, and **`dependents_expand`** when partial roots apply.

Use `--json` when the caller is a script, CI job, or code agent.

## Development

Public Podbay checkouts run the root Go checks through this script.

For a quick Go-only loop:

```bash
gofmt -l .
go vet ./...
go test ./...
```

Build a local binary:

```bash
go build -o podbay ./cmd/podbay
./podbay validate -f examples/nginx
```

This repository also contains a `.devcontainer` for editing and testing the Go toolchain. That is separate from Podbay itself. Podbay targets Podman at runtime; the dev container may use Docker Desktop or Podman depending on your editor setup.

## Current non-goals

Podbay is deliberately not:

- A Kubernetes competitor.
- A full scheduler.
- A long-running control plane.
- A secret manager.
- A universal Compose implementation.
- A replacement for Podman.

The bet is narrower and sharper: **make small operational stacks legible, deployable, inspectable, and accountable in a world where humans and agents are both continually modifying code and then deploying to prod.**

## Disclaimer

Podbay is provided **as-is**, without warranties or guarantees of any kind.

Podbay orchestrates real container workloads and may:

- start, stop, replace, or remove containers
- modify runtime state on a host
- expose or consume network ports
- mount local filesystems and volumes
- execute container build and runtime operations through Podman

You are responsible for:

- reviewing all `podbay.yaml` contracts
- validating deployment behavior
- securing hosts, networks, secrets, and mounted data
- testing workloads before production use
- complying with your organization’s operational and security requirements

Podbay is a deployment tool, not a managed platform or safety system.

No guarantee is made regarding:

- uptime
- security
- correctness
- fitness for a particular purpose
- compatibility across Podman or operating system versions
- prevention of data loss or service interruption

Always review deploy plans, receipts, diffs, and runtime behavior before relying on Podbay in production environments.

Use at your own risk.

## Maintained by Level Up Labs

Podbay is an open-source project by [Level Up Labs](https://levelupla.io).

We are building solutions for enterprise-grade AI systems and agent-native software delivery.