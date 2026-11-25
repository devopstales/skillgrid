# Project Log

Append-only timeline. Format: `## [YYYY-MM-DD] <op> | <title>`. Newest last.

## [2026-11-22] ingest | projectwiki seeded (onboarding reconciliation)

Seeded from detected repo facts + the `2026-11-22-pi-worktree-plugin`
change briefing. State drift repaired: `state.yaml` was absent while
`change-specs/2026-11-22-pi-worktree-plugin/briefing.md` existed.

## [2026-11-22] create | ADR-0001 adopt-intent-driven-template-patterns

Adopted the `.agents/` shared layout (4 role files: `senior-dev`,
`senior-qa`, `adversarial-author`, `adversarial-reviewer`), the
`adversarial-authoring` skill (2-stage subagent round + `*.council.md`
audit trail), and the `grill-me` skill (5-line stress-test). Not
adopted: multi-CLI parity matrix, `spec-as-source` layer,
`c4-diagrams`, `glossary`, `gherkin-authoring`, `openspec-*`
lifecycle skills (all with revisit conditions in the ADR). Files:
`.agents/agents/{senior-dev,senior-qa,adversarial-author,adversarial-reviewer}.md`,
`.agents/skills/{adversarial-authoring,grill-me}/SKILL.md`,
`INSTALL_TEMPLATE.md`, `decisions/04-adr-0001-*.md`.

## [2026-11-22] create | ADR-0002 git-discipline-as-separate-skill

Adopted a separate `git-discipline` skill (10 phase-transition git
state gates G1–G10 with deterministic checks + prescribed phrasing),
not folded into `_shared/work-unit-commits`. The commit contract stays
focused on commit format; the phase gates are first-class: announced,
measured, independently invokable. Files:
`.agents/skills/git-discipline/SKILL.md`,
`decisions/04-adr-0002-*.md`.

## [2026-11-23] reconcile | pi-worktree-plugin state caught up to artifacts (blueprint → qa)

`state.yaml` said `phase: blueprint`, but the artifacts show execution is complete: blueprint approved (5-round goldfish, `review-ledger.md`), `acceptance.feature` + `tasks.md` sliced, all 11 tickets published to Backlog.md and marked Done (2963747), code built and committed in the standalone plugin repo at `.worktree/pi-worktree-wf_t5` (branch `main`, HEAD `f548ef8`, clean tree). Artifacts win per the resume rule; phase patched to `qa` and the execution ledger materialized at `.skillgrid/work/2026-11-22-pi-worktree-plugin/progress.md` (the missing Phase 3 output + canonical execution signal). Next: `skillgrid:test` (adversarial audit). Open divergence to confirm at the check-in: the recorded build-location decision (`plugins/pi/pi-worktree` inside skillgrid-skills) vs. the actual build location (`.worktree/pi-worktree-wf_t5`).

## [2026-11-23] create | mnemonic-core knowledge pages (sources, concepts, gotcha, entity, architecture, synthesis, preferences, glossary)

Filed the `2026-11-23-mnemonic-core` design (change spec in
`.skillgrid/change-specs/`, briefing drafted after the 2026-11-23 programmer
interview): predecessor studies `sources/skillgrid-mnemonic-v2.md` +
`sources/engram.md`; design concepts `concepts/in-file-vector-layer.md`,
`concepts/single-handler-two-adapters.md`, `concepts/degradation-floor.md`;
the FTS5 upsert trigger trap `gotchas/fts5-external-content-upsert.md`; the
planned tool `entities/mnemonic.md`; draft module map + data model
`architecture/mnemonic-memory-core.md`; design hub
`syntheses/mnemonic-core.md`; programmer build preferences
`preferences/build-preferences.md`; term cluster
`glossary/02-technical-terms.md`. `Degraded: projectwiki` — the installed
`mnemonic` binary (v2) has no `projectwiki` command, so `index.md` was
hand-updated to match the pages; run `projectwiki index` + the gate
(validate/lint/status) when the engine is available.

## [2026-11-23] create | ADR-0003…0011 mnemonic-core locked decisions

Locked in the 2026-11-23 interview, authored from the `madr-minimal`
template (`.agents/skills/_shared/adr-records/`, `adr_style` per config):
0003 per-project sqlite file, 0004 vectors-in-file (BLOB+vec0+brute floor),
0005 C/CGo-free target build (pure-Go ONNX, spike-gated), 0006 local ONNX
default embedder, 0007 single-handler-two-adapters (K6), 0008
memory-core-only v1, 0009 WAL-only concurrency, 0010 multi-tool binary
(`mnemonic <tool> <verb>`), 0011 degradation contract. Files:
`decisions/04-adr-000{3..11}-*.md`. ADR index rows added to
`ASSUMPTIONS.md` § LOCKED (+ VERIFIED active-change line, +2 Open
Questions); per-change manifest at
`.skillgrid/change-specs/2026-11-23-mnemonic-core/adr.md`.

