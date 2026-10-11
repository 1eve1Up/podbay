# Sprint 41: Down means already-down

*2026-10-11 — Podbay public preview*

`podbay down` is the “make it not running” gate. Agents retry it. A stack that is already gone, or a Mac whose `podman machine` is stopped, should exit 0 — not `Error: podman ps: exit status 125`.

---

## The hole after a clean teardown

Two already-down hosts, two outcomes:

```text
# backlogzero — machine up, nothing labeled
Tearing down project "backlogzero"
 No containers labeled podbay.project=backlogzero
 Removing network podbay_backlogzero
Done.

# pinionize — machine stopped
Tearing down project "pinionize"
Error: podman ps: exit status 125
```

Teardown listed project containers first and treated any `podman ps` failure as fatal. `Output()` discarded stderr, so the only signal was 125. A stopped machine is connection refused, not an empty label set. A stale “no such container” race during `ps` is the same class of “nothing to remove.”

---

## What we shipped

- **`runner.ProjectPs`** — capture stderr; retry once on exit 125; treat `no such container` / `does not exist in database` as an empty list.
- **Teardown-only unavailable** — `Cannot connect to Podman` / missing socket exits 0 with `Podman is not running; nothing to tear down`. `--json` stays `status: ok` and adds warn **`teardown_podman_unavailable`**.
- **Observability stays fail-closed** — `diff` / `ps` / `explain` still error when the engine is down.
- **Real list failures still fail** — template errors and other `ps` problems exit 1 and now include stderr.

```bash
podbay down
podbay down --json
```

---

## What this is not

- Not `podman machine start` from Podbay.
- Not treating a stopped engine as empty runtime for `diff` / `ps` / `explain`.
- Not volume removal when the VM is off (`--volumes` warns and skips).

---

## Why this sits on the critical path

```text
validate → deploy → diff → logs → down
```

`down` is the cleanup arrow. If it fails when nothing is running, the next attempt does not start clean — the agent stops on a 125. Classify empty / stale / unavailable separately from a real list failure.

---

## What's next

Still deferred: broader codebase heuristics, env/contract cache, parallel / `--no-probes` explain, causal diagnosis in `explain`. The loop can now leave. Speed and diagnosis can wait.

Docs: [agent-loop.md](agent-loop.md), [cli-json.md](cli-json.md), [contract.md](contract.md).
