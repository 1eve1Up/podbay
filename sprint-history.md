# Sprint 31: Diff runtime efficiency

2026-06-10

**Repo:** Podbay (Go tree at monorepo root).

**Carries from Sprint 30:** Deploy phase extraction complete; diff N× inspect and no-op **`LoadHostSubst`** deferred from Sprints 28–30.

---

## Sprint goal

Replace N per-service **`podman inspect`** subprocesses in **`podbay diff`** with a batched runtime snapshot, wire through **`internal/diff`**, document in **`docs/architecture.md`** — no CLI or JSON behavior changes.

---

### What happened

We added **`ParseInspectMany`**, **`InspectContainers`**, and project ps helpers in **`internal/runtimestate`**, wired **`diff.ComputeWithContainerStates`**, removed no-op **`LoadHostSubst`**, and documented the diff subprocess model. All ten **`feature/PIN-3101`** … **`feature/PIN-3110`** merges on **`main`**; **`go test ./...`** green (**436 insertions / 51 deletions** across **10** files, **`1ad372b`** … **`ff4f0e1`**).

---

## Retrospective

### Meta

- **Date / time**: 2026-06-10 (UTC, sprint wrap)
- **Scope**: Diff hot-path perf (refactor-plan §3); no CLI or JSON behavior changes.

### 5 whys (why diff inspect batching stayed deferred through Sprint 30)

1. **Why still open after deploy extraction?** Sprints 28–30 prioritized docs, spec split, and deploy phases first.
2. **Why queue behind deploy?** Sprint 30 wanted named deploy phases before diff profiling targets existed.
3. **Why no urgency on 2-service demos?** Subprocess savings invisible next to deploy/build; debt was O(N) architecture.
4. **Why tolerate duplicate ps?** Extras already used one ps; per-service inspect in **`Compute`** was the dominant cost.
5. **Root lesson:** Serial runtimestate API landings before diff wiring kept drift semantics stable.

### Actions

- [x] Batch inspect parsing + **`InspectContainers`** + merged ps/extras (**PIN-3101** … **PIN-3103**).
- [x] **`ComputeWithContainerStates`** and **`ReportContractResult`** batch path (**PIN-3104**, **PIN-3105**).
- [x] Tests, architecture doc, exit bar (**PIN-3106** … **PIN-3110**).
- [ ] Cross-invocation env/contract cache (deferred).
- [ ] ps/explain inspect batching (deferred).

# Sprint 30: Deploy pipeline phase extraction

2026-06-09

**Repo:** Podbay (Go tree at monorepo root).

**Carries from Sprint 29:** Agent **`docs/`** corpus complete; **Finding 3** (deploy phase extraction) deferred from Sprints 28 and 29.

---

## Sprint goal

Extract **`Deploy()`** into numbered phases (**networks → volumes → services → receipt**), slim **`deploy.go`** to a thin orchestrator, and document phases in **`docs/architecture.md`**—no CLI or JSON behavior changes.

---

### What happened

We split **`internal/deploy/deploy.go`** from **544** lines to a **55-line** orchestrator plus **`deploy_context.go`**, **`networks.go`**, **`volumes.go`**, and **`services.go`**, with **`writeDeployReceipt`** in **`receipt.go`**. **`docs/architecture.md`** gained a deploy pipeline table. All ten **`feature/PIN-3001`** … **`feature/PIN-3010`** merges landed on **`main`**; **`go test ./...`**, **`go vet`**, and **`gofmt`** green (**629 insertions / 501 deletions** across **7** files, **`7bbbbf7`** … **`b8be56e`**).

---

## Retrospective

### Meta

- **Date / time**: 2026-06-10 (UTC, sprint wrap)
- **Scope**: Deploy hot-path refactor (Finding 3); no CLI or JSON behavior changes.

### 5 whys (why `Deploy()` stayed a monolith after Sprint 29)

1. **Why still open after the docs sprint?** Sprints 28–29 deferred **Finding 3** while spec, import layering, and **`docs/`** shipped first.
2. **Why defer when CLI/spec were split?** Docs had higher ROI with zero runtime risk; sibling files masked that **`Deploy()`** still inlined orchestration.
3. **Why did that matter?** Agents profiling validate → deploy had to scan ~260 lines for one phase boundary.
4. **Why two deferrals?** Mechanical refactor queued behind visible doc work twice—no JSON drift forced the issue.
5. **Root lesson:** Split the deploy pipeline right after doc/navigation sprints, before perf work.

### Actions

- [x] **`deployContext`** + numbered phases (**PIN-3001** … **PIN-3007**).
- [x] **`docs/architecture.md`** deploy pipeline subsection (**PIN-3008**).
- [x] Integration tests + exit bar (**PIN-3009**, **PIN-3010**).
- [x] Diff inspect batching and **`LoadHostSubst`** cleanup (**Sprint 31**).

# Sprint 29: Docs corpus + README split

2026-06-08

**Repo:** Podbay (Go tree at monorepo root).

**Carries from Sprint 28:** **`docs/architecture.md`** and **`docs/contract-change-checklist.md`** were referenced in **CONTRIBUTING** but not yet committed; **`docs/glossary.md`** and a **README split** were explicitly deferred. No CLI or JSON behavior changes were in scope.

---

## Sprint goal

Move deep reference out of **README** into focused **`docs/`** pages (**`glossary`**, **`contract`**, **`cli-json`**, **`agent-loop`**, plus Sprint 28 architecture/checklist carry-over), slim **README** to pitch + quick start + a docs index, and rewire **PRD** / **CONTRIBUTING** / **RELEASES** links—no CLI flags, JSON envelopes, or exit code changes.

---

### What happened

We slimmed **README** from **~513** lines to **179** (agent loop, contract reference, JSON tables, and “when to use” moved out; **Documentation** table + quick start retained). **PRD**, **CONTRIBUTING**, and **RELEASES** now point at **`docs/`** instead of README anchors. The six-file **`docs/`** corpus (**768** lines: **`architecture`**, **`contract-change-checklist`**, **`glossary`**, **`contract`**, **`cli-json`**, **`agent-loop`**) landed in follow-up commit **`c3b3ced`** the same day, completing links introduced in exit commit **`d29e15e`**. **`gofmt`**, **`go vet`**, **`go test ./...`**, and **`pinion build`** green on **`main`** (**36 insertions / 435 deletions** across **4** files in **`d29e15e`**; **`PIN-2910`** exit bar only—no Go tree changes this sprint).

---

## Retrospective

### Meta

- **Date / time**: 2026-06-08 (UTC, sprint wrap)
- **Scope**: Documentation navigation and README split; no CLI or JSON behavior changes.

### 5 whys (why README stayed a monolith after Sprint 28)

