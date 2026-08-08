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

See `RELEASES.md` for release notes, known limitations, non-goals, and migration guidance.

Podbay uses calendar-based release versions. Calendar versions identify releases; they are not a substitute for compatibility promises. Until a future `v2026.x-stable` or `v1.0` compatibility commitment, assume the following:

- **Contract stability:** the `podbay.yaml` schema may evolve between public preview releases.
- **Receipt format stability:** receipts are machine-readable and versioned with `format_version`, but fields may still evolve before a stable compatibility commitment.
- **CLI compatibility:** core commands are intended to remain scriptable, especially with `--json`, but flags and output details may still change during public preview.
- **Migration policy:** release notes will call out breaking changes and provide migration guidance when contract, receipt, or CLI behavior changes.

## What Podbay does today

- Defines a stack in `podbay.yaml`: services, builds, images, ports, volumes, networks, profiles, `depends_on` (child→parent startup order and health gates), optional `dependents` (each parent lists every service that **`depends_on`** that parent—validated in both directions), optional **`--dependents`** for transitive downstream partial expansion, health checks, requirements, and Podman-specific parity settings.
- Adopts existing Compose or Dockerfile-only projects with `podbay init --from-codebase` (Compose preferred, then Dockerfile stub → first-pass `podbay.yaml` → orient) or explicit `podbay import compose` for Compose migration.
- Validates before running with `podbay validate`, including dependency graph, port checks, paths, commands, profiles, network rules, and healthy-dependency requirements.
- Deploys with Podman using deterministic names and labels: `podbay_<project>_<service>` plus `podbay.project` / `podbay.service`.
- Writes deploy receipts with `podbay deploy --receipt` (file or directory store): success evidence and health-gate attempt receipts, listable via `podbay receipt list` / `last-ok` / `handoff`.
- Compares contract vs runtime with `podbay diff`, receipt vs receipt with `podbay diff before.json after.json`, or `--vs-last-ok` against a receipt store.
- Explains runtime state with `podbay explain`, including health probes and dependency context.
- Emits versioned JSON for agents and CI on key commands: `validate`, `deploy`, `diff`, `receipt`, `teardown`, `down`, and `logs`.
- Handles practical Podman parity issues that otherwise waste operator and agent time: named volume `:U`, Podman Machine DNS, `host-gateway`, `host.docker.internal` / `host.containers.internal`, network MTU, and health/log failure hints.

## Documentation

| Doc | Contents |
| --- | --- |
| [docs/architecture.md](docs/architecture.md) | Import pipeline, package DAG, agent loop semantics |
| [docs/contract.md](docs/contract.md) | `podbay.yaml` reference, networks, profiles, Compose import |
| [docs/cli-json.md](docs/cli-json.md) | `--json` envelopes, receipts, exit behavior |
| [docs/agent-loop.md](docs/agent-loop.md) | Validate → deploy → diff automation and partial-deploy demos |
| [docs/blog-sprint-38-init-from-codebase.md](docs/blog-sprint-38-init-from-codebase.md) | Sprint 38: `init --from-codebase` adoption blog |
| [docs/blog-sprint-39-dockerfile-from-codebase.md](docs/blog-sprint-39-dockerfile-from-codebase.md) | Sprint 39: Dockerfile-only from-codebase blog |
| [docs/glossary.md](docs/glossary.md) | Terminology across CLI, JSON, and Go |
| [docs/contract-change-checklist.md](docs/contract-change-checklist.md) | Which layers to touch per change type |

Contributors: see [CONTRIBUTING.md](CONTRIBUTING.md) for verification steps and architecture links.

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

Machine-readable gates and partial-deploy demos: [docs/agent-loop.md](docs/agent-loop.md) and [docs/cli-json.md](docs/cli-json.md).

Minimal contract shape and `podbay init`: [docs/contract.md](docs/contract.md).

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

Podbay is an open-source project by [Level Up Labs](https://leveluplabs.ai).

We are building solutions for enterprise-grade AI systems and agent-native software delivery.