## [2026-11-23] create | ADR-0012 go-module-path + research errata

`findings.md` (Research §, 2026-11-23): the ADR-0004/0005 spike-target
modules were wrong — `unum-cloud/onnx` 404s; the CGO-free stack is
`knights-analytics/hugot` Go backend (GoMLX, explicitly designed for
all-MiniLM-L6-v2) + `ncruces/go-sqlite3` + `sqlite-vec-go-bindings/ncruces`
(vec0, FTS5, no CGO); MiniLM ONNX = HF `onnx/model.onnx` 86.3 MiB
apache-2.0. Created `decisions/04-adr-0012-go-module-path.md` (module path
`github.com/devopstales/skillgrid/mnemonic`, same as the old module —
branch handover `release/2` → `feat/v2-rewrite`; no cross-branch imports
constraint) + LOCKED row + Open-Questions replacement. Errata (factual
module-identity corrections, ADRs untouched per IRON RULE):
`concepts/in-file-vector-layer.md` (vec0 binding),
`concepts/degradation-floor.md` (ONNX runtime row). `Degraded: projectwiki`
(engine down — index hand-kept).

## [2026-11-23] interview | mnemonic-code brainstorm: frontier closed (6 decisions)

Ran the `skillgrid:brainstorm` (New Function) interview for the `code` tool
via `skillgrid:questioning`. Resolved: Q1 GitNexus-core 7-tool surface
(`code_search`, `code_context`, `code_impact`, `code_path`, `code_explain`,
`code_index`, `code_status`); Q2 pure-Go tree-sitter — the programmer
overrode the CGo-build-tag option and pointed at `odvcencio/gotreesitter`,
which IS the pure-Go runtime (the predecessor's v0.55.1 binding) — no build
fork needed, ADR-0005 holds; Q3 chunk + symbol embeddings (the predecessor's
two-table model, over the chunks-only recommendation); Q4 code files only
(docs/configs stay out of the index); Q5 core + LSP pass + watch
(predecessor-lite machinery); Q6 knowledge base = in-file graph + hybrid
search, generated docs deferred (consistent with ADR-0008 + Q1). Files:
`glossary/02-technical-terms.md` (code-tool term cluster),
`decisions/0013-pure-go-treesitter.md` +
`decisions/0014-unified-embedding-space.md` (renumbered from 0012/0013 —
`04-adr-0012-go-module-path.md` from the parallel mnemonic-core session took
0012) + LOCKED rows, the `2026-11-23-mnemonic-code` state entry (phase
intake). The code tool's briefing inherits the corrected CGO-free stack from
the parallel session's `2026-11-23-mnemonic-core/findings.md` (hugot GoMLX
backend, ncruces/go-sqlite3, sqlite-vec-go-bindings/ncruces). Next: Goldfish
shootdown readiness gate → 7 fresh lenses → ADR manifest + briefing.

## 2026-11-23 — ADR-0015: CGO-free stack selection (mnemonic-core)

Vector-stack gate resolved (programmer chose option A after the
"what did the old systems use?" check — both old generations ran
brute-force cosine over BLOBs in a pure-Go driver). Filed
`decisions/04-adr-0015-cgo-free-stack-selection.md` (accepted):
`ncruces/go-sqlite3` + `sqlite-vec-go-bindings/ncruces` (tag `sqlitevec`)
+ `knights-analytics/hugot` GoMLX backend (tag `onnx`), `-tags ORT` escape,
brute-force BLOB floor retained (falsifier (b)). Refines ADR-0004/0005/0006;
supersedes none. ADR-0013/0014 are taken by the parallel
`2026-11-23-mnemonic-code` session (treesitter / unified space) — hence 0015.
LOCKED row + `index.md` catalog row added; the "Embedding + vec stack
selection" open question replaced by the code-tool ANN-scale question.

## 2026-11-23 — Phase-4 spikes run (mnemonic-core): both PASS, ADR-0015 amended

Opt-in "now" gate accepted. Spike 1 (MiniLM on hugot GoMLX): PASS — dim 384,
cross-backend cosine(GoMLX, ORT) = 1.000000 (independent ORT 1.22.0
reference), p50 81 ms/sentence non-simd (target ≤ 3 s), semantic sanity
0.5842 vs 0.0511. Spike 2 (vec + FTS5 in one CGO-free build): PASS but the
**asg017 sqlite-vec ncruces binding proved unpairable** (wasm/host-table
drift across v0.15.0/v0.17.1/v0.38.0, live-tested) — the working mechanism
is ncruces v0.38.0's official **`ext/vec1`** + `ext/fts5`, registered
per-connection, verified in one session. ADR-0015 (uncommitted at the time)
amended: Decision Outcome → `ext/vec1`, spike record added, binding moved
to Not-adopted; LOCKED row + index row updated; `findings.md` § Spike
Results records the tables + blueprint impact. Falsifiers (a)/(b) not
triggered.