1. **Why did agents and contributors still load a 500+ line README?** Pitch, quick start, contract reference, JSON envelopes, and the partial-deploy loop all lived in one file.
2. **Why no split when Sprint 28 documented architecture?** Sprint 28 prioritized **`internal/spec`** / **`volumemount`** code splits; **`docs/architecture.md`** was linked from **CONTRIBUTING** before the file existed in git.
3. **Why defer glossary and README split to Sprint 29?** Sprint 28 explicitly queued them behind the spec package refactor and import layering.
4. **Why does that matter?** Duplicate prose across **README**, **PRD**, and **RELEASES** inflated context cost and drift risk whenever JSON or partial-selection semantics changed.
5. **Root lesson:** Once package boundaries are stable, **navigation docs belong in focused files**—README should sell and onboard; deep reference belongs in **`docs/`** with one glossary and cross-links.

### Actions

- [x] **`docs/glossary.md`**, **`docs/contract.md`**, **`docs/cli-json.md`**, **`docs/agent-loop.md`** (**`c3b3ced`**).
- [x] **`docs/architecture.md`**, **`docs/contract-change-checklist.md`** (Sprint 28 carry-over; **`c3b3ced`**).
- [x] Slim **README** to **179** lines with docs index (**`d29e15e`**).
- [x] Rewire **PRD** / **CONTRIBUTING** / **RELEASES** to **`docs/`** (**`d29e15e`**).
- [x] Exit bar: **`gofmt`**, **`go vet`**, **`go test ./...`**, **`pinion build`** green (**PIN-2910**, **`d29e15e`**).
- [x] **`internal/deploy`** phase extraction (Sprint 30; **`7bbbbf7`** … **`b8be56e`**).

# Sprint 28: Architecture clarity + spec navigation + import layering

2026-06-08

**Repo:** Podbay (Go tree at monorepo root).

**Carries from Sprint 27:** Host env **expansion** is unified on **`expand.ExpandService`**. Partial **selection** was unified in Sprint 26 on **`spec.ObservabilityActiveServices`**. Deferred from Sprint 27: **`spec.go` split**, **`docs/architecture.md`**, and **`SplitVolumeMount`** decoupling.

---

## Sprint goal

Document the import pipeline, split **`internal/spec/spec.go`**, extract **`internal/volumemount`**, and sync OSS version strings—no CLI or JSON behavior changes.

---

### What happened

We synced **PRD / CONTRIBUTING / SECURITY** to **`v2026.6.1`**, split **`internal/spec`** into four focused files (**`spec.go`** is 3-line package doc), extracted **`internal/volumemount`**, and broke **`composeimport → runner`** coupling. **`docs/architecture.md`** and **`docs/contract-change-checklist.md`** were wired in **CONTRIBUTING** here and committed in Sprint 29 (**`c3b3ced`**). All tests pass on **`main`** (**752 insertions / 736 deletions** across **25** file touches; twelve **`feature/PIN-2801`** … **`feature/PIN-2812`** merges).

---

## Retrospective

### Meta

- **Date / time**: 2026-06-08 (UTC, sprint wrap)
- **Scope**: Architecture docs + spec navigation + import layering; no CLI or JSON behavior changes.

### 5 whys (why the import pipeline stayed undocumented after Sprint 27)

1. **Why did contributors think composefile, spec, and emit were accidental duplication?** The three layers were intentional but never documented in **`docs/`**.
2. **Why no architecture doc sooner?** Sprints 26–27 prioritized agent-loop correctness; docs deferred while code drift was active.
3. **Why did `spec.go` stay a monolith?** Types, YAML, graph, and profiles accumulated in one file as features landed.
4. **Why did `composeimport` import `runner`?** **`SplitVolumeMount`** lived in the Podman adapter and was reused at import time for convenience.
5. **Root lesson:** Document boundaries once the agent loop is stable—invest in architecture docs and package splits before the next feature sprint.

### Actions

- [x] Architecture + checklist doc content (**PIN-2801** … **PIN-2802**); **`docs/architecture.md`** / **`docs/contract-change-checklist.md`** committed Sprint 29 (**`c3b3ced`**).
- [x] Version sync + **`internal/spec`** split + **`internal/volumemount`** (**PIN-2803** … **PIN-2810**).
- [x] Package docs + exit bar (**PIN-2811** … **PIN-2812**).
- [x] **`docs/glossary.md`**, README split (Sprint 29; **`d29e15e`** / **`c3b3ced`**).
- [ ] **`internal/deploy`** phase extraction (deferred).

# Sprint 27: Unify `expandService` on the agent loop

2026-06-08

**Repo:** Podbay (Go tree at monorepo root).

**Carries from Sprint 26:** The CLI god module is split (`main.go` is ~80 lines) and partial service **selection** is unified on **`spec.ObservabilityActiveServices`**, but host env **expansion** was still copy-pasted in **`internal/validate`**, **`internal/deploy`**, and **`internal/explain`**—deploy expanded **`AnsibleVaultPaths`**; validate and explain did not.

---

## Sprint goal

Extract a single **`expand.ExpandService`** in **`internal/expand/service.go`** and replace all private **`expandService`** copies so **validate → deploy → explain** (and receipt/status call paths) apply the same host `${VAR}` substitution to service fields—without changing CLI flags, JSON envelopes, or exit codes.

---

### What happened

We extracted **`expand.ExpandService`** and **`expand.ExpandStrings`** into **`internal/expand`**, added unit tests (including **`AnsibleVaultPaths`** regression coverage), and migrated **validate**, **deploy** (+ **receipt**), and **explain** (+ **status**) to the shared helper. **107** lines of private **`expandService`** / **`expandStrs`** / **`expandMap`** duplicates were deleted. All **`go test ./...`**, **`go vet ./...`**, and **`gofmt`** checks pass on **`main`** (**123 insertions / 114 deletions** across **7** files from **`676a4b7`** through **`d8850d4`**; six **`feature/PIN-2701`** … **`feature/PIN-2706`** merges; **`PIN-2707`** verification-only).

---

## Retrospective

### Meta

- **Date / time**: 2026-06-08 (UTC, sprint wrap)
- **Scope**: Unify host env expansion on the validate → deploy → explain agent loop; no CLI or JSON behavior changes.

### 5 whys (why `expandService` drifted after Sprint 26)

1. **Why could validate pass while deploy used different expanded values?** Three private **`expandService`** copies applied host substitution with different field lists—deploy expanded **`AnsibleVaultPaths`**; validate and explain did not.
2. **Why three copies?** **validate**, **deploy**, and **explain** each inlined expansion next to the checks or runtime actions that consume expanded service fields.
3. **Why inlined instead of shared?** **`internal/expand`** already owned **`LoadHostSubst`** and **`String`**, but no **`Service`**-level helper existed when those gates were built.
4. **Why defer until Sprint 27?** Sprint 26 closed partial **selection** drift; expansion looked package-local until refactor-plan Finding 7b flagged the same cross-gate risk class.
5. **Root lesson:** **Expansion is part of the agent contract**—when multiple JSON-stable gates reason about the same **`spec.Service`** shape, host substitution must live in one place beside selection.

### Actions

