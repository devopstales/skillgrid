# Briefing — Project Init (Claude OS parity, slice 1)

> **STATUS:** `accepted` (2026-10-02)

**Topic:** 2026-10-02-mnemonic-project-init
**Date:** 2026-10-02
**Classification:** standard (T2)
**Build shape:** Smallest usable whole
**Queued behind:** `2026-10-02-mnemonic-memory-checkpoint` (serial: do not take `current_change` until that change ships or parks)
**Next in Claude OS parity program:** `2026-10-02-mnemonic-llm-provider` (P2 shared LLM; after this init change)
**Findings:** `08-second-brain-roadmap.md`, `09-claude-os-deep-dive.md` (S5 installer / S6 lean boot file), `docs/NOTES.md` Init section

## Problem / Intent

Claude OS’s remaining “magic” is `/claude-os-init`: after two minutes the project has a boot file, an index, and docs in memory. Skillgrid already has the onboarding *skill* (config + lean AGENTS sentinel) and `skillgrid index`, but there is no deterministic `skillgrid init` that finishes the job. A new repo still needs the agent to invent the last mile.

## Purpose & Success Criteria

- **Purpose:** An agent or a human can run project init and get a lean boot file, a forced code index, and local docs in the existing store — then work without re-explaining the project.
- **Success criteria:** See Requirements. A temp repo after `skillgrid init` has the preamble + sentinel, a populated index, and upserted observations for every default path that exists; a second run does not duplicate the sentinel; `--docs` adds extra files.
- **Out of scope:** Four Claude OS knowledge bases; shared LLM provider (roadmap J — separate spec `2026-10-02-mnemonic-llm-provider`); human browse (roadmap G); session-end extraction (H); cross-project pattern synthesis (I); Kanban/services dashboard; a skills marketplace; a slash-command zoo; replacing `skillgrid:onboarding`’s interview.

## Context

Fits existing patterns: **yes-with-notes**.

`skillgrid index` already opens the project handle and indexes a directory. `skillgrid:onboarding` already writes `.skillgrid/config.yaml`, upserts the `<!-- skillgrid:start -->` block (`_shared/agent-config/block.md`), and `mem_save`s `skillgrid-init/{project}`. Init is a **sequencer** over those seams, plus a preamble writer and a file-walk ingest. Notes: (1) do not invent a second store; (2) do not move the interview into Go; (3) AGENTS preamble lives *above* the sentinel so the every-prompt block stays lean.

Locked: Go 1.22+, no new dependencies without an ADR, serial development, spec-zone before code-zone, repo is source of truth. ADR-0012 (one SQLite store) is the knowledge-model constraint.

## Approaches Considered

- **Chosen: A — Orchestrator CLI.** `skillgrid init` sequences AGENTS write, existing index, and observation ingest. The onboarding skill interviews, then runs the CLI.
- **Rejected: B — Skill-only.** No deterministic floor; ingest and AGENTS shape depend on the model.
- **Rejected: C — Replace onboarding.** Extra churn; the interview and config merge already work.

## Requirements

1. **Init command:** `skillgrid init` is a first-class command that sequences boot-file write, code index, and doc ingest.
   - **Current:** `main.go` has `index`, `install`, `policy init` — no project `init`.
   - **Target:** `skillgrid init [--force] [--docs path]...` exists, prints a result (written / indexed / ingested / skipped / errors), exit 0 on partial success if boot file was written, non-zero only if the boot file could not be written.
   - **Acceptance:** `go test` for the command on a temp repo: usage, flags, exit codes.
   - **Acceptance scenario:** `happy path init writes boot file and reports counts` → `acceptance.feature`

2. **Boot file:** Init writes a short working-agreement preamble above the existing Skillgrid sentinel; the sentinel is upserted, never duplicated.
   - **Current:** Onboarding skill upserts the sentinel; NOTES.md wants a longer working agreement; nothing writes that preamble from the CLI.
   - **Target:** File is `AGENTS.md` if it exists or neither AGENTS/CLAUDE exist; else `CLAUDE.md`. Preamble is above `<!-- skillgrid:start -->`. Sentinel replace-in-place. `--force` rewrites the preamble; merge leaves user text outside both regions.
   - **Acceptance:** First run creates both regions; second run without `--force` keeps a single sentinel; `--force` refreshes the preamble only.
   - **Acceptance scenario:** `happy path init upserts preamble and sentinel` → `acceptance.feature`

3. **Forced code index:** Init always runs the existing index pipeline for the project directory.
   - **Current:** `skillgrid index` is a separate command; onboarding does not run it.
   - **Target:** Init calls the same indexer the `index` command uses (no second pipeline). A repo with at least one indexable file ends with a non-empty file list in the store.
   - **Acceptance:** After init on a Go (or other indexed) temp repo, `skillgrid index status` (or the store query the tests already use) shows files > 0.
   - **Acceptance scenario:** `happy path init indexes the project` → `acceptance.feature`

4. **Default ingest:** Existing default paths become observations with stable topic keys; missing defaults are skipped.
   - **Current:** Onboarding saves a few architecture/config observations; it does not walk `docs/` or artifacts.
   - **Target:** For each existing path among `README.md`, `docs/` (regular files under it), `.skillgrid/ASSUMPTIONS.md`, `.skillgrid/ARCHITECTURE.md`, `.skillgrid/artifacts/` (regular files), upsert `mem_save` with `topic_key` `init/docs/<relpath>`, type `architecture` for ASSUMPTIONS/ARCHITECTURE/artifacts, type `discovery` for README/docs. Same key on re-run updates, does not duplicate.
   - **Acceptance:** Temp repo with README + one docs file yields two observations at those keys; a repo without `docs/` still exits 0 and reports skipped.
   - **Acceptance scenario:** `happy path init ingests default paths` → `acceptance.feature`