## [2026-11-23] ship | pi-worktree plugin — Phase 7 editorial package + archive

The `@devopstales/pi-worktree` plugin (the worktree-standard reference
implementation) reached **PASS** at the qa-verdict gate and shipped to
`change-archive/2026-11-22-pi-worktree-plugin/`.

- **Gate decision (A=wire, E=fix, bench=accept):** the three core invariants
  (R7 `applyMergeDefaults`, R8 `checkBranchAllowList`, R9 `observeWorktreeRoot`
  via `isWorktreeContext`) are wired into `runWorktrunk`; the R10 wt-approval
  flow is fixed (probe `wt hook <phase>` for the piped exit-0 case, store under
  a PTY via `script -qec`, core `handleWtApproval` ask→store→retry-once). Both
  are in plugin commit `f1b58b8` on top of the 11-ticket build `f548ef8` +
  fix-pass `9376e19`.
- **Evidence:** `bun test` 48/48, `e2e` 6/6, trivy 0, jscpd 1.66% (advisory),
  drift guard green. Score **8.5 / target 8.5 (grade A)**, floor
  met-with-fixes (3 gotchas deferred as DW with re-entry paths).
- **3 durable gotchas** reflected to `gotchas/`: git C-quoting of non-ASCII
  paths, wt's piped non-interactive runtime, and `write-tree`'s cache-tree
  index side effect.
- **Rollback:** `git reset --hard f548ef8^` in the plugin repo (the plugin is a
  standalone repo under `.worktree/`); spec-zone = revert the skillgrid commits.

## 2026-11-23 — Brainstorm complete: Goldfish shootdown + briefing (mnemonic-code)

All 7 Goldfish lenses completed (fresh `reviewer` subagents, R37 isolation,
Mode BRAINSTORM): 3 🔴 / 38 🟡 / 13 🟢 raw → deduplicated to 16 🟡 groups +
🟢 nits. Blockers all resolved at user gates: B1 (model swap strands corpus)
→ two-phase model stamp + coverage backfill; B2 (first index hours) →
batch-32 backfill + synchronous structural phase / background embedding
(spike-corrected: 81 ms/sentence non-simd → ~54 min for 40k items, not the
original 5.5–22 h estimate); B3 (tier degradation invisible) → persisted
per-language tiers + span-free uids + counted cascade. Q5 machinery
re-confirmed after evidence: LSP **keep** (real JSON-RPC client in v1,
despite the prod-stub finding), watch **keep** with fixes (hash-confirm,
death reporting, non-blocking re-index, re-embed on change). Wrote
`change-specs/2026-11-23-mnemonic-code/briefing.md` (STATUS `draft`,
R1–R13 with acceptance scenario pointers, approaches A–E, pre-mortem,
timebox with query-core shrink slice, 16 🟡 + nits recorded) and `adr.md`
manifest (in-force: 0001/0002/0003–0011 + 0015 + 04-adr-0012; new durable:
0013, 0014; supersessions: none). Inherited verified stack from
`2026-11-23-mnemonic-core/findings.md` (hugot v0.8.6 GoMLX, ncruces v0.38.0
+ ext/vec1 + ext/fts5, go ≥ 1.27; asg017 vec0 binding not adopted).
Open questions for the blueprint gate: v1 language set, gotreesitter pin,
memory-core tag/commit pin, two-domain model eval (at arrival), skill-flip
spec, code-domain latency SLO. Note: questioning exit-check clarity scores
were not recorded; the clarity gate passed by user confirmation at the
interview close + readiness gate. Next: user approves the briefing
(hand-edit STATUS → `approved`) → phase intake→blueprint.

## [2026-11-25] create | ADR-0016 skill-registry-index-location

Pinned the skill registry index location from the 2026-11-25 gentle-ai
study: `.skillgrid/projectwiki/skill-registry.md` (not gentle-ai's
`.atl/` managed dir — skillgrid keeps every durable artifact under
`.skillgrid/`). The registry pattern (index of name, full description,
scope, exact `SKILL.md` path; subagent handoffs pass exact paths, "delegators
pass paths, not digested rules") is adopted; the refresh/generator and the
handoff contract belong to the implementing change, not this ADR. Files:
`decisions/0016-skill-registry-index-location.md`, `ASSUMPTIONS.md` (LOCKED),
`index.md` (decisions row).