- [x] Add **`internal/expand/service.go`** with **`ExpandService`** deploy superset (**PIN-2701**).
- [x] Add **`internal/expand/service_test.go`** (**PIN-2702**).
- [x] Migrate **validate**, **deploy** (+ **receipt**), **explain** (+ **status**) to **`expand.ExpandService`** (**PIN-2703** … **PIN-2705**).
- [x] Remove private expand helpers; export **`expand.ExpandStrings`** for deploy network DNS (**PIN-2706**).
- [x] Exit bar: **`gofmt`**, **`go vet`**, **`go test ./...`**, **`pinion build`** green (**PIN-2707**).
- [ ] **Finding 2** — split **`internal/spec/spec.go`** into types/graph/yaml files.
- [ ] **`docs/architecture.md`** — package diagram and import-pipeline phases.
- [ ] **Finding 8** — move **`SplitVolumeMount`** out of **`internal/runner`**.

# Sprint 26: Split CLI god module + unify partial service selection

2026-06-08

**Repo:** Podbay (Go tree at monorepo root).

**Carries from Sprint 25:** The partial-deploy agent loop is documented end-to-end (`examples/ci-partial-agent-loop-demo.sh`), but **`cmd/podbay/main.go`** was still a **~1,062-line god module** and **`validate`** / **`deploy`** still inlined copy-pasted partial service resolution instead of calling **`spec.ObservabilityActiveServices`**.

---

## Sprint goal

1. Split **`cmd/podbay/main.go`** by command group so agents and contributors can navigate one file per concern—without changing CLI behavior, flags, JSON envelopes, or exit codes.
2. Unify partial service selection in **`internal/validate`** and **`internal/deploy`** on **`spec.ObservabilityActiveServices`** so validate → deploy → diff cannot drift on which services are in scope.

---

### What happened

We split the god module into **seven command-group files** (`contract.go`, `init_cmd.go`, `receipt_cmd.go`, `validate.go`, `deploy.go`, `lifecycle.go`, `observability.go`) plus an **80-line** `main.go` (registration + `version` only), with a comment directing new commands to group files. **`internal/validate`** and **`internal/deploy`** now call **`spec.ObservabilityActiveServices`**; the spec doc comment states it is the single implementation for all commands. All existing integration tests passed unchanged on **`main`** (**1,077 insertions / 1,013 deletions** across **11** files in **`4c39f79`**; nine **`feature/PIN-2601`** … **`feature/PIN-2609`** merges).

---

## Retrospective

### Meta

- **Date / time**: 2026-06-08 (UTC, sprint wrap)
- **Scope**: Mechanical CLI refactor and partial-selection consolidation; no CLI or JSON behavior changes.

### 5 whys (why `main.go` stayed a god module after Sprint 25)

1. **Why was agent navigation still painful?** Every CLI command, JSON emit helper, and contract loader lived in one file—high merge conflict and context-window cost for any single-command change.
2. **Why one file?** Podbay grew command-by-command in `main.go` before `import_cmd.go` established the per-file pattern; observability and lifecycle helpers accumulated in place.
3. **Why defer the split until Sprint 26?** Sprint 25 prioritized the partial-deploy **agent loop** (orchestration + docs); refactor-plan Finding 1 was explicitly queued behind gate completeness.
4. **Why did validate/deploy duplicate partial selection?** Those packages predated **`ObservabilityActiveServices`**, which observability commands adopted first; the inline blocks looked “local” until the agent loop proved cross-gate drift risk.
5. **Root lesson:** **File boundaries are part of the agent contract**—when JSON-stable gates share semantics, extract shared resolution **and** split the CLI surface in the same sprint window, or every feature sprint pays a navigation tax.

### Actions

- [x] Extract command-group files and slim `main.go` (**PIN-2601** … **PIN-2606**).
- [x] Unify **`internal/validate`** and **`internal/deploy`** on **`spec.ObservabilityActiveServices`** (**PIN-2607**, **PIN-2608**).
- [x] Exit bar: **`gofmt`**, **`go vet`**, **`go test ./...`**, **`pinion build`** green (**PIN-2609**).
- [x] **Finding 7b** — extract duplicated `expandService` to `internal/expand/service.go` (Sprint 27).
- [ ] **Finding 2** — split `internal/spec/spec.go` into types/graph/yaml files.
- [ ] **`docs/architecture.md`** — package diagram and import-pipeline phases.

# Sprint 25: End-to-end partial-deploy agent loop

2026-06-02

**Repo:** Podbay (Go tree at monorepo root).

**Carries from Sprint 24:** Every partial-deploy gate emitted versioned JSON, including structured **`deploy_health_*`** failures—but operators still stitched **`ci-receipt-demo.sh`**, **`ci-partial-logs-demo.sh`**, and **`ci-deploy-health-fail-demo.sh`** by hand.

---

## Sprint goal

Ship one runnable recipe and docs that chain **validate → deploy → diff → logs → down** on shared partial roots, plus a **deploy_health_* → logs → explain → down** failure branch—without new CLI JSON shapes.

---

### What happened

We shipped **`examples/ci-partial-agent-loop-demo.sh`** (`happy` / `fail`), **`agent_loop_demo_integration_test.go`**, and README / PRD / **RELEASES** (`v2026.6.0`) on **`main`** (**190 insertions / 1 deletion** across **5** files in **`f0b6ff3`**).

---

## Retrospective

### Meta

- **Date / time**: 2026-06-02 (UTC, sprint wrap)
- **Scope**: Agent-loop orchestration and documentation; CLI JSON envelopes unchanged.

### 5 whys (why three demos persisted after Sprint 24)

1. **Why no single CI recipe?** Gates worked separately; nothing chained them on the same service roots or documented health-failure recovery.
2. **Why after Sprint 24?** Structured deploy failures shipped first; full-loop polish was an explicit fork.
3. **Why Sprint 25?** Orchestration only—no **`clijson`** changes—so safe once the deploy gate was done.
4. **Why matter?** Adoption still required mental glue despite correct per-command JSON.
5. **Root lesson:** **Composition is part of the contract** when every gate is machine-readable.

# Sprint 24: Structured deploy health-gate failures (`deploy --json`)

2026-05-22

**Repo:** Podbay (Go tree at monorepo root).

**Carries from Sprint 23:** The **partial-deploy agent loop** was complete on success and evidence paths; **`podbay deploy --json`** still collapsed every runtime failure—including health-gate timeouts—into a single **`deploy_error`** issue.

---

## Sprint goal

Make **`podbay deploy --json`** emit **structured, per-service health-gate failures** so agents and CI can branch on stable codes and service names when deploy fails after containers start but health never passes.

---

### What happened

We shipped **`HealthGateFailure`** and **`health_wait`** refactor in **`internal/deploy`**, **`clijson`** deploy health issue codes (**`deploy_health_timeout`**, **`deploy_health_probe_failed`**, **`deploy_external_dep_unhealthy`**), **`deploy_json_integration_test.go`**, **`examples/unhealthy-health/`** and **`examples/ci-deploy-health-fail-demo.sh`**, and **README** / **PRD** / **RELEASES** (`v2026.5.3`) on **`main`** (**554 insertions / 54 deletions** across **15** files from **`3b22759`** through **`4e28afa`**; nine feature branch merges).

---

## Retrospective

### Meta

