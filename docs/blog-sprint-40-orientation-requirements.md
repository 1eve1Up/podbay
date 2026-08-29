# Sprint 40: Orientation can name the requirements

*2026-08-29 — Podbay public preview*

Show me a repo I've never seen, and in 30 seconds I understand what runs, what depends on what, **what the requirements are**, and whether I should trust it. Sprint 39 opened Dockerfile arrive. Sprint 40 makes that arrive surface speak the third clause.

---

## The hole after Dockerfile from-codebase

`podbay init --from-codebase` already wrote a first-pass `podbay.yaml` from Compose or a Dockerfile and pointed at onboard / validate. `onboard --json` answered what runs and what depends on what. It did not name ports, expose, health, or build vs image. Dockerfile stubs were build-only, so the YAML was empty too—and validate stayed `ok` because the health warn fires only when host ports exist.

Hand-tighten was folklore. Agents had to read YAML (or invent it).

---

## What we shipped

### Requirements on the orientation graph

`onboard --json` and the additive `orientation` block on `explain --json` now skim, per in-scope service:

- published `ports`
- `expose`
- `health` (`http` / `exec` / omitted when absent)
- `source` (`build` or `image`)

Compose from-codebase and greenfield nginx light up immediately. No second YAML read.

### Declared Dockerfile instructions, not invented binds

When `--from-codebase` takes the Dockerfile path:

- `EXPOSE` → contract `expose:`
- `HEALTHCHECK` → `health.exec` (same CMD / CMD-SHELL / NONE argv rules as Compose import)

It does **not** invent `80:80` from `EXPOSE 80`. Published host ports stay hand-tighten.

### Gaps are a field list

Dockerfile `kind: init` success JSON lists `extracted` vs `gaps` (`expose`, `health`, `published_ports`). When ports or health are still missing, next actions still point at onboard / validate and add a hand-tighten hint. Init does not fail. Validate rules for worker-shaped services stay unchanged.

```bash
podbay init --from-codebase
podbay onboard -f podbay.yaml --json
podbay validate -f podbay.yaml --json
```

Offline demo: [`examples/ci-dockerfile-from-codebase-demo.sh`](../examples/ci-dockerfile-from-codebase-demo.sh) (bare stub gaps + EXPOSE/HEALTHCHECK skim).

---

## What this is not

- Not language or package-manager scanning.
- Not inventing published ports from `EXPOSE`.
- Not a global validate fail on services without ports or health.
- Not auto-deploy or diagnosis.

---

## Why this sits on the critical path

```text
init --from-codebase → onboard → validate → deploy → receipt / handoff → explain
```

Arrive was open. The 30-second North Star still failed on “what the requirements are.” Naming them on the existing orientation dialect beats polishing a loop whose first JSON document is empty.

---

## What's next

Still deferred: broader codebase heuristics, env/contract cache, parallel / `--no-probes` explain, causal diagnosis in `explain`. The front door names requirements; the house can get faster or diagnostic later.

Docs: [agent-loop.md](agent-loop.md), [contract.md](contract.md), [cli-json.md](cli-json.md).
