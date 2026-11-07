# Development Workflow — Implementation Blueprint (steps, files, verification)

**Status:** PROPOSED
**Tier:** T2
**Build shape:** Tracer thread — the page (core artifact) is created first and attested, then the cross-link + glossary wire it into the vault, then the gate verifies end-to-end: a thin path (documented → discoverable → indexed → verified) that works first, then thickens.
**Change classification:** standard (L2 — full suite + build; the gate builds the fresh `mnemonic` binary from `mnemonic-v2/`).
**Goal:** Land the approved development-workflow plan as an attested projectwiki vault page (`workflows/development-workflow.md`), cross-linked from the SDD pipeline concept page and glossary-indexed, adding no new drift to the pre-ship projectwiki gate.
**Architecture:** Adapt the approved plan (in-session content) into a single OKF `BusinessWorkflow` vault page carrying a resolvable frontmatter `anchors` list (so the page is attested → fresh, not a drift source). Add one cross-link line to `sdd-pipeline.md`; add one glossary row. Rebuild the projectwiki index in the same edit as each vault write and run the no-new-drift gate on a freshly-built `mnemonic` binary (the installed 0.2.0 predates the kb→projectwiki rename, commit 5686caba, and exposes `kb`, not `projectwiki`).
**Tech Stack:** Markdown (OKF v0.2 frontmatter, agent-style v0.9.0 prose); the projectwiki engine via a freshly-built `mnemonic` Go binary from `mnemonic-v2/`; shell verification (`test -f`, `grep`, command exit codes). No Go/JS application code changes.
**Spec:** `.skillgrid/change-specs/2026-11-07-skillgrid-dev-workflow/briefing.md`
**Critic:** mandatory (Tier T2) — see Plan Review below.

## Hypothesis

**Claim:** The approved development-workflow plan can be adapted into an attested projectwiki vault page (`workflows/development-workflow.md`) that is faithful to the plan's substance, cross-linked from `sdd-pipeline.md`, glossary-indexed, and introduces no new drift into the pre-ship projectwiki gate.

**Right condition:** The page exists with `type: BusinessWorkflow` frontmatter + a resolvable `anchors` list + all `## Step 0`…`## Step 10` sections; the cross-link + glossary term are present; `projectwiki index` runs in the same edit; `projectwiki validate --strict` reports the new page FRESH (only the 2 known pre-existing drifts remain — no new drift); `projectwiki lint` is clean; the R6 mutation test (break anchor → page drifts → restore → fresh) passes.

**Wrong condition:** Any of: the page is missing a step section; the frontmatter is wrong or the anchors do not resolve (the page becomes drifted); the cross-link/glossary term is absent; `projectwiki index` fails; `projectwiki validate --strict` reports the new page DRIFTED (new drift, drift count rises above the 2-drift baseline); `projectwiki lint` reports new findings; the R6 mutation test fails (the validator does not detect the broken anchor).

**Thinnest MVP:** The page file with frontmatter + a resolvable anchor + `## Context` + `## Step 0`…`## Step 10` headings (a faithful skeleton) — proves the plan's structure maps onto an attested vault page that adds no drift.

**Door check:** Task 1 (create the page) — after the same-edit index, if the page is not FRESH (anchors do not resolve) or a step section cannot be faithfully mapped, the blueprint is invalidated (revisit anchor selection or structure) and the user decides whether to revise or abandon.

## Must-Haves (goal-backward verification)

**Truths** (observable behaviors that must hold):
- `test -f .skillgrid/projectwiki/workflows/development-workflow.md` exits 0; the frontmatter carries `type: BusinessWorkflow`, `title: "Development Workflow"`, and an `anchors:` list with both resolvable anchors (config schema + sdd-structure phase order) (R1).
- The page contains all 11 step headings `## Step 0` … `## Step 10`, each naming its Skill / Subagents / MCP / CLI / Files / Gates (R1, R2).
- Per-step fidelity spot checks pass — for each of Steps 0–10 the page names the same skill, subagent pattern (counts + roles), MCP tools, files, and user gates as the approved plan (R2 spot-check list); specifically Step 7/7.5 names `review-ledger` + the `review: record round <N>` commit format (the sub-step most likely to vanish, since it is a `###` not a top-level heading).
- `grep -q 'development-workflow' .skillgrid/projectwiki/workflows/sdd-pipeline.md` exits 0 (R3).
- `grep -q 'Development Workflow' .skillgrid/projectwiki/glossary/01-business-terms.md` exits 0 (R4).
- `projectwiki index` (fresh binary) exits 0 and indexes the new page (R5).
- `projectwiki validate --strict` (fresh binary, non-JSON) reports the new page FRESH and the 2 pre-existing drifts only — no new drift from this change (R5). `backstop` — not confirmable from the diff alone; requires the runtime gate.
- `projectwiki lint` (fresh binary) exits 0 with no new findings (R5).
- R6 mutation (throwaway): baseline (new page fresh, 2 drifts) → break one anchor → re-index → new page drifts (3 drifts) → restore → re-index → new page fresh (2 drifts). `backstop` — runtime mutation test.