- **Date / time**: 2026-05-23 (UTC, sprint wrap)
- **Scope**: **`deploy --json`** structured health-gate failures complete the agent deploy gate; preflight validate JSON and success deploy shapes unchanged.

### 5 whys (why **`deploy --json`** health failures stayed unstructured after Sprint 23)

1. **Why could agents not branch on deploy failure at a health gate?** **`deploy --json`** emitted one **`deploy_error`** issue with a human-oriented message string—no **`service`** or stable health code.
2. **Why was that left after Sprint 23?** Sprint 23 closed the partial-deploy loop on **success and evidence** (`logs` batch JSON); structured health-gate failures were explicitly deferred from that sprint.
3. **Why defer to Sprint 24?** Runtime health waits needed a typed failure path from **`waitServiceHealth`** without conflating with preflight **`DeployFromValidateResults`** validate issues.
4. **Why does that ordering matter?** Without structured health failure, agents could **`logs`** / **`explain`** after deploy but still had to parse opaque strings to know **which service** failed and **why** (timeout vs probe).
5. **Root lesson:** The **deploy gate** must expose structured runtime validation outcomes—the same JSON discipline as preflight **`validate`** and post-deploy **`logs`**—or the agent loop breaks on the most expensive failure path (containers started, health never passed).

# Sprint 23: Partial-deploy log evidence (`logs` selection + batch `--json`)

2026-05-18

**Repo:** Podbay (Go tree at monorepo root).

**Carries from Sprint 22:** **`podbay import compose --json`** success shipped. **`validate`**, **`deploy`**, **`diff`**, **`ps`**, **`explain`**, and **`teardown` / `down`** already shared optional service roots and **`--dependents`**; **`podbay logs`** still required a **single** service name and could not return structured evidence for the same resolved set in one invocation.

---

## Sprint goal

Close the last major gap in the **multi-agent partial-deploy loop**: **`podbay logs`** accepts the **same optional service roots and `--dependents`** as other contract commands, and **`logs --json`** returns **one versioned document** with log evidence for **every service in that resolved set** (`log_entries[]`), without N CLI calls or stderr scraping.

---

### What happened

We shipped **`internal/logs`** (`ActiveServices`, human multi-service output, batch **`CaptureBytes`**), a refactored **`logsCmd`** using **`loadContractWithDeployServices`** and **`spec.ObservabilityActiveServices`**, **`clijson.FromLogsBatchSuccess`** / **`LogsFailurePartial`** with **`log_entries[]`** and additive **`deploy_services`** / **`dependents_expand`**, stable codes **`logs_resolve_error`** and **`logs_follow_multi_service`**, subprocess integration tests, **`examples/two-service/`** and **`examples/ci-partial-logs-demo.sh`**, and **README** / **PRD** / **RELEASES** (`v2026.5.2`) updates on **`main`** (**394 insertions / 76 deletions** across **12** files from **`8bff9bc`** through **`e316547`**; nine **`feature/PIN-2301`** … **`feature/PIN-2308`** merges).

---

## Retrospective

### Meta

- **Date / time**: 2026-05-18 (UTC, sprint wrap)
- **Scope**: **`logs`** partial selection and batch **`--json`** complete the agent evidence path after partial deploy; single-service top-level **`service`** / **`log_body`** preserved when one target resolves.

### 5 whys (why **`logs`** lagged partial selection and batch JSON)

1. **Why could agents not collect post-deploy log evidence for a partial set in one step?** **`logs`** accepted only **one** service name and emitted **one** container’s text (or one **`log_body`**), while **`diff`** / **`explain`** already honored the same roots and **`--dependents`**.
2. **Why was that left after Sprints 17–20?** Those sprints scoped **partial deploy**, **observability**, and **lifecycle** first; Sprint 21 added **single-service** **`logs --json`** without multi-target selection.
3. **Why defer batch JSON until Sprint 23?** **`--json` + `--follow`** and multi-stream human tailing needed an explicit contract; one-shot batch capture could ship without streaming semantics.
4. **Why does that ordering matter?** Without **`logs`** parity, the **partial-deploy agent loop** still broke at **evidence**—operators and CI had to guess service names or call **`logs`** repeatedly after **`deploy`** partial roots.
5. **Root lesson:** Commands used as **evidence** in the agent loop must share the same **selection and JSON discipline** as gates—or document a deliberate exception. **`logs`** was the last exception.

### Actions

- [x] Ship **`logs`** partial roots, **`--dependents`**, and batch **`logs --json`** with docs (**this sprint**).
- [x] Release **PIN-2301** … **PIN-2309** on **`main`** with per-target **`feature/PIN-###`** branches and **`go test ./...`** on transitions (**this sprint**).
- [ ] Optional: dedupe **`--dependents`** help strings (cosmetic).
- [ ] Optional: **`logs`** **`--json` + `--follow`** behind a written contract + tests.

# Sprint 22: Machine-readable success for `podbay import compose` (`--json`)

2026-05-16

**Repo:** Podbay (Go tree at monorepo root).

**Carries from Sprint 21:** **`podbay logs --json`** shipped; **`podbay import compose --json`** still had **failure-only** JSON while success emitted raw YAML.

---

## Sprint goal

Add a **documented success path** for **`podbay import compose --json`**: one versioned JSON document on stdout on success (`kind: import_compose`, `status: ok`, `contract_yaml`, `service_count`, optional `project` / `output_path`), with **`-o`** writing YAML before JSON.

---

### What happened

We shipped **`FromImportComposeSuccess`**, additive **`Document`** fields, **`import compose`** success **`--json`** wiring (file write before JSON when **`-o`** is set), subprocess integration tests, and **README** / **PRD** / **RELEASES** updates on **`main`** (**194 insertions / 24 deletions** in commit **`7cddb16`**).
---

## Retrospective

### Meta

- **Date / time**: 2026-05-16 (UTC, sprint wrap)
- **Scope**: **`import compose --json`** success matches other CLI JSON surfaces; Sprint 16 failure JSON unchanged.

### 5 whys (why **`import compose`** success lagged other **`--json`** commands)

1. **Why could agents not treat a successful import as a structured CLI outcome?** **`--json`** was **failure-only**; success emitted **raw YAML** on stdout.
2. **Why leave it that way after Sprint 16?** Sprint 16 scoped **stable codes on failure** first; success needed an explicit **`contract_yaml`** / **`-o`** contract.
3. **Why defer past later sprints?** **Partial deploy**, **observability**, **lifecycle**, and **`logs`** JSON reduced operator confusion before the **import** success gap bit automation.
4. **Why does that ordering matter?** Without **`logs --json`**, claiming “JSON everywhere” would still fail at the **post-deploy** read path.
5. **Root lesson:** **Define success and failure machine-readable shapes** for any command in the agent/CI gate—or document an explicit exception.

### Actions

- [x] Ship **`import compose --json`** success path and docs (**this sprint**).
- [x] Run **`pinion retro`** and PB-1 bundle (**this sprint**).
- [ ] Optional: dedupe **`--dependents`** help strings (cosmetic).
- [ ] Optional: **`logs`** **`--json` + `--follow`** behind a written contract + tests.


