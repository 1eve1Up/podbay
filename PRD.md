# PRD: Podbay

## Title

**Podbay: Compose for Multi-Agent Software Delivery**
*A Runtime Contract Layer for AI-Built Applications*

## 1. Executive Summary

Podbay defines a new layer in the software stack:

> A runtime contract that makes multi-agent-built systems legible, verifiable, and deployable.

Today:

- Agents write code
- Tools run containers
- Nobody owns runtime truth

Podbay fixes that.

- It is **not** an orchestrator.
- It is **not** a wrapper.

It is the system of record for how an application is supposed to exist at runtime—and whether that is actually true.

## Implementation Status: v2026.5.1 Public Preview

`v2026.5.1` is the current public May 2026 preview release. It is usable for narrow Podman stacks, but the `podbay.yaml` contract is not yet 1.0-stable.

Podbay uses calendar-based release versions. Calendar versions identify releases; they are not a substitute for compatibility promises. Until a future `v2026.x-stable` or `v1.0` commitment:

- **Contract stability:** `podbay.yaml` may evolve between public preview releases.
- **Receipt format stability:** receipts are machine-readable and versioned with `format_version`, but fields may still evolve before a stable commitment.
- **CLI compatibility:** core commands are intended to stay scriptable, especially with `--json`, but flags and output details may still change during public preview.
- **Migration policy:** release notes will call out breaking changes and provide migration guidance when contract, receipt, or CLI behavior changes.

Shipped in public preview through `v2026.5.1`:

- Podman execution backend
- preflight validation
- Compose import for a documented subset
- deploy receipts
- contract-vs-runtime and receipt-vs-receipt diff
- versioned JSON output for automation
- factual runtime inspection through `podbay explain`

Not shipped in public preview through `v2026.5.1`:

- Quadlet/systemd compilation
- causal/root-cause diagnosis in `podbay explain`
- automatic rollback, SBOM/provenance, or cryptographic guarantees from receipts
- built-in multi-agent locking, merge semantics, or workflow coordination
- Kubernetes replacement behavior, cluster scheduling, or a long-running control plane

## Addendum: shipped on `main` after `v2026.5.0` tag

- **Compose `include:` (v1 subset)** during `podbay import compose`: local relative paths only, merged before `extends:`, bounded depth and cycle detection. See [README](README.md) (Import from Compose) and [RELEASES](RELEASES.md).
- **`podbay import compose --json`:** failure-only versioned JSON (`kind: import_compose`, stable `issues[].code` values) for compose read/parse, include cycle/depth/escape/unsupported, and generic contract translation failures; success imports still emit YAML only. See [README](README.md).
- **`podbay validate` / `podbay deploy` by service name:** optional service roots after the contract path (`-f` with trailing names, or `path svc [svc…]`) select explicit targets within the active `--profile` set; by default the effective set is **only** those names. **`--dependents`** expands to the transitive closure of profile-active services that **`depends_on`** any service already in the set. **`dependents:`** must mirror **`depends_on`** in both directions. Deploy pre-waits on existing `depends_on` targets outside the active set (started/healthy) without redeploying them; validate/deploy **`--json`** may list `deploy_services` and `dependents_expand` when applicable. **`podbay diff`**, **`podbay ps`**, and **`podbay explain`** accept the same roots and **`--dependents`** for contract/runtime views (default remains full profile-active when no roots). **Receipt pair** diff (two decoded receipt files) is unchanged. See [README](README.md).
- **`podbay teardown` / `podbay down` partial selection:** same optional service roots and **`--dependents`** as **`validate`** / **`deploy`**; partial mode removes only matching containers, skips project network removal while any project-labelled container remains, and rejects **`--volumes` / `-v`** until you run a full teardown (omit service names). **`--json`** may include **`deploy_services`** and **`dependents_expand`** when partial roots apply. See [README](README.md) and [RELEASES](RELEASES.md).
- **`podbay logs --json`:** one versioned JSON document per invocation (`kind: logs`); success includes captured **`log_body`** from a single non-follow **`podman logs`** run. **`--json`** cannot be combined with **`--follow`**. See [README](README.md) and [RELEASES](RELEASES.md).

## 2. Problem Statement

### The New Reality

Software is increasingly built by multiple agents:

- codegen agents
- test agents
- security agents
- deploy agents
- human reviewers in the loop

But there is no shared contract for:

- What is the system?
- What must be true?
- What depends on what?
- What is healthy vs. merely running?

### Current Tools Fail

- **Docker Compose** → "runs things" but no guarantees
- **Podman** → primitives, no contract layer
- **Kubernetes** → overkill, cluster-first, not agent-native
- **systemd** → lifecycle, but no system-level intent

### Result

Systems that "work" but cannot be trusted, reasoned about, or safely evolved—especially under multi-agent development.

## 3. Vision

Podbay is the **runtime contract layer** for AI-built applications.

It enables:

- Agents to coordinate via shared operational truth
- Systems to be defined by requirements, not just containers
- Deployments to be validated, not assumed
- Operators to understand systems in seconds, not hours

## 4. North Star