**Artifacts** (files that must exist with real content, not stubs):
- [`.skillgrid/projectwiki/workflows/development-workflow.md` — the full workflow page: frontmatter + anchors + Context + Steps 0–10 + Cross-cutting tool usage + Subagent summary + Assumptions & contingencies]
- [`.skillgrid/projectwiki/workflows/sdd-pipeline.md` — one added cross-link line (modify)]
- [`.skillgrid/projectwiki/glossary/01-business-terms.md` — one added glossary row (modify)]

**Key links** (critical connections that must work together):
- The new page's frontmatter `anchors` must resolve against live files (the first anchor pins `.skillgrid/config.yaml` tokens `schema`/`skillgrid/v1`; the second pins `.agents/skills/_shared/rules/sdd-structure.md` token `Phase Order`) — if either does not resolve, the page is drifted and the gate fails (R5 door).
- The cross-link in `sdd-pipeline.md` must reference the new page's rel_path (`development-workflow.md`) so it is not a dangling reference (R3).
- The glossary row's term (`Development Workflow`) must match the page's title so the term resolves to the page (R4).
- `projectwiki index` must run in the same edit as each vault write (same-edit rule) so the gate sees the new page (R5).

**One-way-door decisions** (hard to reverse — flag for explicit user approval):
- None (a new Markdown page + two one-line edits + a re-derivable index; trivially reversible).

## Global Constraints

- OKF v0.2 frontmatter: `type: BusinessWorkflow`, `title: "Development Workflow"` (ADR-0048 zone). Mirror `workflows/sdd-pipeline.md`'s `type`/`title` shape, **plus an `anchors:` list it lacks** — that page has none, which is exactly why it is one of the 2 baseline drifts; copying its frontmatter verbatim would mint a drifted page.
- agent-style v0.9.0 prose (ADR-0051).
- Same-edit rule: every vault page write is followed by `projectwiki index` (freshly-built binary).
- `index/` projections are engine-owned — never hand-edited.
- No changes to `.agents/skills/`, `_shared/rules/sdd-structure.md`, or `config.yaml` (out of scope per briefing).
- No new ADR; no new dependency (Go: zero; Node: zero).
- The pre-ship gate runs on a freshly-built `mnemonic` binary from `mnemonic-v2/` (the installed 0.2.0 is stale — it predates the kb→projectwiki rename, commit 5686caba; it exposes `kb`, not `projectwiki`).
- The 2 pre-existing unresolved drifts (`workflows/sdd-pipeline.md`, `decisions/04-adr-0050-harness2-host-integration.md`) are a baseline — not from this change; the gate's pass criterion is **"no new drift"** (user decision 2026-11-07).

## Threat Matrix

| Boundary | Applicability | Design response | Planned RED test |
|---|---|---|---|
| Shell command: `go build -o /tmp/mnemonic-pw ./cmd/mnemonic` (fresh binary) | Applicable: the gate depends on a binary built from source | Build to a temp path (never overwrite `~/.local/bin/mnemonic`); verify the built binary exposes `projectwiki validate` (a real subcommand) before running the gate | Task 4 Step 1: build fails, or `projectwiki validate --help` shows no `--strict` flag → the gate cannot run |
| Mnemonic tool contract: `projectwiki index/validate/lint` on the fresh binary | Applicable: the no-new-drift gate is the pre-ship check | Fresh binary resolves to project `skillgrid-v2` (verified); the new page carries two resolvable `anchors` so it is attested → `ok`; gate pass = new page `ok` + per-page drift count stays at the 2-drift baseline | Task 1 door check + Task 4: `validate --strict` reports the new page drifted (an anchor unresolved) → new drift → RED |
| Pre-existing drift baseline (2 unresolved grounded pages) | Applicable: `validate --strict` exits 1 on them regardless of this change | Treat the 2 drifts as a baseline; the gate asserts no NEW drift (per-page `drift (` count stays 2, new page `ok`) rather than exit 0 | Task 4 Step 3 (R6 mutation): `grep -c ' drift ('` → break one anchor → 3 → restore → 2 (validator not weakened) |