# Sprint 21: Machine-readable `podbay logs` (`--json`)

2026-05-14

**Repo:** Podbay (Go tree at monorepo root).

**Carries from Sprint 20:** Partial lifecycle and profile-aligned **`logs`** shipped; automation still could not consume **`logs`** as versioned JSON.

---

## Sprint goal

Ship **`podbay logs --json`** with **`kind: logs`**, stable **`issues[].code`** values on failure, **`log_body`** on success (one-shot capture; **`--json`** incompatible with **`--follow`**), integration tests, and README / PRD / RELEASES alignment.

---

### What happened

We shipped **`podbay logs --json`**: **`KindLogs`**, **`FromLogsSuccess`** / **`LogsFailure`**, stable codes, **`runner.LogsBytes`**, **`logsCmd`** wiring (contract/profile resolution **before** **`EnsurePodman`** for accurate JSON without Podman), **`logs_json_integration_test.go`**, and documentation updates on **`main`** (**402 insertions / 8 deletions** in commit **`806aadd`**). Pinion **PIN-175**–**PIN-180** were run to **`released`** with **`pinion retro`** closing the sprint (PB-1 bundle under Pinion **`artifacts/sprint-21/`**).

## Retrospective

### Meta

- **Date / time**: 2026-05-14 (UTC, sprint wrap)
- **PIN / work units**: **PIN-175** … **PIN-180** (Pinion coordination); Podbay implementation delivered as a **single merge** to **`main`**.
- **Scope**: Close the **JSON** gap on **`logs`** so the agent/operator path matches **`validate`**, **`deploy`**, **`diff`**, **`explain`**, **`receipt`**, **`teardown`**, and **`down`**.

### Stats

- **Pinion merge snapshot** (per-unit merge rows): **0** lines added / net (coordination-only merges); **~970** estimated input tokens summed across units.
- **Git (Podbay):** **402** insertions, **8** deletions, **11** files (see **`806aadd`**).

### 5 whys (why **`logs`** lagged other JSON commands)

1. **Why could CI not consume `logs` outcomes structurally?** It only exposed streamed **`podman`** text—no **`clijson`** envelope.
2. **Why no envelope?** **`logs`** was modeled as a thin wrapper, not a peer to **`diff`** / **`teardown`** for automation.
3. **Why did that survive Sprint 20?** Sprint 20 scoped **`logs`** to **help + profiles** and deferred multi-stream / follow complexity.
4. **Why defer `--json` then?** **`--json` + `--follow`** needs an explicit streaming contract; shipping unspecified behavior would break parsers.
5. **Root lesson:** Commands used as **evidence** in the agent loop must emit the **same JSON discipline** or document a deliberate exception—otherwise “machine-readable Podbay” is still a lie at **`logs`**.

### Actions

- [x] Ship **`logs --json`** and docs (**this sprint**).
- [x] Run **`pinion retro`** and record closure (**this sprint**).
- [ ] Optional: dedupe partial-selection **`--dependents`** help across commands (cosmetic).
- [ ] Optional: **`logs`** multi-service or **`--json` + `--follow`** behind a written contract + tests.

### Notes

- Pinion PB-1 path (local dev tree): **`pinion/pinion/sprints/artifacts/sprint-21/`** when present; **`pinion retro`** appended **`## Retrospective`** to **`pinion/pinion/sprints/sprint-21.md`** and cleared **`project.active_sprint`**.

# Sprint 20: Partial lifecycle (teardown, down, logs alignment)

2026-05-13

**Repo:** Podbay (Go tree at monorepo root).

**Carries from Sprint 19:** Observability commands (`diff`, `ps`, `explain`) share the same partial roots and `--dependents` resolution as `validate` / `deploy`. **`teardown`**, **`down`**, and **`logs`** still assume **full-project** or **single-service** shapes, so operators tearing down or tailing logs after a partial deploy must mentally switch models.

---

## Sprint goal

Bring **lifecycle** commands in line with **partial selection**:

1. **`podbay teardown` and `podbay down`** accept the **same optional contract path, `--profile`, service roots, and `--dependents`** as `validate` / `deploy`, and remove **only** containers for services in that resolved set (not every labeled container in the project).
2. Define and document **network**, **named volume**, and **“leftover containers”** behavior when only a subset is torn down (so we do not surprise users or strand the stack in an inconsistent state).
3. **`podbay logs`:** at minimum, **reuse the same contract path and profile resolution** as other commands and keep **one** log target; extend help and validation so partial-deploy users are not confused. **Multi-service tailing or `--follow` across multiple containers** is optional and may be explicitly deferred if it balloons scope.

---

### What happened

We shipped **partial `podbay teardown` and `podbay down`** with the same contract path, **`--profile`**, optional service roots, and **`--dependents`** as **`validate` / `deploy`**, using **`spec.ObservabilityActiveServices`** for the resolved service set. **Partial** teardown removes only matching containers, **skips project network removal** while any project-labelled container remains, **rejects `--volumes` / `-v`**, and documents volume behavior for full teardown only. **`teardown` / `down` `--json`** gained additive **`deploy_services`** and **`dependents_expand`**. **`podbay logs`** gained clearer long help (profiles, root `-f` vs **`--follow`**, single-container scope). **README**, **PRD**, and **RELEASES** describe the lifecycle story; **`tools/run-sprint-tests.sh`** header notes partial lifecycle coverage.

# Sprint 19: Partial-selection observability (`diff` / `ps` / `explain`)

2026-05-13

**Repo:** Podbay (Go tree at monorepo root).

**Carries from Sprint 18:** Partial `validate` / `deploy` share an explicit target map plus optional `--dependents`. Contract-vs-runtime **`diff`**, **`ps`**, and **`explain`** still treat **all profile-active services** as the expected set, so a partial deploy can look like widespread “missing” drift. This sprint aligns those commands with the **same resolved active set** when the caller passes the same roots and flags.

---

## Sprint goal

Ship **observability parity**: `podbay diff`, `podbay ps`, and `podbay explain` (contract mode) accept the **same optional service roots and `--dependents` flag** as `validate` / `deploy`, using one **spec-level** resolution helper so the expected service set cannot drift. Default with **no** roots remains **full profile-active** (today’s behavior). **Receipt pair diff** stays unchanged.

---

### What happened

We shipped **one spec helper** (`spec.ObservabilityActiveServices`) and threaded it through **`internal/diff`**, **`internal/ps`**, and **`internal/explain`** so the **expected** service slice cannot drift from **`validate` / `deploy`**. Contract **`diff`** gained partial roots + **`--dependents`**; **receipt pair** mode is preserved by treating **two args as receipts only when both files decode as deploy receipts** (avoiding the old “always two-arg receipt” collision with `diff path svc`). **`ps`** and **`explain`** match the same positional rules as deploy/validate; **`explain`** dropped the separate `loadContractForExplain` / second-positional “focus” path in favor of **partial roots**—dependency context now appears when **partial selection collapses to exactly one** service (preserving the common `explain … web` shape as a single-root partial). **`clijson`** gained **`FromDiffWithPartial`**; **`ps` / `explain` JSON** carry **`deploy_services`** / **`dependents_expand`** when applicable.

