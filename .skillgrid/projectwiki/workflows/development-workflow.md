---
title: "Development Workflow"
type: BusinessWorkflow
anchors:
  - path: .skillgrid/config.yaml
    all_of: [schema, "skillgrid/v1"]
  - path: .agents/skills/_shared/rules/sdd-structure.md
    all_of: ["Phase Order"]
---

# Development Workflow

## Context

This page is the ordered, step-by-step execution guide for Delivery changes:
for each step, which skill to invoke, which subagents to dispatch, which MCP
tools to use, which files are written, and where the user gates are. It is
the executable complement to the `sdd-pipeline.md` concept page — that page
states the *what and why*; this page states the *how*, top to bottom, for any
Delivery change (new function or new project) in this repo.

**Applies to Delivery changes only.** Specialist and Q&A requests do NOT
enter the pipeline — they run the named specialist skill (or an answer)
directly and stop.

Phase order follows `.agents/skills/_shared/rules/sdd-structure.md`
`## Phase Order` (the only full chain). The workflow relies on these config
facts (`.skillgrid/config.yaml`, schema `skillgrid/v1`):

- `mnemonic.enabled: true` — memory + code-index MCP usage is live
- `ticketing.type: hermes-kanban` — board `skillgrid`, assignee `default`
- `bdd.enabled: true` — stack javascript
- `rules.tiers.default: T2` — rigor tier (fast-track enabled)
- test: `go test ./...`
- build/setup: `go build ./...`
- format: `gofmt -w ./...`
- advisory (non-blocking): trivy, jscpd

## Step 0 — Session start & routing

- **Skill:** `using-skillgrid` (the router). Announce `Using [skill] to [purpose]`.
- **Subagents:** 0.
- **MCP:** `mem_session_start` (once per session; reuse the returned sid for all later `mem_*` calls), `mem_context`/`mem_search` (recall project state before work), `code_status` (index health — if not `fresh`, start `code_index` now so later phases have data).
- **CLI:** `node scripts/phase-router.mjs all` — JSON per in-flight change: `{phase, next, marker, drift, ledger, blocked}`.
- **Files:** read `.skillgrid/config.yaml`, `.skillgrid/meta/state.yaml`, `.skillgrid/projectwiki/ASSUMPTIONS.md` § LOCKED (ADR index — open the linked `decisions/04-adr-NNNN-slug.md` for a decision you touch), AGENTS.md § Rules, and the glossary (`glossary/01-business-terms.md` + `02-technical-terms.md`).
- **Gates:** none (auto-route; exactly one first skill, no catalog scan). Classify the request: Delivery (→ Step 1; bounded paths stop after chat design approval; trivial/small → fast-track) / Specialist (run the named skill directly — out of the pipeline) / Q&A (answer — out of the pipeline).

## Step 1 — Brainstorm

- **Skill:** `brainstorming`.
- **Subagents:** 0.
- **MCP:** `mem_save` (phase-transition upsert, `topic_key: skillgrid/<topic>/briefing`).
- **CLI:** —
- **Files:** `.skillgrid/change-specs/YYYY-MM-DD-<topic>/briefing.md` (line 3 MUST be `> **STATUS:** \`draft\` (YYYY-MM-DD)`) + `adr.md` (per-change ADR Review Manifest, pointers only).
- **Gates:** HARD GATE — no implementation skill, code, or scaffolding until the user approves the briefing; only then → Step 2.

### Step 1a — Classify

Classify the delivery path out loud first: Prototype / Bounded / New Project
/ New Function. One-way ratchet: hidden complexity upgrades the path, nothing
downgrades it.

### Step 1b — Requirements + assumptions

Interview the design tree (agent answer-select tool, one question at a
time); question with existing flows and prior art where the codebase answers.
VERIFIED/INFERRED facts → `ASSUMPTIONS.md`; resolved terms → glossary;
`state.yaml` → `changes.<topic>: {phase: spec, status: in_progress}`. An
advisory state lock guards the topic: `node scripts/state-lock.mjs acquire
<topic> [session-id]` (exit 1 = surface conflict warning).

