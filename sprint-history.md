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