# Sprint 18: Bidirectional `dependencies` / `depends_on`, `--dependents`, partial deploy semantics

2026-05-12

**Repo:** Podbay (Go tree at monorepo root).

**Carries from Sprint 17:** Sprint 17 shipped partial deploy with **`ExpandRedeployPeers`** (one-hop **parent** pull when a child is an explicit CLI target). Sprint 18 **replaces** that default expansion model, tightens the contract, and adds **`--dependents`** for **transitive downstream** expansion.

---

## Sprint goal

1. **Contract:** For every edge **X** `depends_on` **Y**, **Y**’s YAML **`dependencies`** (`RedeployPeers`) must list **X**, and every **`dependencies`** entry must reverse-match **`depends_on`**. **`podbay validate`** enforces both directions.
2. **Partial deploy / validate:** Explicit CLI targets alone define the default active set (**no parent pull**). Optional **`--dependents`** adds the **transitive closure** of services that `depends_on` any service already in the working set (BFS downstream). Full deploy (no service names) unchanged.
3. **CLI:** Flag name **`--dependents`** only (no `--dependencies` deploy flag).
4. **Compose:** Import derives **`dependencies`** where needed; emit round-trips bidirectional consistency.
5. **Docs / examples / release:** Breaking change called out; fixtures and example YAML updated.

---

### What happened

We shipped a **coherent partial-deploy model**: **`ExpandDependentsTransitive`** replaced Sprint 17’s default **parent-pull** expansion; **`podbay validate`** now rejects half-specified graphs (**`dependencies_missing_inverse`** and stray inverse entries) so every child→parent edge is mirrored on the parent’s **`dependencies`** list. **`podbay deploy`** / **`validate`** share the same **`--dependents`** flag; **`clijson`** exposes **`dependents_expand`** when partial roots use that flag. **`podbay import compose`** derives **`dependencies`** so typical imports validate without hand-editing; docs, **RELEASES**, and **examples** (including **flowboard** edges through **caddy**) were brought in line. The work stayed **serial by design** (spec → compose → validate → deploy → CLI → JSON → docs → exit bar) so the repo never sat in a long-lived inconsistent half-migration.

# Sprint 17: Partial `podbay deploy` / `validate` by service name

2026-05-12

**Repo:** Podbay (Go tree at monorepo root).

**Carries from Sprint 16:** **`podbay import compose --json`** gave agents stable failure codes on import; **`deploy`** and **`validate`** still assumed **full profile-active** stacks—no explicit **service roots** at the CLI for targeted redeploys.

---

## Sprint goal

Ship **`podbay deploy`** (and matching preflight / **`podbay validate`**) so callers can pass **one or more service names** after the contract path is resolved. The effective set starts from **explicit roots** (deduped) within the **profile-active** subset; **`depends_on`** orders the active set and (on deploy) **pre-waits** on deps outside the set—it does **not** add parents unless **`dependencies:`** expansion applies. Omitting names keeps **full-stack** behavior.

---

### What happened

We shipped partial **`podbay deploy`** / **`podbay validate`** with explicit service roots, external **`depends_on`** pre-wait on deploy, and optional parent **`dependencies:`** (**`RedeployPeers`**) for **one-hop** co-redeploy when a **listed dependent** is an explicit CLI target—tightened in Sprint 18 after operator confusion around **`depends_on`** vs **`dependencies`**. **`spec.ServicesForDeployTargets`**, **`ExpandRedeployPeers`** (removed in Sprint 18), **`TopologicalOrderSubset`**, **`deploy.Options.DeployServices`**, matching validate outcomes, CLI positionals, **`clijson`** **`deploy_services`**, tests, and **README** / **RELEASES** / **PRD** updates shipped; **`tools/run-sprint-tests.sh`** was updated or verified as needed.

# Sprint 16: `import compose` machine-readable failures

2026-05-12

**Repo:** Podbay (Go tree at monorepo root).

**Carries from Sprint 15:** Local Compose **`include:`** merge shipped; agents and CI still lacked a **first-class import gate** comparable to **`validate --json`** for parse/include failures.

---

## Sprint goal

When **`podbay import compose`** cannot produce a contract—**parse errors**, **`include:`** resolution failures (cycle, depth, escape, unsupported shape), or **import-time rejections**—callers must be able to consume **stable issue codes** and a **structured `--json` envelope** aligned with existing CLI JSON patterns (`validate`, `diff`), without scraping human-oriented stderr. Successful import behavior stays unchanged.

---

### What happened

Sprint 16 delivered **failure-only JSON** for **`podbay import compose --json`**: **`composefile.ImportFailure`** carries stable codes through **`Load` / `Parse` / `include:`**; **`clijson.FromImportComposeError`** maps those (and a generic **`import_contract_error`** fallback) into the shared **`Document`** shape; the CLI prints one JSON object to **stdout** and exits **1** on failure, matching the README contract. **`go run ./cmd/podbay`** subprocess tests cover missing file, include cycle, depth, path escape, bad YAML, and unsupported URL include. **`./tools/run-sprint-tests.sh`** was green at closeout.

# Sprint 15: Compose `include:` (documented subset)

2026-05-12

**Repo:** Podbay (Go tree at monorepo root).

**Carries from Sprint 14:** **`v2026.5.0`** public-preview baseline (trust docs, root Go CI, verification script, release notes) closed; Sprint 15 returns to **Compose parity depth** with **`profiles:`** already modeled—**`include:`** was the next merge slice.

---

## Sprint goal

Add **Compose `include:`** support for a **narrow, documented subset** so importing or hand-authoring stacks that split definitions across files produces a **single flattened Podbay contract** (or a **clear import/validate failure**). **`podbay validate`**, **`deploy`**, **`diff`**, and **`--json`** must stay consistent with the supported semantics.

---

### What happened

Sprint 15 delivered **local-file Compose `include:`**: **`composefile.Load`** resolves and merges includes (relative paths only, bounded depth, cycle detection, path escape rejection) **before** **`ResolveExtends`**, matching the README v1 subset. **`examples/compose-include/`** and **`TestImportComposeIncludePipelineValidate`** give a demo and CI-friendly proof without Podman. **`tools/run-sprint-tests.sh`** now runs **`gofmt -l .`** at the Podbay repo root before **`go vet`** / **`go test`**, aligning public verification with **`.github/workflows/go.yml`**.

# Sprint 14: OSS release readiness

2026-05-11

**Repo:** Podbay (Go tree at monorepo root).

**Carries from Sprint 13:** External networks and prior Compose/runtime depth existed, but **OSS adoption blockers** (license, security/contributing docs, root CI, honest shipped-vs-planned story) still blocked a credible public preview.

---

## Sprint goal

Make Podbay credible to publish as **`v2026.5.0`**: the first public May 2026 release—usable, secure enough to evaluate, easy to verify from a fresh clone, honest about what is shipped, and clear that the contract is not yet 1.0-stable. Close the next-30-day OSS readiness gaps before adding more Compose parity or runtime surface area.

