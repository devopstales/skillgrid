# Findings — skillgrid-workflow-updates

## Research: What can Skillgrid learn from GSD Core?

> **Decision this serves:** Scope the `skillgrid-workflow-updates` change: which GSD Core mechanisms are already in Skillgrid, which five are worth borrowing, and which GSD patterns must not be copied.
>
> **Type:** competitive (position against a named SDD framework) + technical (which mechanisms fit this repo)
> **Mode:** research (single pass)
> **Date:** 2026-10-02 · **Status:** complete
> **Primary corpus:** local checkout `~/git/ai-test/gsd-core` (OpenGSD `gsd-core`, MIT) read this run; Skillgrid skills and state files read this run.

## Executive Summary

GSD Core is a prompt-and-file SDD loop — Discuss → Plan → Execute → Verify → Ship — whose stated problem is **context rot**: quality degrades as the main session fills, so heavy work must run in fresh-context subagents while durable state lives on disk. [1][2][3]

Skillgrid already absorbed the parts of that idea that match this repo: `effort:` frontmatter and an honest-overhead “when not to use the pipeline” rule (explicitly borrowed from GSD), [7] skill byte ceilings, [8] fresh workers with file artifacts and disjoint waves, [9] and hook-enforced gates. [10] Mnemonic SQLite is ahead of GSD’s markdown `.planning/` store and its opt-in intel JSON / global-learnings files. [2][5][11]

**Do this change around the remaining gap:** GSD makes plan quality and context headroom **mechanical**, not prompt-only. The five mechanisms worth taking, in this order, are (1) an agent-facing context monitor, (2) a deterministic state CLI so `state.yaml` is not a narrative blob, (3) a plan-checker plus requirements-coverage and scope-reduction recovery, (4) a Nyquist requirement-to-test contract before implementation, and (5) two-stage skill routers so the eager catalog is not ~70 entries every turn. [5][6]

**Biggest caveat:** GSD’s catalog is ~71 skills and ~46 capability packs, with absent-means-enabled defaults and worktrees on by default. [1] Copying that surface would fight Skillgrid’s serial, one-change rule. [11] The lesson is the *mechanisms*, not the catalog.

## Findings

### What GSD Core is

GSD Core describes itself as a light-weight meta-prompting, context-engineering, and spec-driven development system for Claude Code, OpenCode, Cursor, and other runtimes. [1] The installer (`npx @opengsd/gsd-core`) is required for cross-runtime compatibility; copying `agents/` or `commands/` is explicitly discouraged. [1]

The architecture is four layers: command files → workflow orchestrators → fresh-context agents → `gsd-tools.cjs` CLI, with all durable state in `.planning/` (`PROJECT.md`, `REQUIREMENTS.md`, `ROADMAP.md`, `STATE.md`, `config.json`, phase folders). [2] Design principles that matter for this comparison: [2]

- Fresh context per spawned agent (up to 200k tokens).
- Thin orchestrators that load compact context, spawn, collect, and update state — they do not do the heavy work.
- File-based state (no database).
- Absent = enabled feature flags.
- Defense in depth: plan-check before execute, atomic commits per task, post-execution verification, UAT as a final gate.

The local checkout lists 71 skills under `skills/` and 46 packs under `capabilities/` (runtimes plus domains such as `tdd`, `audit`, `mempalace`, `graphify`). [1]

### Context rot and the phase loop

GSD’s explanation of context rot is the reason the framework exists: as the window fills, the model keeps answering while early constraints lose attention. `/clear` loses continuity. [3] The structural answer is fresh-context subagents; the orchestrator never touches source files. [3] Spec-driven artifacts (`CONTEXT.md`, `RESEARCH.md`, `PLAN.md`) and meta-prompted workflows make a fresh agent reason about the right thing. [3]

The loop is Discuss → (optional UI design) → Plan → Execute → Verify → Ship. [4] Each step guards a failure the previous step cannot: Discuss captures implementation decisions so the planner does not guess; Plan research + plan-check before code; Execute gives each worker only its `PLAN.md`; Verify checks goals and decisions, not just task completion; Ship archives and advances `STATE.md`. [4] A good phase is one sentence, bounded research, a handful of non-overlapping plans, and a testable done. [4]

GSD is explicit about overhead: the loop is wrong for a typo or a one-turn rename; `/gsd-quick` and `/gsd-fast` exist for that. [3] Skillgrid already copied this honest-overhead posture into `effort-budgets.md`. [7]

### What Skillgrid already has

| GSD idea | Skillgrid today | Source |
|---|---|---|
| Honest overhead / when not to use the loop | `effort: low\|standard\|max`; table that sends one-turn work off the pipeline | [7] |
| Do not fork the orchestrator to save context | Same rule, citing GSD #921 | [7] |
| Skill / workflow size ceilings | `skill-size-budget.json`: 22 KB standard, 38 KB large, 40 KB extra-large (GSD workflows: 38 / 54 / 90 KB, ratchet tighten-only) | [8][2] |
| Fresh workers + file artifacts + disjoint waves | Specs under `.skillgrid/specs/`; `Owns:` / `Needs:` waves; fan-out ledger | [9] |
| QA drift + four-state gate | `qa` PASS / CONCERNS / FAIL / WAIVED; state/structure/byte-budget drift | [9] |
| Session continuity | `resume` from checkpoint + spec artifacts; memory checkpoints on stop | [9][10] |
| Durable memory and code intelligence | Mnemonic SQLite (observations, FTS, graph) — stronger than GSD `.planning/intel/*.json` and opt-in global learnings | [5][11] |
| Serial one-change | Locked constraint; `state.yaml` names one `current_change` | [11][12] |

GSD’s “global learnings store” and “queryable codebase intelligence” are opt-in file dumps (copy learnings at phase end; `stack.json` / `api-map.json` / etc. in `.planning/intel/`, stale after 24h). [5] Skillgrid should not replace Mnemonic with those files. [11]

### Remaining GSD mechanisms (ranked)

**1. Agent-facing context monitor.** The statusline shows usage to the human; the agent does not see it. GSD’s post-tool hook reads `/tmp/claude-ctx-{session}.json` and injects `additionalContext` at ≤35% remaining (WARNING: wrap up, do not start complex work) and ≤25% remaining (CRITICAL: stop, `/gsd-pause-work`). Debounce is 5 tool uses; severity escalation bypasses debounce. Hooks fail silently and never block the tool. [5][6] A later utilization guard (`/gsd-health --context`) warns at 60% used and goes critical at 70%. [5]

Skillgrid hooks record tool calls and claim memory checkpoints on stop / session idle. [10] They do not tell the running agent that attention is about to degrade.

**2. Deterministic state mutations.** `gsd-tools` owns `state validate` / `state sync` / `state planned-phase`: `STATE.md` is compared to the filesystem and can be reconstructed from disk. Drift is a bug. [2][5] Compound `init` commands load compact JSON context per workflow so orchestrators do not read the whole tree. [2][5]

Skillgrid’s `.skillgrid/state.yaml` `progress` field is one concatenated narrative log. [12] Drift scripts exist (`state-drift-check.mjs` / `ship-drift-check.mjs` per ASSUMPTIONS), [11] but there is no command an agent must call to advance phase, plan count, or blockers. That is the failure mode GSD’s CLI was built to prevent. [5]

**3. Plan-checker, coverage gate, scope-reduction recovery.** After planning, a plan-checker verifies eight quality dimensions and loops up to three times. [5] A requirements-coverage gate blocks planning if a ROADMAP requirement ID never appears in a `PLAN.md`. [5] Scope reduction has three layers: planner prohibition, checker dimension, orchestrator re-injection. [5] A research gate blocks plan-phase while `RESEARCH.md` still has unresolved open questions. [5]

Skillgrid’s approval gate is a human Go / Revise after `tasks.md`. [9] It does not catch a planner that silently dropped a requirement.

**4. Nyquist validation.** Named after the sampling theorem: every requirement must have a feedback signal before code. Each requirement maps to a test command; missing scaffolding is Wave 0; the plan-checker treats Nyquist as an eighth dimension; `/gsd-validate-phase` may add tests later and must not change implementation code. [5]

Skillgrid already has acceptance features and TDD. [9] The missing contract is “this requirement’s runnable signal is this command,” written before the first implementation commit.

**5. Two-stage skill routers.** GSD v1.40 (`#2792`) shows six namespace meta-skills (`workflow`, `project`, `quality`, `context`, `manage`, `ideate`) instead of ~86 eager entries. Their own count: ~2,150 tokens down to ~120. Descriptions are pipe-separated keyword tags ≤60 characters. On runtimes with non-recursive skill loaders the installer nests concrete skills under the routers; on Claude/Cursor/Codex the layout stays flat because nested names are uninvokable. [2][5]

`using-skillgrid` already routes. The eager listing in a Cursor session is still the full catalog.

**Smaller, later.** Node repair classifies a failed task as RETRY / DECOMPOSE / PRUNE with a default budget of two attempts. [5] Verifier milestone-scope filtering marks later-phase gaps as deferred, not failed. [5] Read-before-edit is an advisory PreToolUse hook for runtimes without Claude’s built-in guard. [5]

### What not to copy

- Markdown-only `.planning/` as the memory layer — ADR-0012 already chose SQLite. [11]
- The 70+ skill / 46-capability catalog and absent-equals-enabled defaults. [1][2]
- Worktree-by-default parallelism — Skillgrid is serial, one change at a time. [11]
- ORM schema-drift (Prisma / Drizzle / …) — this Go tree has no such pair. The *pattern* (two artifacts that must move together, checked at plan time and execute time) is already implied by the coverage and Nyquist items above. [5]

## Cross-Dimension Insights

GSD and Skillgrid solved the same three failures — context rot, session amnesia, unverified “done” — with different stores. GSD put everything in inspectable markdown and then had to invent `gsd-tools` so agents would not corrupt `STATE.md`. [2][5] Skillgrid put memory in SQLite and left `state.yaml` as a pointer plus a prose log. [12] The interesting borrow is therefore not GSD’s files; it is GSD’s **refusal to let the model be the state machine**.

The other cross-cut: Skillgrid already copied GSD’s *rhetoric* (`effort:`, context rot, byte budgets) [7][8] while leaving GSD’s *gates* (plan-check, Nyquist, coverage, scope recovery, agent-facing headroom) as prose in skills. [5][9] A change that only edits more skill text will not close that gap.

## Contrary Evidence

The strongest reason not to take any of this: Skillgrid’s pipeline is already longer than GSD’s five-step loop (brainstorming → blueprints → slicing → approval → execution → qa → review → ship → reflect). [9] Adding plan-check, Nyquist, and a state CLI can become ceremony on top of ceremony. GSD itself says the loop is unjustified for one-turn work. [3] Any borrow must stay behind the existing effort-budget table [7] and must not become a sixth mandatory phase for `trivial` / `small` fast-track changes.

A second contrary: GSD’s file-only state is a feature for git visibility and “no server.” [2] Skillgrid already rejected that for *memory*. Using a small CLI to mutate `state.yaml` is consistent with “repo is source of truth” [11] only if the CLI writes the file the agent would have written — it must not hide state in SQLite.

## Recommendations

Bound to this change’s briefing. Interview still chooses how many of the five land in v1 of the change.

1. **Treat the five mechanisms as the candidate scope, in the listed order.** Confidence **high** on the ranking relative to GSD’s own docs: context monitor and state CLI are infrastructure (hooks + one command); plan-check / Nyquist / routers change planner and skill-layout behavior. [5][6][10]
2. **Do not copy GSD memory or intel files.** Confidence **high**: ADR-0012 and the second-brain findings already settled the store. [11]
3. **Keep `effort:` and skill-size budgets as-is.** Confidence **high**: they are already the GSD borrow, including the “do not fork the orchestrator” rule. [7][8]
4. **Leave current_change on `2026-10-02-mnemonic-project-init` until that change ships or parks.** Confidence **high**: locked serial constraint. [11][12] This spec is queued, not current.
5. **Node repair and the research gate are follow-ups**, not the first slice, unless the interview pulls them forward. Confidence **medium**: they are real GSD features [5] but they assume the plan-checker and findings contract already exist.

## Open Questions

- Which of the five mechanisms are in the first slice vs. later changes? (Interview.)
- Is the state CLI a `skillgrid` subcommand (fits the single binary) or a small Node script next to the existing drift checkers?
- For the context monitor: which harness events are actually available on Cursor / OpenCode / Kilo today, and what is the utilization signal (tokens vs. percent)?
- For two-stage routers: Cursor’s skill loader is recursive/flat per GSD’s matrix [2] — does nesting help here, or only shorter `description:` tags plus `using-skillgrid`?
- Does Nyquist attach to `acceptance.feature` scenarios, to `#### Gates` in `tasks.md`, or both?

## Source Appendix

Every inline `[n]` resolves here. Sources are the files opened this run. GSD paths are in `~/git/ai-test/gsd-core`. Skillgrid paths are in this repo.