5. **Extra docs flag:** Repeatable `--docs <path>` ingests additional files or directories the same way.
   - **Current:** No flag.
   - **Target:** Each extra path that exists is ingested; a missing extra path is reported as an error line but does not fail the command if the boot file was written.
   - **Acceptance:** `--docs extra.md` creates `init/docs/extra.md`; missing `--docs` path appears in the report and does not duplicate or crash.
   - **Acceptance scenario:** `happy path init ingests extra --docs path` → `acceptance.feature`

6. **Skill hands off to CLI:** After the existing onboarding interview, the skill runs `skillgrid init` (with `--force` only when the user asked to rebuild).
   - **Current:** Onboarding writes config/AGENTS/mnemonic saves and stops.
   - **Target:** Skill body adds one step: run `skillgrid init` (pass through `--docs` if the user named extra paths). It does not re-implement ingest in prose.
   - **Acceptance:** The onboarding SKILL.md names `skillgrid init` as the deterministic finish; no second ingest procedure in the skill.
   - **Acceptance scenario:** `happy path onboarding skill calls skillgrid init` → `acceptance.feature`

## Implementation Decisions

- **Modules to build/modify:** new `init` command next to `index`/`policy`; thin helpers for preamble upsert and file-walk ingest; onboarding SKILL.md one-step addition; `main.go` usage line.
- **Interfaces:** `skillgrid init [--force] [--docs path]...` (cwd = project). Result shape: `{ boot_file, preamble: written|kept|forced, sentinel: upserted, indexed: N, ingested: N, skipped: [paths], errors: [messages] }`. Human text on stdout; `--json` not required this slice.
- **Data flow:** cwd → open existing project handle → write boot file → run index pipeline → walk paths → `mem_save` upserts → print report.
- **Error handling:** Boot-file write failure is fatal (non-zero). Index or ingest failures append to `errors` and the command still exits 0 if the boot file landed. Missing default paths are `skipped`, not errors. Missing `--docs` paths are errors (listed) but non-fatal.
- **Dependencies:** Reuse index pipeline and memory save. No new modules, no new libraries, no four-KB schema.

## Testing Decisions

- **What makes a good test:** Temp directory with a fake git/project handle (same fixtures as other CLI tests). Assert files on disk, observation topic keys, index file count, exit code, and that a second run does not duplicate the sentinel.
- **Modules to test:** `init` command (primary). Skill change is a string/contract assertion on SKILL.md.
- **Prior art:** `policy_cmd_test.go`, `index_status_test.go`, doctor temp-db tests.
- **Edge cases:** no README; empty docs; CLAUDE.md-only repo; second run; `--force`; missing `--docs` path; index failure after boot file written.

## Impact on Global Docs

- `.skillgrid/artifacts/00-prd.md`: None this draft (add the init command to the CLI surface when the change executes).
- `.skillgrid/ASSUMPTIONS.md`: None this draft (no new ADR; keep-store restates ADR-0012).
- `.skillgrid/ARCHITECTURE.md`: None this draft (init is another CLI verb on the existing Distribution Surface).

## Clarity Report

| Dimension           | Score | Min  | Status | Notes |
|---------------------|-------|------|--------|-------|
| Goal Clarity        | 0.86  | 0.75 | pass   | Slice 1 is project init, cli+skill, one store |
| Boundary Clarity    | 0.84  | 0.70 | pass   | G/H/I and dashboard are later serial changes |
| Constraint Clarity  | 0.80  | 0.65 | pass   | ADR-0012, no new deps, merge + lean preamble |
| Acceptance Criteria | 0.78  | 0.70 | pass   | Temp-repo CLI checks above |
| **Clarity**         | 0.17  | ≤0.20| pass   | Residual unclarity |

**Interview log:**

| Round | Question summary | Decision locked |
|-------|------------------|-----------------|
| 1 | Which Claude OS surface? When? | All remaining surfaces as a serial program; start after webui-rewrite |
| 2 | First slice? What is a KB? Store model? Trigger? | init-first; KB = Claude OS named pile, not ours; keep-store; cli+skill |
| 3 | AGENTS shape, re-run, ingest paths | lean-plus-preamble; merge; defaults-plus-flag |
| 4 | Approach | A — orchestrator CLI |

## Open Questions & Assumptions

- **Assumption:** Preamble content is the NOTES working-agreement headings (overview, env, standards, security, deps, architecture, DoD), filled from detected stack where cheap, placeholders left for the skill interview to complete — not a 351-line dump.
- **Assumption:** Init does not write `.skillgrid/config.yaml`; the skill still does that before calling the CLI. A bare `skillgrid init` on a repo with no config still writes AGENTS, indexes, and ingests.
- **Question:** None that block the spec. Blueprint can pick the exact preamble template file path.

## Decisions (ADR)

- `.skillgrid/artifacts/04-adr-0012-sqlite-as-second-brain.md` — one store; init does not add KBs.
- No new ADR. Orchestrator CLI vs skill-only is reversible and unsurprising.

## Terms

- [Knowledge Base (KB)](.skillgrid/artifacts/02-technical-terms.md) — Claude OS pile; not a Skillgrid object (added this interview).
- [Project Init](.skillgrid/artifacts/02-technical-terms.md) — `skillgrid init` + onboarding skill (added this interview).
- [Second Brain](.skillgrid/artifacts/02-technical-terms.md), [Distribution Surface](.skillgrid/artifacts/02-technical-terms.md), [Topic Key](.skillgrid/artifacts/02-technical-terms.md) — relied on, unchanged.