---

### What happened

Sprint 14 converted the OSS-readiness review into a concrete public-preview baseline: Apache-2.0 **`LICENSE`**, **`SECURITY.md`**, **`CONTRIBUTING.md`**, a minimal issue template, root Go CI, contributor-friendly **`tools/run-sprint-tests.sh`** behavior (Podbay Go unconditional), **`v2026.5.0`** stability framing in **PRD** / docs, **`RELEASES.md`**, **`examples/ci-receipt-demo.sh`**, inspection artifacts, and sprint proof.

# Sprint 13: External networks (scoped `external:`)

2026-05-11

**Repo:** Podbay (Go tree at monorepo root).

**Carries from Sprint 12:** Compose **`extends:`** / **`x-*`** import parity landed; contracts could still not honestly attach to **pre-existing Podman networks** for a documented **`external:`** subset.

---

## Sprint goal

Support a **documented subset** of Compose-style **`external:`** networks so imported or hand-written **`podbay.yaml`** either **attaches services to pre-existing Podman networks** as declared or **fails at import or validate** with actionable errors—**no** silent join to the wrong bridge. **`podbay validate`** / **`validate --json`** must match deploy semantics for the supported subset.

---

### What happened

Shipped **Compose `external:` → contract → validate → deploy → teardown** for a documented subset: external networks must **pre-exist** in Podman; **`podbay down`** does **not** remove them. Import maps **`name:`** and **`external: { name: … }`**.

# Sprint 12: Compose parity v4 (`extends:` + `x-*` extensions)

2026-05-11

**Repo:** Podbay (Go tree at monorepo root).

**Carries from Sprint 11:** Multi-internal **Podman** networks and validate/deploy alignment shipped; **`extends:`** and **`x-*`** Compose patterns still blocked many real imports.

---

## Sprint goal

Extend **`podbay import compose`** so common **Compose v3-style inheritance and extension fields** are handled for a **documented subset**: resolve or merge **`extends:`** into a single effective service, and define behavior for **`x-*`** keys so real-world Compose files import without hand-merging YAML. Imported **`podbay.yaml`** must stay **`spec.Load`‑compatible**, flow through **`validate` / `validate --json`**, and **fail closed** when the graph exceeds the supported subset.

---

### What happened

Shipped **`extends:`** resolution at **`composefile.Load`**: same-file and cross-file references, with top-level **`networks:`** / **`volumes:`** / **`configs:`** / **`secrets:`** merged from referenced files when missing in the primary document; **cycles**, **max depth (16)**, and **URL `file:`** fail clearly. **`x-*`** and unknown keys are **ignored** by decoding (README). **`TestImportComposeExtendsValidate`** covers import → **`spec.Load`** → **`validate`**. Same serial-PIN / coordination-merge pattern as Sprint 11–13 retros on stats and pytest cost.

# Sprint 11: Multi-network runtime (scoped Podman topology)

2026-05-11

**Repo:** Podbay (Go tree at monorepo root).

**Carries from Sprint 10:** Receipt–receipt **diff v2** landed; **`deploy`** still **flattened multi-network contracts** to a single project bridge in too many cases.

---

## Sprint goal

Make **`podbay deploy`** honor **`networks:`** in the contract for a **defined subset** of topologies so contracts are not silently flattened to a **single project bridge** when they declare **multiple attachable networks**. **Validate** (human + **`--json`**) must agree: supported graph or **actionable failure**—no quiet mismatch.

---

### What happened

Shipped **scoped multi-network** behavior: per-logical Podman bridges **`podbay_<project>_<key>`**, **`deploy`** attaches each service per **`services.*.networks:`**, **`validate`** rejects unsupported drivers and missing membership when multiple networks exist, **Compose import** carries **split** stacks Sprint 8 used to reject, and **`teardown`** removes project logical networks when **`networks:`** is set. **Podman-backed** tests skip when **`podman`** is absent.

# Sprint 10: Richer receipt–receipt diff (v2 comparable fields)

2026-05-10

**Repo:** Podbay (Go tree at monorepo root).

**Carries from Sprint 9:** Compose **`configs:`** / **`secrets:`** (+ optional Vault) import shipped; receipt pairs still compared only a **minimal v1** field set.

---

## Sprint goal

Extend **`podbay diff <receipt-a> <receipt-b>`** so the comparison catches **more of what actually changed between deploy truths** than Sprint 7’s minimal v1. Add a **documented v2** comparable set—prioritize **per-service environment** and **volume / mount identity**—with **human** and **`--json`** output from **one internal model**, stable **issue codes**, and **additive** JSON / **`format_version`** discipline.

---

### What happened

Delivered **v2 receipt diff**: optional **env** and **mount** snapshots on **`ServiceRecord`**, comparison with stable codes and **`warn`**-level **`receipt_diff_*_incomparable`** when only one side recorded env/mounts (no false drift on legacy receipts), **additive** **`clijson`** fields (**`receipt_pair_diff_version`**, **`env_value_display_policy`**), default JSON env redaction plus **`--receipt-diff-show-env`**. Nine sequential feature merges plus README follow-up; **`go test ./...`** green at closure.

# Sprint 9: Compose parity v3 (`configs:` / `secrets:` + optional Ansible Vault)

2026-05-10

**Repo:** Podbay (Go tree at monorepo root).

**Carries from Sprint 8:** Compose parity **v2** (**`networks:`**, long-form **`ports:`**) shipped; **`configs:`** / **`secrets:`** and optional **Vault** materialization were still out of scope.

---

## Sprint goal

Extend **`podbay import compose`** so common top-level **`configs:`** and **`secrets:`** (and per-service references) map into the contract as **file-backed mounts** with **`spec.Load`‑compatible** output, flowing through **`validate` / `validate --json`**. **Ansible Vault** is an **optional** decrypt path at deploy time via **`ansible-vault`**; missing Ansible or password yields **clear, actionable errors**—no Podbay-specific crypto format.

---

### What happened

Shipped **Compose parity v3**: **`composefile`** models **`configs:`** / **`secrets:`**; **`composeimport.ToContract(f, composeDir)`** resolves **`file:`** paths relative to the Compose directory, appends read-only binds, detects Vault ciphertext, sets **`spec.Service.AnsibleVaultPaths`**; **deploy** uses **`ansible-vault view`** into **`0600`** temps with safe cleanup; tests across **`composefile`**, **`composeimport`**, **`deploy`**; **README** documents parity, unsupported cases, and optional Ansible.

# Sprint 8: Compose parity v2 (networks + port forms)

2026-05-10

**Repo:** Podbay (Go tree at monorepo root).

**Carries from Sprint 7:** Receipt–receipt **`diff`** shipped; Compose import still hard-failed or dropped too much on **`networks:`** and long-form **`ports:`**.

---

## Sprint goal

Extend **`podbay import compose`** so more real-world Compose files import without hand-editing: **top-level `networks:`** becomes representable, and **`ports:`** accepts **long-form** mappings—not only short **`host:container`** strings. Emitted **`podbay.yaml`** must remain **`spec.Load`‑compatible**, flow through **`validate` / `validate --json`**, and stay honest about runtime gaps (e.g. single project bridge) rather than silent wrong deploys.

