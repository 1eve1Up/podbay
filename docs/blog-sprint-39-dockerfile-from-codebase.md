# Sprint 39: Dockerfile-only repos can arrive too

*2026-08-08 — Podbay public preview*

Show me a repo I've never seen. Sprint 38 opened the door for Compose. Sprint 39 widens it for the repo that only has a `Dockerfile`.

---

## The hole after Compose from-codebase

`podbay init --from-codebase` already turned a Compose tree into a first-pass `podbay.yaml` and pointed at onboard / validate. Dockerfile-only apps still failed discovery—even though the contract and runner already understand `build.context` / `build.dockerfile`.

Adoption width is not the same as loop latency. We deferred the cache and probe track again and shipped the next front-door slice.

---

## What we shipped

### Compose first, Dockerfile fallback

Unchanged Compose discovery order. When that misses (or you pass **`--dockerfile`**), discovery looks for:

1. `Dockerfile`
2. `dockerfile`

`--compose` and `--dockerfile` are mutually exclusive. Compose still wins when both files exist.

### Honest single-service stub

```bash
podbay init --from-codebase
# Dockerfile-only tree → first-pass podbay.yaml
podbay onboard -f podbay.yaml --json
podbay validate -f podbay.yaml --json
```

The stub is one `app` service with build context + dockerfile and a local image tag. No invented ports or health checks—validate / hand-tighten remains required. Same next-step dialect as Compose and greenfield init.

### Machine-readable origin

`--json` success includes `source_kind` (`compose` or `dockerfile`) and `compose_source` or `dockerfile_source`. Neither found → `codebase_discovery_not_found`. Overwrite still fail-closed (`init_target_exists`).

Offline demo: [`examples/ci-dockerfile-from-codebase-demo.sh`](../examples/ci-dockerfile-from-codebase-demo.sh).

---

## What this is not

- Not language or package-manager scanning.
- Not LLM contract codegen.
- Not auto-deploy or diagnosis.
- Not a claim that the stub is production-ready without validate.

---

## Why this sits on the critical path

```text
init --from-codebase → onboard → validate → deploy → receipt / handoff → explain
```

Compose converters were in. Dockerfile-only repos were still inventing YAML by hand. Widening arrive beats polishing a loop they cannot enter.

---

## What's next

Still deferred: broader codebase heuristics, env/contract cache, parallel / `--no-probes` explain. The front door is wider; the house can get faster later.

Docs: [agent-loop.md](agent-loop.md), [contract.md](contract.md), [cli-json.md](cli-json.md).