### Step 1c — ADR manifest / in-force ADRs

Maintain the per-change ADR Review Manifest (`adr.md`, pointers only). The
ADR bar — write an ADR only when all three hold: hard to reverse, surprising
without context, real trade-off. ADR file `decisions/04-adr-NNNN-slug.md`
(frontmatter `status`/`date`/`supersedes`), pointer row in `ASSUMPTIONS.md`
§ LOCKED. Never mutate an accepted ADR — supersede it.

## Step 2 — Blueprint

- **Skill:** `writing-blueprints`. Reads `briefing.md`, `findings.md` (if present), the `adr.md` manifest + in-force ADRs.
- **Subagents:** critic (mandatory T2/T3 — structured critique: issues + severity for architectural gaps, missing edge cases, untested assumptions, scope creep, unclear interfaces) + plan-review (T2+, preferred for 3+ tasks — failure modes, scope coherence, feasibility, ordering, test strategy, ADR compliance; verdict `READY FOR EXECUTION | NEEDS REVISION`). Sequential, two distinct dispatches; in-controller Self-Review is always on and is not a subagent.
- **MCP:** `mem_save` (phase-transition upsert, `topic_key: skillgrid/<topic>/blueprint`).
- **CLI:** —
- **Files:** `.skillgrid/change-specs/YYYY-MM-DD-<topic>/blueprint.md`. Header MUST read `# <Topic> — Implementation Blueprint (steps, files, verification)`; carries Change Classification (`trivial|small|standard|risky|high-risk` → verification floor L1–L4 — this selects what `qa` runs), `SATISFIES:` lines, `> ⚠ one-way:` tags, and commands cited from config for verification.
- **Gates:** user approves the final blueprint — the decide-then-fix loop (present every critic finding with options + benefits + weaknesses; update per decision; re-present). One-way-door checkpoints STOP for explicit approval; the Owed-Decision Gate forces a user pick (resolve / not-a-real-decision / build-on-assumption → `**Status: ASSUMED**`). Handoff choice after approval: Subagent-Driven (→ Step 4 via `slicing` for 3+ tasks, direct for 1–2) or Inline (→ `simple-execution`).

## Step 3 — Acceptance tests

- **Skill:** `acceptance-test-authoring`. Prerequisite: `blueprint.md` exists AND passed the two-stage validation — never write `acceptance.feature` without a validated blueprint.
- **Subagents:** 0.
- **MCP:** —
- **CLI:** `cucumber-js --dry-run` (scenarios present + pending) + gherkin-lint exit 0 (pre-project lint: `extract-gherkin.cjs` then `npx gherkin-lint --config .../gherkin-lintrc.json` on the extracted features).
- **Files:** `.skillgrid/change-specs/YYYY-MM-DD-<topic>/acceptance.feature` — Markdown headings = capability/requirement/scenario; ` ```gherkin ` fences hold steps only. Every requirement carries an RFC 2119 keyword, one happy path + ≥1 edge + ≥1 failure scenario, and a `#### Gates` block: `G<n>` with `CHECK:` (repo command) + `EXPECT:` (success-only marker) + `EVIDENCE: pending`. If `acceptance-tests/` does not exist, scaffold it verbatim from `acceptance-test-authoring/references/javascript/`.
- **Gates:** pre-project lint passes (mandatory before slicing). Zone rule: commit specs before code — the pre-commit zone guard enforces it; never stage spec+code in one commit.

## Step 4 — Slicing + ticketing

- **Skill:** `slicing` (4.1), then `ticketing` (4.2).
- **Subagents:** 0.
- **MCP:** —
- **CLI:** hermes kanban, board `skillgrid` (see 4.2).
- **Files:** `.skillgrid/change-specs/YYYY-MM-DD-<topic>/tasks.md` + tickets published to the tracker.
- **Gates:** 4.3 user gate — the execution approach (Subagent-Driven vs Inline), mandatory, never auto-executed.

### Step 4.1 — Slicing