---

### What happened

Shipped **Compose parity v2**: long-form **`ports:`** normalized to short strings (reject **`mode: host`** / missing **`published`**); structured **`networks:`** and **`services[].networks`**; **`ToContract`** fills **`spec.Contract.Networks`** for a single shared logical network and fails on **external** or **split** multi-network graphs; **`TestImportComposeNetworksAndLongPortsValidate`**; **README** documents supported vs unsupported import vs deploy networking.

# Sprint 7: Receipt–receipt diff (`diff <receipt> <receipt>`)

2026-05-09

**Repo:** Podbay (Go tree at monorepo root).

**Carries from Sprint 6:** Machine-readable **`teardown` / `down`** shipped; comparing **two deploy receipts** for drift was still deferred.

---

## Sprint goal

Ship **comparing two deploy receipts** so CI and agents can detect **what changed between two recorded deploy truths**—**human output by default** and optional **`--json`** using the same **`clijson`** spirit as contract **`diff --json`**: stable **`format_version`**, parseable **issues** with stable codes, and **documented exit codes**.

---

### What happened

Delivered **`podbay diff <receipt-a> <receipt-b>`** (human + **`--json`**) via **`receipt.CompareReceipts`**, **`receipt.FormatReceiptDiff`**, and **`clijson.FromReceiptDiff`** with additive **`receipt_pair`** on **`kind: diff`**, **`format_version: 1`**, and stable **`receipt_diff_*`** codes. Contract **`diff`** (zero or one path) unchanged. **README** documents both modes, exit codes, and CI gates; tests in **`internal/receipt`**, **`internal/clijson`**, **`cmd/podbay`**.

# Sprint 6: Machine-readable teardown (`down` / `teardown --json`)

2026-05-09

**Repo:** Podbay (Go tree at monorepo root).

**Carries from Sprint 5:** **`podbay diff --json`** shipped; **teardown** paths were still text-only for automation.

---

## Sprint goal

Ship **structured, versioned JSON** for **`podbay teardown`** and **`podbay down`** (same behavior; two entrypoints) so CI and agents can **bring a stack down** with the same **`clijson`** envelope style as **`validate`**, **`deploy`**, and **`diff`**: stable **`format_version`**, parseable **issues** with stable codes, and **documented exit codes**. Human output stays the default.

---

### What happened

Delivered **`teardown --json`** and **`down --json`** sharing **`newTeardownLikeCmd`**, **`teardown.Execute`** with **`Quiet: true`** in JSON mode, **`clijson.FromTeardown`** / **`TeardownLoadError`** with **`kind: teardown`**, **`format_version: 1`**, mirrored text exit semantics (including **network rm warning → exit 0** with **`teardown_network_warning`** in **`issues[]`**), and stable **`teardown_*`** fatal codes. **README** CI path and **`cmd/podbay/teardown_json_integration_test.go`** (no live Podman).

# Sprint 5: Machine-readable drift (`diff --json`)

2026-05-08

**Repo:** Podbay (Go tree at monorepo root).

**Carries from Sprint 4:** **`podbay import compose`** migration path shipped; **ongoing** contract↔runtime checks still required scraping **`diff`** text.

---

## Sprint goal

Ship **structured, versioned JSON** for **`podbay diff`** so CI and agents can detect **contract ↔ runtime drift** without parsing text. Human **`diff`** stays the default; **`--json`** uses a stable envelope aligned with **`validate` / `deploy` / `receipt`** (`format_version`, minimal v1 fields).

---

### What happened

Shipped **`podbay diff --json`** with **`DriftResult`**, **`clijson` `KindDiff`** / **`FromDiff`**, mirrored exit codes, single source of truth for drift facts, integration tests around **`emit*`** helpers, and **README** CI gate. Retro noted **per-PIN `lines_added: 0`** artifacts when transitions ran after merges—real diff ~1.8k insertions across the sprint.

# Sprint 4: Compose ingestion and migration parity

2026-05-08

**Repo:** Podbay (Go tree at monorepo root).

**Carries from Sprint 3:** **`validate` / `deploy` / `receipt` JSON** shipped; repos still lacked a **credible Compose → Podbay** path without hand-translating full stacks.

---

## Sprint goal

Ship a **credible Compose → Podbay migration path** so repos (and tooling that emits Compose) can reach **`podbay validate` / `deploy` / JSON outcomes** without hand-translating full stacks—**`podbay import compose`**, a **defined v1 subset**, validation hook, proof, and **README** migration section.

---

### What happened

All seven planned units shipped: **`internal/composefile`**, **`composeimport.ToContract` / `MarshalContract`**, **`podbay import compose`** (stdout default, **`-o`**), integration through **`validate.NewRunOutcome`** and **`clijson.FromValidate`**, README migration and v1 parity notes; **`go test ./...`** green on Podbay; **`project.active_sprint`** cleared until the next sprint. 

# Sprint 3: Machine-readable validate, deploy, and receipts

2026-05-08

**Repo:** Podbay (Go tree at monorepo root).

**Carries from Sprint 2:** **`ps`**, **`logs`**, and **`down`** shipped; **validate**, **deploy**, and **receipt** paths still lacked a **shared versioned JSON envelope** for CI and agents.

---

## Sprint goal

Ship **structured, versioned JSON** for **`podbay validate`** and **`podbay deploy`** (failures first; optional compact success metadata where useful) plus a **minimal receipt read path** for agents (`--json`, stable envelope), closing the operator loop for parseable outcomes.

---

### What happened

*(No `## Retrospective` / “What happened” block in `sprint-3.md`.)* Per plan: shared **`clijson`** envelope, **`validate --json`**, **`deploy --json`**, **`receipt` + `--json`**, repo-wide proof, and **README** CI/agent path beside the operator demo.

# Sprint 2: Operator observability (`ps` + `logs`)

2026-05-08

**Repo:** Podbay (Go tree at monorepo root).

**Carries from Sprint 1:** Podbay **Phase 0** still needed first-class **`ps`** / **`logs`** (and **`down`**) in contract vocabulary.

---

## Sprint goal

Ship first-class **`podbay ps`** and **`podbay logs`** so operators and agents stay inside the **project / service / profiles** vocabulary instead of raw **`podman`** for inventory and diagnostics—plus **`podbay down`** as a **teardown** alias and proof/docs.

---

### What happened

All eight tasks released on Podbay: **`podbay ps`** (table + **`--json`** with **`format_version`**), **`podbay logs`** with **`--follow`**, **`--tail`**, **`--since`**, **`podbay down`**, **`internal/ps`**, **`runtimestate`**, **`runner.Logs`**, and **README** operator flow; **`go test ./...`** green.

# Sprint 1: Pinion bootstrap

2026-05-08

**Repo:** Go tree at the monorepo root.

**Carries from prior work:** *(none in this sequence — bootstrap sprint.)*

---

## Sprint goal

Bootstrap Pinion.

---

### What happened

**Complete** per sprint file.