| [n] | Claim/finding it supports | Publisher | Pub date | Accessed | Confidence |
|-----|---------------------------|-----------|----------|----------|------------|
| [1] | GSD product statement; five-step loop; installer-required; 71 skills / 46 capabilities in this checkout | [gsd-core/README.md](file:///Users/paladm/git/ai-test/gsd-core/README.md) + `skills/`, `capabilities/` listing | 2026-09-02 (tree mtime) | 2026-10-02 | high |
| [2] | Layered architecture; fresh-context / thin orchestrator / file state / absent=enabled; workflow byte budgets; two-stage routing + runtime nesting caveats | [gsd-core/docs/ARCHITECTURE.md](file:///Users/paladm/git/ai-test/gsd-core/docs/ARCHITECTURE.md) | 2026-09-02 | 2026-10-02 | high |
| [3] | Context-rot definition; fresh subagents; spec + meta-prompting; `.planning/` + `STATE.md`; hooks/headroom; quick/fast escape hatches | [gsd-core/docs/explanation/context-engineering.md](file:///Users/paladm/git/ai-test/gsd-core/docs/explanation/context-engineering.md) | 2026-09-02 | 2026-10-02 | high |
| [4] | Phase-loop rationale; milestone vs phase; good phase scope | [gsd-core/docs/explanation/the-phase-loop.md](file:///Users/paladm/git/ai-test/gsd-core/docs/explanation/the-phase-loop.md) | 2026-09-02 | 2026-10-02 | high |
| [5] | Nyquist, plan-check, coverage, scope reduction, research gate, context monitor + utilization guard, CLI tools, STATE consistency, global learnings, intel JSON, namespace routers, node repair | [gsd-core/docs/FEATURES.md](file:///Users/paladm/git/ai-test/gsd-core/docs/FEATURES.md) §§15–22, 35, 59, 64–65, 69, 71, 89–90, 123–124 | 2026-09-02 | 2026-10-02 | high |
| [6] | Context-monitor thresholds, debounce, fail-open, bridge file | [gsd-core/docs/context-monitor.md](file:///Users/paladm/git/ai-test/gsd-core/docs/context-monitor.md) | 2026-09-02 | 2026-10-02 | high |
| [7] | Skillgrid already borrowed GSD effort signal and honest-overhead table | [.agents/skills/_shared/rules/effort-budgets.md](.agents/skills/_shared/rules/effort-budgets.md) | 2026-10-02 (working tree) | 2026-10-02 | high |
| [8] | Skillgrid skill-size ceilings 22/38/40 KB | [.agents/skills/_shared/skill-size-budget.json](.agents/skills/_shared/skill-size-budget.json) | 2026-10-02 (working tree) | 2026-10-02 | high |
| [9] | Skillgrid pipeline, artifacts, approval gate, waves | [docs/user-guide/03-workflow-usage.md](docs/user-guide/03-workflow-usage.md) | 2026-10-02 (working tree) | 2026-10-02 | high |
| [10] | Skillgrid hooks: git + Stop + capture + checkpoint; no agent-facing headroom warning | [docs/user-guide/04-hooks.md](docs/user-guide/04-hooks.md) | 2026-10-02 (working tree) | 2026-10-02 | high |
| [11] | Locked serial constraint; ADR-0012 SQLite store; drift-guard existence | [.skillgrid/ASSUMPTIONS.md](.skillgrid/ASSUMPTIONS.md) | 2026-10-02 (working tree) | 2026-10-02 | high |
| [12] | `current_change` is `2026-10-02-mnemonic-project-init`; `progress` is a concatenated log | [.skillgrid/state.yaml](.skillgrid/state.yaml) | 2026-10-02 (working tree) | 2026-10-02 | high |

---

## Research: What can Skillgrid learn from BMAD?

> **Decision this serves:** Add a second source to `skillgrid-workflow-updates`: which BMAD mechanisms Skillgrid already absorbed, which remaining ones are worth taking, and which BMAD surface must not be copied. Merge with the GSD ranking where they overlap.
>
> **Type:** competitive + technical
> **Mode:** research (single pass)
> **Date:** 2026-10-02 · **Status:** complete
> **Primary corpus:** local install `~/git/ai-test/BMAD` (an *installed* BMad workspace: `.agents/skills/`, `.opencode/commands/`, `_bmad/` config — no upstream README at repo root). Skillgrid `based_on` provenance and pipeline skills read this run.
> **Caveat:** This is not the BMad Method source tree. Claims are about this install (tree mtime 2026-09-13). Module docs URLs in the help CSV were not fetched this run.

## Executive Summary

BMAD in this checkout is a **modular method install**: Core + BMad Method (BMM) + Test Architecture Enterprise (TEA) + BMad Builder (BMB) + BMAD Loop + Creative Intelligence (CIS). 57 skills, a CSV help catalog with `phase` / `preceded-by` / `followed-by` / `required` / `outputs`, four-layer TOML config, named agent personas, and Python (`uv run`) resolvers. [B1][B2][B3]

Skillgrid already lists BMAD as a source of logic [B4] and has absorbed the parts that fit: `research` / `code-research` from `bmad-deep-recon`, `qa` from TEA trace + test-levels, `parallel-code-review` and verification-gap from `bmad-code-review` / `bmad-review`, `ticketing` from sprint-planning ideas, `onboarding` from `bmad-project-context`. [B4][B5] The four-state QA gate (PASS / CONCERNS / FAIL) is the same vocabulary BMAD’s sprint-planning readiness gate uses. [B6][B13]

**Remaining BMAD edge is not more phases.** It is (1) a **machine-readable next-step graph** plus completion-from-markers, (2) an **append-only memlog** from which SPEC/briefing is *derived*, (3) a named **correct-course** impact pass when the plan is wrong mid-build, (4) a **shared elicitation checkpoint** any skill can pause into, (5) a **deterministic sprint-status file** written by a script after a readiness gate, (6) **review lenses** that apply to docs and code, (7) **skill evals** (baseline / variant / trigger), and (8) a **deferred-work sweep** that re-checks the ledger against the live tree. [B2][B7][B8][B9][B10][B11][B12][B14]

**Overlap with the GSD ranking:** BMAD sprint-readiness ≈ GSD plan-checker; TEA ATDD red-phase ≈ GSD Nyquist; the help CSV ≈ GSD two-stage routers (graph of *what’s next* vs namespaces of *which family*). Interview should merge those, not add a sixth and seventh mandatory phase. [B3][B6][B15]

**Biggest caveat:** Do not import BMM’s brief → PRD → UX → architecture → epics → sprint waterfall, CIS personas, or party-mode theater. [B1][B3][B16] Skillgrid’s pipeline is already longer than BMM’s ship path. [B13]

## Findings

### What this BMAD checkout is

Root has no README. Layout is an installer drop: `_bmad/config.toml` (installer-managed, overwritten on reinstall; durable pins live in `_bmad/custom/`), `_bmad/{core,bmm,tea,bmb,cis,bmad-loop}/`, `_bmad/scripts/{resolve_config,resolve_customization,memlog,render_skill}.py`, 57 skills under `.agents/skills/`, matching OpenCode slash commands. [B1][B2]

BMM required path (from the help CSV): product brief or PRFAQ → PRD → (optional UX) → architecture → epics/stories → sprint planning (readiness PASS/CONCERNS/FAIL, then `sprint-status.yaml`) → build → optional code-review / walkthrough / e2e / retrospective. Anytime skills: spec, correct-course, project-context, sprint status. [B3]

TEA adds teach-me-testing, risk-based test design, framework/CI scaffold, ATDD red-phase before implementation, automate, test-review (0–100), NFR evidence audit, traceability gate. [B15]

BMB builds/analyzes/converts agents, workflows, and modules; eval-runner grades skills. [B2][B11] Loop is automation-native sweep + interactive resolve of CRITICAL escalations against a frozen spec. [B10][B17] CIS is creative personas (storyteller, design-thinking, brainstorming, TRIZ, innovation, presentation) — out of Skillgrid’s engine scope. [B1]

Help (`bmad-help`) reads `_bmad/_config/bmad-help.csv`, resolves config via the Python merger, and recommends the next skill from **artifact presence plus completion markers**. File presence means “started,” not “done.” A draft marker keeps the skill in progress. Recommend only what is relevant; offer to run the one next skill. Recommend a **fresh context window** per skill. [B2]

### Already in Skillgrid

| BMAD idea | Skillgrid today | Source |
|---|---|---|
| Deep recon epistemics / type packs | `research`, `code-research` (`based_on: bmad-deep-recon`) | [B5] |
| Traceability + test-level framework | `qa` (`based_on: BMAD:bmad-testarch-trace` + test-levels) | [B5] |
| Multi-lens code review + verification-gap | `parallel-code-review`, `qa/references/verification-gap.md` | [B5] |
| Sprint / ticket planning | `ticketing` (`based_on: BMAD:bmad-sprint-planning`) | [B5] |
| Project context / AGENTS block | `onboarding` (`based_on: BMAD:bmad-project-context`) | [B5] |
| PASS / CONCERNS / FAIL | QA four-state gate (plus WAIVED) | [B6][B13] |
| Interview until shared model | `interviewing` design-tree + clarity gate (from mattpocock grilling, not BMAD AE) | [B18] |
| Close-of-cycle retro | `reflect` after ship | [B19] |
| Durable memory | Mnemonic SQLite — stronger than BMAD `.memlog.md` per skill | [B7][B20] |

### Remaining mechanisms (ranked)

**1. Machine-readable skill graph + help.** The CSV is the catalog: module, skill, menu-code, phase, precedes/follows, required, output-location, outputs. Help orients from that graph and from artifact markers, not from conversation memory. [B2][B3] Skillgrid’s `using-skillgrid` routes in prose; `resume` infers phase from which spec files exist. There is no committed graph an agent can query for “what is required next, and is it actually finished?” Complements GSD’s six namespace routers: GSD shrinks *eager listing*; BMAD help shrinks *what to do next*.

**2. Append-only memlog; derived contracts.** `memlog.py` is write-only, chronological, no edit/delete, atomic rename. Frontmatter is descriptive only; “done” is an event line. `bmad-spec` treats `.memlog.md` as canonical; `SPEC.md` and companions are **re-derived every run**. Capability IDs (`CAP-N`) are stable; hand-edits to SPEC.md are overwritten. That lets PRD / UX / architecture / epics land in any order without merge-fighting the kernel. [B7] Skillgrid briefing/blueprint are hand-edited. Mnemonic stores decisions, but it is not the single writer of `briefing.md`. The borrow is “interview/log is canonical; the briefing is a render,” not “replace SQLite with a markdown log.”

**3. Correct course.** Mid-sprint, load PRD / epics / architecture / UX / spec, assess impact, write a sprint-change proposal with a handoff (start over / update PRD / redo architecture / fix stories). [B8] Skillgrid has no named skill for “the plan is wrong while `tasks.md` is in flight.” `resume` continues; `reflect` only runs after ship. [B13][B19]

**4. Shared elicitation checkpoint.** `bmad-advanced-elicitation` is invoked by other skills at a pause. It serves a five-method menu from a CSV (pre-mortem, first principles, red team, Socratic, …) via a picker script so the catalog never enters context whole. Apply / reject / proceed; the enhanced version is handed back. [B9] Skillgrid interviewing is front-loaded. Later phases have no shared “pressure this draft” skill.

**5. Readiness gate + script-owned status file.** Sprint-planning first asks “is this implementable?” (PASS / CONCERNS / FAIL). On PASS a script writes `sprint-status.yaml`; the skill’s judgment is which files are epics and how to reconcile orphans — parsing and merging are not LLM work. Status / validate / fix are separate intents. Headless returns JSON. [B6] Same lesson as GSD’s `gsd-tools state *`: the model must not be the state machine. Skillgrid `state.yaml` `progress` is still a narrative blob. [B20]

**6. Review lenses, not only code reviewers.** `bmad-review` selects lenses by `applies_to` / `when`, fans independent lenses, then dependent `after` lenses. Canonical finding shape: location, trigger_condition, guard_snippet, potential_consequence. Works on diffs, PRDs, specs, prose. Adversarial lens requires ≥10 findings. Never uninvited. [B12] Skillgrid’s parallel-code-review is the code half; there is no first-class doc-lens path for a briefing or blueprint.

**7. Skill evals.** Four modes: baseline (skill vs bare model, same input), variant (does this section earn its place?), quality (rubric), trigger (does `description:` fire on the right queries). Isolated working dirs; `state_prefix` places a case mid-workflow in one shot. [B11] Skillgrid `eval` is retrieval ablation, not skill quality. [B20]

**8. Deferred-work sweep.** Loop sweep is read-only and automation-only (`BMAD_LOOP_MODE=1`). Every open `DW-n` is verified against the tree; statuses are assumed stale. Partition: already_resolved (file:line or commit), bundles (one-session goals), blocked (named future story), skip, decisions (human options with a recommendation). Orchestrator closes resolved entries. [B10] Skillgrid backlog items can sit `needs-triage` after the code moved on.

**Smaller, later.** Frozen-spec escalate → `bmad-loop-resolve` (interactive, writes `resolution.json`, then re-drive). [B17] Per-skill `customize.toml` + team + user merge, activation hooks, `persistent_facts`. [B1][B16] TEA NFR evidence audit. [B15] PRFAQ / Working Backwards as an alternative to a product brief. [B3] Party-mode “clash, don’t reconcile” is already Skillgrid’s teams-vs-fan-out rule; the theatrical party is not. [B16]

### What not to copy

- BMM’s long planning waterfall and a second `_bmad-output/` tree beside `.skillgrid/`. [B3]
- Named personas (Mary / John / Winston / Murat) and CIS coaches as the default UX. [B1]
- Party mode as a product feature (HTML keepsakes, grudges, fourth-wall fiction). [B16]
- Python/`uv` as the skill runtime — Skillgrid’s binary is Go; a new resolver is an ADR. [B1][B20]
- Re-importing deep-recon, TEA trace, or verification-gap as new skills — they are already here. [B5]

## Cross-Dimension Insights

BMAD and GSD both refuse to let the model own sequencing and state. GSD does it with `gsd-tools` + `STATE.md` sync. [GSD §] BMAD does it with a help CSV, a sprint-status script, and a memlog that only appends. [B2][B6][B7] Skillgrid already refused the markdown *memory* store (ADR-0012) but still lets the model own *pipeline* state (`state.yaml` prose, phase inferred from files). [B20] The combined lesson is one CLI + one graph, not two frameworks.

The memlog idea is the one BMAD has that GSD does not: **the interview log is the source; the spec is a view.** [B7] That is closer to “Mnemonic is the index, the repo file is the render” than to GSD’s `CONTEXT.md`. If Skillgrid adopted it, `briefing.md` would be regenerated from observations / a change memlog, not patched by the agent.

Help-CSV completion detection (“presence ≠ done; look for a finalization marker”) is the same failure `resume` already documents, made queryable. [B2][B13]

## Contrary Evidence

BMAD’s surface is large (57 skills, six modules, persona menus). [B1][B2] Adding help-CSV + memlog + correct-course + elicitation + evals on top of Skillgrid’s existing chain is ceremony. The GSD findings already warned that Skillgrid’s pipeline is longer than GSD’s five steps. [GSD contrary] Any BMAD borrow must attach to an existing skill (`using-skillgrid`, `resume`, `interviewing`, `slicing`, `qa`, `reflect`) rather than become a new module pack.

A second contrary: `.memlog.md` as the briefing SoT fights “the committed briefing is the design.” Dual-write is already the rule (file wins, Mnemonic mirrors). A memlog that *replaces* `briefing.md` would invert that. The safe form is: memlog or observations accumulate; `briefing.md` is still the committed contract, regenerated then committed.

Party mode’s “don’t reconcile voices” is useful for debugging; as a default product loop it wastes tokens. Skillgrid already split fan-out (verdict) vs teams (investigation). [B16]

## Recommendations

Bound to this change’s briefing. Merge with the GSD five; do not stack a second five.

1. **Fold BMAD (1) and (5) into the GSD state-CLI + router work.** A `skillgrid` command that validates/syncs `state.yaml` should also be able to answer “next required skill / is this artifact complete?” from a small committed graph (help-CSV shape, Skillgrid phase names). Confidence **high**. [B2][B6]
2. **Add correct-course as a named mid-change skill** (or a `resume` mode) that writes a change proposal across briefing / blueprint / tasks / ADRs. Confidence **high** — there is no current owner. [B8]
3. **Treat memlog-as-canonical as an interview decision, not a default.** If yes: append-only change log + regenerate briefing; Mnemonic remains the cross-change store. Confidence **medium** — conflicts with “hand-edited briefing is the spec” unless the regenerate-then-commit rule is explicit. [B7]
4. **Elicitation checkpoint and doc-lenses are cheap attach points** (`interviewing` / `writing-blueprints` / `qa` already pause). Do not ship party mode. Confidence **medium**. [B9][B12]
5. **Skill evals and deferred-work sweep are follow-ups.** Eval needs cases and an adapter; sweep needs a ledger Skillgrid does not have (Backlog tasks are the closest). Confidence **medium**. [B10][B11]
6. **Do not copy BMM/CIS/BMB as modules.** Confidence **high**. [B1][B3]

## Open Questions

- Is the skill graph a committed CSV/YAML under `.agents/` or generated from skill frontmatter?
- Does correct-course live as `resume` (mid-change) or a new skill?
- Is `briefing.md` allowed to be regenerated from a memlog/Mnemonic, or must it stay hand-authored?
- Do skill evals belong in this change or in a later Hub-content change?
- TEA NFR audit vs existing QA threat-matrix — merge or skip?

## Source Appendix

`[Bn]` citations. BMAD paths are under `~/git/ai-test/BMAD`. Pub date is that tree’s mtime (2026-09-13) unless noted.

| [n] | Claim/finding it supports | Publisher | Pub date | Accessed | Confidence |
|-----|---------------------------|-----------|----------|----------|------------|
| [B1] | Modules, personas, installer-managed config vs custom pins; 57 skills | [_bmad/config.toml](file:///Users/paladm/git/ai-test/BMAD/_bmad/config.toml) + `.agents/skills/` listing | 2026-09-13 | 2026-10-02 | high |
| [B2] | Help CSV schema; completion = markers not presence; fresh window; catalog rows | [bmad-help/SKILL.md](file:///Users/paladm/git/ai-test/BMAD/.agents/skills/bmad-help/SKILL.md) + [_config/bmad-help.csv](file:///Users/paladm/git/ai-test/BMAD/_bmad/_config/bmad-help.csv) | 2026-09-13 | 2026-10-02 | high |
| [B3] | BMM phase order, required gates, anytime skills | [_bmad/bmm/module-help.csv](file:///Users/paladm/git/ai-test/BMAD/_bmad/bmm/module-help.csv) | 2026-09-13 | 2026-10-02 | high |
| [B4] | Skillgrid names BMAD as a logic source | [docs/user-guide/00-start-here.md](docs/user-guide/00-start-here.md) + [02-skills.md](docs/user-guide/02-skills.md) | 2026-10-02 | 2026-10-02 | high |
| [B5] | `based_on` BMAD on research, qa, parallel-code-review, ticketing, onboarding | `.agents/skills/**/SKILL.md` frontmatter | 2026-10-02 | 2026-10-02 | high |
| [B6] | Sprint readiness PASS/CONCERNS/FAIL; script owns `sprint-status.yaml` | [bmad-sprint-planning/SKILL.md](file:///Users/paladm/git/ai-test/BMAD/.agents/skills/bmad-sprint-planning/SKILL.md) | 2026-09-13 | 2026-10-02 | high |
| [B7] | Memlog invariants; SPEC.md derived; CAP-N stability | [_bmad/scripts/memlog.py](file:///Users/paladm/git/ai-test/BMAD/_bmad/scripts/memlog.py) + [bmad-spec/SKILL.md](file:///Users/paladm/git/ai-test/BMAD/.agents/skills/bmad-spec/SKILL.md) | 2026-09-13 | 2026-10-02 | high |
| [B8] | Correct-course loads planning artifacts and writes a change proposal | [bmad-correct-course/SKILL.md](file:///Users/paladm/git/ai-test/BMAD/.agents/skills/bmad-correct-course/SKILL.md) | 2026-09-13 | 2026-10-02 | high |
| [B9] | Advanced elicitation menu, picker script, apply/reject/proceed | [bmad-advanced-elicitation/SKILL.md](file:///Users/paladm/git/ai-test/BMAD/.agents/skills/bmad-advanced-elicitation/SKILL.md) | 2026-09-13 | 2026-10-02 | high |
| [B10] | Deferred-work sweep partitions; ledger statuses assumed stale | [bmad-loop-sweep/SKILL.md](file:///Users/paladm/git/ai-test/BMAD/.agents/skills/bmad-loop-sweep/SKILL.md) | 2026-09-13 | 2026-10-02 | high |
| [B11] | Skill eval modes: baseline, variant, quality, trigger | [bmad-eval-runner/SKILL.md](file:///Users/paladm/git/ai-test/BMAD/.agents/skills/bmad-eval-runner/SKILL.md) | 2026-09-13 | 2026-10-02 | high |
| [B12] | Review lenses, finding shape, docs+code, never uninvited | [bmad-review/SKILL.md](file:///Users/paladm/git/ai-test/BMAD/.agents/skills/bmad-review/SKILL.md) | 2026-09-13 | 2026-10-02 | high |
| [B13] | Skillgrid pipeline, approval gate, resume-from-files | [docs/user-guide/03-workflow-usage.md](docs/user-guide/03-workflow-usage.md) | 2026-10-02 | 2026-10-02 | high |
| [B14] | Loop module rows (setup + sweep) | [_bmad/bmad-loop/module-help.csv](file:///Users/paladm/git/ai-test/BMAD/_bmad/bmad-loop/module-help.csv) | 2026-09-13 | 2026-10-02 | high |
| [B15] | TEA ATDD / NFR / trace order | [_bmad/tea/module-help.csv](file:///Users/paladm/git/ai-test/BMAD/_bmad/tea/module-help.csv) | 2026-09-13 | 2026-10-02 | high |
| [B16] | Party mode: clash don’t reconcile; session/subagent/agent-team; theatrical close | [bmad-party-mode/SKILL.md](file:///Users/paladm/git/ai-test/BMAD/.agents/skills/bmad-party-mode/SKILL.md) | 2026-09-13 | 2026-10-02 | high |
| [B17] | Loop-resolve: interactive frozen-spec repair, resolution.json | [bmad-loop-resolve/SKILL.md](file:///Users/paladm/git/ai-test/BMAD/.agents/skills/bmad-loop-resolve/SKILL.md) | 2026-09-13 | 2026-10-02 | high |
| [B18] | Interviewing is design-tree grilling, based_on mattpocock | [.agents/skills/planning/interviewing/SKILL.md](.agents/skills/planning/interviewing/SKILL.md) | 2026-10-02 | 2026-10-02 | high |
| [B19] | Reflect is terminal, post-ship only | [.agents/skills/lifecycle/reflect/SKILL.md](.agents/skills/lifecycle/reflect/SKILL.md) | 2026-10-02 | 2026-10-02 | high |
| [B20] | Locked store + serial; state.yaml pointer; eval is retrieval | [.skillgrid/ASSUMPTIONS.md](.skillgrid/ASSUMPTIONS.md) + [state.yaml](.skillgrid/state.yaml) | 2026-10-02 | 2026-10-02 | high |

---

## Research: How BMAD scripts make the method repeatable

> **Decision this serves:** The GSD/BMAD state-CLI and plan-check items are not “write more skill prose.” They are “put the repeating half in a script with a JSON contract.” This section names that contract so the briefing can require it.
>
> **Type:** technical
> **Mode:** research (single pass)
> **Date:** 2026-10-02 · **Status:** complete
> **Primary corpus:** `~/git/ai-test/BMAD/_bmad/scripts/` and per-skill `scripts/` (66 Python files this run); Skillgrid `deterministic-boundary.md`, `skill-write`, pipeline SKILL.md files.

## Executive Summary

BMAD does not trust the model to merge config, parse epics, assign IDs, serve a catalog, append a memlog, or write `sprint-status.yaml`. Those jobs live in scripts. The skill line names *judgment*; the script owns *the rest*. [S1][S4][S5]

This checkout has **5 shared platform scripts** under `_bmad/scripts/` and **16 skill `scripts/` directories**. Typical method scripts print **only JSON** (argparse failures included), write atomically (`temp` → `fsync` → `replace`), and ship tests beside the script. [S1][S2][S4][S8] A scanner (`scan-scripts.py`) fails skills that use `input()`, lack argparse, lack JSON output, or lack tests. [S8]

Skillgrid already wrote the same rule in `_shared/rules/deterministic-boundary.md` and `skill-write`. [S9][S10] QA, ship, citations, and design skills actually call scripts. **Planning and lifecycle skills almost never do** — `using-skillgrid`, `interviewing`, `writing-blueprints`, `slicing`, `resume` have no `scripts/` invocation in their SKILL.md. [S11] That is why `state.yaml` `progress` is still a narrative blob and resume still infers phase by reading files in prose.

**Do this:** adopt BMAD’s *script contract* (JSON stdout, atomic write, tests, no `input()`) on Node/Go — not Python/`uv` as a new runtime — and extract the repeating halves of the pipeline skills first: state validate/sync, phase/completion from artifacts, catalog/help paging, requirement-ID coverage, briefing/memlog append. [S1][S4][S6][S9]

## Findings

### The shared platform (always called first)

Every BMAD skill’s “On Activation” runs `uv run {project-root}/_bmad/scripts/resolve_customization.py` then `resolve_config.py`. Both dump merged TOML as JSON. [S1][S2] Shared library `config_utils.py` does a strict structural merge (scalars override, tables deep-merge, arrays of tables keyed by `code`/`id` replace). [S3] `memlog.py` is append-only, write-only, atomic. [S5] `render_skill.py` snapshots a skill with config tokens substituted and a content hash — the running agent reads a frozen copy, not a live file the user can half-edit mid-run. [S2]

`sys.dont_write_bytecode = True` is set so install trees do not grow `__pycache__`. [S1]

### Per-skill scripts: judgment vs machine

The sprint-planning docstring is the pattern in one paragraph: *“The LLM decides which files are epics (discovery is judgment); this script owns everything after that decision: parsing, key derivation, ordering, status preservation, … and validation.”* Writes are atomic; post-write validation rolls the file back if it fails; `--dry-run` reports drift (`in_sync` / `illegal` / `orphans`) without writing. Subcommands: `generate`, `status`, `validate`. [S4]

Other repeating jobs extracted the same way:

| Script | Repeating job the model must not redo | Contract |
|---|---|---|
| `pick_methods.py` / `brain.py` | Serve a CSV catalog without loading it all; `--all` is a deliberate dump | lean TSV default, `--json`; `list` refuses neither `--category` nor `--all` [S6] |
| `recon_kit.py` | Citation `[n]` vs appendix, memlog tally, staleness windows, slug, HTML-escape sources | one JSON object; exit 0/1/2 [S7] |
| `lint_spine.py` | Duplicate/non-monotonic AD-n, missing fields, TBD/TODO, unpinned versions | JSON findings; exit always 0 — caller judges [S12] |
| `git_evidence.py` | Commit/file churn over a range (merges counted separately) | JSON only, including argparse errors; **measures, never judges** [S13] |
| `sprint_status.py` (retrospective) | Read the same key grammar as `sprint_plan.py` | shared regex contract [S4] |
| `scan-scripts.py` | Enforce the script contract on other skills | AST: no `input()`, argparse, `json.dumps`, exit codes, tests exist [S8] |

Catalog servers exist so a 200-row methods CSV never enters the skill body. [S6] That is the same token-economics argument Skillgrid’s deterministic-boundary file already makes. [S9]

### The agentic script contract (what “repeatable” means)

From the scripts and the scanner, the contract is: [S4][S7][S8][S13]

1. **Stdout is machine-shaped.** JSON (or lean TSV). Argparse failures are JSON too, not usage text — `git_evidence.py` even disables `-h` so help cannot break the contract. [S13]
2. **Stderr is diagnostics.** `recon_kit.py` states this explicitly. [S7]
3. **Exit codes are a vocabulary.** 0 = pass, 1 = findings/fail, 2 = usage. Some linters always exit 0 and put findings in JSON so the *caller* decides severity. [S7][S12]
4. **Writes are atomic** and reversible if validation fails. [S4][S5]
5. **No `input()`.** Flags only. Blocks agents. [S8]
6. **Tests sit next to the script** (`scripts/tests/test_*.py`). Status-definition comments are byte-pinned to the template so comment and parser cannot drift. [S4][S8]
7. **PEP 723** inline deps (`# /// script`) so `uv run` needs no venv. [S8]
8. **The skill names the split.** “You judge X; the script does Y.” If the script fails, do not guess: tell the user and offer the fix flow. [S4]

### What Skillgrid already does vs the gap

The rule is written: identical input → identical output ⇒ script, not a skill line. Example in that file is `state-drift-check.mjs`. [S9] `skill-write` refuses English if-statements. [S10]

Scripts that pipeline skills actually call today: QA drift / size budget / qa-gate, `state-lock.mjs` on ship, `sources.py` for citations, `sdd-workspace` / `review-package` on execution. Design skills are script-heavy (search, tokens). [S11]

**Missing from the spine:** planning and resume still describe deterministic procedures in prose — which spec files exist, how to advance `state.yaml`, how to list the next skill, how to check requirement IDs appear in `tasks.md`. A `rg` of planning `SKILL.md` files for `Run \`node` / `scripts/` returned no matches. [S11]

That is the concrete form of the earlier GSD/BMAD lesson (“do not let the model be the state machine”). BMAD paid it down with one script per repeating job. Skillgrid wrote the convention and applied it at the QA/hook edge, not at the planner.

## Cross-Dimension Insights

The GSD state CLI, the BMAD help CSV, and these scripts are one mechanism: **a typed I/O boundary**. The agent supplies a judgment (which files, which topic) and receives a structured envelope. Without that envelope, every later skill re-parses markdown and drifts.

Catalog-paging scripts (`pick_methods`, `brain`) are the implementation of two-stage routers for *data*, not just skills: never load the whole library. [S6] Skillgrid’s `effort:` and skill-size budget shrink SKILL.md; they do not page the method catalog the agent is choosing from.

`lint_spine` / `recon_kit citations` are Nyquist/plan-check as grep: IDs and `[n]` markers are countable. [S7][S12] A Skillgrid `requirement-coverage.mjs` that greps `acceptance.feature` IDs against `tasks.md` is the same move.

## Contrary Evidence

A script per skill can become a second codebase. BMAD has 66 Python files and a Python/`uv` toolchain Skillgrid has not locked. [S14] The Skillgrid convention already says `.mjs` or bash, and “no new dependency without an ADR.” [S9][S15] Steal the *contract*, implement it in Node (existing `yaml` package + drift scripts) or as `skillgrid` subcommands in Go. Do not add `uv` as a pipeline runtime.

Some BMAD scripts are builder-only (`count_tokens`, `scaffold-*`, `scan-path-standards`). Those are BMB quality gates, not the method loop. First slice is the five platform-shaped jobs: config/state, memlog/append, catalog page, parse/validate status, citation/ID lint.

`scan-scripts.py`’s “must have json.dumps if >20 lines” is a heuristic and would false-flag a TSV catalog server. [S8][S6] Keep the *intent* (structured, testable, no prompts), not that AST check verbatim.

## Recommendations

1. **Add a script-contract requirement to this change.** Every new pipeline helper is a `.mjs`: JSON (or documented TSV) on stdout, atomic writes, tests, no interactive prompts. Confidence **high**. Locked: JS, not Python/`uv`, not a new Go CLI. [S4][S8][S9]
2. **Extract these first (maps onto briefing reqs 2, 3, 7):** `state validate|sync|advance`; `phase-from-artifacts` (resume); `requirement-coverage` (plan-check); a help/catalog pager if routers land. Confidence **high**. [S4][S6][S11]
3. **Keep `deterministic-boundary.md` as the rule; add BMAD’s split sentence to skill-write.** “The agent decides *which*; the script owns *everything after*.” Confidence **high**. [S4][S10]
4. **Do not adopt Python/`uv` or `scan-scripts.py` as-is.** Port the checks we care about into the existing skill-size / skill-write review. Confidence **high**. [S8][S15]
5. **Memlog script is optional** until the interview allows derived briefings. If yes, one append-only `.mjs` writer, same atomicity as `memlog.py`. [S5]

## Open Questions

- ~~Node vs Go?~~ **Locked 2026-10-02:** JS `.mjs` scripts; steal BMAD logic; not Python/`uv`; not a new Go CLI for these helpers.
- Does `phase-from-artifacts` become the only way `resume` and `using-skillgrid` learn the phase?
- Should QA grow a “pipeline skill called a script for every deterministic step” lint, or is skill-write + review enough?

## Source Appendix

| [n] | Claim/finding it supports | Publisher | Pub date | Accessed | Confidence |
|-----|---------------------------|-----------|----------|----------|------------|
| [S1] | `resolve_config.py` four-layer TOML → JSON; no bytecode | [_bmad/scripts/resolve_config.py](file:///Users/paladm/git/ai-test/BMAD/_bmad/scripts/resolve_config.py) | 2026-09-13 | 2026-10-02 | high |
| [S2] | Customization merge + project-root walk; `render_skill.py` snapshot/hash | [resolve_customization.py](file:///Users/paladm/git/ai-test/BMAD/_bmad/scripts/resolve_customization.py) + [render_skill.py](file:///Users/paladm/git/ai-test/BMAD/_bmad/scripts/render_skill.py) | 2026-09-13 | 2026-10-02 | high |
| [S3] | Strict TOML load + keyed array merge | [_bmad/scripts/config_utils.py](file:///Users/paladm/git/ai-test/BMAD/_bmad/scripts/config_utils.py) | 2026-09-13 | 2026-10-02 | high |
| [S4] | Sprint plan: LLM discovers epics, script owns parse/merge/validate; atomic write; JSON only | [bmad-sprint-planning/scripts/sprint_plan.py](file:///Users/paladm/git/ai-test/BMAD/.agents/skills/bmad-sprint-planning/scripts/sprint_plan.py) | 2026-09-13 | 2026-10-02 | high |
| [S5] | Memlog append-only / write-only / atomic / no status field | [_bmad/scripts/memlog.py](file:///Users/paladm/git/ai-test/BMAD/_bmad/scripts/memlog.py) | 2026-09-13 | 2026-10-02 | high |
| [S6] | Catalog pager; refuse full dump | [pick_methods.py](file:///Users/paladm/git/ai-test/BMAD/.agents/skills/bmad-advanced-elicitation/scripts/pick_methods.py) + [brain.py](file:///Users/paladm/git/ai-test/BMAD/.agents/skills/bmad-brainstorming/scripts/brain.py) | 2026-09-13 | 2026-10-02 | high |
| [S7] | recon_kit: citations, tally, staleness, slug; JSON + exit 0/1/2 | [recon_kit.py](file:///Users/paladm/git/ai-test/BMAD/.agents/skills/bmad-deep-recon/scripts/recon_kit.py) | 2026-09-13 | 2026-10-02 | high |
| [S8] | scan-scripts: no input(), argparse, json.dumps, tests, PEP 723 | [scan-scripts.py](file:///Users/paladm/git/ai-test/BMAD/.agents/skills/bmad-workflow-builder/scripts/scan-scripts.py) | 2026-09-13 | 2026-10-02 | high |
| [S9] | Skillgrid deterministic-boundary rule + state-drift example | [.agents/skills/_shared/rules/deterministic-boundary.md](.agents/skills/_shared/rules/deterministic-boundary.md) | 2026-10-02 | 2026-10-02 | high |
| [S10] | skill-write enforces script-first | [.agents/skills/craft/skill-write/SKILL.md](.agents/skills/craft/skill-write/SKILL.md) | 2026-10-02 | 2026-10-02 | high |
| [S11] | Planning SKILL.md files do not invoke scripts; QA/ship/citations do | `rg` over `.agents/skills/planning/**/SKILL.md` and lifecycle | 2026-10-02 | 2026-10-02 | high |
| [S12] | lint_spine: mechanical AD-n / placeholder checks; exit 0 | [lint_spine.py](file:///Users/paladm/git/ai-test/BMAD/.agents/skills/bmad-architecture/scripts/lint_spine.py) | 2026-09-13 | 2026-10-02 | high |
| [S13] | git_evidence: measure only; JSON argparse errors | [git_evidence.py](file:///Users/paladm/git/ai-test/BMAD/.agents/skills/bmad-retrospective/scripts/git_evidence.py) | 2026-09-13 | 2026-10-02 | high |
| [S14] | 66 Python files, 16 skill script dirs in this checkout | `find` over `~/git/ai-test/BMAD` | 2026-09-13 | 2026-10-02 | high |
| [S15] | No new dependency without an ADR | [.skillgrid/ASSUMPTIONS.md](.skillgrid/ASSUMPTIONS.md) locked constraints | 2026-10-02 | 2026-10-02 | high |

## Research: What can Skillgrid learn from colemedin-skills?

> **Decision this serves:** Scope the remaining `skillgrid-workflow-updates` edge after GSD and BMAD: which Cole mechanisms are already in Skillgrid, which meta-layer pieces are worth taking, and which catalog items must not be copied.
>
> **Type:** competitive (position against a named AI-layer catalog) + technical (which mechanisms fit this repo)
> **Mode:** research (single pass)
> **Date:** 2026-10-02 · **Status:** complete
> **Primary corpus:** local checkout `~/git/ai-test/colemedin-skills` (Cole Medin / `coleam00/skills`, MIT) read this run; Skillgrid skills, hooks, and `docs/user-guide/08-concepts.md` read this run.

## Executive Summary

Cole’s repo is not a framework. It is 34 markdown skills plus six copy-in hooks, built around one ticket loop: **prime → plan → implement → validate → review → commit → PR**. [C1] Skillgrid already absorbed that spine: `08-concepts.md` names the PIV loop, `slicing` is `based_on colemedin-skills:piv-slice-epic`, and the user guide already states Cole’s hook line — “A rule asks; a hook guarantees.” [C2][C3][C4]

**Do not take another PIV catalog.** The remaining lesson is the **meta-layer** around the loop: measure whether always-loaded rules still earn their place, keep `AGENTS.md` true after a change, evolve the AI layer from a real run (not from taste), and enforce *declared* file coupling so “read before edit” is not advisory. [C5][C6][C7][C8]

Cole’s own description budget is the sibling of GSD’s routers: 34 skills cost ~4,400 always-on tokens because only descriptions load. [C1] Skillgrid’s eager catalog is larger. Shrink descriptions and route; do not import Cole’s 34 names.

**Biggest caveat:** Cole’s distinctive skills are still mostly *procedures*. Ablation and opportunity-scan become Skillgrid-shaped only if the repeating half is a JS script (map the layer, grep session events, check path claims) and the model does the judgment. Do not copy the Python/`uv` hooks. [C8][C9]

## Findings

### What colemedin-skills is

The checkout (`coleam00/skills`) ships a Claude Code plugin of 34 skills from Cole’s Agentic Coding course. Install paths: plugin marketplace, `npx skills add`, or copy the markdown. Nothing here is a runtime. [C1]

The catalog has four rings: [C1]

| Ring | Skills | Skillgrid equivalent |
|------|--------|----------------------|
| Prime | `prime-codebase`, `prime-backend`, `prime-frontend` | `onboarding` / mnemonic `code_explore` / `orient` |
| Plan | `plan-create-prd`, `plan-architecture`, `piv-slice-epic`, `plan-create-stories` | `interviewing`, `writing-blueprints`, `slicing`, `ticketing` |
| PIV loop | plan / implement / validate / review / fix / commit / PR / full-loop | pipeline phases + `qa` + code-review + `work-unit-commits` + `ship` |
| Meta | rules-create, rules-check-drift, ablate-ai-layer, skills-create, hooks-create, opportunity-scan, execution-report, evolution-review, second-brain-audit | `skill-write`, `reflect`, Mnemonic + ADR-0011; **gap is the rest** |

Plus worktrees, issue RCA, dark-factory autonomy, browser/desktop tools. [C1]

### Already in Skillgrid (do not re-import)

- **PIV as the spine.** Documented in `08-concepts.md`; every Skillgrid phase maps onto a step. [C2]
- **Vertical slices that fit one fresh context.** `slicing` cites `colemedin-skills:piv-slice-epic`. [C3]
- **Hook doctrine.** Cole: “A rule asks. A hook guarantees.” Skillgrid already uses that sentence in the user guide and `99-logics.md`. [C4][C8]
- **Overlapping hooks, already JS.** Cole `stop_tests_must_pass` ≈ `hooks/stop-tests.js`; `session_start_context` ≈ `hooks/session-start.js`; `post_tool_use_log` ≈ `hooks/tool-call-capture.js`. Skillgrid also has zone/cwd/commit-msg/gate-stop that Cole does not. [C4][C8]
- **Memory engine.** Cole’s `second-brain-audit` teaches state=replace vs event=append for markdown notes. [C10] Skillgrid already locked this in the store: bi-temporal observations + AUDN supersede (ADR-0011 / ADR-0012). Do not add a markdown vault audit as a second brain.
- **Prime as dump-the-tree.** Cole `prime-codebase` runs `git ls-files` / `tree` and reads entry points. [C11] Skillgrid’s mnemonic `code_explore` is the replacement. Learn the *gate* (do not plan unprimed), not the file dump.
- **Execution report / process retro.** Cole splits `system-execution-report` (what diverged) from `system-evolution-review` (bugs in the process). [C12][C13] Skillgrid `reflect` does both after ship. Mid-change process repair is the BMAD `correct-course` gap, not a missing Cole skill name.

### Distinctive remaining: the meta-layer

#### 1. Wrong rules are worse than missing rules

`rules-check-drift` treats `CLAUDE.md` / `AGENTS.md` as a **steering document**, not documentation. After a change it flags only: a claim that is now false, an architecture-map path that moved, or a new durable invariant (one line). Advisory. Lean by default. Most diffs need no edit. [C6]

Skillgrid has `state-drift-check.mjs` for `state.yaml` vs spec folder, and `ASSUMPTIONS.md` as the locked-constraint source that `AGENTS.md` mirrors. Nothing checks whether the always-loaded `### Rules` block or the architecture map is still true after the diff. A stale “Go 1.22+” or a moved artifact path in `AGENTS.md` misleads every future session.

This is the cheapest Cole take. Mechanical half (path existence, “lives at” claims vs `git ls-files`) is a `.mjs`; the model judges “is this still an invariant.” Fits scripts-over-guessing. Natural attach: `qa` / `requesting-code-review` / pre-merge, not a new phase.

#### 2. Declared file coupling, not “please read first”

Cole’s `pre_tool_use_dependencies.py` is the hook Skillgrid does not have. You declare couplings once in `dependencies.json` (API route ↔ schema, model ↔ migration). On Read, remember the path for the session. On Edit, refuse (or inject) until those files were read. Fails open if the config is missing. [C8][C14]

GSD’s “read before edit” is advisory. Skillgrid’s precommit zone guard is *zone* coupling (spec vs code), not *file* coupling. This hook makes the pairing structural.

Port the *logic* as `hooks/*.js` next to the existing dispatcher. Same fail-open posture as ADR-0021. Do not ship Cole’s Python/`uv` shebang.

#### 3. Opportunity-scan: fix the AI layer, not only the code

Two targets, one question: [C7]

- **Reactive:** point at one run’s artifacts (plan, report, RCA, diff) and ask “what in the AI layer would have prevented this?”
- **Proactive:** aggregate a window of session logs (`jq` / `sort | uniq -c`, never ingest transcripts) and ask “what do I keep doing by hand?”

Output maps each finding to a primitive: rule · skill · hook · script · MCP. Discovery, not eval.

Skillgrid `reflect` writes a sourced retro after ship. It does not classify the next durable change as rule vs hook vs script. Mnemonic `session_events` already is the proactive log window — do not invent `logs/agent-actions.jsonl` as a second trail.

Attach as a named `reflect` section or a follow-up skill. The repeating half (list events, count repeated tool/correction patterns) is a script; the primitive mapping is judgment.

#### 4. Ablate always-loaded rules

`ablate-ai-layer` is the experiment Skillgrid does not run. Model upgrades retire workarounds; reading `AGENTS.md` will not tell you which lines still earn attention. Both arms, many runs, same probe task, throwaway worktrees from HEAD, working tree untouched. Default scope is **always-loaded only** — skills/hooks/path-scoped rules cost nothing until they fire, so deleting them buys no context. [C5]

Cole’s map (`always-loaded` / `on-demand` / `enforcement`) is the vocabulary GSD routers and BMAD skill-evals were missing. Skillgrid already budgets skill *bytes*; it does not measure whether the always-loaded sentinel still changes outcomes.

This is a follow-up, not first-slice. Expensive, user-picked probe, JS port of `map_layer` at most. Complements honest-overhead: if a rule does not change the probe, it is dead weight.

#### 5. State vs event — apply to always-loaded claims, not to Mnemonic

Cole’s second-brain idea is already locked in SQLite. The part that still applies: **a stale fact in the always-loaded file is the bug; a stale fact in an archive is harmless.** [C10] That is `AGENTS.md` / `ASSUMPTIONS.md` `### Locked constraints`, not `.skillgrid/archive/`. `rules-check-drift` is the operational form. Do not add a notes-vault skill.

#### 6. Prime as a gate

Cole’s prime skills exist so the agent does not plan on an empty model of the repo. [C11] Skillgrid already has mnemonic orientation. The lesson is sequencing: `using-skillgrid` / `resume` should refuse to plan when `code_explore` / `mem_context` has not run this session — not another `git ls-files` dump. Optional Jira/Confluence attach in Cole prime is out of scope.

### Description budget

Cole publishes the cost: ~4,400 tokens for 34 descriptions. [C1] That number is why “copy the catalog” is the wrong move even when the skills are good. Merge with GSD two-stage routers (briefing req 5): fewer eager entries, shorter descriptions, bodies on demand.

### Hooks measurement (cited, not fetched)

Cole’s hooks README cites *Agentic Harness Engineering* (arXiv:2604.25850): a self-written 9 KB system prompt scored *below* the seed (67.4% vs 69.7%); measured gain came from memory, tools, and middleware — enforcement, not instruction. [C8] **This run did not fetch the paper.** Treat the percentages as Cole’s citation, confidence **medium**. The operational claim is already locked in Skillgrid: if ignoring it is a production incident, write a hook.

## Cross-Dimension Insights

GSD’s plan-check, BMAD’s readiness/scripts, and Cole’s rules-drift/ablate are one stack:

| Layer | Who owns it | Cole name |
|-------|-------------|-----------|
| Always-loaded text | must stay *true* and *short* | rules-check-drift + ablate |
| On-demand procedure | skill body | already Skillgrid |
| Repeating parse/write | JS script | BMAD contract (locked) |
| Must-never-skip | hook | Cole coupling + existing Skillgrid gates |

Opportunity-scan is how the stack grows without taste: a failure becomes the smallest primitive that would have caught it. That is the missing input to `skill-write` and to “should this be a hook.”

Cole’s fail-open coupling hook is the same posture as Skillgrid’s new-hook rule (ADR-0021): no config → allow. A missing `dependencies.json` must not block edits.

## Contrary Evidence

Cole is a *personal* catalog (“the skills I actually use”), not a method with a state machine. [C1] Importing names (`piv-validate`, `piv-run-full-loop`) would fork Skillgrid’s phase vocabulary for no gain.

`build-dark-factory` and unattended full-loop autonomy fight the serial, human-gated pipeline. Out of scope. [C1]

Ablation is stochastic: two runs of the same arm can differ more than the arms differ. [C5] A Skillgrid port that reports one pair as truth would be worse than not running it.

`prime-codebase` plus Jira/Confluence is the opposite of mnemonic-first orientation. Copying it would reintroduce dump-the-tree after ADR-0012.

Python/`uv` hooks would split the hook runtime Skillgrid already standardized on Node. [C9][C4]

## Recommendations

1. **Treat Cole as the meta-layer source, not a third catalog.** Do not add PIV skill names. Confidence **high**. [C1][C2]
2. **Propose `rules-check-drift` as an interview candidate** (pre-merge / QA): keep `AGENTS.md` true, never longer. Mechanical path-claim check in `.mjs`; model writes the one-line fix. Confidence **high**. [C6]
3. **Propose a declared file-coupling hook** (JS, fail open, `dependencies.json`). Complements GSD read-before-edit. Confidence **high**. [C8][C14]
4. **Fold opportunity-scan into `reflect` (or a follow-up),** reactive first: “what in the AI layer would have prevented this?” Map to rule / skill / hook / script. Proactive scan of `session_events` later. Confidence **medium**. [C7]
5. **Park ablation** as a follow-up after BMAD skill-evals. Port `map_layer` (always-loaded vs on-demand vs enforcement) if anything; do not run worktree bake-offs in the first slice. Confidence **medium**. [C5]
6. **Do not copy** dark factory, prime-as-dump, drive-screen, Jira/Confluence prime, Python hooks, or a markdown second-brain audit. Confidence **high**. [C1][C10][C11]

## Open Questions

- Does rules-drift attach to `qa`, `requesting-code-review`, or a pre-merge hook?
- Is file coupling a first-slice hook or a follow-up after state/plan-check scripts?
- Reactive opportunity-scan in `reflect` vs a separate skill the human invokes when a run went sideways?
- Who authors the first `dependencies.json` couplings (schema↔migration, briefing↔ASSUMPTIONS, AGENTS sentinel↔locked constraints)?

## Source Appendix

| [n] | Claim/finding it supports | Publisher | Pub date | Accessed | Confidence |
|-----|---------------------------|-----------|----------|----------|------------|
| [C1] | 34 skills, PIV loop, ~4400-token descriptions, catalog rings, dark factory | [README.md](file:///Users/paladm/git/ai-test/colemedin-skills/README.md) | checkout | 2026-10-02 | high |
| [C2] | Skillgrid documents the PIV loop as the pipeline spine | [docs/user-guide/08-concepts.md](docs/user-guide/08-concepts.md) | 2026-10-02 | 2026-10-02 | high |
| [C3] | `slicing` based_on `colemedin-skills:piv-slice-epic` | [.agents/skills/planning/slicing/SKILL.md](.agents/skills/planning/slicing/SKILL.md) | 2026-10-02 | 2026-10-02 | high |
| [C4] | “A rule asks; a hook guarantees” + existing JS hook set | [docs/user-guide/04-hooks.md](docs/user-guide/04-hooks.md) + [08-concepts.md](docs/user-guide/08-concepts.md) | 2026-10-02 | 2026-10-02 | high |
| [C5] | Ablate always-loaded only; both arms; worktrees; user picks probe | [ablate-ai-layer/SKILL.md](file:///Users/paladm/git/ai-test/colemedin-skills/.claude/skills/ablate-ai-layer/SKILL.md) | checkout | 2026-10-02 | high |
| [C6] | Wrong rules worse than missing; rules-file-only; lean; pre-merge | [rules-check-drift/SKILL.md](file:///Users/paladm/git/ai-test/colemedin-skills/.claude/skills/rules-check-drift/SKILL.md) | checkout | 2026-10-02 | high |
| [C7] | Reactive run vs proactive logs; map to rule/skill/hook; aggregate don’t ingest | [opportunity-scan/SKILL.md](file:///Users/paladm/git/ai-test/colemedin-skills/.claude/skills/opportunity-scan/SKILL.md) | checkout | 2026-10-02 | high |
| [C8] | Hook doctrine; six hooks; coupling guard; arXiv citation (paper not fetched) | [hooks/README.md](file:///Users/paladm/git/ai-test/colemedin-skills/hooks/README.md) | checkout | 2026-10-02 | high (arxiv %: medium) |
| [C9] | Cole hooks are Python/`uv`; Skillgrid hooks are JS | Cole `hooks/*.py` vs repo `hooks/*.js` | checkout | 2026-10-02 | high |
| [C10] | State=replace vs event=append; always-loaded rot is the bug | [second-brain-audit/SKILL.md](file:///Users/paladm/git/ai-test/colemedin-skills/.claude/skills/second-brain-audit/SKILL.md) | checkout | 2026-10-02 | high |
| [C11] | prime-codebase dumps `git ls-files` / tree; optional Jira/Confluence | [prime-codebase/SKILL.md](file:///Users/paladm/git/ai-test/colemedin-skills/.claude/skills/prime-codebase/SKILL.md) | checkout | 2026-10-02 | high |
| [C12] | Execution report: divergences, challenges, validation | [system-execution-report/SKILL.md](file:///Users/paladm/git/ai-test/colemedin-skills/.claude/skills/system-execution-report/SKILL.md) | checkout | 2026-10-02 | high |
| [C13] | Evolution review: process bugs, not code bugs | [system-evolution-review/SKILL.md](file:///Users/paladm/git/ai-test/colemedin-skills/.claude/skills/system-evolution-review/SKILL.md) | checkout | 2026-10-02 | high |
| [C14] | Coupling hook: Read remembers, Edit blocks/injects; fail open | [hooks/pre_tool_use_dependencies.py](file:///Users/paladm/git/ai-test/colemedin-skills/hooks/pre_tool_use_dependencies.py) | checkout | 2026-10-02 | high |

## Research: What can Skillgrid learn from Superpowers?

> **Decision this serves:** Scope the remaining `skillgrid-workflow-updates` edge after GSD, BMAD, and Cole: Superpowers is the catalog Skillgrid already ported — what mechanism is still missing?
>
> **Type:** competitive + technical
> **Mode:** research (single pass)
> **Date:** 2026-10-02 · **Status:** complete
> **Primary corpus:** local checkout `~/git/ai-test/superpowers` (`obra/superpowers`, MIT; working tree dated 2026-09-09) read this run; Skillgrid skills, hooks, and `skill-anatomy.md` read this run.

## Executive Summary

Superpowers is a 14-skill methodology (brainstorm → worktree → plan → subagent/execute → TDD → review → finish) plus a session-start bootstrap that *forces* the agent to check skills before any action. [P1][P2] Skillgrid already imported the catalog: thirteen skills declare `based_on: superpowers:…`, including the router (`using-skillgrid` ← `using-superpowers`). [P3]

**Do not take another Superpowers skill.** The remaining lesson is what Superpowers itself calls the product: **the bootstrap is the entire integration.** Skills on disk without a SessionStart inject that the harness actually consumes are dead weight. [P4][P5] Second: **skill-behavior TDD** — watch an agent fail without the skill, then write the skill that closes that rationalization — which Skillgrid `skill-write` does not do (BMAD skill-evals were already a follow-up). [P6][P7]

**Biggest caveat:** this checkout is a 2026-09-09 snapshot. README already lists more harnesses than Skillgrid ships. Copying the 14-harness surface would fight the serial, JS-hooks, Cursor/OpenCode/Kilo install Skillgrid already has. Steal the *invariants* (bootstrap shapes, acceptance test, actions-not-tools), not the plugin folders.

## Findings

### What Superpowers is

Jesse Vincent / Prime Radiant. MIT. Same markdown skills everywhere; per-harness thin layer for install + tool mapping + SessionStart. Philosophy: TDD, systematic over ad-hoc, complexity reduction, evidence over claims. [P1]

The workflow Skillgrid already mapped onto phases: brainstorming, isolated-workspace, writing-blueprints, subagent-execution / simple-execution, TDD, requesting/receiving-code-review, ship (`finishing-a-development-branch`). [P1][P3]

### Already in Skillgrid (do not re-import)

| Superpowers | Skillgrid |
|-------------|-----------|
| using-superpowers | `using-skillgrid` (router + 1% rule + red-flag table) |
| brainstorming | `brainstorming` |
| writing-plans | `writing-blueprints` (incl. junior-engineer / bite-sized TDD steps) |
| executing-plans | `simple-execution` |
| subagent-driven-development | `subagent-execution` |
| dispatching-parallel-agents | `parallel-execution` |
| using-git-worktrees | `isolated-workspace` |
| test-driven-development | `test-driven-development` |
| systematic-debugging | `structured-debugging` |
| verification-before-completion | `test-driven-verification` (+ `qa`) |
| requesting / receiving-code-review | same names |
| finishing-a-development-branch | `ship` |
| writing-skills (anatomy, SDO “no process in description”) | `skill-write` + `_shared/rules/skill-anatomy.md` |

SDO is already locked: “If the description contains workflow steps, the agent may follow the summary and never read the body.” [P8] Superpowers is stricter (description = *when* only). Skillgrid keeps what+when. That is a deliberate adaptation, not a gap. [P8][P6]

`verification-before-completion` (“no completion claims without fresh evidence”) is already the verification ladder / 99-logics line “A green suite is not the verdict.” [P9]

### Distinctive remaining

#### 1. Bootstrap is the product

Superpowers’ porting guide states it as an invariant: at session start, inject the full `using-superpowers` body (wrapped `<EXTREMELY_IMPORTANT>`), plus a per-harness tool map. **Without that inject, the skills are inert.** [P4]

The SessionStart hook emits **one** JSON shape per platform — Cursor `additional_context`, Claude nested `hookSpecificOutput`, else SDK `additionalContext` — because Claude reads both fields and would double-inject. That contract is tested. [P10][P11]

New-harness acceptance is one user line: “Let’s make a react todo list.” A working port auto-triggers `brainstorming` before any code. Copying skill files, `npx skills` shims, or opt-in-per-session are rejected as non-integrations. [P5]

Skillgrid already *intends* this: `hooks/session-start.js` comments that the router description is a soft guarantee and SessionStart must close the gap. [P12] Two facts show it is not yet the product:

- The script reads `.agents/skills/using-skillgrid/SKILL.md`. The router lives at `.agents/skills/lifecycle/using-skillgrid/SKILL.md`. Missing file → INFO degrade, no router inject. [P12]
- Cursor’s `hooks-cursor.json` SessionStart is `cursor-session-start.sh`, which injects **Mnemonic prime**, not `using-skillgrid`. Superpowers Cursor SessionStart is the bootstrap. [P13][P14]

This is the Superpowers-shaped form of Cole’s always-loaded layer and GSD’s “skills must fire.” A context monitor warns the window is filling; a bootstrap makes the *first* turn use the router.

#### 2. Skills name actions, not tools

Porting adds a tool-mapping reference. It never rewrites `SKILL.md` to name `Task` vs `task` vs a vendor tool. [P4] Skillgrid already has `using-skillgrid/references/*-tools.md`. Many skill bodies still name Cursor/MCP tools. The invariant belongs next to GSD two-stage routers: the eager catalog stays harness-agnostic; the map is per install.

#### 3. Skill-behavior TDD (pressure scenarios)

`writing-skills`: writing a skill *is* TDD on process docs. Test case = pressure scenario with a subagent. RED = agent violates the rule without the skill (document the rationalizations). GREEN = skill present, agent complies. If you did not watch the failure, you do not know the skill teaches the right thing. [P6]

Eval harness (`superpowers-evals` / Drill) drives real tmux sessions of Claude Code / Codex / Gemini and judges compliance with an LLM verifier. Plugin *infra* tests stay in-repo (`tests/hooks`, `tests/opencode`, …). [P5][P7]

Skillgrid `skill-write` is the deterministic-boundary skill (agent decides *which*; script owns the rest). It does not run a baseline-fail pressure pass. BMAD findings already parked “skill evals” as a follow-up. Superpowers is the method and the split (behavior eval vs hook-shape unit test).

Do not clone Drill or add `uv`/tmux as a pipeline runtime. A later slice can be: one pressure scenario + one JS hook-shape test (Superpowers `test-session-start.sh` logic in `scripts/test-*.mjs`).

#### 4. Never edit the user’s personal config to “install”

Superpowers ports ship through the harness install (plugin / marketplace). They must not patch `~/.claude`, `settings.json`, or bashrc. [P4] Skillgrid `skillgrid setup` *does* write MCP/plugin config — that is this product’s installer, not a Superpowers gap. Do not adopt the “zero-dependency plugin only” rule; Mnemonic is a binary by ADR-0012.

### Cross-Dimension Insights

| This change already | Superpowers name |
|---------------------|------------------|
| GSD two-stage routers / shorter descriptions | SDO + actions-not-tools |
| Cole always-loaded vs on-demand | Bootstrap = the one always-loaded skill |
| BMAD skill evals (follow-up) | writing-skills TDD + Drill |
| Scripts over guessing | Hook-shape tests for SessionStart JSON |
| Cole rules-drift | Superpowers treats skill text as behavior-shaping *code* (eval before reword) |

`Rulings, not stalls` and two-stage spec-then-quality review are already in `subagent-execution`. Bite-sized 2–5 minute TDD steps are already in `writing-blueprints`. Finishing-branch menus are already in `ship`.

### Contrary Evidence

Importing Superpowers’ 14-harness plugin tree would recreate GSD’s catalog problem. Skillgrid’s install surface is Cursor / OpenCode / Kilo plus a Go CLI. [P1]

Visual-companion telemetry and the 94% PR-rejection contributor voice are Superpowers-the-repo, not Skillgrid-the-method. [P1][P5]

Superpowers is zero-dependency markdown. Skillgrid is a method *plus* an engine. “Never add a binary” would contradict ADR-0012.

Description = when-only would fight Skillgrid’s locked anatomy (what + when, no process). Keep the Skillgrid shape; keep Superpowers’ “no workflow in the description” rule we already have. [P8]

### Recommendations

1. **Treat Superpowers as the bootstrap + eval source, not a fifth catalog.** Confidence **high**. [P3][P4]
2. **Propose a harness-bootstrap requirement** (interview candidate): SessionStart injects `using-skillgrid` on every supported harness, one JSON shape per platform, tested. Fix the wrong router path. Cursor should get router + mnemonic prime, not prime alone. Acceptance test: a clean session with a “let’s build X” prompt routes to `brainstorming` before code. Confidence **high**. [P4][P5][P12][P13]
3. **Keep skill-evals as the existing follow-up**, now with Superpowers’ method: pressure scenario (fail without skill → pass with skill); hook-shape unit tests in `.mjs`. Do not add Drill/`uv`/tmux in the first slice. Confidence **medium**. [P6][P7]
4. **Do not copy** the 14-skill names, visual companion, contributor slop rhetoric, or “no installer writes.” Confidence **high**.

### Open Questions

- Does the bootstrap requirement attach to this change’s first slice (with GSD routers) or to a harness-install follow-up?
- Cursor SessionStart: one hook that emits router + prime, or two hooks?
- Is “Let’s build X → brainstorming” an acceptance scenario in `acceptance.feature`, or only a harness-port checklist?

## Source Appendix

| [n] | Claim/finding it supports | Publisher | Pub date | Accessed | Confidence |
|-----|---------------------------|-----------|----------|----------|------------|
| [P1] | 14-skill workflow, philosophy, harness list, telemetry | [README.md](file:///Users/paladm/git/ai-test/superpowers/README.md) | 2026-09-09 checkout | 2026-10-02 | high |
| [P2] | 1% rule, invoke before any action, red-flag table | [using-superpowers/SKILL.md](file:///Users/paladm/git/ai-test/superpowers/skills/using-superpowers/SKILL.md) | checkout | 2026-10-02 | high |
| [P3] | Thirteen Skillgrid skills `based_on: superpowers:…` | `rg based_on: superpowers` on `.agents/skills/**/SKILL.md` | 2026-10-02 | 2026-10-02 | high |
| [P4] | Bootstrap is the entire integration; actions not tools; no user-file edits | [docs/porting-to-a-new-harness.md](file:///Users/paladm/git/ai-test/superpowers/docs/porting-to-a-new-harness.md) | checkout | 2026-10-02 | high |
| [P5] | “todo list” acceptance test; skills are behavior-shaping code; Drill evals | [CLAUDE.md](file:///Users/paladm/git/ai-test/superpowers/CLAUDE.md) | checkout | 2026-10-02 | high |
| [P6] | Skill writing is TDD; pressure scenario; SDO when-not-what | [writing-skills/SKILL.md](file:///Users/paladm/git/ai-test/superpowers/skills/writing-skills/SKILL.md) | checkout | 2026-10-02 | high |
| [P7] | In-repo tests are plugin infra (hooks, harness manifests), not skill-behavior | `tests/` layout this run | 2026-09-09 | 2026-10-02 | high |
| [P8] | Skillgrid already forbids process in descriptions; keeps what+when | [.agents/skills/_shared/rules/skill-anatomy.md](.agents/skills/_shared/rules/skill-anatomy.md) | 2026-10-02 | 2026-10-02 | high |
| [P9] | Evidence before completion claims | [verification-before-completion/SKILL.md](file:///Users/paladm/git/ai-test/superpowers/skills/verification-before-completion/SKILL.md) | checkout | 2026-10-02 | high |
| [P10] | Three SessionStart JSON shapes; Cursor vs Claude vs SDK | [hooks/session-start](file:///Users/paladm/git/ai-test/superpowers/hooks/session-start) | checkout | 2026-10-02 | high |
| [P11] | Hook output shapes are tested | [tests/hooks/test-session-start.sh](file:///Users/paladm/git/ai-test/superpowers/tests/hooks/test-session-start.sh) | checkout | 2026-10-02 | high |
| [P12] | Skillgrid session-start.js intends router inject; looks up missing path | [hooks/session-start.js](hooks/session-start.js) | 2026-10-02 | 2026-10-02 | high |
| [P13] | Cursor SessionStart is mnemonic prime, not the router | [hooks/hooks-cursor.json](hooks/hooks-cursor.json) + [hooks/cursor-session-start.sh](hooks/cursor-session-start.sh) | 2026-10-02 | 2026-10-02 | high |
| [P14] | Superpowers Cursor SessionStart is the bootstrap hook | [hooks/hooks-cursor.json](file:///Users/paladm/git/ai-test/superpowers/hooks/hooks-cursor.json) | checkout | 2026-10-02 | high |

## Research: What can Skillgrid learn from mattpocock-skills?

> **Decision this serves:** Scope the remaining `skillgrid-workflow-updates` edge after GSD, BMAD, Cole, and Superpowers: Matt Pocock’s catalog is the one Skillgrid absorbed most deeply — what is still missing?
>
> **Type:** competitive + technical
> **Mode:** research (single pass)
> **Date:** 2026-10-02 · **Status:** complete
> **Primary corpus:** local checkout `~/git/ai-test/mattpocock-skills` (`mattpocock/skills`, MIT) read this run; Skillgrid `based_on` frontmatter, `code-standards.md`, `skill-write`, `99-logics.md` read this run.

## Executive Summary

Matt’s repo is composable, small skills — not a framework that owns the process. It exists to fix four failure modes: misalignment (grill), jargon (ubiquitous language + ADRs), flying blind (TDD / diagnose), and a ball of mud (deep modules). [M1] Skillgrid already imported the spine: `interviewing` ← grilling, ADRs ← domain-modeling, `research`, `slicing` ← to-tickets, `prototype`/`sketch`/`spike` ← prototype, `onboarding` ← setup, `ticketing` ← triage. Deep-module vocabulary is already in `code-standards.md`, `writing-blueprints/references/codebase-design.md`, `ponytail`, and `99-logics.md`. [M2][M3]

**Do not take another Matt catalog.** The remaining lesson is *enforcement and two missing user interrupts*, not more primitives:

1. **`wait-what`** — a named human interrupt: “that last message did not land; re-pitch in the project vocabulary.” Skillgrid has no such skill. [M4]
2. **Deep modules as a gate**, not a glossary. Agents are good at creating shallow modules. The words are already here; blueprint + review do not fail a change that multiplies tiny busy interfaces. [M1][M3][M5]
3. **Writing-for-agents pointer economics** — context pointer vs context load vs cognitive load; user-invoked vs model-invoked as a *load* trade. This is the same object as GSD routers, Cole always-loaded, and Superpowers SDO. Skill-write already cites Matt’s video; the catalog still pays context load for skills that should be user-only. [M6][M7]

**Biggest caveat:** Matt’s README positions GSD/BMAD as process-owners that steal control. [M1] Skillgrid *is* a process owner. Do not import “skills only, no pipeline.” Steal the small-skill *craft*, keep the pipeline.

## Findings

### What mattpocock-skills is

Checkout: `mattpocock/skills`. Claude plugin or `npx skills add`. Run `/setup-matt-pocock-skills` once per repo (tracker, labels, docs path). User-invoked skills orchestrate; model-invoked skills hold reusable discipline. A user-invoked skill may call a model-invoked one, never another user-invoked one. [M1][M7]

Engineering user-invoked: ask-matt, grill-with-docs, triage, improve-codebase-architecture, setup, to-spec, to-tickets, implement, wayfinder. Model-invoked: prototype, diagnosing-bugs, research, tdd, domain-modeling, codebase-design, code-review, resolving-merge-conflicts, wizard. Productivity: grill-me, handoff, teach, to-questionnaire, wait-what, grilling, writing-for-agents. In-progress extras (loop-me, retro, implement-spec, TS deep-modules, writing-beats/shape/fragments) are not the daily set. [M1]

### Already in Skillgrid (do not re-import)

| Matt | Skillgrid |
|------|-----------|
| grilling / grill-me / grill-with-docs | `interviewing` (design tree, frontier, terms + ADRs written *during*) |
| domain-modeling + CONTEXT.md | `architectural-decision-records` + `01`/`02` terms + `ASSUMPTIONS.md` |
| research | `research` (`based_on` Matt + BMAD epistemics) |
| to-tickets | `slicing` (`based_on` to-tickets + Cole PIV) |
| prototype | `prototype` / `sketch` / `spike` |
| setup-matt-pocock-skills | `onboarding` |
| triage | `ticketing` |
| tdd / diagnosing-bugs / code-review / implement | TDD, structured-debugging, requesting-code-review, execution |
| codebase-design glossary | `_shared/rules/code-standards.md` + blueprint `codebase-design.md` + `ponytail` |
| user- vs model-invoked | `skill-write` + `skill-anatomy.md` |
| writing-for-agents (video) | `skill-write` cites Trigger/Structure/Steering/Pruning, leading words, deletion test, context pointers |
| handoff | `resume` + structured session handoff (already shipped) |

`grill-with-docs` is two Skill-tool calls (grilling + domain-modeling). [M8] Skillgrid folded that into one `interviewing` skill. Do not split it back.

### Distinctive remaining

#### 1. wait-what — the human interrupt

Six lines: stop, re-pitch, ASD-STE100 Simplified Technical English, use the ubiquitous language. User-invoked only. [M4]

Skillgrid has `resume` (files say where you left off) and Cole opportunity-scan (what in the AI layer would have prevented this). Neither is “I did not understand that last turn.” Cheap, high leverage, matches `99-logics` “Questioning until has a shared vocabulary.”

#### 2. Deep modules as a gate

Matt’s #4 failure mode: agents accelerate entropy; they create shallow modules (busy interfaces, many tiny pieces). Fix: care about design every day — `to-spec` quizzes seams before the spec; `improve-codebase-architecture` surveys hot spots and grills one deepening. [M1][M5][M9]

Skillgrid already has the vocabulary and the user wrote it into `99-logics.md`. What is missing is a *fail*: a blueprint that does not name the seam it will test at, or a review that does not flag a new shallow module, still passes. `to-spec`’s seam quiz (prefer existing seams, highest seam, fewest seams, check with the user) is the mechanical attach — it merges with Nyquist (req 4): the test command lives at the module interface, not past it. [M9]

Do **not** port `setup-ts-deep-modules` / dependency-cruiser. That is a TypeScript package-boundary linter. This repo is Go; a new dep needs an ADR. Steal the four *ideas* (entry-point boundary, intra-package freedom, tests through the interface, no cycles) if a later Go check lands — not this change’s first slice. [M10]

`improve-codebase-architecture` (HTML report in `$TMPDIR`, then grill one candidate) is a specialist follow-up, not a pipeline phase. [M5]

#### 3. Pointer economics (same object as routers / always-loaded)

`writing-for-agents`: a **context pointer** (skill description, AGENTS.md line) states the material and the branches that should load it. Always-loaded pointers spend **context load** every turn; user-invoked skills spend **cognitive load** (the human is the index). Router skills exist to cure piled-up cognitive load: one user-invoked name that *hints* at the others and cannot fire them. [M6][M7]

Skillgrid’s `using-skillgrid` is a *model-invoked* router over the pipeline — Superpowers-shaped, always-on. Matt’s `ask-matt` is a *user-invoked* router over user-invoked skills. [M1] Both are valid. The remaining move is to mark more Skillgrid orchestrators user-invoked so they drop out of the eager catalog (GSD req 5), and to write descriptions as pointers (branches, not workflow) — already anatomy, not yet applied to the ~70-skill eager list.

#### 4. wayfinder — decision tickets, not build tickets

Work bigger than one session is a **map** of questions whose resolution is a decision. Fog of war, frontier = unblocked unclaimed children, claim-by-assign, map is an index not a store. [M11]

Skillgrid’s serial one-change + spec folder is a different unit. Do not import tracker-native wayfinder maps. The *shape* (destination, decisions so far, not yet specified, out of scope) is already the briefing. A follow-up only if a change is truly larger than one folder.

### Cross-Dimension Insights

| This change already | Matt name |
|---------------------|-----------|
| GSD routers / Cole always-loaded / Superpowers SDO | context pointer + two loads |
| Nyquist (req 4) | to-spec seam quiz + “interface is the test surface” |
| Cole rules-drift | CONTEXT.md / terms must stay true (already interviewing) |
| Interviewing | grilling — done |
| Superpowers skill-evals | writing-for-agents completion-criterion levers |

Matt is the *craft* source (how to write a skill, how to name a module). GSD/BMAD/Superpowers are the *machinery* sources. Cole is the *meta-layer*. Do not flatten Matt into another script extract.

### Contrary Evidence

“Approaches like GSD, BMAD, and Spec-Kit … take away your control.” [M1] Skillgrid chose the opposite: a locked pipeline. Importing Matt’s anti-framework stance would undo this change.

`to-spec` says “Do NOT interview; just synthesize.” [M9] Skillgrid locks intent first (`99-logics`). Use the seam quiz, not the skip-interview.

`CONTEXT.md` as one file is weaker than Skillgrid’s split (ASSUMPTIONS + terms + ADR files). Do not merge them back.

`loop-me`, `teach`, `wizard`, `to-questionnaire`, and the in-progress writing-* skills are out of scope for a Go SDD pipeline.

Handoff-to-temp-dir duplicates `resume`. Do not add a second handoff format.

### Recommendations

1. **Treat Matt as already-imported craft.** Do not add grill / ticket / prototype / CONTEXT.md names. Confidence **high**. [M2]
2. **Propose `wait-what` as an interview candidate** — a tiny user-invoked skill (or `resume` mode): re-pitch the last turn in project vocabulary. Confidence **high**. [M4]
3. **Propose a deep-module gate** on blueprint + review: name the seam and the interface test; fail a change that adds a shallow module without saying so. Reuse existing `codebase-design.md`; merge with Nyquist. Confidence **high**. [M3][M5][M9]
4. **Apply pointer economics when routers land** (req 5): user-invoked orchestrators drop eager descriptions; model-invoked descriptions list branches, not workflow. Confidence **medium**. [M6][M7]
5. **Park** wayfinder, architecture-survey HTML, TS boundary linter, to-spec-without-interview. Confidence **high**. [M1][M10][M11]

### Open Questions

- Is `wait-what` a new skill or a `resume` / interviewing mode?
- Does the deep-module gate attach to `writing-blueprints`, `requesting-code-review`, or both?
- Which current model-invoked skills should flip to user-invoked to cut eager context load?

## Source Appendix

| [n] | Claim/finding it supports | Publisher | Pub date | Accessed | Confidence |
|-----|---------------------------|-----------|----------|----------|------------|
| [M1] | Four failure modes; composable vs GSD/BMAD; user- vs model-invoked catalog | [README.md](file:///Users/paladm/git/ai-test/mattpocock-skills/README.md) | checkout | 2026-10-02 | high |
| [M2] | Skillgrid `based_on` Matt: interviewing, research, ADRs, slicing, onboarding, ticketing, prototype | `rg based_on: mattpocock` on `.agents/skills/**/SKILL.md` | 2026-10-02 | 2026-10-02 | high |
| [M3] | Deep-module vocabulary already in Skillgrid + `99-logics.md` | [code-standards.md](.agents/skills/_shared/rules/code-standards.md) + [99-logics.md](docs/user-guide/99-logics.md) | 2026-10-02 | 2026-10-02 | high |
| [M4] | wait-what: re-pitch, STE, CONTEXT vocab; user-invoked | [wait-what/SKILL.md](file:///Users/paladm/git/ai-test/mattpocock-skills/skills/productivity/wait-what/SKILL.md) | checkout | 2026-10-02 | high |
| [M5] | Architecture survey: hot spots, deletion test, HTML in temp, then grill | [improve-codebase-architecture/SKILL.md](file:///Users/paladm/git/ai-test/mattpocock-skills/skills/engineering/improve-codebase-architecture/SKILL.md) | checkout | 2026-10-02 | high |
| [M6] | Context pointer, two loads, completion criteria, leading words | [writing-for-agents/SKILL.md](file:///Users/paladm/git/ai-test/mattpocock-skills/skills/productivity/writing-for-agents/SKILL.md) | checkout | 2026-10-02 | high |
| [M7] | User- vs model-invoked as load trade; router skills hint, never fire | [SKILL-MECHANICS.md](file:///Users/paladm/git/ai-test/mattpocock-skills/skills/productivity/writing-for-agents/SKILL-MECHANICS.md) | checkout | 2026-10-02 | high |
| [M8] | grill-with-docs = grilling + domain-modeling | [grill-with-docs/SKILL.md](file:///Users/paladm/git/ai-test/mattpocock-skills/skills/engineering/grill-with-docs/SKILL.md) | checkout | 2026-10-02 | high |
| [M9] | to-spec: no interview; quiz seams first; prefer existing/highest/fewest | [to-spec/SKILL.md](file:///Users/paladm/git/ai-test/mattpocock-skills/skills/engineering/to-spec/SKILL.md) | checkout | 2026-10-02 | high |
| [M10] | TS dependency-cruiser entry-point rules (not this repo’s stack) | [setup-ts-deep-modules/SKILL.md](file:///Users/paladm/git/ai-test/mattpocock-skills/skills/in-progress/setup-ts-deep-modules/SKILL.md) | checkout | 2026-10-02 | high |
| [M11] | wayfinder: decision-ticket map, fog of war, frontier, index not store | [wayfinder/SKILL.md](file:///Users/paladm/git/ai-test/mattpocock-skills/skills/engineering/wayfinder/SKILL.md) | checkout | 2026-10-02 | high |

## Research: What can Skillgrid learn from gstack?

> **Decision this serves:** Scope the remaining `skillgrid-workflow-updates` edge after GSD, BMAD, Cole, Superpowers, and Matt: gstack is a large persona factory — which power tools fit this repo?
>
> **Type:** competitive + technical
> **Mode:** research (single pass)
> **Date:** 2026-10-02 · **Status:** complete
> **Primary corpus:** local checkout `~/git/ai-test/gstack` (`garrytan/gstack`, MIT; working tree dated 2026-09-13) read this run; Skillgrid QA/review/commits `based_on` and hooks read this run.

## Executive Summary

gstack is Garry Tan’s “virtual engineering team”: 23 specialists plus eight power tools, slash commands, Markdown, MIT — plus a Bun/`bin/` toolchain, generated `SKILL.md` from a template, a per-skill `gstack-skill-start` preamble, GBrain, telemetry, and a 10-host installer. [G1] The sprint is Think → Plan → Build → Review → Test → Ship → Reflect. [G1]

Skillgrid already absorbed the pieces that match this repo: `qa` `based_on gstack:cso`, `parallel-code-review` `based_on gstack:review`, `work-unit-commits` `based_on` gstack’s WIP context block. [G2] User sovereignty (“AI recommends, user decides”) is already the Skillgrid gate. [G3]

**Do not take the factory.** Twenty-three named personas, `/autoplan`, GBrain, Aside/browser, iOS QA, and Bun skill-start are a second product. The remaining lesson the user locked is **scope pushback** — office-hours / CEO-review’s Hold Scope and Scope Reduction, not the titles. [G11] Freeze/careful stay research-proposed; skill-start bind folds into Superpowers req 13.

1. **Scope pushback (locked as briefing req 21).** When the user piles independent outcomes, stop and cut. Reframe, recommend the narrowest wedge, default Hold/Reduce. Do not import `/office-hours` or Expansion-as-default. [G11]
2. **`/freeze`** (proposed) — session-scoped edit lock to one directory. [G4]
3. **`/careful`** (proposed) — warn (or hard-deny) destructive bash. [G5]
4. **Skill-start instruction bind** — honor instruction blocks only from the start command you just ran, matching session id. Fold into req 13. [G6]
5. **2KB agents-digest** — already `99-logics.md` + `AGENTS.md`. Do not add a third always-loaded file. [G1]

**Biggest caveat:** ETHOS “Boil the Ocean” (completeness is cheap; prefer the 150-LOC full version) fights Skillgrid’s effort budgets, honest-overhead, and YAGNI. [G3][G7] Do not import it.

## Findings

### What gstack is

Checkout: `garrytan/gstack`. Install clones into `~/.claude/skills/gstack` and runs `./setup`. Team mode auto-updates. Hosts: Claude, Codex, OpenCode, Cursor, Factory, Kiro, Slate, OpenClaw, Hermes, GBrain. [G1]

Skills are **generated** from `SKILL.md.tmpl` (`bun run gen:skill-docs`). Every skill begins with a preamble that execs `gstack-skill-start` and reads `KEY: value` STATUS lines. Missing protocol → degraded defaults, continue the user’s task. [G6]

GBrain is a separate memory/code index (PGLite/Supabase). `/learn` manages session learnings. [G1] Skillgrid already locked SQLite Mnemonic (ADR-0012). Do not add GBrain.

### Already in Skillgrid

| gstack | Skillgrid |
|--------|-----------|
| `/cso` (OWASP + STRIDE, confidence gate) | `qa` `based_on gstack:cso` + `owasp-security` / threat-matrix |
| `/review` (prod bugs, auto-fix, advisory simplify) | `parallel-code-review` `based_on gstack:review` |
| WIP context block / snapshot | `work-unit-commits` |
| User sovereignty / generation-verification | Human gates after slicing, ship options, interviewing |
| `/investigate` Iron Law | `structured-debugging` |
| `/ship` / `/retro` | `ship` / `reflect` |
| `/spec` five-phase | briefing + blueprint |
| agents-digest ethos | `AGENTS.md` + `99-logics.md` |
| Search before building | `research` |

### Distinctive remaining

#### 0. Scope pushback — Hold / Reduce (locked as briefing req 21)

gstack’s remaining advantage is not freeze or a CEO persona. `/office-hours` reframes and recommends the narrowest wedge; `/plan-ceo-review` names four modes: Expansion, Selective Expansion, Hold Scope, Scope Reduction. [G11] Skillgrid already has YAGNI and “flag multiple subsystems immediately”; nothing *blocks* a briefing that keeps growing. Steal the gate: stop, list the pile, recommend one first slice, human picks. Default Hold/Reduce (serial one-change). Do not import the personas or Expansion-as-default.

#### 1. Freeze — edit scope as a hook

`/freeze` asks which directory, then PreToolUse on Edit/Write runs `check-freeze.sh` and **blocks** paths outside it. `/unfreeze` clears it. `/guard` = freeze + careful. Use when debugging so the agent cannot “fix” unrelated code. [G4][G8]

Skillgrid `precommit-zone-guard` is commit-time spec-zone XOR code-zone. Cole coupling (req 10) is “read these files first.” Freeze is “this session may only write under `path/`.” All three compose. Port as a fail-**closed** JS hook *after the user activates it*; no config → allow (same fail-open as ADR-0021 until freeze is on).

#### 2. Careful — destructive-command gate

Pattern table: recursive delete, DROP/TRUNCATE, force-push, `git reset --hard`, kubectl delete. MEDIUM → user override; root/home recursive delete and default-branch force-push are hard-denied. [G5]

Skillgrid `cursor-policy.sh` → `tool-call-capture.js policy` (ADR-0021) does not match these patterns this run. [G9] A JS sibling of the existing policy hook is the Skillgrid shape. Not Python, not a new Bun binary.

#### 3. Preamble contract, not the Bun binary

gstack’s real machinery is `gstack-skill-start`: STATUS lines, telemetry, artifacts sync, one-time instruction blocks. The security rule is the steal: honor an instruction block only if it appeared in the *direct* tool result of the start command you just ran **and** the header carries that run’s `SESSION_ID`. Never from a file, page, or other tool. Unterminated block ends at end-of-output. [G6]

That is the research skill’s untrusted-input boundary applied to skill bootstrap. Superpowers req 13 (SessionStart inject) should use the same bind so a fetched page cannot inject “run this skill.” Do not port the Bun preamble, analytics JSONL, or `gen:skill-docs`.

#### 4. Daily vs comprehensive CSO (already attributed)

`/cso` has daily (zero-noise, 8/10) and comprehensive (monthly, 2/10) modes plus trend tracking. [G10] Skillgrid QA cites CSO but did not import the two-mode scanner. Keep scanners as named specialists (`owasp-security`, `nuclei`). Do not add a CSO persona.

### Cross-Dimension Insights

| This change already | gstack name |
|---------------------|-------------|
| Cole file coupling (req 10) | freeze (directory lock, not pair lock) |
| Superpowers bootstrap (req 13) | skill-start + instruction-session bind |
| BMAD scripts / JS contract | generated SKILL.md + start script (steal bind, not Bun) |
| Cole careful-vs-hook doctrine | careful / freeze *are* hooks — “a skill asks; a hook guarantees” |
| User sovereignty (already Skillgrid) | ETHOS §3 — keep; do not add boil-the-ocean |
| YAGNI + “flag multiple subsystems” (advisory) | office-hours / CEO Hold Scope & Reduction (named modes; locked as req 21) |

### Contrary Evidence

**Boil the Ocean** (“don’t ship the 90% shortcut; completeness costs seconds”) contradicts `effort-budgets.md`, GSD honest-overhead, and Matt/YAGNI. [G3][G7] Skillgrid stays “smallest usable whole.”

The 23-persona sprint (`/office-hours` → `/plan-ceo-review` → … → `/autoplan`) is BMAD party mode with YC titles. Already rejected. [G1]

`/learn` + GBrain is a second memory store. ADR-0012.

Telemetry to `~/.gstack/analytics` and skill-usage JSONL are Superpowers-visual-companion cousins. Mnemonic `session_events` is enough.

`/qa` that *fixes* bugs in the QA skill fights Skillgrid’s QA-as-gate (report, don’t implement).

### Recommendations

1. **Treat gstack as power-tool source, not a seventh catalog.** Confidence **high**. [G1][G2]
2. **Lock scope pushback (briefing req 21):** pile of independently shippable outcomes → stop, recommend one wedge, human picks hold / cut / split. Default Hold/Reduce. No CEO persona. Confidence **high**. [G11]
3. **Propose freeze** (not locked): user-activated, session-scoped write lock to one directory; JS hook; off → allow. Complements Cole coupling. Confidence **high**. [G4]
4. **Propose careful** (not locked): destructive-command patterns on the existing policy hook; hard-deny the two catastrophic cases. Confidence **high**. [G5][G9]
5. **Fold instruction-session bind into Superpowers req 13** — do not make a new req. Confidence **high**. [G6]
6. **Do not copy** boil-the-ocean, 23 personas, `/office-hours`, `/plan-ceo-review`, autoplan, GBrain, Bun skill-start, Aside/iOS, CSO persona, QA-that-fixes. Confidence **high**.

### Open Questions

- Freeze + Cole coupling: one hook with two modes, or two hooks?
- Careful: extend `tool-call-capture.js policy`, or a new `hooks/careful.js`?
- Is freeze first-slice (with coupling) or a follow-up?

## Research: Project-context skill (AI Labs / Karpathy loop)

> **Decision this serves:** Whether `skillgrid-workflow-updates` should add a project-context skill. **Status:** optional, not decided (briefing req 20). Interview has not picked skip / map / map + reconcile.
>
> **Source:** [AI Labs — Karpathy loop](https://www.youtube.com/watch?v=qLfSDQ5NGh0) (project-context skill at ~04:21). [K1]
> **Date:** 2026-10-02 · **Status:** captured, not locked

The video skill is one growing file for one app: what it does, which pages it has, conventions, and what to avoid. Only the short description stays in the window. The full body loads when the task needs it. A fresh feature agent starts empty; the skill is how it still knows the app. [K1]

Skillgrid already split that job on purpose. Do not add a dump skill. A second `SKILL.md` that copies those files is a second source that will drift.

### How to use it in Skillgrid (if taken)

- `AGENTS.md` is the short description that stays loaded. It points. It does not store the project.
- `using-skillgrid` already says: read config, resume, then the domain model before other work.
- A fresh feature agent loads only the file that matches the task.
- Onboarding already comes from BMAD `bmad-project-context`. It writes the lean block and the first facts. Interview and blueprint keep those files current. Ship and mnemonic keep the second brain current.

### What already replaces that skill

| That skill holds | Use this instead |
|---|---|
| What the app does | `.skillgrid/artifacts/00-prd.md` and the Product section of `.skillgrid/ASSUMPTIONS.md` |
| Pages / how it is built | `.skillgrid/ARCHITECTURE.md` and the code index |
| Shared words | `.skillgrid/artifacts/01-business-terms.md` and `02-technical-terms.md` |
| Hard decisions | `.skillgrid/artifacts/04-adr-NNNN-slug.md` (index in `ASSUMPTIONS.md`) |
| What is locked | `ASSUMPTIONS.md` → Locked constraints, mirrored under `### Rules` in `AGENTS.md` |
| Stack, test runner, tracker | `.skillgrid/config.yaml` |
| What to avoid, after a real miss | One line in `AGENTS.md`, or a mnemonic note. Do not preempt. |
| Always-on pointer | The `## Skillgrid` block in `AGENTS.md` |
| Load the rest only when needed | Open the file, or `mem_search` → `mem_get_observation` |

### What would change (three options — not picked)

1. **Skip.** Keep the existing split. No new skill.
2. **Map only.** A thin `project-context` skill that says when to open which file. It must not copy those files.
3. **Map + post-ship reconcile.** After ship, update `ARCHITECTURE.md` and the terms from the real repo. Handwritten lines stay. Conflicts are flagged, not overwritten. Onboarding merge is the start of that. It is not yet a loop after every feature.

Habits the loop learns belong in `AGENTS.md` (one line after a real miss) or in mnemonic. They do not belong in a second project-context blob.

### Do not do (even if taken)

- Do not put pages, conventions, and avoid-lists into one `SKILL.md`. That is the bloated `AGENTS.md` Skillgrid already forbids.
- Do not replace `ASSUMPTIONS.md`, `ARCHITECTURE.md`, the terms, or the ADRs with a skill body.
- Do not replace Mnemonic with a markdown dump. Mnemonic is the index. The files stay the source.
- Do not import a one-file `CONTEXT.md`.

The rule stays: the agent decides which file to open. The file holds the fact.

### Open questions

- Skip, map-only, or map + post-ship reconcile?
- If map: new skill vs a section on `using-skillgrid` / `onboarding`?
- If reconcile: attach to `ship` or `reflect`? First slice or a later change?

## Research: What can Skillgrid learn from test-book (book production pipeline)?

> **Decision this serves:** Whether `skillgrid-workflow-updates` should borrow structural patterns from a second, non-SDD skill ecosystem (`~/git/test-book/.agents/skills`, 15 skills, book production). **Status:** captured, not locked — no new requirement proposed; patterns are candidates for existing reqs or follow-ups.
>
> **Source:** local checkout `~/git/test-book/.agents/skills` read this run (2026-10-04); Skillgrid skills and state files read this run.
> **Date:** 2026-10-04 · **Status:** complete

### Executive summary

test-book and Skillgrid solved the same two failures — *unverified "done"* and *session amnesia* — with opposite shapes. Skillgrid is **gate-centric**: `SKILL.md` carries spine + `Common Rationalizations` / `Red Flags` / `Verification`, quality is a single `qa` gate (four-state) + `floor.md`, state is `.skillgrid/state.yaml`. test-book is **artifact-centric**: `SKILL.md` is a thin orchestrator; the real contract is a declarative `manifest.yaml` (phase → `prompt` / `gate` / `outputs` / `next`) + a `PROJECT_STATE.yaml` `gates:` block + per-phase prompt files, and quality is a **multi-context scoring protocol**, not one gate skill.

Six patterns are portable to Skillgrid. None replaces an existing mechanism; each tightens one.

### The six borrowable patterns (ranked by ROI)

**1. Evaluator-independence grade (highest leverage).**
test-book's `evaluator-protocol.md` formalizes *who is allowed to approve*: the evaluator sees **blind inputs only** (no prior scores, no threshold, no writer self-report), is graded **A / B / C** on independence, and **Grade C is diagnostic only — it cannot approve**. Seeing the target score is an integrity failure (`DEGRADED`) that invalidates the result.

Skillgrid's `qa` skill runs the suite **and** renders the four-state verdict in one context. It has the floor rule (weakest dimension caps the verdict) but **no independence grade for the evaluator itself**. A `qa` run where the agent wrote the code under test is structurally indistinguishable from an independent review.

**Borrow:** add an `Independence: A | B | C` line + a `DEGRADED` integrity check to `qa` and `requesting-code-review`. Grade A = evaluator did not author the artifact and did not see the pass threshold before scoring. Grade B = saw the threshold but did not author. Grade C = authored the artifact or saw a prior score → **diagnostic only, cannot render PASS / CONCERNS; the verdict is advisory until a Grade-A context re-runs it.** Not a new skill. Not a new gate. A frontmatter-style line on the existing verdict.

Maps to: `qa` (attach), `requesting-code-review` (attach). No new req; strengthens the existing QA gate that req 9 (rules-drift) and the deep-module gate (req 16) already target.

**2. `manifest.yaml` as the pipeline spine (medium leverage, large diff).**
test-book's `references/pipeline/manifest.yaml` encodes the whole phase chain as data: each phase is `{prompt, gate, outputs, next}`. `SKILL.md` says "load manifest, run current phase." The DAG is data, not prose. Advancing a phase is a file read + a gate check, not a re-interpretation of 21 KB of `sdd-structure.md`.

Skillgrid's `sdd-structure.md` (21 KB) encodes the same chain (`proposal → specs → tasks → apply → verify → archive`) in prose. `state.yaml` has `pipeline.current_phase` but no per-phase `gate` / `outputs` / `next` registry.

**Borrow:** per-change `manifest.yaml` under `.skillgrid/specs/<change>/` encoding the phase chain for *that* change. Overlaps req 2 (deterministic state mutations) and req 7 (skill graph + completion markers). If req 7 lands as a committed graph, `manifest.yaml` is the natural shape. **Do not** make `manifest.yaml` a second store: it points at the same artifacts (`briefing.md`, `blueprint.md`, `tasks.md`, `acceptance.feature`) that `state.yaml` already references.

Maps to: req 2 (state CLI reads it), req 7 (graph shape). Follow-up, not first-slice.

**3. Per-skill `evals/evals.json` (medium leverage, small diff).**
test-book ships machine-checkable evals per skill: `copy-editing/evals/evals.json` has `prompt` → `expected_output` → `assertions[]`. Skillgrid's `Verification` sections are prose checklists, not re-runnable evals.

**Borrow:** an `evals/evals.json` convention for the highest-stakes skills — `qa`, `acceptance-test-authoring`, `research`. Each eval is `{prompt, expected_output, assertions: []}`. This is the skill-level counterpart to req 14 (skill-behavior TDD, named follow-up): req 14 is "a skill change is proven by a pressure scenario"; `evals.json` is the *format* that pressure scenario is written in. If req 14 lands, `evals.json` is its artifact.

Maps to: req 14 (format). Follow-up.

**4. Per-specialist packet schema (medium leverage, small diff).**
test-book's `agent-registry.yaml` gives every specialist role a packet: `{mission, timing, score_floor, inputs, outputs, gates}`. Skillgrid's `subagent-execution` and `parallel-code-review` dispatch subagents without a per-role packet schema — the mission, inputs, and gates live in the dispatch prose.

**Borrow:** a `packet` schema for subagent dispatch: `{mission, timing, inputs, outputs, gates}` per role. This makes the dispatch checkable: a subagent that does not receive its declared `inputs` or does not produce its declared `outputs` is a finding, not a vibes call.

Maps to: `subagent-execution` (attach), `parallel-code-review` (attach). No new req; a convention under `_shared/`.

**5. Checkpoint-then-choice recovery (small leverage, small diff).**
test-book's `host-contract.md`: "Retry at most twice with a concrete correction. If no measurable progress, save a checkpoint and give a choice: retry / adjust / stop. Never restart the whole book automatically." On resume it reconciles state vs files ("a chapter heading or empty file is not a completed chapter") and persists the precise resume step.

Skillgrid's `resume` continues from files and `rigor-tiers.md` has a fix-loop cap, but the *checkpoint-then-choice* pattern is not named: on a stalled retry, the agent re-runs rather than stopping and offering retry / adjust / stop.

**Borrow:** a named recovery pattern in `resume`: on resume, reconcile `state.yaml` vs artifacts; persist the precise resume step; cap retries at 2 with a concrete correction; then offer **retry / adjust / stop** instead of auto-restart. The "a file exists is not a file is done" reconciliation rule is the sharp part.

Maps to: `resume` (attach). No new req; a section.

**6. `allowed-tools` frontmatter (low leverage, trivial diff).**
test-book's `humanizer` SKILL.md frontmatter lists `allowed-tools: Read, Write, Edit, Grep, Glob, AskUserQuestion`. Skillgrid frontmatter has `license` / `effort` / `metadata` but no tool scoping.

**Borrow:** optional `allowed-tools` in SKILL.md frontmatter for skills that should stay read-only (`research`, `code-research`, `codebase-inspection`) or write-only. The hook dispatcher (req 23 code-index policy) can honor it: a read-only skill that calls `Edit` is a finding.

Maps to: req 23 (policy hook honors it). Trivial; fold into req 23 if it lands.

### What Skillgrid already does better (do not regress)

- **Anti-rationalization tables** (`Common Rationalizations` / `Red Flags`). test-book has no equivalent; do not remove them to make room for the borrows above.
- **`floor.md` / verification-ladder** — more rigorous gate machinery than test-book's 8.5 threshold + calibration. The floor rule stays; the independence grade (borrow 1) *adds to* it, not replaces it.
- **`cite-dont-restate.md`** (cite ADR by ID). test-book restates contracts inline; do not regress to inline restatement when adding the `manifest.yaml` (borrow 2) — the manifest points at artifacts, it does not restate them.
- **Script-first deterministic boundary** (`skill-write` + `skill-size-budget.mjs`). test-book is all-markdown, no script extraction. The JS-only runtime lock (round 1) is not in tension with any test-book borrow.
- **`_shared/` conventions as single source of truth**. test-book duplicates editorial rules across skills; the packet schema (borrow 4) and `evals.json` format (borrow 3) should land in `_shared/`, not per-skill.

### Cross-dimension insights

The deepest difference is **where quality lives**. Skillgrid puts it in one gate skill (`qa`) that runs the suite and renders the verdict in the same context. test-book splits it: an **adversarial audit phase** (7 mandatory passes: existence, voice, over-explanation, human-mess, failure, repetition, agent-pi) runs *before* any scoring, and the **scoring protocol** runs in a separate blind context. Skillgrid's `qa` is both auditor and scorer in one pass. The independence grade (borrow 1) is the minimal version of test-book's split; a full "adversarial audit phase before `qa`" is a larger change and not proposed here.

The second insight: test-book's **score calibration rule** ("subtract 0.8 from each lens median unless an external critique exists") is a concrete anti-inflation mechanism. Skillgrid's floor rule ("weakest dimension caps the verdict") is structural, not statistical — it catches a single weak dimension but not uniform inflation across all dimensions. The 0.8 discount is a cheap add to `floor.md` if the independence grade (borrow 1) lands: Grade-C self-evaluation gets an additional discount on top of the floor.

### What not to copy

- **YAML manifest as the *only* pipeline definition.** Skillgrid's `sdd-structure.md` carries the phase semantics (what each phase *means*, what it must contain). A `manifest.yaml` (borrow 2) carries the *shape* (which phase, what gate, what outputs, what next). Both are needed; the manifest does not replace the semantics doc.
- **Multi-lens scoring with 8.5 thresholds** as the gate mechanism. Skillgrid's four-state gate + floor rule is more rigorous and already effort-budget-aware. Do not import a numeric threshold system.
- **`PROJECT_STATE.yaml` as a second state file.** Skillgrid has `state.yaml` + spec-folder artifacts. test-book's `PROJECT_STATE.yaml` is a per-project copy of a template with a `gates:` block. The `gates:` block idea (borrow 2, folded into req 7's graph) is the borrow; the separate file is not.
- **Book-domain prompt files** (`adversarial-audit.md`, `drafting.md`, `literary-barrier-loop.md`). Domain-specific; the *structure* (separate audit phase, calibration discount, independence grade) is the borrow, not the content.

### Recommendations

1. **Add evaluator-independence grade to `qa` and `requesting-code-review`** (borrow 1). Smallest diff, highest leverage. No new req. Confidence **high**.
2. **If req 7 (skill graph) lands, shape it as a per-change `manifest.yaml`** with a `gates:` block (borrow 2). Follow-up, not first-slice. Confidence **high**.
3. **If req 14 (skill-behavior TDD) lands, define its artifact as `evals/evals.json`** (borrow 3). Follow-up. Confidence **medium-high**.
4. **Add a per-specialist packet schema to `subagent-execution` and `parallel-code-review`** (borrow 4). Convention under `_shared/`. No new req. Confidence **medium-high**.
5. **Add checkpoint-then-choice recovery to `resume`** (borrow 5). A section, not a new skill. Confidence **medium**.
6. **If req 23 (code-index policy) lands, honor optional `allowed-tools` frontmatter** (borrow 6). Trivial. Confidence **medium**.

### Open questions

- Independence grade: is it a frontmatter line on the verdict, a field in `state.yaml`, or a section in the QA report? (Probably a section in the QA report + a line in `state.yaml` `progress` for traceability.)
- `manifest.yaml` (borrow 2): one manifest per change, or a repo-level manifest for the pipeline + per-change overrides? (Probably per-change; the repo-level `sdd-structure.md` stays the semantics doc.)
- `evals.json` (borrow 3): does the QA gate run them, or are they a separate `skill-eval` invoke? (Probably separate; QA is the change gate, evals are the skill gate.)
- Packet schema (borrow 4): does it go in `_shared/rules/subagent-packet.md` or is it frontmatter on the dispatching skill? (Probably a rule doc; frontmatter is for the skill, not the dispatch.)
- Checkpoint-then-choice (borrow 5): is the checkpoint a `state.yaml` field, a `RUN_REPORT.md` line, or a Mnemonic observation? (Probably `state.yaml` `progress` + Mnemonic; no new file.)

## Source Appendix

| [n] | Claim/finding it supports | Publisher | Pub date | Accessed | Confidence |
|-----|---------------------------|-----------|----------|----------|------------|
| [G1] | 23 specialists, 8 power tools, hosts, sprint, agents-digest, GBrain | [README.md](file:///Users/paladm/git/ai-test/gstack/README.md) | 2026-09-13 checkout | 2026-10-02 | high |
| [G2] | Skillgrid `based_on` gstack CSO / review / WIP context | `qa`, `parallel-code-review`, `work-unit-commits` SKILL.md | 2026-10-02 | 2026-10-02 | high |
| [G3] | Boil the ocean; search first; user sovereignty | [ETHOS.md](file:///Users/paladm/git/ai-test/gstack/ETHOS.md) | checkout | 2026-10-02 | high |
| [G4] | freeze: PreToolUse blocks Edit/Write outside a directory | [freeze/SKILL.md](file:///Users/paladm/git/ai-test/gstack/freeze/SKILL.md) | checkout | 2026-10-02 | high |
| [G5] | careful: destructive patterns; override vs hard-deny | [careful/SKILL.md](file:///Users/paladm/git/ai-test/gstack/careful/SKILL.md) + README power tools | checkout | 2026-10-02 | high |
| [G6] | skill-start preamble; instruction bind to this run + SESSION_ID; degrade | [SKILL.md](file:///Users/paladm/git/ai-test/gstack/SKILL.md) router preamble | checkout | 2026-10-02 | high |
| [G7] | Skillgrid effort budgets / when-not-to-use-the-pipeline | `.agents/skills/_shared/rules/effort-budgets.md` | 2026-10-02 | 2026-10-02 | high |
| [G8] | guard = careful + freeze | [guard/SKILL.md](file:///Users/paladm/git/ai-test/gstack/guard/SKILL.md) | checkout | 2026-10-02 | high |
| [G9] | Skillgrid hooks have no rm -rf / DROP / force-push patterns | `rg` over `hooks/` | 2026-10-02 | 2026-10-02 | high |
| [G10] | CSO daily 8/10 vs comprehensive 2/10 | [cso/SKILL.md](file:///Users/paladm/git/ai-test/gstack/cso/SKILL.md) | checkout | 2026-10-02 | high |
| [G11] | Office-hours / CEO-review: reframe + Hold Scope / Scope Reduction | [plan-ceo-review/SKILL.md](file:///Users/paladm/git/ai-test/gstack/plan-ceo-review/SKILL.md) + README | checkout | 2026-10-02 | high |
| [K1] | Project-context skill: app-specific memory bank; short description always loaded; full body on demand | [AI Labs — Karpathy loop](https://www.youtube.com/watch?v=qLfSDQ5NGh0) (~04:21) | 2026-10-02 | 2026-10-02 | high |

---

## Research: Should Skillgrid run execution in subagents (GSD `gsd-execute-phase` model)?

> **Decision this serves:** Whether the `subagent-execution` / `simple-execution` skills should adopt GSD's subagent-driven execution model wholesale, and which of its mechanisms fit the serial lock.
>
> **Type:** technical (mechanism comparison against two named systems) + competitive (position against gsd-core's orchestrator/worker engine)
> **Mode:** research (single pass, delegated fan-out over both checkouts)
> **Date:** 2026-10-07 · **Status:** complete
> **Primary corpus:** local checkouts `~/git/ai-test/gsd-core` (`skills/gsd-execute-phase/SKILL.md`, `gsd-core/workflows/execute-phase.md`, `execute-plan.md`, `agents/gsd-executor.md`) and `~/git/ai-test/gstack` (`README.md`, `ship/SKILL.md`, `spec/SKILL.md`, `autoplan/SKILL.md`) read this run; Skillgrid `.agents/skills/execution/{subagent-execution,simple-execution,parallel-execution}/SKILL.md` read this run.

### Executive Summary

**gsd-core's `gsd-execute-phase` is a deterministic orchestrator/worker engine, and "subagent-driven" is its default and canonical mode, not an option.** The orchestrator coordinates — it discovers plans, groups them into dependency waves, spawns `gsd-executor` subagents (fresh context, ~10–15% of the window for the orchestrator itself), and collects results. It never touches source files. Inline execution is a documented, conditional fallback: `--interactive` flag, small plans (≤2 tasks, ~14K-token spawn overhead), or runtimes without a reliable `Agent()` completion signal.

**gstack is the opposite paradigm.** It is a role-based slash-command process: the main session builds inline; subagents are *hired* for specific sub-tasks (adversarial review + Codex cross-model pass, `/document-release` subagent, `claude -p` in a fresh worktree for spec→build). There is no wave/orchestrator fan-out at all; parallelism lives at the sprint level via an external tool (Conductor), not inside a skill.

**Skillgrid's `subagent-execution` already has GSD's per-task shape** — fresh implementer per task, per-task reviewer, two-axis final review — so the gap is not "switch to subagents" (already done) but the three orchestrator mechanisms the skill lacks: (1) filesystem-as-truth completion, (2) post-merge build+test gate per wave, (3) separate goal verifier + gap-closure loop. Wave fan-out and worktree-by-default isolation are the GSD parts that fight the serial one-change lock.

### Findings

#### gsd-core: orchestrator/worker mechanics

- **Orchestrator coordinates, not executes.** `execute-phase.md` states the core principle: each subagent loads the full execute-plan context; the orchestrator holds ~10–15% of a 200K window; subagents get a fresh window. "No polling (Agent blocks). No context bleed."
- **Unit of work = a plan** (`*-PLAN.md`), grouped into **waves** by dependency analysis. Waves run sequentially; plans within a wave run in parallel (via `run_in_background: true` `Agent()` calls, one at a time to avoid `.git/config.lock` races) or sequentially if `parallelization=false` or if the intra-wave file-overlap check finds shared `files_modified`.
- **Isolation is the only fan-out branch point**: `ISOLATION ∈ {harness-worktree, orchestrator-worktree, none}`. In worktree mode each executor commits on a per-agent branch and must commit `SUMMARY.md` before returning.
- **Filesystem is the source of truth, not the return message.** The executor writes `SUMMARY.md` to disk; `gsd-executor.md` states the orchestrator reads it from disk after the agent returns and "does NOT read your return message for the file content." Because backgrounded agents' completion signals may never arrive (a known runtime quirk), the orchestrator **spot-checks, never waits**: `SUMMARY.md` exists + `git log --grep=<plan-id>` finds commits ⇒ done.
- **Stall surveillance**: on a fixed interval, if no `SUMMARY.md` and no expected-branch commits after a threshold, the orchestrator pauses and offers continue / kill+retry / kill+switch-to-inline.
- **Post-merge build & test gate after each wave** — explicitly justified against the "generator self-evaluation blind spot" (an agent reports Self-Check: PASSED but the merge breaks). Tracking advances only if the test run exits 0 (timeout = inconclusive, not pass).
- **End-of-phase verification is a separate `gsd-verifier` subagent** that checks the phase **GOAL** (not just task completion) and cross-references requirement IDs, producing `VERIFICATION.md`. Gaps route to a `--gaps` planning pass, then `--gaps-only` re-execution, then re-verify — the **gap-closure loop**.
- **Rework machinery**: a failure classifier (`quota-exceeded` / a known handoff bug treated as success after spot-check / `unknown-failure`); checkpoints spawn a **fresh continuation agent** (not resume — resume serialization breaks with parallel tool calls); a node-repair budget (default 2) before escalation.
- **Inline fallbacks**: `--interactive` (sequential inline, no subagents); `TASK_COUNT <= INLINE_THRESHOLD` (default **2**) runs inline to avoid spawn overhead; runtimes without a reliable completion signal default to sequential inline.

#### gstack: human-steered process, subagents as hired roles

- **The main session is the default executor.** Each slash-command skill is "executable instructions, not reference." The user (or Conductor) advances the sprint command-by-command: Think → Plan → Build → Review → Test → Ship → Reflect.
- **Subagents are hired for specific sub-tasks, not as the work unit**: `/ship` + `/review` run an adversarial Claude subagent **plus** a Codex pass (cross-model gate); large diffs (200+ lines) add a structured Codex review with a P1 gate; `/ship` dispatches a `/document-release` subagent (non-blocking); `/spec --execute` spawns `claude -p` in a **fresh worktree**.
- **No wave model**: `/autoplan` chains CEO → Design → DX → Eng review "NEVER in parallel." Parallelism is 10–15 concurrent *sprints* via Conductor (isolated sessions/workspaces), not concurrent tasks inside one change.
- **Gates are named skills, not an orchestrator**: `/qa` (real browser, fix → regression test → re-verify), `/cso` (8/10 confidence), `/spec` (7/10 score gate), `gstack-verify-gate` (Stop hook), `gstack-evidence` (verification ledger cited by `/ship`).

#### Skillgrid: current state vs the gap

| GSD `gsd-execute-phase` mechanism | Skillgrid today | Verdict |
|---|---|---|
| Fresh implementer per task + per-task review | `subagent-execution`: fresh implementer subagent per task, task reviewer (spec + quality), two-axis final review | **Already present** |
| Orchestrator holds ~10–15% context; workers get fresh windows | Same principle, stated in the skill's "Why subagents" | **Already present** |
| Return message is not the signal; spot-check `SUMMARY.md` + `git log` | Parent re-verify re-runs the task's `#### Gates` oracles (Decompose & Gate), but completion trusts the implementer's report + ledger; no artifact/commit spot-check for lost completion signals | **Gap (a)** |
| Post-merge build+test gate after each wave | Only the end-of-change `skillgrid:qa` gate; nothing between tasks/waves | **Gap (b)** |
| Separate `gsd-verifier` checks the phase GOAL; gap-closure loop (`--gaps` → re-verify) | `skillgrid:qa` + two-axis final review check the code, not the briefing goal against a Validation matrix; no named gap-closure loop (req 6 correct-course is the closest, and it is queued) | **Gap (c)** |
| Waves: sequential waves, parallel plans-in-wave, worktree isolation | `subagent-execution` is sequential by design; `parallel-execution/concurrent-leaves` is the lightweight fan-out case; lease/wave machinery "is a future orchestrator skill, out of scope" (subagent-execution:264-267); "worktree-by-default parallelism" is out of scope (briefing line 20); serial one-change is a locked constraint | **Rejected** |
| Inline fallback for ≤2 tasks / no-reliable-signal runtimes | `simple-execution` line 22: "If subagents are available, use skillgrid:subagent-execution instead" | **Already present (as a separate skill)** |

### Recommendations

1. **One requirement (req 24), three parts, with the cheap part first.** Part (a) filesystem-as-truth is an edit to two existing skills (spot-check artifacts + git log before marking complete; a mismatch means not done) — no new machinery, lands in the first slice. Parts (b) post-wave gate and (c) goal verifier + gap-closure loop are queued behind the first slice. Confidence **high**.
2. **Reject wave fan-out and worktree-by-default.** Both fight the serial one-change lock and the existing "worktree-by-default parallelism" out-of-scope line. `parallel-execution/concurrent-leaves` stays the lightweight case; the full orchestrator (4+ leaves, lease/wave state) remains a future skill. Confidence **high**.
3. **Do not demote `simple-execution` as part of this.** It is already the documented no-subagent-harness fallback; changing the default split is a separate decision the interview left out of scope. Confidence **high**.
4. **The goal verifier (part c) should be the independent reader of the req 19 Validation matrix** — same evaluator-independence posture as the test-book borrow 1 (→ `qa`). It re-runs the matrix rows against the live tree; gaps cannot be parked without a ledger ruling. Confidence **medium-high** (depends on req 19 landing first).

### Contrary Evidence

- "Everything in subagents" taken literally would import GSD's wave engine and worktree isolation — the two parts that break the serial lock. The defensible read is subagents-for-*work* (already true), not subagents-for-*coordination*.
- Part (a)'s value depends on the harness actually losing completion signals (documented for Claude Code's backgrounded agents; unverified for OpenCode/Cursor/Kilo). If no supported harness loses them, part (a) degrades from "catch lost signals" to "defense in depth" — still consistent with the parent re-verify rule, but the urgency drops.
- Part (c) adds a second full-context subagent run (the most expensive model per model-selection) at the end of every standard change. Against the effort-budget table, it should apply to `standard`/`max` only, not the `trivial`/`small` fast-track.

### Open Questions

- Part (b): does the post-wave gate run `testing.runner` + `commands.build` only, or also lint? And on `trivial`/`small` fast-track or `standard`/`max` only (effort-budget table)?
- Part (c): is the goal verifier a new `verification` subagent dispatched by `subagent-execution` / a `parallel-execution` variant, or a mode inside `skillgrid:qa`? How do its named gaps map to Backlog ticket IDs (new tickets vs a correct-course briefing patch, req 6)?
- Part (a): is the spot-check a one-line rule in the skills, or a tested `.mjs` helper per req 8's script contract (`git log --grep=<task-id>` + artifact existence → JSON verdict)? If a script, the task ID needs a greppable convention in commit messages (the `[skillgrid-context]` block already names the task ID — confirm it is always present).

### Source Appendix

| [n] | Claim/finding it supports | Publisher | Pub date | Accessed | Confidence |
|-----|---------------------------|-----------|----------|----------|------------|
| [E1] | Orchestrator coordinates not executes; ~10–15% window; waves sequential / plans parallel; `run_in_background` one-at-a-time | [gsd-core/skills/gsd-execute-phase/SKILL.md](file:///Users/paladm/git/ai-test/gsd-core/skills/gsd-execute-phase/SKILL.md) + [gsd-core/workflows/execute-phase.md](file:///Users/paladm/git/ai-test/gsd-core/gsd-core/workflows/execute-phase.md) | 2026-10-07 | 2026-10-07 | high |
| [E2] | `SUMMARY.md` on disk is truth; return message not read; spot-check never wait; stall surveillance intervals | [gsd-core/workflows/execute-phase.md](file:///Users/paladm/git/ai-test/gsd-core/gsd-core/workflows/execute-phase.md) + [gsd-core/agents/gsd-executor.md](file:///Users/paladm/git/ai-test/gsd-core/gsd-core/agents/gsd-executor.md) | 2026-10-07 | 2026-10-07 | high |
| [E3] | Post-merge build+test gate per wave; TEST_EXIT≠0 ⇒ not complete; generator self-evaluation blind spot | [gsd-core/workflows/execute-phase.md](file:///Users/paladm/git/ai-test/gsd-core/gsd-core/workflows/execute-phase.md) | 2026-10-07 | 2026-10-07 | high |
| [E4] | `gsd-verifier` subagent checks phase GOAL + requirement IDs; `VERIFICATION.md`; `--gaps` / `--gaps-only` gap-closure loop | [gsd-core/workflows/execute-phase.md](file:///Users/paladm/git/ai-test/gsd-core/gsd-core/workflows/execute-phase.md) | 2026-10-07 | 2026-10-07 | high |
| [E5] | Inline fallbacks: `--interactive`, `INLINE_THRESHOLD` default 2, no-reliable-signal runtimes; failure classifier; fresh continuation agent (not resume) | [gsd-core/skills/gsd-execute-phase/SKILL.md](file:///Users/paladm/git/ai-test/gsd-core/skills/gsd-execute-phase/SKILL.md) + [gsd-core/workflows/execute-plan.md](file:///Users/paladm/git/ai-test/gsd-core/gsd-core/workflows/execute-plan.md) | 2026-10-07 | 2026-10-07 | high |
| [E6] | Worktree isolation modes; per-agent branch; SUMMARY committed before return; intra-wave file-overlap forces sequential | [gsd-core/workflows/execute-phase.md](file:///Users/paladm/git/ai-test/gsd-core/gsd-core/workflows/execute-phase.md) | 2026-10-07 | 2026-10-07 | high |
| [E7] | gstack: main session is default executor; sprint command chain; subagents for review (adversarial + Codex), docs, spec→build worktree; no wave model; Conductor for parallel sprints | [gstack/README.md](file:///Users/paladm/git/ai-test/gstack/README.md) + [ship/SKILL.md](file:///Users/paladm/git/ai-test/gstack/ship/SKILL.md) + [spec/SKILL.md](file:///Users/paladm/git/ai-test/gstack/spec/SKILL.md) + [autoplan/SKILL.md](file:///Users/paladm/git/ai-test/gstack/autoplan/SKILL.md) | 2026-09-13 checkout | 2026-10-07 | high |
| [E8] | Skillgrid per-task shape already present; sequential by design; lease/wave machinery "future orchestrator skill, out of scope" | [.agents/skills/execution/subagent-execution/SKILL.md](.agents/skills/execution/subagent-execution/SKILL.md) | 2026-10-07 (working tree) | 2026-10-07 | high |
| [E9] | `simple-execution` is the no-subagent fallback (line 22) | [.agents/skills/execution/simple-execution/SKILL.md](.agents/skills/execution/simple-execution/SKILL.md) | 2026-10-07 (working tree) | 2026-10-07 | high |
| [E10] | Serial one-change lock; "worktree-by-default parallelism" out of scope (line 20); effort-budget table | [.skillgrid/ASSUMPTIONS.md](.skillgrid/ASSUMPTIONS.md) + [briefing.md](briefing.md) + [.agents/skills/_shared/rules/effort-budgets.md](.agents/skills/_shared/rules/effort-budgets.md) | 2026-10-07 (working tree) | 2026-10-07 | high |