`tasks.md` from `slicing/templates/tasks.md`: `> **STATUS:** \`sliced\``
banner, epic summary, tickets each with `SATISFIES:` scenario + ASCII-only
title, dependency graph (mermaid), waves, optional per-ticket
`Precondition:` / `Reversibility:` / `Fails-when:` lines, and the
`## Delivery Strategy` section with the 4 literal guard lines:

```
Decision needed before apply: {Yes|No}
Chained PRs recommended: {Yes|No}
Chain strategy: {stacked-to-main|feature-branch-chain|size-exception|pending}
400-line budget risk: {Low|Medium|High}
```

Fast-track changes carry `Fast-track: {trivial|small} (waiver in
briefing.md)`. `state.yaml` → `phase: slicing`. Commit (spec zone).

### Step 4.2 — Ticketing

Publish to the tracker (CLI-only; never hand-edit tracker files). Config:
`type: hermes-kanban`, board `skillgrid`, assignee `default`. In order:
discovery (`hermes --version`; `hermes kanban boards list`; board missing →
`hermes kanban boards create skillgrid`; duplicate check via
`hermes kanban --board skillgrid list`) — **discovery failure (auth,
unresolvable board) = hard STOP**, never continue into publication. Then
publish: the epic, then each ticket with `--parent`, `--skill
subagent-execution`, and a body naming its blockers; verify each with
`hermes kanban --board skillgrid show <task-id>`. Record the `Tracker ID`
line under each ticket in `tasks.md` (the one permitted hand-edit) and
commit.

### Step 4.3 — User gate

Present exactly two options and wait:

1. **Implement** → `subagent-execution` (or `simple-execution`)
2. **Revise** → back to `brainstorming` and/or `writing-blueprints`

## Step 5 — Execution