## File Structure

- `.skillgrid/projectwiki/workflows/development-workflow.md` — the new workflow page (frontmatter + anchors + Context + Steps 0–10 + Cross-cutting tool usage + Subagent summary + Assumptions & contingencies). The core deliverable.
- `.skillgrid/projectwiki/workflows/sdd-pipeline.md` — one added line in the intro cross-referencing the new page as the step-by-step execution guide.
- `.skillgrid/projectwiki/glossary/01-business-terms.md` — one added row defining the `Development Workflow` term.

(Task 4 builds a temp binary and runs `projectwiki` commands — no repo file changes.)

---

### Task 1: Create the attested development-workflow page

> **Checkpoint:** Door check for the blueprint hypothesis — if the page is not FRESH after the same-edit index, stop and revisit anchor selection.

**Files:**
- Create: `.skillgrid/projectwiki/workflows/development-workflow.md`

**Interfaces:**
- Consumes: the approved development-workflow plan (session-local content — the controller provides it in the task brief; this change exists to persist it as a vault page per ADR-0013, since a `local://` plan dies with the session). The plan supplies the step structure, per-step Skill/Subagents/MCP/CLI/Files/Gates, cross-cutting contracts, subagent summary table, and assumptions & contingencies.
- Produces: an attested `BusinessWorkflow` vault page (all later tasks + the gate depend on its existence and freshness).
- Seam: none (in-process — a Markdown page; its interface is the OKF frontmatter + section headings).
- Deletion test: deleting the page re-opens the gap the briefing names (no step-by-step execution guide in `workflows/`); the cross-link (Task 2) would dangle.
- Adapters: 1 (the page itself) — no port to build.

**SATISFIES:** `happy path workflow-page-exists-with-all-steps`, `happy path step-fidelity-spot-checks`

- [ ] **Step 1: Write the failing check** (RED)

Run:
```bash
test -f .skillgrid/projectwiki/workflows/development-workflow.md; echo "exit=$?"
```
Expected: `exit=1` (file absent).

- [ ] **Step 2: Write the page**

