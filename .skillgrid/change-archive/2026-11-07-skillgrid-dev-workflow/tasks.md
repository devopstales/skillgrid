# Tasks — Development Workflow execution guide

> **STATUS:** `sliced` (2026-11-07)

> Sliced from `.skillgrid/change-specs/2026-11-07-skillgrid-dev-workflow/blueprint.md`.
> Vertical tracer-bullet tickets, dependency-ordered, sized for one fresh agent context window.
> **Docs-only change** — no Go/JS code; the "code" is Markdown (an attested vault page +
> a cross-link + a glossary row) + shell verification + a freshly-built `mnemonic` binary
> for the no-new-drift projectwiki gate.

## Epic Summary

Land the approved development-workflow plan as an **attested** projectwiki vault page
(`workflows/development-workflow.md`) — the ordered step-by-step execution guide for
Delivery changes (step → skill → subagents → MCP → CLI → files → gates, Steps 0–10) —
cross-linked from the SDD pipeline concept page and glossary-indexed, **adding no new
drift** to the pre-ship projectwiki gate. The page carries two resolvable frontmatter
anchors (config schema + the sdd-structure phase-order source) so it is attested → `ok`,
not a drift source. The gate runs on a freshly-built `mnemonic` binary (the installed
0.2.0 predates the kb→projectwiki rename and exposes `kb`, not `projectwiki`).

Fast-Track: **standard** — standard classification (L2: full suite + build; the gate
builds the fresh binary). No new dependency, no one-way door, no public contract.

## Delivery Strategy

| Field | Value |
|-------|-------|
| Estimated changed lines | ~350–450 (4 tickets; one ~300-line page + two one-line edits + the gate) |
| 400-line budget risk | Low (the single large unit, TICKET-01, is one coherent Markdown page; the other units are one-line edits + a verification task) |
| Chained PRs recommended | No (one change, serial) |
| Suggested split | 4 work units (W1 page → W2 cross-link → W3 glossary → W4 gate) |
| Delivery strategy | no-gate |
| Chain strategy | N/A (single commit stream, serial development) |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: N/A
400-line budget risk: Low

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Author the attested Development Workflow page (frontmatter + 2 anchors + Context + Steps 0–10 + cross-cutting + subagent summary + assumptions) — door check | PR 1 | `test -f .skillgrid/projectwiki/workflows/development-workflow.md && /tmp/mnemonic-pw projectwiki validate --strict` (page `ok`, no new drift) | The freshly-built binary + the live vault | `.skillgrid/projectwiki/workflows/development-workflow.md` |
| 2 | Cross-link from `sdd-pipeline.md` (one intro line) | PR 2 | `grep -q 'development-workflow' .skillgrid/projectwiki/workflows/sdd-pipeline.md` | n/a (text check) | the one added line in `sdd-pipeline.md` |
| 3 | Add the `Development Workflow` glossary row | PR 3 | `grep -q 'Development Workflow' .skillgrid/projectwiki/glossary/01-business-terms.md` | n/a (text check) | the one added row in `01-business-terms.md` |
| 4 | No-new-drift gate (build fresh binary + index + validate --strict + lint) + R6 drift-detection mutation | PR 4 | `/tmp/mnemonic-pw projectwiki index && /tmp/mnemonic-pw projectwiki validate --strict && /tmp/mnemonic-pw projectwiki lint --json` | The freshly-built binary + the live vault | none (verification; the R6 mutation is fully reverted) |

> The four plain-text guard lines above are the **contract** — `skillgrid:subagent-execution` matches them literally.

## Tickets

