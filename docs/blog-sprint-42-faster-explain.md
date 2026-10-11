# Sprint 42: Faster explain without quieter facts

*2026-10-11 — Podbay public preview*

Arrive names the requirements. `down` leaves. Mid-loop `explain` still burned the 30-second Clarity budget: re-parse the contract and env files at every helper, then walk HTTP and exec probes one service at a time under a 5s cap.

---

## The hole after a honest enter and leave

A five-service stack could wait 5s × N on the default path. Agents who only needed inspect + orientation after deploy paid that too. `Report`, `ReportJSON`, and onboard each called `LoadHostSubst`. `spec.Load` and dotenv parses had no content memo. Size and mtime looked like a cheap hit — until a same-size rewrite in one clock tick served yesterday’s map.

---

## What we shipped

- **Env-file memo** — parse `.env` / `host_env_files` by content digest. Overlay live `os.Environ` every call. Never write process env to `.podbay`.
- **Contract memo** — `spec.Load` reuses a parse when path + digest match; a byte change is a miss. Different paths do not share one `Contract`.
- **One subst per explain** — `prepareExplain` loads host subst once for text, JSON, and onboard.
- **`explain --no-probes`** — inspect and orientation stay; HTTP/exec do not run. `--json` adds `probes_skipped`. Default explain still probes.
- **Parallel default probes** — in-scope services probe concurrently. Per-probe cap stays 5s. Inspect error and `--no-probes` still skip.

```bash
podbay explain -f examples/nginx --json
podbay explain -f examples/nginx --json --no-probes
```

---

## What this is not

- Not a disk cache of environment secrets.
- Not `--no-probes` as the silent default.
- Not a shorter stack-wide probe budget.
- Not causal diagnosis or remediation in `explain`.

---

## Why this sits on the critical path

```text
validate → deploy → logs → explain → down
```

After arrive and down, the expensive arrow is explain. Memo the files. Load subst once. Skip probes only when asked. Parallelize the rest. Honesty is a digest miss and a live process-env overlay.

---

## What's next

Still deferred: broader codebase heuristics, causal diagnosis in `explain`. The loop can enter, leave, and look around without a serial stopwatch. Diagnosis can wait.

Docs: [agent-loop.md](agent-loop.md), [cli-json.md](cli-json.md), [glossary.md](glossary.md).
