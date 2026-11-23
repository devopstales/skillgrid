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