> **Precondition (all tickets):** the approved development-workflow plan content is available
> (session-local — the controller provides it in the TICKET-01 brief; this change exists to
> persist it per ADR-0013). The fresh binary is built on first use (TICKET-04 Step 1, or
> pre-built by the controller for TICKET-01's door check).
>
> **Tracker IDs are placeholders** (`TASK-XXX`) — `skillgrid:ticketing` publishes the real
> IDs to the configured tracker (hermes kanban, board `skillgrid`) at execution.

### TICKET-01 — Author the attested Development Workflow page (door check)
- **Tracker ID:** TASK-XXX  (milestone 2026-11-07-skillgrid-dev-workflow)
- **Scope:** Create `.skillgrid/projectwiki/workflows/development-workflow.md` — OKF frontmatter (`type: BusinessWorkflow`, `title: "Development Workflow"`, two resolvable `anchors`: `.skillgrid/config.yaml` all_of `[schema, "skillgrid/v1"]` + `.agents/skills/_shared/rules/sdd-structure.md` all_of `["Phase Order"]`), `# Development Workflow` title, `## Context`, `## Step 0`…`## Step 10` (each: Skill / Subagents / MCP / CLI / Files / Gates; sub-steps 1a/1b/1c as `###` under Step 1, 7.5 as `###` under Step 7 — NOT top-level), `## Cross-cutting tool usage`, `## Subagent summary`, `## Assumptions & contingencies` — faithful to the approved plan. Run the same-edit `projectwiki index` + the door check (`validate --strict` → page `ok`, no new drift). **The door check** — if the page is not `ok` (an anchor unresolved), stop and revisit anchor selection.
- **Acceptance:** `test -f` passes; frontmatter carries `type: BusinessWorkflow` + `title: "Development Workflow"` + both anchors; all 11 `## Step` headings present; `projectwiki validate --strict` reports the page `ok` with the 2 pre-existing drifts only (per-page `drift (` count stays 2).
- **SATISFIES:** happy path workflow-page-exists-with-all-steps, happy path step-fidelity-spot-checks
- **Files:** `.skillgrid/projectwiki/workflows/development-workflow.md`
- **Size:** ~300 (L, one coherent page — held whole as the door check)
- **Blocks:** TICKET-02, TICKET-04
- **Blocked by:** none
- **Reversibility:** reversible (a new Markdown file)
- **Fails-when:** the page is not `ok` after the same-edit index (an anchor unresolved → new drift), OR a `## Step 0`…`## Step 10` heading is missing, OR a per-step spot check (R2) fails.

### TICKET-02 — Cross-link from the SDD pipeline concept page
- **Tracker ID:** TASK-XXX  (milestone 2026-11-07-skillgrid-dev-workflow)
- **Scope:** Add one line to the intro of `.skillgrid/projectwiki/workflows/sdd-pipeline.md` referencing `development-workflow.md` as the step-by-step execution guide (Steps 0–10). Run the same-edit `projectwiki index`.
- **Acceptance:** `grep -q 'development-workflow' .skillgrid/projectwiki/workflows/sdd-pipeline.md` exits 0; `projectwiki index` exits 0.
- **SATISFIES:** happy path pipeline-page-cross-links
- **Files:** `.skillgrid/projectwiki/workflows/sdd-pipeline.md` (one added line)
- **Size:** ~2 (S)
- **Blocks:** TICKET-04
- **Blocked by:** TICKET-01 (the link target must exist so it is not dangling)
- **Reversibility:** reversible (one line)
- **Fails-when:** the grep does not match, OR the link references a non-existent rel_path.

### TICKET-03 — Add the `Development Workflow` glossary row
- **Tracker ID:** TASK-XXX  (milestone 2026-11-07-skillgrid-dev-workflow)
- **Scope:** Add one row to `.skillgrid/projectwiki/glossary/01-business-terms.md` defining `Development Workflow` (the ordered step-by-step execution guide for Delivery changes; the executable complement to the SDD Pipeline concept page) with Definition / Use When / Avoid per the table format. The term matches the page title (resolvable). Run the same-edit `projectwiki index`.
- **Acceptance:** `grep -q 'Development Workflow' .skillgrid/projectwiki/glossary/01-business-terms.md` exits 0; `projectwiki index` exits 0.
- **SATISFIES:** happy path glossary-term-present
- **Files:** `.skillgrid/projectwiki/glossary/01-business-terms.md` (one added row)
- **Size:** ~4 (S)
- **Blocks:** TICKET-04
- **Blocked by:** none (independent of TICKET-01/02)
- **Reversibility:** reversible (one row)
- **Fails-when:** the grep does not match, OR the row is missing a Use-When or Avoid cell.

### TICKET-04 — No-new-drift projectwiki gate + R6 drift-detection mutation
- **Tracker ID:** TASK-XXX  (milestone 2026-11-07-skillgrid-dev-workflow)
- **Scope:** Build the fresh binary (`cd mnemonic-v2 && go build -o /tmp/mnemonic-pw ./cmd/mnemonic`); assert `projectwiki validate --help` shows `--strict` (a real subcommand). Run the gate from the repo root: `projectwiki index` (exit 0), `projectwiki validate --strict` (new page `ok` + the 2 pre-existing drifts only — per-page `drift (` count = 2, no new drift; exit 1 on the baseline is expected), `projectwiki lint --json` (exit 0, no issues). Run the R6 throwaway mutation: baseline per-page drift count 2 → break one anchor → 3 → restore → 2 (counter via `grep -c ' drift ('`). Confirm `git status --porcelain .skillgrid/projectwiki/` is clean (mutation reverted). Record the verdict for the QA report.
- **Acceptance:** `index` exit 0; `validate --strict` new page `ok` + exactly 2 pre-existing `drift (` lines; `lint` exit 0 no issues; R6 mutation 2→3→2; vault clean after.
- **SATISFIES:** happy path kb-indexed-and-valid, failure kb-validate-detects-anchor-drift
- **Files:** none (verification — temp binary + `projectwiki` commands; the verdict is recorded for the QA `report.md`, owned by the QA phase)
- **Size:** ~0 (S, verification)
- **Blocks:** none (the final gate)
- **Blocked by:** TICKET-01, TICKET-02, TICKET-03 (all vault edits complete before the gate re-derives)
- **Reversibility:** n/a (verification; the R6 mutation is fully reverted)
- **Fails-when:** the fresh build fails or lacks `projectwiki validate`, OR `validate --strict` reports the new page `drift` (new drift), OR `lint` reports issues, OR the R6 mutation does not move 2→3→2.

## Execution Notes

- **Waves (serial):** W1 (T1) → W2 (T2) → W3 (T3) → W4 (T4). T2 depends on T1 (the link target); T4 depends on T1–T3; T3 is independent of T1/T2 (could run in a parallel worktree, but serial is the default).
- **The fresh binary is a shared prerequisite:** TICKET-04 Step 1 builds it; TICKET-01's door check needs it too — the controller pre-builds it (or TICKET-01 builds it on first use per the blueprint's forward reference). Never overwrite `~/.local/bin/mnemonic`.
- **The 2 pre-existing drifts** (`workflows/sdd-pipeline.md`, `decisions/04-adr-0050-harness2-host-integration.md`) are a baseline — NOT from this change. The gate's pass criterion is **no new drift** (user decision 2026-11-07), not `validate --strict` exit 0.
- **Serial development:** one change at a time (AGENTS.md § Rules). `state.yaml` `pipeline.current_change` is updated to `2026-11-07-skillgrid-dev-workflow` at the blueprint phase commit (the controller owns it, not a ticket).
- **QA gate (per skillgrid:qa):** L2 — full suite + build (`go test ./...` + `go build ./...` as a safety net; no Go code changes) + the doc-specific structural checks + the no-new-drift projectwiki gate. The R6 mutation is the drift-detectability backstop.
- **Definition of Done (per AGENTS.md § Rules):** the page is attested (`ok`, two resolvable anchors), all 11 steps are present + faithful (R2 spot checks), the cross-link + glossary term resolve, the same-edit rule held (index after each vault write), and the no-new-drift gate is green.
