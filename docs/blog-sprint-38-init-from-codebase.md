# Sprint 38: One command from a Compose repo to a Podbay contract

*2026-08-07 — Podbay public preview*

Show me a repo I've never seen. In thirty seconds I want a contract I can validate—not a scavenger hunt through `import compose` folklore.

Sprint 38 ships that front door: **`podbay init --from-codebase`**.

---

## The hole after orientation

Sprint 37 gave agents and humans a shared **orientation** surface: `podbay onboard --json`, additive orientation on `explain`, and init next steps pointing at onboard / validate. That answers “what is this stack and what should I do next?”—**once a `podbay.yaml` exists**.

Brownfield reality was uglier:

1. Greenfield `podbay init` still wrote a fixed nginx stub.
2. Migration meant remembering `podbay import compose … -o podbay.yaml`.
3. Until something wrote the contract, orientation could not fire.

The PRD measures **adoption**—percent of users replacing Compose in single-host setups. Speeding up validate→deploy with an env cache does not fix a missing front door. We skipped the door-knob polish and shipped the door.

---

## What we shipped

### Discover, don't invent

`composefile.Discover` looks for well-known Compose filenames in a documented order:

1. `compose.yaml`
2. `compose.yml`
3. `docker-compose.yaml`
4. `docker-compose.yml`

Pass **`--compose <path>`** when the repo uses a non-standard name. No package.json scanning, no Dockerfile heuristics, no LLM codegen this sprint—Compose-first on purpose.

### One command, same translator

```bash
podbay init --from-codebase
# or: podbay init --from-codebase /path/to/repo
# or: podbay init --from-codebase --compose path/to/compose.yaml -f podbay.yaml
```

Under the hood this is the **same** pipeline as `podbay import compose`: composefile → composeimport → emit. We did not fork a second dialect. The file is written, overwrite is refused, and next steps point at the Sprint 37 dialect:

```text
podbay onboard -f podbay.yaml --json
podbay validate -f podbay.yaml --json
```

Bare `podbay init` (no flags) still creates the nginx starter. Greenfield and brownfield stay distinct.

### Agents get JSON, not prose

```bash
podbay init --from-codebase --json
```

Success is `kind: init` with `contract_path`, `compose_source`, `service_count`, and ordered `next_actions`. Failures use stable codes—among them `compose_discovery_not_found`, `init_target_exists`, and the existing import load/translate codes—so CI and agents do not scrape stderr stories.

### Proof you can run offline

```bash
go build -o ./podbay ./cmd/podbay
PODBAY_BIN=./podbay ./examples/ci-from-codebase-demo.sh
```

That demo copies the in-repo Compose include fixture, runs from-codebase → onboard → validate, then asserts overwrite fail-closed. No Podman required.

---

## What this is not

- **Not magic.** First-pass contracts still need `validate` and hand-tighten. Import docs said that; from-codebase packages discovery, it does not claim full Compose parity.
- **Not “understand any repo.”** No language or package-manager heuristics yet. Dockerfile-only stubs are a natural follow-on.
- **Not diagnosis or auto-deploy.** Next steps are onboard / validate—the same rule-based gates as orientation and handoff.

`podbay import compose` remains the explicit migration tool when you already know the Compose path.

---

## Why this sits on the critical path

Receipts 2.0 (Sprints 34–36) made deploys evidence, history, and intelligence. Orientation (Sprint 37) made arrive/mid-loop context callable. Adoption (Sprint 38) makes the **first contract** callable from the repo you already have.

The loop agents actually want:

```text
init --from-codebase → onboard → validate → deploy → receipt / handoff → explain
```

Without the first arrow, everything else is a product for people who already converted.

---

## Try it

```bash
# In a directory that has docker-compose.yml (or compose.yaml, …)
podbay init --from-codebase --json
podbay onboard -f podbay.yaml --json
podbay validate -f podbay.yaml --json
```

Docs: [agent-loop.md](agent-loop.md) (brownfield arrive), [contract.md](contract.md) (init / import), [cli-json.md](cli-json.md) (`kind: init`).

---

## What's next

Still deferred: Dockerfile-only / broader codebase stubs, env/contract cache, parallel / `--no-probes` explain, richer receipt timelines. The front door is open; the house can get faster and wider later.
