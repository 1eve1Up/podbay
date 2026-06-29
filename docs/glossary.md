# Podbay glossary

Short definitions for terms used across CLI flags, JSON fields, Go packages, and docs. For architecture and package boundaries see [architecture.md](architecture.md). Deep reference: [contract.md](contract.md), [cli-json.md](cli-json.md), [agent-loop.md](agent-loop.md).

---

## Partial selection

| Term | Meaning |
| --- | --- |
| **Service roots** | Optional service names passed after the contract path (or after `-f`) on validate, deploy, diff, ps, explain, logs, teardown. Select explicit targets within the profile-active set. |
| **`--dependents`** | CLI flag. When combined with service roots, expands the working set to the transitive downstream closure: every profile-active service that **`depends_on`** any service already in the set. |
| **`deploy_services`** | JSON field on validate, deploy, diff, ps, explain, teardown when partial roots were passed. Lists the explicit service names from the CLI. |
| **`dependents_expand`** | JSON field when partial roots were combined with **`--dependents`**. Indicates downstream expansion was applied. |
| **Profile-active set** | Services that match the active **`--profile`** (or default profile rules). Partial selection always operates within this set. |
| **`ObservabilityActiveServices`** | Go helper in `internal/spec` — single implementation of partial service selection for validate, deploy, diff, ps, explain, logs, and teardown. |

Without service roots, validate, deploy, diff, ps, and explain use the **full profile-active set** (teardown/down with no roots removes all project containers).

---

## Graph and dependents

| Term | Meaning |
| --- | --- |
| **`depends_on`** | Child→parent edges in `podbay.yaml`. Defines startup order and health gates (`service_started` / `service_healthy`). |
| **`dependents`** | YAML field on a **parent** service listing every profile-active child that **`depends_on`** that parent. Must mirror **`depends_on`** in both directions; validate rejects missing or stray entries. |
| **`RedeployPeers`** | Go field name on `spec.Service` for the YAML **`dependents`** key (historical name; YAML authors use **`dependents`**). |
| **`ExpandDependentsTransitive`** | Go helper that walks **`depends_on`** downstream from explicit roots within the profile-active map. |
| **Bidirectional validation** | Every **`depends_on`** edge must appear on the parent's **`dependents`** list, and every **`dependents`** entry must **`depends_on`** its parent. |

Partial deploy **pre-waits** on **`depends_on`** targets outside the active set (existing containers must be started/healthy) without redeploying them.

---

## Health probes

| Term | Meaning |
| --- | --- |
| **`health.timeout`** | Contract per-try probe deadline for **deploy** health gates (may be longer during cold start / `start_period`). |
| **Explain probe budget** | **`podbay explain`** runs **single-shot** HTTP/exec probes capped at **5s per probe** (shorter than deploy when contract timeout is large). See [agent-loop.md](agent-loop.md) failure playbook. |

---

## Host expansion

| Term | Meaning |
| --- | --- |
| **`LoadHostSubst`** | Builds the `${VAR}` substitution map: process environment, then contract-directory env files (default `.env.example` then `.env`, or **`host_env_files`**). |
| **`ExpandService`** | Applies host substitution to runtime-facing service fields (ports, volumes, env, health URLs, etc.) before validate checks or deploy/run. Shared across validate, deploy, and explain. |
| **Host-side substitution** | Expansion using the host env map. Diff and ps do not expand service fields for comparison. |

---

## Import pipeline (three phases)

These are **intentional layers**, not duplicate models. See [architecture.md](architecture.md).

| Phase | Package | Question |
| --- | --- | --- |
| **Foreign input** | `internal/composefile` | What did upstream Compose say? |
| **Operational contract** | `internal/spec` | What should run? Is it valid? Did it drift? |
| **Migration output** | `internal/composeimport` (`emit_types`) | What first-pass `podbay.yaml` should the agent commit? |

Runtime commands use **`spec`** only. **`import compose`** uses composefile → translate → spec → emit.

---

## JSON vocabulary

| Term | Meaning |
| --- | --- |
| **`format_version`** | Integer schema version on versioned JSON documents (currently **1**). |
| **`kind`** | Discriminator: `validate`, `deploy`, `diff`, `receipt_read`, `teardown`, `import_compose`, `logs`, etc. |
| **`status`** | Usually **`ok`** or **`failed`**. |
| **`issues[]`** | Structured failures with **`code`**, **`level`**, **`message`**, optional **`service`**. |
| **`issues[].code`** | Stable machine identifier (e.g. `deploy_health_timeout`, `import_include_cycle`). See [cli-json.md](cli-json.md). |
| **`contract_path`** | Absolute path to the contract file in most kinds; for **`import_compose`**, names the **Compose source** file, not output `podbay.yaml`. |
| **`contract_yaml`** | On successful **`import_compose --json`**, the generated Podbay contract as a UTF-8 string. |
| **`drift`** | On **`diff --json`**, **`true`** when runtime does not match the contract. |

---

## Related docs

- [architecture.md](architecture.md) — package DAG and agent loop
- [contract-change-checklist.md](contract-change-checklist.md) — which layers to touch per change type
- [contract.md](contract.md) — `podbay.yaml` reference
- [cli-json.md](cli-json.md) — `--json` envelopes and exit behavior
- [agent-loop.md](agent-loop.md) — validate → deploy → diff automation
