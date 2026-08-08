# Podbay contract reference

Canonical reference for `podbay.yaml`: schema, networks, profiles, host env substitution, and Compose import. For JSON output see [cli-json.md](cli-json.md). For automation patterns see [agent-loop.md](agent-loop.md). Terminology: [glossary.md](glossary.md).

---

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

Create a **greenfield** starter with:

```bash
podbay init
```

For a **brownfield** repo, prefer one-command adoption:

```bash
podbay init --from-codebase
# or: podbay init --from-codebase /path/to/repo
# or: podbay init --from-codebase --compose path/to/compose.yaml
# or: podbay init --from-codebase --dockerfile path/to/Dockerfile
podbay onboard -f podbay.yaml --json
podbay validate -f podbay.yaml
```

`--from-codebase` discovers the first well-known Compose file (`compose.yaml`, `compose.yml`, `docker-compose.yaml`, `docker-compose.yml`) and runs the same import pipeline as `import compose`. If no Compose file is found, it falls back to a well-known Dockerfile (`Dockerfile`, then `dockerfile`) and writes a **single-service build stub** (`build.context` / `build.dockerfile` plus an image tag). It prints onboard / validate next steps and refuses to overwrite an existing contract. Use **`--compose`** or **`--dockerfile`** to override (mutually exclusive). Use **`--json`** for a machine-readable `kind: init` outcome (`source_kind`, `compose_source` or `dockerfile_source`, `service_count`, `next_actions`, stable `issues[].code` on failure — including `codebase_discovery_not_found` when neither source exists).

Treat the result as a **first-pass** contract (validate / hand-tighten)—not full Compose parity, not language/package-manager scanning, and not magic.

---

## Import from Compose

Podbay can also generate a first-pass contract from an **explicit** Compose file path (same translator as `init --from-codebase`):

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

**Trust:** Included files are part of the deployment contract surface—review them like the primary Compose file ([`SECURITY.md`](../SECURITY.md)).

Import intentionally rejects or requires manual adjustment for features Podbay does not claim to support yet, including non-bridge drivers on **internal** networks, Compose IPAM blocks, `build` without `image`, unsupported `depends_on` conditions, ephemeral published ports, Swarm-only `deploy`, Swarm secret/config drivers, and arbitrary Compose extensions beyond the documented import parity notes.

Treat import as a migration assistant, not magic. The winning loop is:

```bash
podbay import compose docker-compose.yml -o podbay.yaml
podbay validate -f podbay.yaml
# tighten the contract by hand
podbay deploy -f podbay.yaml --receipt .podbay/receipts/demo/
podbay receipt list .podbay/receipts/demo
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

### Command overview

| Command | Purpose |
| --- | --- |
| `podbay init` | Create a greenfield starter `podbay.yaml` (nginx template). |
| `podbay init --from-codebase [dir]` | Discover Compose (preferred) or Dockerfile under `dir` (default `.`) and write a first-pass `podbay.yaml`. Optional **`--compose`** / **`--dockerfile`**. Use **`--json`** for `kind: init` success/failure. |
| `podbay import compose <file>` | Convert an explicit Compose file into a first-pass Podbay contract. Use **`--json`** for versioned JSON on stdout (**success** or **failure**; stable **`issues[].code`** on failure). |
| `podbay validate` | Load the contract and run preflight checks. Optional **service names** after the contract path (or after `-f`) select **explicit targets** within the `--profile` active set; by default the checked set is **exactly** those names (no automatic parent pull). Pass **`--dependents`** to include the **transitive closure** of profile-active services that **`depends_on`** any service already in the working set. **`dependents`** on a service **P** must list every profile-active child that **`depends_on` P**, and every **`dependents`** entry must **`depends_on`** its parent—validate fails if either direction is wrong. **`depends_on`** still defines startup order and, for edges **outside** the active set, **`podbay deploy`** **pre-waits** on existing containers without redeploying them. Use **`--json`** for CI/agents. |
| `podbay deploy` | Same selection rules and **`--dependents`** flag as **`validate`**, then build/start that subset. Receipt lists deployed services only. |
| `podbay receipt <file>` | Read and validate a deploy receipt (human summary includes `deploy_id` / `contract_digest` / `status` / `failure` when present). |
| `podbay receipt list <dir>` | Inventory receipt JSON files in a directory (newest first; `--status ok|failed`; `--json` → `kind: receipt_list`). |
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

Do **not** pass a contract path as a positional argument when you already set `-f` / `--file` for the contract (same rule as before). With `-f`, any **additional** positionals are **service names** for partial validate/deploy (see [agent-loop.md](agent-loop.md)).

---

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

---

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

---

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

---

## Profiles

Profiles behave like Docker Compose profiles:

- Services without `profiles:` are active by default.
- Services with `profiles:` are active only when a matching `--profile` is passed.

Example:

```bash
podbay validate -f examples/flowboard --profile observability
podbay deploy   -f examples/flowboard --profile observability
```

### Re-pass `--profile` on observability commands

`--profile` is **call-time only**. Contract-mode `diff`, `ps`, `explain`, and `logs` do **not** remember which profiles were used at deploy, and containers are not labeled with profile names.

Use the **same** `--profile` set on every command that should see those services:

```bash
podbay deploy  -f . --profile observability
podbay diff    -f . --profile observability
podbay ps      -f . --profile observability
podbay explain -f . --profile observability
```

If you omit `--profile` after a profiled deploy, profile-gated containers still running under the project label are reported as **unexpected** / unknown (for example `podbay_<project>_prometheus` after `deploy --profile observability`). That is expected with the current CLI scope rules—not a sign that `profiles:` in `podbay.yaml` is invalid.

Combine multiple profiles the same way as deploy:

```bash
podbay deploy  --profile crane --profile observability
podbay diff    --profile crane --profile observability
```

Receipt-pair `diff` (`podbay diff before.json after.json`) compares receipt `profiles` fields and rejects `--profile`.

---

## Environment substitution

Host-side `${VAR:-default}` substitution applies to fields such as ports, volumes, environment values, DNS, extra hosts, `user`, Ansible Vault paths, and HTTP health URLs.

By default, Podbay reads host substitution values from:

1. process environment
2. `.env.example` under the contract directory, when present
3. `.env` under the contract directory, when present

Later files override earlier ones. Use `host_env_files` to choose a different set.

This differs from Docker Compose in one important way: Podbay deliberately lets `.env.example` participate in host-side substitution because many agent-built repos commit useful local defaults there. When a value must differ inside a container, use a dedicated variable name. The Flowboard example uses `PODBAY_OLLAMA_BASE_URL` for this reason.

---

## Related docs

- [architecture.md](architecture.md) — import pipeline and package boundaries
- [contract-change-checklist.md](contract-change-checklist.md) — which layers to touch per change type
- [glossary.md](glossary.md) — terminology
- [cli-json.md](cli-json.md) — JSON envelopes
- [agent-loop.md](agent-loop.md) — partial selection and automation