> "Show me a repo I've never seen, and in 30 seconds I understand:
>
> - what runs
> - what depends on what
> - what the requirements are
> - whether I should trust it"

If Podbay achieves this, it wins.

## 5. Core Principles

### 5.1 Contract > Configuration

Podbay defines what *the requirements are*, not just what to run.

### 5.2 Systems > Containers

Services are part of a dependency graph, not a flat list.

### 5.3 Validation > Execution

Deployment is meaningless without verification.

### 5.4 Explicit > Implicit

No hidden behavior. No magic defaults that matter.

### 5.5 Agent-Readable First

Humans read it. Agents reason over it.

## 6. Target Users

### Primary

- AI-first developers building agent-driven systems
- DevOps / Platform engineers running single-host or edge workloads
- Founders building "AI appliances" (like Flowboard)

### Secondary

- Enterprise teams standardizing internal tooling
- Red Hat / Podman ecosystem users avoiding Kubernetes overhead

## 7. Core Product

### 7.1 Podbay Contract File

`podbay.yaml`

Defines:

- services
- dependents (inverse of `depends_on`; contract field)
- health checks
- resources
- volumes
- networking
- requirements
- validation gates

### 7.2 Core Capabilities

#### A. System Definition

- Declarative runtime contract
- Service graph (`depends_on`, `dependents`)
- Explicit operational expectations

#### B. Validation Engine (Core Differentiator)

- Pre-deploy validation
- Post-deploy verification
- requirement enforcement
- Failure explanation

#### C. Runtime Translation

Shipped today:

- Podman

Planned / not shipped in public preview through `v2026.5.1`:

- Quadlet / systemd

Design constraint:

- Deterministic mapping (no hidden behavior)

#### D. Explainability Layer

Shipped today: factual runtime inspection.

- "What is running?"
- "What changed?"
- "What is broken?"

Planned / not shipped in public preview through `v2026.5.1`: causal diagnosis.

- "What should I do next?"

## 8. Multi-Agent Context

Podbay acts as **shared truth** across agents.

| Agent | Role |
| --- | --- |
| Builder Agent | Updates contract when runtime changes |
| Reviewer Agent | Ensures code and contract match |
| Test Agent | Derives tests from requirements |
| Deploy Agent | Executes contract deterministically |
| Security Agent | Evaluates exposure and constraints |
| Ops Agent | Uses contract to reason about system state |

## 9. Key Workflows

### 9.1 Define System

```bash
podbay init
```

→ creates baseline contract

### 9.2 Validate Before Deploy

```bash
podbay validate
```

Outputs:

```text
✔ Ports available
✔ Volumes writable
✖ API health check not defined
⚠ Service dependency incomplete
```

### 9.3 Deploy

```bash
podbay deploy
```

- compiles contract
- executes deployment
- runs validation gates

### 9.4 Diagnose

```bash
podbay explain
```

Current `v2026.5.1` behavior: factual runtime state, health probes, dependency context, and unexpected containers. It does not infer root cause.

Future diagnostic direction:

```text
API is running but unhealthy.
Reason:
- Postgres dependency is reachable
- API health endpoint returns 500
- Likely cause: missing migration step
```

### 9.5 Drift Detection

```bash
podbay diff
```

```text
Expected: 3 services
Actual: 4 containers running
Unexpected: debug container
```

## 10. Non-Goals (Critical)

- Not a Kubernetes competitor
- Not a full orchestration system
- Not a CI/CD pipeline
- Not a config management tool
- Not "yet another compose"

## 11. MVP Scope

### Must Have

- `podbay.yaml` schema
- basic service + dependency model
- Podman execution backend
- preflight validation
- health checks
- simple requirement checks
- CLI (`init` / `validate` / `deploy` / `explain`)

### Nice to Have

- Quadlet/systemd generation
- richer requirement DSL
- drift detection
- agent hooks

## 12. Success Metrics

- **Adoption** — % of users replacing Compose in single-host setups
- **Engagement** — frequency of `podbay validate` before deploy
- **Trust Signal** — "I won't deploy without Podbay" sentiment
- **Clarity** — time-to-understand-system < 30 seconds

## 13. Competitive Positioning

| Tool | What it does | Why Podbay wins |
| --- | --- | --- |
| Docker Compose | Runs containers | No validation, no contract |
| Podman | Container runtime | No system abstraction |
| Kubernetes | Full orchestration | Overkill for single-host |
| systemd | Process lifecycle | No system-level intent |

## 14. Risks

1. **"Just another wrapper" perception**
   - *Mitigation:* lead with validation, not deployment
2. **Overengineering**
   - *Mitigation:* enforce strict MVP discipline
3. **Edge-case complexity (SELinux, volumes, networking)**
   - *Mitigation:* explicit validation + diagnostics

## 15. Future Vision

- Standard contract format for agent-built systems
- Integration into IDEs and agent frameworks
- Runtime truth layer across environments
- Potential bridge to cluster environments (optional)

## 16. Final Positioning

> Podbay is where agents agree on reality.

Not:

- how code is written
- how containers are run

But:

- what the ongoing requirements are for the system to be considered correct

## One-line Summary

> Podbay turns "it runs on my machine" into "this system is operationally verifiable."