Create `.skillgrid/projectwiki/workflows/development-workflow.md`, adapting the approved plan into a vault page (faithful to the plan's substance — re-stated as durable repo content, not a verbatim session-plan paste):
- Frontmatter (OKF v0.2):
  ```yaml
  ---
  title: "Development Workflow"
  type: BusinessWorkflow
  anchors:
    - path: .skillgrid/config.yaml
      all_of: [schema, "skillgrid/v1"]
    - path: .agents/skills/_shared/rules/sdd-structure.md
      all_of: ["Phase Order"]
  ---
  ```
  (The first anchor pins the config schema the workflow relies on; the second pins the SDD phase-order source the Steps follow. Both verified 2026-11-07 to resolve → the page is attested/fresh. If either source loses a pinned token, the page drifts — signaling the guide is stale. User decision 2026-11-07: two anchors, not schema-only.)
- `# Development Workflow` title.
- `## Context` — what the workflow is, that it applies to Delivery changes only (Specialist/Q&A do not enter the pipeline), and the config facts it relies on (`mnemonic.enabled`, `ticketing.type: hermes-kanban` board `skillgrid`, `bdd.enabled` stack javascript, `rules.tiers.default: T2`, `rules.fast_track.enabled`, test `go test ./...`, build `go build ./...`, format `gofmt -w ./...`, advisory trivy + jscpd).
- `## Step 0` … `## Step 10` — exactly 11 top-level step headings (R1's structural contract). Each step names Skill / Subagents / MCP / CLI / Files / Gates, faithful to the approved plan (the R2 spot-check list pins the per-step substance). The plan's sub-steps land as `###` subsections under their parent step — `Step 1a/1b/1c` under `## Step 1`; `Step 7.5` (review-ledger) under `## Step 7` — and are NOT promoted to top-level `## Step` headings (that would break R1's 11-heading check).
- `## Cross-cutting tool usage` — memory (`mem_*`), the Code Index Ladder (binding order), the knowledge base (projectwiki read/write contracts, same-edit rule, pre-ship gate), the commit protocol (work-unit-commits), and the resume procedure.
- `## Subagent summary` — the plan's subagent table (counts + roles per step).
- `## Assumptions & contingencies` — the hermes-kanban discovery STOP, the MCP build-drift fallbacks, the qa-gate path drift + direct-run contingency, the state.yaml v1/v2 note, and the parallel-worktrees rule.

- [ ] **Step 3: Run the same-edit index + door check**

(If the fresh binary is not yet built, build it first per Task 4 Step 1.) Run:
```bash
/tmp/mnemonic-pw projectwiki index && /tmp/mnemonic-pw projectwiki validate --strict
```
Expected: `index` exit 0 (new page indexed); `validate --strict` reports the `workflows/development-workflow.md` line as `ok` (any reason — `fresh` on a clean store, `baseline_current` if a baseline exists) and the 2 pre-existing drifts only (per-page drift count stays 2 — no new drift). The page line must NOT read `drift`.
- If the new page is NOT `ok` (an anchor unresolved → drifted), STOP (door check failed): fix the anchor to a resolvable token/path, re-index, and re-check. If it still will not resolve, the hypothesis is invalidated — stop and surface to the user.

- [ ] **Step 4: Verify structure** (GREEN)

Run:
```bash
f=.skillgrid/projectwiki/workflows/development-workflow.md
rc=0
test -f "$f" && grep -q 'type: BusinessWorkflow' "$f" && grep -q 'title: "Development Workflow"' "$f" && grep -q 'anchors:' "$f" || rc=1
for i in $(seq 0 10); do grep -q "^## Step $i" "$f" || { echo "MISSING Step $i"; rc=1; }; done
echo "structure-exit=$rc"
```
Expected: no `MISSING Step N` lines; `structure-exit=0`.

- [ ] **Step 5: Commit**

```bash
git add .skillgrid/projectwiki/workflows/development-workflow.md
git commit -m "docs(projectwiki): add Development Workflow execution guide page"
```

### Task 2: Cross-link from the SDD pipeline concept page

**Files:**
- Modify: `.skillgrid/projectwiki/workflows/sdd-pipeline.md` (add one line in the intro)

**Interfaces:**
- Consumes: the new page's rel_path (`development-workflow.md`) from Task 1.
- Produces: a discoverable pointer from the concept page to the execution guide.
- Seam: none (in-process).
- Deletion test: deleting the line removes the only pointer to the execution guide (the page becomes undiscoverable from the concept page).
- Adapters: 1.

**SATISFIES:** `happy path pipeline-page-cross-links`

- [ ] **Step 1: Write the failing check** (RED)

Run:
```bash
grep -q 'development-workflow' .skillgrid/projectwiki/workflows/sdd-pipeline.md; echo "exit=$?"
```
Expected: `exit=1` (no reference yet).

- [ ] **Step 2: Add the cross-link**

In `sdd-pipeline.md`'s intro (the paragraph stating the per-phase mechanics live in the pipeline skills + `_shared/rules/sdd-structure.md`), add one line:
> For the step-by-step execution guide (which skill, subagent, MCP tool, file, and user gate per phase, Steps 0–10), see `development-workflow.md`.

- [ ] **Step 3: Run the same-edit index + verify** (GREEN)

Run:
```bash
/tmp/mnemonic-pw projectwiki index && grep -q 'development-workflow' .skillgrid/projectwiki/workflows/sdd-pipeline.md; echo "exit=$?"
```
Expected: `index` exit 0; grep `exit=0`.

- [ ] **Step 4: Commit**

```bash
git add .skillgrid/projectwiki/workflows/sdd-pipeline.md
git commit -m "docs(projectwiki): cross-link SDD pipeline to Development Workflow guide"
```

### Task 3: Add the glossary term

**Files:**
- Modify: `.skillgrid/projectwiki/glossary/01-business-terms.md` (add one row)

**Interfaces:**
- Consumes: the term `Development Workflow` (matches the page title from Task 1).
- Produces: a resolvable glossary entry for the new term.
- Seam: none (in-process).
- Deletion test: deleting the row removes the term from the business glossary (unresolvable).
- Adapters: 1.

**SATISFIES:** `happy path glossary-term-present`

- [ ] **Step 1: Write the failing check** (RED)

Run:
```bash
grep -q 'Development Workflow' .skillgrid/projectwiki/glossary/01-business-terms.md; echo "exit=$?"
```
Expected: `exit=1` (term absent).

- [ ] **Step 2: Add the glossary row**

In `01-business-terms.md`, add one row following the existing table format (Term | Definition | Use When | Avoid):
> `Development Workflow` | The ordered step-by-step execution guide for Delivery changes: which skill to invoke, subagents to dispatch, MCP tools to use, files to write, and where the user gates are, per SDD phase (Steps 0–10); the executable complement to the SDD Pipeline concept page. | Authoring or executing a Delivery change top-to-bottom; onboarding a new session to the pipeline. | Reaching for it for Specialist or Q&A requests (they do not enter the pipeline); treating it as a replacement for the per-phase skill bodies.

- [ ] **Step 3: Run the same-edit index + verify** (GREEN)

Run:
```bash
/tmp/mnemonic-pw projectwiki index && grep -q 'Development Workflow' .skillgrid/projectwiki/glossary/01-business-terms.md; echo "exit=$?"
```
Expected: `index` exit 0; grep `exit=0`.

- [ ] **Step 4: Commit**

```bash
git add .skillgrid/projectwiki/glossary/01-business-terms.md
git commit -m "docs(projectwiki): add Development Workflow term to business glossary"
```

### Task 4: Pre-ship projectwiki gate (no-new-drift) + drift-detection mutation

**Files:**
- No repo file changes in this task (verification — builds a temp binary and runs `projectwiki` commands over the vault). The gate verdict is recorded for the QA `report.md`, which the QA phase (Step 6) owns — not written here.

**Interfaces:**
- Consumes: the vault state after Tasks 1–3 (new page + cross-link + glossary, each indexed in its same edit).
- Produces: the pre-ship gate verdict (no new drift) + the R6 drift-detection proof.
- Seam: none (in-process).
- Deletion test: n/a (verification).
- Adapters: n/a.

**SATISFIES:** `happy path kb-indexed-and-valid`, `failure kb-validate-detects-anchor-drift`

- [ ] **Step 1: Build the fresh binary** (RED if it fails)

Run:
```bash
cd mnemonic-v2 && go build -o /tmp/mnemonic-pw ./cmd/mnemonic && /tmp/mnemonic-pw projectwiki validate --help 2>&1 | grep -qi strict
echo "build-exit=$?"
```
Expected: build exit 0; `projectwiki validate --help` shows the `--strict` flag. (Assert on a real subcommand — `projectwiki --help` is itself an unknown subcommand whose error string contains the usage list, so `grep validate` would pass by accident.) If the build fails or `projectwiki`/`validate` is absent, the gate cannot run — stop (the installed binary is stale; the fresh build is the gate's prerequisite). A missing subcommand surfaces as exit 2, distinct from drift's exit 1.

- [ ] **Step 2: Run the gate** (GREEN = no new drift)

Run (from the repo root):
```bash
/tmp/mnemonic-pw projectwiki index
/tmp/mnemonic-pw projectwiki validate --strict
/tmp/mnemonic-pw projectwiki lint --json
```
Expected:
- `index` exit 0 (new page indexed). (This re-index is a harmless pre-gate sync — Tasks 1–3 already indexed their writes and `lint` re-indexes internally before reporting; kept as belt-and-suspenders.)
- `validate --strict` reports the `workflows/development-workflow.md` line as `ok` (any reason) and the 2 pre-existing drifts only (`workflows/sdd-pipeline.md`, `decisions/04-adr-0050-harness2-host-integration.md`) — exactly 2 per-page `drift (` lines, **no new drift from this change**. (It exits 1 on the 2 pre-existing drifts — that is the expected baseline, not a gate failure.)
- `lint` exit 0 with no issues (match on `count == 0` / no issues in the JSON — the field order and indent are not contractual, so do not match a literal string).

- [ ] **Step 3: R6 drift-detection mutation** (throwaway, not a permanent test)

Run:
```bash
# Count per-page drift lines ONLY: `grep -c 'drift'` also matches the summary line
# "N checked, M drifted" and is off-by-one in every state (observed 3→4→3, not 2→3→2).
# The pattern ' drift (' matches only per-page status lines → 2/3/2.
/tmp/mnemonic-pw projectwiki validate --strict | grep -c ' drift ('   # expect 2 (baseline: 2 pre-existing)
# break one anchor in the new page: edit the first anchor all_of to [schema, "does-not-exist-xyz"]
/tmp/mnemonic-pw projectwiki index
/tmp/mnemonic-pw projectwiki validate --strict | grep -c ' drift ('   # expect 3 (new page now drifts)
# restore the anchor to [schema, "skillgrid/v1"]
/tmp/mnemonic-pw projectwiki index
/tmp/mnemonic-pw projectwiki validate --strict | grep -c ' drift ('   # expect 2 (new page fresh again)
```
Expected: per-page drift count goes 2 → 3 → 2, proving the validator detects a broken anchor in the new page (not weakened by it) and the page returns to `ok` on repair.

- [ ] **Step 4: Record the gate verdict + confirm the vault is clean**

Record the gate result (no new drift; new page `ok`; lint clean; R6 mutation 2→3→2) for the QA report (the `report.md` write is owned by the QA phase, Step 6 — not by this task). The R6 mutation must leave the vault unchanged (anchor restored); scope the check to the vault (the repo has unrelated in-flight dirt from other changes):
```bash
git status --porcelain .skillgrid/projectwiki/   # expect clean (the R6 mutation is fully reverted; no vault file left modified)
```

---

## Plan Review / Critic

- **Critic subagent (ran, Tier T2 — mandatory):** verdict **sound-with-minors** — 1 MAJOR, 4 MINOR, 4 NIT, no BLOCKER.
- **Plan-review subagent (ran, 4 tasks ≥ 3):** verdict **READY-WITH-MINORS** — no blockers/majors, 1 MINOR + 6 NIT. (Both independently executed the R6 mutation to verify the counter finding.)
- **User-decide-then-fix (loop ran twice):**
  - **Round 1 (pre-ship gate):** the installed-binary-stale + 2-pre-existing-drifts finding (the plan's R5 "all three exit 0" is not achievable) was surfaced and resolved as the **no-new-drift gate** (attest the new page; tolerate the 2 pre-existing drifts as a baseline). `briefing.md` R5, R6, the CLI assumption, and the Dependencies line were updated to match.
  - **Round 2 (review findings):** the combined critic + plan-review findings were surfaced; the user decided the **anchor scope** (add a second anchor pinning the sdd-structure phase-order source, not schema-only). All remaining fixes applied inline:
    - **MAJOR:** R6 drift counter `grep -c 'drift'` → `grep -c ' drift ('` (the summary line `N checked, M drifted` also contains "drift" → observed 3→4→3, now 2/3/2) — Task 4 Step 3 + Threat Matrix row 3.
    - **MINOR:** Task 4 Step 4 clean-tree check scoped to `.skillgrid/projectwiki/` (unrelated in-flight dirt exists); sub-steps 1a/1b/1c + 7.5 homes stated (`###` under Step 1 / Step 7) + a Step 7/7.5 `review-ledger` spot-check added; second anchor (phase-order source); Task 4 Step 1 build check asserts a real subcommand (`validate --help | grep -qi strict`).
    - **NIT:** `ok (fresh)` → `ok` (any reason); `lint` matched on `count == 0` not a literal string; structure-exit rc accumulator; "Files: None" reworded (report owned by QA); redundant re-index noted; `state.yaml` pointer owner line; mirror-frontmatter wording (sdd-pipeline has no anchors); acceptance-test step named in the Execution Handoff.
- **Status:** final blueprint presented for user approval.

## Execution Handoff

- **Acceptance tests (after blueprint approval, before slicing):** the SATISFIES lines reference `acceptance.feature`, which does not yet exist (the folder holds only `adr.md`, `briefing.md`, `blueprint.md`). Canonical order: `skillgrid:acceptance-test-authoring` writes `acceptance.feature` (the 6 scenarios: 5 happy-path + 1 failure) once this blueprint is approved, then slicing.
- **Slicing:** 4 tasks → invoke `skillgrid:slicing` to produce `tasks.md` (vertical tracer-bullet tickets with dependency edges: Task 2 depends on Task 1; Task 4 depends on Tasks 1–3; Task 3 is independent of Tasks 1–2). The fresh-binary build (Task 4 Step 1) is a prerequisite for Tasks 1–3's same-edit index steps — the controller pre-builds it (or Tasks 1–3 build it on first use per their forward reference).
- **Execution approach:** offer Subagent-Driven (recommended) vs Inline Execution.
- **state.yaml (owner: the controller, with the blueprint phase commit — no task owns it):** update the v1 `pipeline.current_change` → `2026-11-07-skillgrid-dev-workflow` (drift-free per `state-drift-check.mjs` v1 branch; do not create a v2 `changes:` map — that would register 16 presence drifts against the in-flight changes).