- **Skill:** `subagent-execution` (inline equivalent: `simple-execution` — same loop, same ledger, no subagents; parallel variant for 2+ independent domains: `parallel-execution` — one subagent per domain in one dispatch, provably-disjoint `Owns:` globs, full `go test ./...` after every wave).
- **Subagents:** implementer per ticket (strictly sequential — max 1 in flight, never parallel implementers) + task reviewer per ticket (spec compliance + quality verdict). Fix rounds get a scoped re-reviewer; rounds 4–5 swap in a fresh implementer on a more capable model.
- **MCP:** —
- **CLI:** test commands (`go test ./...`, scoped subsets via the Code Index Ladder's `code_affected`).
- **Files:** implementation commits + the `.skillgrid/meta/.sdd/<plan>/progress.md` ledger (line 1 MUST be `# SDD ledger — plan: <plan-file-path>`).
- **Gates:** the four stops — everything else is a ruling, not a stall: (1) irreversible/destructive op, (2) security-sensitive action, (3) side effect outside the worktree (merge, push to shared branch, publish), (4) plan so broken every path forward is a guess. Plus: unmet `Precondition:` checkpoints; `Reversibility: one-way` tickets insert a human checkpoint; never start on main/master without consent.

The per-ticket loop (strictly sequential): claim the ticket (`hermes kanban
--board skillgrid claim <task-id>` — atomic) → generate the task brief (the
brief is the single source of requirements — never paste the whole plan) →
dispatch the implementer (TDD: RED — one minimal failing test for the
ticket's `SATISFIES` scenario, run and watch it fail → GREEN — minimal code,
suite green → TRIANGULATE — second test, different inputs → REFACTOR) →
commit per `work-unit-commits` → generate the review package (`git log
--oneline` + `diff --stat` + `diff -U10` from the pre-dispatch BASE, never
`HEAD~1`) → dispatch the task reviewer (returns Spec Compliance
✅/❌/⚠️ + Critical/Important/Minor + quality verdict) → fix loop ≤ 5 rounds
(Minor findings never enter the loop: `Task <N>: minor (deferred):
<one-liner>`) → tracker `request-review` → `complete` (closing commit footer
`Closes <task-id>`) → ledger line (`Task <N>: complete (commits
<base7>..<head7>, review clean)`) + `mem_save` execution-progress → mark
`[x]` in `tasks.md`. The edit-scope guard: a subagent that must touch a file
outside the ticket's `Files` set halts and checkpoints. A bug found
mid-ticket goes to `structured-debugging` (root cause first; 3-fix rule: ≥3
failed fixes → STOP and discuss architecture with the human).

## Step 6 — QA

- **Skill:** `qa`. Coordinator-run gate.
- **Subagents:** 0 (no subagents dispatched).
- **MCP:** —
- **CLI:** full suite + build (`go test ./...` + `go build ./...`), plus the QA scripts (`scripts/traceability.mjs`, `scripts/tdd-evidence.mjs`) and the 9.5 gate chain (`node .agents/skills/qa/scripts/qa-gate.mjs .`).
- **Files:** `report.md` (the QA gate report, from `qa/templates/report.md`).
- **Gates:** the 9.5 deterministic gate chain (chains `state-drift-check.mjs`, `ship-drift-check.mjs check <baseRef>`, `skill-size-budget.mjs check`, `check-adr-invariants.mjs`, `okf-lint.mjs`, `status-banner.mjs check`, `traceability.mjs`, `tdd-evidence.mjs`, `review-ledger-lint.mjs`, `windows.mjs check`, `delivery-guard.mjs`; worst-scope-wins composite) + the four-state verdict `PASS | CONCERNS | FAIL | WAIVED` in `report.md` `## Gate Decision` (`**Verdict:**` line). WAIVED is human-only (named human + recorded risk).

Steps, in order: read the change (`briefing.md`, `acceptance.feature`,
`blueprint.md`, `tasks.md`, `findings.md`) → write the Test Plan into
`report.md` (commit, spec zone) → Goal-Backward Verification (Truth /
Artifact / Key Link / Data Flow; statuses `VERIFIED` /
`PRESENT_BEHAVIOR_UNVERIFIED` / `UNVERIFIED` — the latter routes to human) →
Verification-Gap Audit (regression / missing-adoption / broken-verification /
missing-oracle) → traceability → TDD-Evidence Audit → Assertion-Quality Audit
(tautology / ghost loop / no-code-call = CRITICAL) → Test-Quality Audit
(counter-test + P0 triangulation; cross-check the executed set against
changed-file impact — a changed file with no covered test = verification
gap) → Security Audit (`trivy fs . --scanners vuln --severity CRITICAL,HIGH,MEDIUM,LOW`,
advisory) → Code-Quality Gate (P0 pass 100% / P1 95%; jscpd advisory) →
9.5 gate chain → Render Gate (verdict + open WARNING/SUGGESTION rows to
`WINDOWS.md`; state drift renders as WARNING, never affects the verdict;
commit the QA half, spec zone). Route: PASS → Step 7; CONCERNS/FAIL → fix
loop, re-run Steps 3–9; WAIVED → Step 7.

## Step 7 — Code review

- **Skill:** `code-review`.
- **Subagents:** exactly 3 parallel reviewers — Standards (glossary terms + in-force ADRs), Spec (per-scenario `PASS/FAIL/MISSING` + Scope Creep + `Spec met? Yes|No|Partially`), Security (CWE + OWASP category + Confidence per finding) — dispatched in parallel with fresh blind contexts; the coordinator (not a subagent) verifies each finding at `file:line`, dedups, rejects unverified, and renders exactly one verdict per axis. Escalation: diff ≥ 50 changed lines or high-risk (or Tier T3) → the `code-review` fan-out (core five + conditional specialists + red team LAST, sequentially).
- **MCP:** —
- **CLI:** —
- **Files:** `review.md` (review notes, from `code-review/templates/review.md`): `## Passes` (per axis), `## Findings` (Critical/Important/Minor; max 5 Minor per axis), `## Independence` grade table, `## Verdict` — floor across axes `met | met-with-fixes | not met` + `Unfixed Important count (must be 0 to proceed)`. Commit (spec zone).
- **Gates:** user (unfixed Important blocks proceeding).

### Step 7.5 — Review ledger

- **Reference:** `_shared/craft/review-ledger.md`. After EVERY review round, before moving on.
- **Subagents:** 0 (validated later by `scripts/review-ledger-lint.mjs` inside qa-gate).
- **MCP:** —
- **CLI:** —
- **Files:** append to `.skillgrid/change-specs/YYYY-MM-DD-<topic>/review-ledger.md`: `## Round <N> — <YYYY-MM-DD>` + frozen candidate (base/head hashes, file count, `git hash-object` content hash) + reviewer name + independence grade + budget + Findings table `| ID | Severity | File:Line | Finding | Disposition | Fix Commit | Lineage |` (IDs `R<N>-F<NNN>`; dispositions `fixed|parked|waived`) + Dispositions list + Round Verdict (`clean|findings-dispositioned|invalidated`).
- **Gates:** —
- **Commit:** `review: record round <N> (...)` with the `[skillgrid-context]` block — the ledger records findings + dispositions + lineage. A commit landing or file edit mid-review → round `invalidated: <reason>`, re-dispatch on the new candidate. `parked` requires a ruling + named human; `waived` requires named human + accepted risk.

## Step 8 — Receiving code review

- **Skill:** `receiving-code-review`. Per finding: READ → UNDERSTAND → VERIFY → EVALUATE → RESPOND → IMPLEMENT, one at a time, TDD each (failing test first for behavior fixes).
- **Subagents:** 0.
- **MCP:** —
- **CLI:** —
- **Files:** fix commits (triage + fix the in-scope set).
- **Gates:** —
- Triage buckets: Fix now / Defer / Human look / Noise. Any item unclear → STOP, ask before implementing; a conflict with a prior user decision → stop and discuss first. Deferred items → tracker (`hermes kanban create`) or notes in `tasks.md`. Validation after fixes: full `go test ./...` + build, then Step 9.

## Step 9 — Ship

- **Skill:** `ship`. Iron law: no merge/push/move until tests green ON THE INTEGRATED TREE.
- **Subagents:** 0.
- **MCP:** `mem_save` (ship context: `topic_key: skillgrid/<topic>/ship`).
- **CLI:** `git` (test on the integrated tree, env detection, base-branch resolution from the `Chain strategy:` line, then the user-chosen menu: merge locally / push+PR / keep).
- **Files:** the ship commits + `.skillgrid/change-archive/YYYY-MM-DD-<topic>/` (the mechanical move: snapshot to a temp dir → `git mv` the change-specs folder → `diff -r` readback; ONLY empty diff passes; shell-only, never model Read/Write).
- **Gates:** —
- Pre-mutation checks (rulings, not user gates): QA hard gate — read `report.md` `**Verdict:**` before the move (PASS/WAIVED proceed; CONCERNS needs explicit human "ship it anyway"; FAIL/unresolved CRITICAL → `blocked (qa-failed)`); review advisory — `not met` with unfixed Important proceeds only with a human. Then: capture ship context for reflect (incl. ASSUMED-blueprint decision debt), confirm the last work-unit commit carries `[skillgrid-context]`, run the pre-ship vault gate (index → validate --strict → lint), reconcile the vault (same-edit `projectwiki index` after every page write), tracker epic/tickets → done, `state.yaml`: remove the change entry + `progress.completed_changes` +1, release the state lock. Return the ship envelope: GO/NO-GO + gates + evidence + rollback plan + `Next: reflect`.

## Step 10 — Reflect

- **Skill:** `reflect` — the terminal phase, the only one that closes the session.
- **Subagents:** 0 (invokes `environment-retro` as a sub-phase: six categories — Navigation, Automated checks, Coding standards, Steering-file no-ops, Tool economy, Information access; propose-only, never edits code).
- **MCP:** `mem_save` (report: `topic_key: skillgrid/<topic>/report`, type learning) + `mem_session_end` (with `mem_session_summary`; lineage observation IDs recorded from `mem_search`/`mem_get_observation`).
- **CLI:** —
- **Files:** the retro half completed IN PLACE in `.skillgrid/change-archive/YYYY-MM-DD-<topic>/report.md`, from `## Final-State Facts` onward: `## Final-State Facts`, `## Gates`, `## Decisions`, `## Lessons`, `## Patterns`, `## Surprises` (every row sourced: file:line / commit / ticket / scenario), `## Environment Retro`, `## Acceptance Verdict` (`accepted | accepted-with-open-items | rejected`), `## Open Items (→ next change)`, `## Web-cache promotion candidates`, `## Prior-Change Follow-Through`, `## Move Evidence`, `## Overrides / Waivers`. Durable, still-relevant findings are lifted into the vault (ADRs, gotchas, research findings) — every vault page write is followed by `projectwiki index` in the same edit; `index/` projections stay engine-owned. `state.yaml` → `changes.<topic>: {phase: complete, status: complete}`.
- **Gates:** user accepts the verdict (hard preconditions first: folder in `change-archive/` + ship context + `diff -r` empty; QA FAIL → `blocked (qa-failed)`). Web-cache promotion candidates need explicit human confirmation before any write (unconfirmed expire); a `rejected` verdict is surfaced to the human.
- Return: `## Change Closed` (verdict, gates, sourced learnings, open items, lineage observation IDs) + `## Key Learnings`.

## Cross-cutting tool usage

**Memory — Mnemonic MCP (`mem_*`, gated by `mnemonic.enabled: true`):**
`mem_session_start` once at Step 0 → `mem_session_summary` + `mem_session_end`
once at Step 10 (reflect is the only phase that closes the session). Recall
before re-deriving: `mem_context` → `mem_search` → `mem_get_observation(id)`
(the only full-content path). Save on every phase transition: `mem_save` with
`topic_key: skillgrid/<topic>/<artifact>` where artifact ∈ `briefing |
blueprint | findings | tasks | execution-progress | report | review | ship`,
`title == topic_key`, `type: architecture|learning`, `scope: project`. In-repo
files win over the memory mirror on disagreement. Under context pressure
(compaction near): STOP → append position to the ledger/spec zone → work-unit
commit → `mem_save` execution-progress.

**Code Index Ladder (binding order; prefer over grep/rg):**
1. `code_status` — freshness `fresh|lag|empty|unknown`; if not `fresh`, run `code_index` first (async run_id; 409 while active).
2. `code_search` — hybrid search (FTS + structural + semantic) to locate the relevant area; `code_explain` for a known symbol's neighborhood.
3. Neighbor edges / blast radius before changing an exported symbol: `code_get_callers` / `code_impact` (the blast radius must stay inside the task's scope, else halt and checkpoint).
4. `code_read` — exact slice for a path + range ALREADY narrowed by search; never whole-file speculative reads.
`code_affected` (by changed files or a base ref) — which tests to run after a change.

**Knowledge base — projectwiki (vault `.skillgrid/projectwiki/`; `kb` skill):**
Read (cheap, any phase): `projectwiki_query` (stage-1 ranked retrieval) →
`projectwiki_read` (full page / `##` section + pin-check state) →
`projectwiki_search` (raw FTS power path). Write (task bookends — query at
start, file concept at end; never a drive-by write): write the page into
`workflows/ | decisions/ | invariants/ | glossary/ | gotchas/ | architecture/`,
then the SAME-EDIT rule — `projectwiki_generate` (CLI equivalent:
`mnemonic projectwiki index`) in the same edit; `index/` projections are
engine-owned — never hand-edit. Pre-ship no-new-drift gate: `projectwiki
index` → `projectwiki validate --strict` (drift → exit 1) → `projectwiki lint`
(target: clean).

**Commit protocol (work-unit-commits, always on):** conventional subjects
(`feat|fix|docs|...`), no AI-attribution trailers, atomic +
independently-revertable, `[skillgrid-context]` block (`Task: / Decisions: /
Remaining: / Tried: [/skillgrid-context]`), spec-zone commits BEFORE
code-zone (the pre-commit zone guard is fatal), tracker footers `Refs:`/`Closes:`
with parseable IDs, protected-ref guard (never commit to `main`/`master`/
`develop`/`trunk`/`release/*`), worktree branch prefix `agent-*`.

**Resume (any point — after compaction or session restart):**
1. `node scripts/phase-router.mjs all` (or `<slug>`) → derived phase from the deepest artifact (artifacts win over `state.yaml`; drift logged to the index log, phase patched).
2. Read: `state.yaml` → `.skillgrid/meta/.sdd/<plan>/progress.md` ledger (`Task <N>: complete` lines) → the latest commit's `[skillgrid-context]` block.
3. MCP fallback layer: `mem_search("skillgrid/<topic>")` → `mem_get_observation`.
4. Tail markers: `report.md` verdict empty/PENDING → `qa`; verdict filled + `review.md` floor unmet → `code-review`; floor met + still in `change-specs/` → `receiving-code-review` → `ship`; in `change-archive/` with retro empty → `reflect`; retro filled → complete.
5. Multiple in-flight changes, none named → ask the user which to resume (default: most recent).

## Subagent summary

| Step | Phase | Subagents (all `general-purpose`) |
|---|---|---|
| 0 | routing | 0 |
| 1 | brainstorming | 0 |
| 2 | blueprint | critic (mandatory T2/T3) + plan-review (T2+, 3+ tasks) — sequential |
| 3 | acceptance tests | 0 |
| 4 | slicing + ticketing | 0 |
| 5 | execution | implementer per ticket (sequential) + task reviewer per ticket (spec + quality); re-reviewer per fix round |
| 6 | qa | 0 (coordinator-run gate) |
| 7 | code review | exactly 3 parallel reviewers (Standards + Spec + Security) |
| 7.5 | review ledger | 0 |
| 8 | receiving code review | 0 |
| 9 | ship | 0 |
| 10 | reflect | 0 |

Orchestration rules (from `using-skillgrid`): max dispatch depth 1 (workers
never dispatch); the orchestrator collects verdicts and edits only to wire up
worker output; fan-out = verdict on a known artifact, team = root cause among
unknowns.

## Assumptions & contingencies

- **Ticketing discovery STOP:** `ticketing.type: hermes-kanban` (board `skillgrid`) is the source of truth. If `hermes kanban` discovery fails at Step 4.2 (auth, unresolvable board), STOP and ask — never continue into publication or publish to a fallback board silently.
- **MCP build drift:** the mnemonic MCP surface varies by build. Where a skill names a tool the mounted server lacks: `projectwiki_*` (or legacy `kb_*`) → use the CLI `mnemonic projectwiki ...` (same engine); `web_cache_*` → skip (research proceeds uncached); `mem_inject_session` → `mem_context`/`mem_search` fallback. `mnemonic.enabled: false` → all memory/index/web-cache usage stops; the pipeline continues.
- **qa-gate path drift (verified in this checkout):** `qa-gate.mjs` resolves `state-drift-check.mjs`, `ship-drift-check.mjs`, `skill-size-budget.mjs`, and `okf-lint.mjs` under `.agents/skills/verification/qa/scripts/`, which does not exist (real location `.agents/skills/qa/scripts/`). Those 4 checks report "script not found" (exit 2). Contingency: run them directly from their real location — the repo-root checks (`check-adr-invariants.mjs`, `status-banner.mjs`, `traceability.mjs`, `tdd-evidence.mjs`, `review-ledger-lint.mjs`, `windows.mjs`, `delivery-guard.mjs`) resolve correctly.
- **state.yaml v1/v2:** this repo currently carries v1 (`pipeline.current_change` + dated journal keys) while skills write v2 (`changes:` map). The drift guard parses both; new writes use v2. If the phase-router reports drift, artifacts win — patch `state.yaml`, commit, continue.
- **Parallel waves:** parallel waves in the execution phase use isolated worktrees (`git worktree add`), never a shared checkout.
