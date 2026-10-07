# Skillgrid workflow updates (GSD + BMAD + Cole + Superpowers + Matt) — Design Briefing (requirements & intent)

> **STATUS:** `draft` (2026-10-04) — Seven-source findings written (GSD, BMAD, Cole, Superpowers, Matt, gstack, **test-book**). Cole / Superpowers / Matt / validation / **scope pushback** locked (rounds 3–8). **First slice cut (round 9): reqs 21 + 19 + 8.** gstack freeze/careful (17–18), project-context map (req 20), **ARCHITECTURE.md enforcement (req 22)**, and **code-index enforcement (req 23)** stay proposed and queued behind the first slice (round 10). test-book borrow patterns (evaluator-independence grade, manifest.yaml, evals.json, packet schema, checkpoint-recovery, allowed-tools) captured 2026-10-04 — no new req; they attach to existing reqs or follow-ups. **Subagent execution req 24 captured 2026-10-07 (round 12): part (a) filesystem-as-truth is in the first slice; parts (b) post-wave gate and (c) goal verifier are queued.**

**Topic:** 2026-10-02-skillgrid-workflow-updates
**Date:** 2026-10-02
**Classification:** standard (T2) — pending interview
**Build shape:** Smallest usable whole (first slice TBD)
**Queued behind:** `2026-10-02-mnemonic-project-init` (serial: do not take `current_change` until that change ships or parks). Sibling queued spec: `2026-10-02-mnemonic-llm-provider`.
**Findings:** `findings.md` — GSD, BMAD, Cole, Superpowers, Matt, gstack, project-context skill (AI Labs / Karpathy loop, req 20, not locked), `## Research: What can Skillgrid learn from test-book (book production pipeline)?` (captured 2026-10-04, not locked), and `## Research: Should Skillgrid run execution in subagents (GSD gsd-execute-phase model)?` (captured 2026-10-07 → req 24, part (a) locked in first slice).

## Problem / Intent

Skillgrid already copied GSD Core’s rhetoric, several BMAD skills, Cole’s PIV spine, Superpowers’ catalog, and Matt’s grill/domain/ticket/deep-module craft, but left *mechanical* sequencing, plan repair, AI-layer honesty, the session-start bootstrap, and two Matt gates as prose or a broken hook. Plans can silently drop requirements, `state.yaml` is a narrative log, the running agent is never told the window is filling, `AGENTS.md` can go stale, `.skillgrid/ARCHITECTURE.md` is listed as an artifact but was never created, agents search with `find` / `ls` / `rg` / `eza` instead of the Mnemonic code index, Cursor starts without the router, a turn that did not land has no named interrupt, and a shallow module still passes review. This change exists to take the remaining mechanisms that fit this repo, merge the overlaps, and refuse the catalogs.

## Purpose & Success Criteria

- **Purpose:** Make the pipeline repeatable: scripts do parse / merge / validate / write; the agent judges and does not guess bookkeeping. Use scripts and templates to generate AGENTS.md brifing.md  blueprint.md for standard format and content. Plan quality, **how the change will be proven**, context headroom, always-loaded-rule honesty, and the first-turn router inject become something a JS script or hook can fail, not something a skill hopes the model remembers.
- **Success criteria:** First slice is **21 + 19 + 8** (round 9). In-scope (locked): Cole 9–11, Superpowers 13, Matt 15–16, **validation 19**, **scope pushback 21**. GSD/BMAD reqs 1–7 stay proposed; req 8 is in the first slice. Follow-ups: 12, 14. gstack freeze/careful (17–18), project-context map (req 20), ARCHITECTURE.md enforcement (req 22), and code-index enforcement (req 23) stay proposed. The gstack take that is on this plan is req 21, not the persona factory.
- **Out of scope (from findings, unless interview overrides):** Replacing Mnemonic with GSD `.planning/` markdown, intel JSON, or a BMAD `.memlog.md` as the *memory* store; importing GSD’s 70+ skill catalog, BMAD’s 57-skill / six-module surface, Cole’s 34 PIV skill names, Superpowers’ 14 skill names / 14-harness plugin tree, or Matt’s remaining catalog (wayfinder maps, `to-spec` skip-interview, `CONTEXT.md` as one file, TS dependency-cruiser, architecture HTML survey, loop-me/teach/wizard); **a single growing `project-context` SKILL.md that copies PRD / ARCHITECTURE / terms / ADRs** (req 20 is optional and undecided — if taken, it is a map only); BMM waterfall (brief → PRD → UX → epics → sprint); CIS / party-mode theater; named personas as the default UX; Python/`uv` as the skill runtime; Cole Python hooks; Drill/`uv`/tmux skill-eval harness in the first slice; dark-factory / unattended autonomy; prime-as-`git ls-files` dump; a markdown second-brain vault audit (Mnemonic + ADR-0011 already own state vs event); absent-equals-enabled defaults; worktree-by-default parallelism; ORM schema-drift for Prisma/Drizzle. Re-importing skills already `based_on` BMAD, Cole, Superpowers, or Matt.

## Context

Fits existing patterns: **yes-with-notes**.

The pipeline, `effort-budgets.md`, skill-size budget, `using-skillgrid` router, QA drift scripts, and hook dispatcher already exist. Notes: (1) do not invent a second memory store; (2) state mutations write the in-repo `state.yaml` the agent would have written — they must not hide project phase in SQLite; (3) any new gate must honor the existing effort-budget table so `trivial` / `small` fast-track stays light; (4) serial development stays locked — this folder is queued, not current; (5) **pipeline helpers are JS scripts** (existing `scripts/*.mjs` / skill `scripts/` pattern). Steal BMAD’s logic and contract; do not add Python/`uv` or a new `skillgrid` Go surface for these jobs.

Locked: Go 1.22+, no new dependencies without an ADR, serial one-change, spec-zone before code-zone, repo is source of truth (ADR-0013), SQLite second-brain store (ADR-0012).

## Approaches Considered

Approach for scope (locked 2026-10-02, rounds 3–5). First-slice cut is still interview.

- **Merge GSD + BMAD + Cole meta-layer + Superpowers bootstrap + Matt gates + validation planning + scope pushback into one mechanical set.** 
  - GSD: context monitor → state CLI → plan-check/coverage/scope-recovery → Nyquist / TEA ATDD → two-stage routers. 
  - BMAD: named correct-course mid-change. 
  - Cole: rules-file drift, declared file-coupling hook, reactive opportunity-scan.
  - Superpowers: SessionStart injects `using-skillgrid`.
  - Matt: `wait-what`; deep-module gate. **Req 19 (locked):** a Validation matrix before implement (happy/edge/failure, check, falsifier) — Must-Haves and Nyquist are not enough. **Req 21 (locked):** pile of independent outcomes → stop, recommend one wedge, human picks; default Hold/Reduce. Ablation and skill-behavior TDD stay named follow-ups. Freeze/careful and the project-context map stay proposed.
  - gstack: Do not import `/plan-eng-review`, `/office-hours`, or Expansion-as-default as is. Get logic from it to reinforce existing skills.
  - devopstales: Use hooks for gates that ai can not pass. Use scripts wher it can for repetable standard outcomes. Like generating standard format AGENTS.md, headers, tstuctured breafing.md, bluerint.md, findings.md with status. Harden user validation of files by waiting until the user change the status in the files. AI can not edit that. Force ticket update, state change wile developpning. Enforece architectional decision validation with user.

## Requirements

Reqs 1–8 are the GSD/BMAD set. **Reqs 9–11, 13, 15–16, 19, and 21 are locked in-scope** (rounds 3–8). **First slice = reqs 21 + 19 + 8** (round 9). Reqs 12 and 14 are named follow-ups. Reqs 17–18 (freeze/careful), 20 (project-context map), 22 (ARCHITECTURE.md enforcement), and 23 (code-index enforcement) stay proposed. **Req 21 applies to this briefing:** reqs 22–23 were added after the cut (round 10), so they are queued, not added to the first slice. **Exception (round 12):** req 24 part (a) (filesystem-as-truth completion in `subagent-execution` / `parallel-execution`) is explicitly in the first slice — it was the named wedge in the round-12 interview and edits only two execution skills; req 24 parts (b)–(c) are queued.

1. **Agent-facing context monitor:** The running agent is warned when remaining context is low; hooks fail open and never block the tool.
   - **Current:** Hooks record tool calls and checkpoint memory on stop / idle. No agent-facing headroom warning.
   - **Target:** Advisory WARNING / CRITICAL at documented remaining-% (GSD’s 35 / 25 is the starting number); debounce; point at `resume` / checkpoint, not at a new ceremony.
   - **Acceptance:** TBD after interview (which harness events, which signal).
   - **Acceptance scenario:** pending `acceptance.feature`

2. **Deterministic state mutations (script-owned):** Phase, current change, and blockers change only through a command that writes `state.yaml`.
   - **Current:** `progress` is a concatenated narrative; drift scripts exist but do not advance state. Planning/`resume` skills describe file-inventory in prose.
   - **Target:** A JS script (same family as `state-drift-check.mjs`) validates pointer fields against the spec folder and writes structured updates (BMAD `sprint_plan.py` logic: agent decides *which* change; script owns parse / merge / validate / write). Atomic write; JSON stdout including errors; `--dry-run`. Agents do not append free prose to `progress` as the state machine.
   - **Acceptance:** TBD (validate / sync / advance equivalents; script tests).
   - **Acceptance scenario:** pending `acceptance.feature`

3. **Plan-check before execute:** A planner that drops a requirement cannot pass slicing.
   - **Current:** Human Go / Revise after `tasks.md`. No mechanical coverage or scope-recovery loop.
   - **Target:** Requirements-coverage check + scope-reduction recovery (prohibit / check / re-inject), max 3 iterations, before the user gate.
   - **Acceptance:** TBD.
   - **Acceptance scenario:** pending `acceptance.feature`

4. **Nyquist contract:** Every in-scope requirement names a runnable test command before implementation.
   - **Current:** Acceptance features and TDD exist; there is no per-requirement command contract written before code.
   - **Target:** Wave-0 scaffolding for missing signals; later validate-phase may add tests only, not production code.
   - **Acceptance:** TBD (attach to `acceptance.feature`, `#### Gates`, or both).
   - **Acceptance scenario:** pending `acceptance.feature`

5. **Two-stage skill routing:** Eager skill listing is a handful of routers, not the full catalog.
   - **Current:** `using-skillgrid` routes; Cursor still sees the full skill list.
   - **Target:** Namespace routers (workflow / project / quality / context / manage / ideate or a Skillgrid-native set) with short keyword descriptions; concrete skills stay directly invocable where the loader requires it.
   - **Acceptance:** TBD (token/entry count before vs after on the target runtime).
   - **Acceptance scenario:** pending `acceptance.feature`

6. **Correct course:** A named mid-change path writes a change proposal when the plan is wrong while tickets are in flight.
   - **Current:** `resume` continues the plan; `reflect` only runs after ship. No impact pass across briefing / blueprint / tasks / ADRs.
   - **Target:** Load those artifacts, assess impact, propose start-over / patch briefing / redo blueprint / reticket — human picks. Does not silently edit the plan.
   - **Acceptance:** TBD (`resume` mode vs new skill).
   - **Acceptance scenario:** pending `acceptance.feature`

7. **Skill graph + completion markers (folded into req 2 + 5):** Next required skill and “started vs done” are queryable.
   - **Current:** `using-skillgrid` routes in prose; `resume` infers phase from file presence.
   - **Target:** A small committed graph (phase, precedes, required, outputs). File presence means started; a finalization marker means done. Help/resume reads the graph.
   - **Acceptance:** TBD (CSV/YAML vs generated from frontmatter).
   - **Acceptance scenario:** pending `acceptance.feature`

8. **Script contract on new pipeline helpers:** Repeating work is a tested script, not skill prose.
   - **Current:** `deterministic-boundary.md` + `skill-write` state the rule. QA/ship/citations call scripts. `using-skillgrid`, `interviewing`, `writing-blueprints`, `slicing`, `resume` do not.
   - **Target:** New helpers (state, phase-from-artifacts, requirement-coverage, catalog page) are `.mjs` next to the skill or under repo `scripts/`, following BMAD’s contract in JS: structured stdout, atomic writes, tests beside the script (`scripts/test-*.mjs`), no interactive prompts. Skill line names the judgment/script split. No Python/`uv`. No new Go CLI for these jobs.
   - **Acceptance:** Each extracted helper has a JS test; the calling SKILL.md is one line to run it plus judgment. `skill-write` quotes “agent decides *which*; script owns everything after.”
   - **Acceptance scenario:** pending `acceptance.feature`

9. **Rules-file drift check:** After a change, `AGENTS.md` (and the locked-constraint mirror) is still true and no longer than it needs to be.
   - **Current:** `state-drift-check.mjs` watches `state.yaml` vs the spec folder. Nothing holds always-loaded rule claims against the diff. Wrong rules are worse than missing ones.
   - **Target:** Pre-merge / QA advisory: flag only a now-false claim, a moved architecture-map path, or one new durable invariant. Mechanical path-claim check is a `.mjs`; the agent writes the one-line fix. Most diffs need no edit.
   - **Acceptance:** TBD (attach to `qa`, `requesting-code-review`, or a pre-merge hook).
   - **Acceptance scenario:** pending `acceptance.feature`

10. **Declared file coupling:** An edit of a coupled file cannot proceed until its declared dependencies were read this session.
    - **Current:** GSD-style “read before edit” is advisory. Precommit zone guard is spec-vs-code, not file-vs-file (schema↔migration, route↔schema).
    - **Target:** A fail-open JS hook plus a committed coupling map (Cole `dependencies.json` logic). No config → allow. Not Python/`uv`.
    - **Acceptance:** TBD (first-slice hook vs follow-up; who authors the first couplings).
    - **Acceptance scenario:** pending `acceptance.feature`

11. **AI-layer opportunity from a real run:** A finished run names the smallest durable change that would have prevented the failure — rule, skill, hook, or script.
    - **Current:** `reflect` writes a sourced retro after ship. It does not map the next primitive.
    - **Target:** Reactive scan first, using the change’s artifacts + Mnemonic `session_events`. Do not add a second `logs/agent-actions.jsonl` trail. Proactive log-window scan stays a follow-up.
    - **Acceptance:** TBD (`reflect` section vs a separate invoke).
    - **Acceptance scenario:** pending `acceptance.feature`

12. **Ablate always-loaded rules (named follow-up on this plan):** Measure whether always-loaded instructions still change outcomes before cutting them.
    - **Current:** Skill-size budgets shrink SKILL.md. Nothing experiments on `AGENTS.md` / always-loaded rules.
    - **Target:** Map the layer (always-loaded / on-demand / enforcement). Same probe task, both arms, throwaway worktrees, working tree untouched. User picks the probe. JS `map_layer` at most in a later slice; no Python/`uv` bake-off in the first slice.
    - **Acceptance:** Follow-up — listed so it is not dropped; not first-slice unless interview pulls it.
    - **Acceptance scenario:** pending `acceptance.feature`

13. **Harness bootstrap:** SessionStart injects `using-skillgrid` on every supported harness so skills are not inert files.
    - **Current:** `session-start.js` intends this but reads a path that does not exist (router lives under `lifecycle/`). Cursor SessionStart injects Mnemonic prime only.
    - **Target:** One JSON shape per platform (Cursor `additional_context`, Claude nested, else SDK), tested in `.mjs`. Cursor gets router + prime. A clean “let’s build X” session routes to `brainstorming` before code. Skills stay action-named, not tool-named. Do not rewrite SKILL.md per harness.
    - **Acceptance:** TBD (first slice with routers vs later harness-install ticket; one hook vs two on Cursor).
    - **Acceptance scenario:** pending `acceptance.feature`

14. **Skill-behavior TDD (named follow-up on this plan):** A skill change is proven by a pressure scenario — fail without the skill, pass with it.
    - **Current:** `skill-write` enforces the deterministic boundary and anatomy. It does not run a baseline-fail pass. Hook-shape tests exist for some scripts, not for SessionStart JSON.
    - **Target:** Pressure scenario for skill text; hook-shape unit tests in `.mjs` (Superpowers `test-session-start.sh` logic). No Drill / `uv` / tmux in the first slice.
    - **Acceptance:** Follow-up — listed so it is not dropped; not first-slice unless interview pulls it.
    - **Acceptance scenario:** pending `acceptance.feature`

15. **wait-what:** A named human interrupt re-pitches the last turn when it did not land.
    - **Current:** `resume` continues from files. Interviewing builds vocabulary up front. Nothing handles “I did not understand that.”
    - **Target:** User-invoked (or a `resume` / interviewing mode): re-pitch in the project terms, short, no new work. Not a second handoff format.
    - **Acceptance:** TBD (new skill vs mode).
    - **Acceptance scenario:** pending `acceptance.feature`

16. **Deep-module gate:** A blueprint names the seam and the interface test; review fails a new shallow module that is not called out.
    - **Current:** Vocabulary lives in `code-standards.md`, blueprint `codebase-design.md`, `ponytail`, and `99-logics.md`. Agents still ship busy interfaces. No gate.
    - **Target:** Attach to `writing-blueprints` + `requesting-code-review`. Prefer existing / highest / fewest seams (`to-spec` quiz). Merges with Nyquist (req 4): the test command is at the interface. No TypeScript dependency-cruiser; no new Go dep without an ADR.
    - **Acceptance:** TBD (blueprint only, review only, or both).
    - **Acceptance scenario:** pending `acceptance.feature`

17. **Edit freeze (proposed from gstack; not locked):** A user-activated session lock blocks writes outside one directory.
    - **Current:** Zone guard is spec-vs-code at commit time. Cole coupling (req 10) is read-before-edit pairs. Nothing scopes “only this folder this session.”
    - **Target:** JS PreToolUse hook. Off → allow. On → block Edit/Write outside the path. Complements Cole coupling. Not Bun, not a persona.
    - **Acceptance:** TBD (own hook vs a mode on the coupling hook; first slice vs follow-up).
    - **Acceptance scenario:** pending `acceptance.feature`

18. **Destructive-command careful (proposed from gstack; not locked):** Shell policy warns or hard-denies catastrophic commands.
    - **Current:** `cursor-policy.sh` / `tool-call-capture.js policy` (ADR-0021) does not match `rm -rf`, `DROP TABLE`, force-push, or `kubectl delete`.
    - **Target:** Extend that policy (or a sibling `hooks/careful.js`). MEDIUM → warn + override; root/home recursive delete and default-branch force-push hard-deny. JS. Fail open if the hook errors.
    - **Acceptance:** TBD (extend existing policy vs new hook).
    - **Acceptance scenario:** pending `acceptance.feature`

19. **Technical validation plan (locked 2026-10-02, round 6):** Before implementation, the change says how it will be proven — not only that tests will exist.
    - **Current:** Briefing lists requirements; blueprint Must-Haves list observable truths; Nyquist (req 4) will name a command; QA runs after code. This briefing itself still says “Acceptance: TBD” on most reqs. There is no matrix of happy / edge / failure, layer, command, and falsifier written *before* slicing. Agents implement, then invent the proof.
    - **Target:** A committed `## Validation` (on the blueprint, or a sibling `validation.md` if interview splits it) that plan-check treats as required. One row per in-scope requirement: happy path, edge, failure; the check (command or hook); the layer (unit / script / gate / browser); what result would falsify it; out-of-scope (what we are *not* proving). Merges with Nyquist (the command column), Must-Haves (truths become rows), deep-module gate (check sits at the interface), and `99-logics.md` (happy / edge / failure; a green suite is not the verdict). A coverage script fails slicing if a requirement has no row. Do not import gstack `/plan-eng-review` persona or autoplan.
    - **Acceptance:** TBD (section on `blueprint.md` vs `validation.md`; whether briefing may keep TBD until this section exists).
    - **Acceptance scenario:** pending `acceptance.feature`

20. **Project-context map (optional, not decided — AI Labs / Karpathy-loop):** A fresh feature agent always knows the project exists, and loads only the file the task needs. Not a second dump of the project.
    - **Current:** `onboarding` already comes from BMAD `bmad-project-context`. `AGENTS.md` is the always-on pointer. `using-skillgrid` already reads config, resume, then the domain model. Observed misses become one line in `AGENTS.md` or a mnemonic note. There is no dedicated map skill. There is no post-ship reconcile that updates `ARCHITECTURE.md` and terms from the real repo while keeping handwritten lines and flagging conflicts.
    - **What already replaces the video skill** (do not copy these into a `SKILL.md`):

      | Video skill holds | Use this instead |
      |---|---|
      | What the app does | `00-prd.md` + `ASSUMPTIONS.md` Product |
      | Pages / structure | `ARCHITECTURE.md` + code index |
      | Shared words | `01-business-terms.md` / `02-technical-terms.md` |
      | Hard decisions | `04-adr-NNNN-slug.md` (index in `ASSUMPTIONS.md`) |
      | What is locked | `ASSUMPTIONS.md` Locked constraints → `AGENTS.md` `### Rules` |
      | Stack | `.skillgrid/config.yaml` |
      | Always-on pointer | `AGENTS.md` `## Skillgrid` block |
      | Load on demand | Open the file, or `mem_search` → `mem_get_observation` |
      | After a real miss | One line in `AGENTS.md`, or a mnemonic note |

    - **Target if taken:** Interview picks one: skip; a thin map skill (when to open which file); or map + post-ship reconcile of `ARCHITECTURE.md` and terms (handwritten lines stay; conflicts flagged, not overwritten). The skill body is a map only. Existing files stay the store.
    - **Do not do (even if taken):** A growing `SKILL.md` that restates PRD / ARCHITECTURE / terms / ADRs. A one-file `CONTEXT.md`. Replacing Mnemonic or the committed artifacts with a skill body.
    - **Acceptance:** TBD. Not first-slice unless interview pulls it.
    - **Acceptance scenario:** pending `acceptance.feature`

21. **Scope pushback (locked 2026-10-02, gstack’s real remaining lesson):** When the user piles independent outcomes into one change, the agent stops and cuts — it does not refine a fat plan.
    - **Current:** Brainstorming already says “flag multiple subsystems immediately” and YAGNI. Interviewing scores Boundary Clarity. Serial one-change and “smallest usable whole” are locked. None of that *blocks* a briefing that keeps growing (this folder is the exhibit: six research sources, 20+ reqs, still one change). gstack’s advantage is the pushback: office-hours reframes and recommends the narrowest wedge; CEO-review has Hold Scope / Scope Reduction as named modes. [gstack README + `plan-ceo-review`]
    - **Target:** A mandatory gate in `brainstorming` / `interviewing` (and again before `writing-blueprints`): if the request or draft briefing names more than one independently shippable outcome, stop. List the pile. Recommend one first slice (smallest usable whole) and queue or drop the rest. Human picks hold / cut / split. Do not write `blueprint.md` until the pick is recorded in the interview log. Default mode is **Hold / Reduce**, not Expand (Skillgrid is serial; gstack Expansion stays out). No CEO persona. No `/office-hours` import.
    - **Acceptance:** TBD (clarity-gate dimension vs a hard stop in `using-skillgrid`; how “independently shippable” is counted).
    - **Acceptance scenario:** pending `acceptance.feature`

22. **ARCHITECTURE.md is created and kept current (proposed, round 10; queued behind the first slice):** `.skillgrid/ARCHITECTURE.md` exists once the repo has structure, and a change that adds or moves a component cannot ship without updating it.
    - **Current:** `AGENTS.md` lists `.skillgrid/ARCHITECTURE.md` as "Live repo/program structure", but the file does not exist in this repo. `onboarding` says it is "created lazily" (and may create it empty). Only the brainstorming **New Project** path writes it from `templates/ARCHITECTURE.md`. The **New Function** path updates it "only when they exist", so a brownfield repo never gets one. `ship` and `reflect` do not mention it. No script checks it. Skills that "read ARCHITECTURE.md if present" silently skip it.
    - **Target:** (a) **Creation:** a deterministic `.mjs` scaffolds `ARCHITECTURE.md` from `templates/ARCHITECTURE.md` plus real repo facts (top-level dirs, modules, entrypoints from the code index), with section stubs the agent fills. Handwritten lines are never overwritten. Onboarding / reconcile and the first brainstorming of a brownfield repo run it when the file is absent or empty. (b) **Updating:** a drift script (same family as `state-drift-check.mjs`; overlaps req 9 rules-drift path claims) compares paths and components named in `ARCHITECTURE.md` with the tree and the change's diff. It flags a named path that no longer exists, and a new top-level component or store in the diff that the file does not name. QA / pre-ship reports it; the agent writes the one-line fix. (c) **User validation:** a structural update made by an agent is marked for review and does not count as accepted until the user confirms it (devopstales: "harden user validation by waiting until the user changes the status"). Not a dump of the code tree. Not a second store: the code index stays the source of facts, and `ARCHITECTURE.md` is the human-readable map.
    - **Acceptance:** TBD (QA WARNING vs ship-blocking; generator inputs: code index vs `git ls-files`; whether a missing file blocks `writing-blueprints` for New Function; how the user-confirmed status is written).
    - **Acceptance scenario:** pending `acceptance.feature`

23. **Code search goes through the Mnemonic code index (proposed, round 10; queued behind the first slice):** When the project is indexed, an agent answers code-structure and code-search questions with the skillgrid MCP (`code_explore`, then `code_search` / `code_read`), not shell `find`, `locate`, `ls -R`, `rg`, `grep -r`, `fd`, `tree`, or `eza --tree`.
    - **Current:** The Mnemonic prime says "Call code_explore before rg. rg is the escape hatch when code_explore is empty." The `mnemonic.mdc` rule describes the code-index ladder. Both are advisory, and agents still reach for `find` / `ls` / `rg` / `eza`. The enforcement point exists: `hooks/hooks-cursor.json` routes `beforeShellExecution`, `beforeMCPExecution`, and `beforeReadFile` through `cursor-policy.sh` → `tool-call-capture.js policy` → `/policy/evaluate` (ADR-0021, effects `block` / `warn` / `guide` / `allow`, fail open). But this repo has no `.skillgrid/policy.yaml`, so every command is allowed with no guidance. `AGENTS.md` has no code-index rule. Native agent search tools (Cursor `Grep` / `Glob`) do not pass through `beforeShellExecution`.
    - **Target:** (a) A shipped default policy rule for `command_exec` that matches the search commands above and returns `guide` with a one-line pointer to `code_explore` / `code_search` / `code_read`. It escalates to `warn` after a repeat in the same session. It never blocks when the index is stale, missing, or Mnemonic is down (fail open, ADR-0021). Escape hatch: `code_explore` returned nothing for the query, or the target is not indexed (non-code files, `.skillgrid/` artifacts, gitignored paths). (b) A tested `.mjs` (or the policy test suite) holds the match list and the allow cases so the rule is not regex prose. (c) A one-line `AGENTS.md` rule and the `using-skillgrid` / `codebase-inspection` text point at the ladder. (d) Measure it: count of shell-search calls vs `code_*` calls per session from Mnemonic `session_events` (no second log), before and after.
    - **Do not do:** Hard-deny every `rg` (it is the named escape hatch). Block `git ls-files` / `ls` of a single directory. Import a second search tool. Add a dependency.
     - **Acceptance:** TBD (`guide` vs `warn` vs `block` threshold; how the hook learns the index is healthy without a slow round-trip; whether native `Grep` / `Glob` can be covered on Cursor or only on OpenCode / Kilo `tool.before`; whether the default ships in `~/.skillgrid/policy.yaml` via install or the repo `.skillgrid/policy.yaml`).
     - **Acceptance scenario:** pending `acceptance.feature`

 24. **Subagent execution: filesystem-as-truth + post-wave gate + goal verifier (proposed, round 12; first part in first slice, parts 2–3 queued):** `subagent-execution` already matches GSD's per-task shape (fresh implementer + per-task review + two-axis final review); the gap is the three GSD mechanisms the skill's orchestrator lacks. Not wave fan-out, not worktree-by-default (both stay out — serial constraint, briefing line 20).
    - **Current:** `subagent-execution` dispatches fresh implementers per task and trusts the return message plus the ledger for completion; verification between tasks is the task reviewer; integration breakage between tasks is only caught by the end-of-change `skillgrid:qa` gate; nothing checks the change *goal* (not just task completion) — the Validation matrix (req 19) is checked for row coverage, not re-verified at the end by an independent reader. `parallel-execution/concurrent-leaves` is the lightweight fan-out case; the lease/wave machinery for 4+ leaves is explicitly "a future orchestrator skill, out of scope" (subagent-execution:264-267).
    - **Target:** (a) **Filesystem-as-truth completion** (in first slice; edits `subagent-execution` + `parallel-execution` only, no new machinery): before a task is marked complete, the orchestrator spot-checks the worker's claimed artifacts on disk and the git log for its commits (the GSD `SUMMARY.md` + `git log --grep=<task-id>` move); a worker whose claims do not match disk or git is treated as not done, and the return message alone is never the completion signal. This strengthens the existing parent re-verify (Decompose & Gate) — the re-run covers the `#### Gates` oracles; this adds the artifact/commit check for harnesses where a subagent completion signal may be lost. (b) **Post-wave build+test gate** (queued behind the first slice): after each wave's tasks land, run `testing.runner` + `commands.build` before the next wave dispatches; a failing gate means the wave's integration is broken — stop, fix, re-run — instead of carrying breakage to the end-of-change QA. (c) **Goal verifier + gap-closure loop** (queued behind the first slice, synergetic with req 19): a separate verifier subagent (fresh context, most capable model per model-selection) re-runs the change's Validation matrix rows against the live tree and checks the briefing goal, not task checkboxes; gaps route to a named gap-closure pass (new tickets or a briefing patch via the correct-course path, req 6), then re-verify. The verifier is advisory-to-blocking: a gap it names cannot be parked without a ledger ruling.
    - **Do not do:** Wave fan-out (waves sequential, plans-in-wave parallel) — fights the serial one-change lock; worktree-by-default isolation for every task; importing GSD's `STATE.md` / `REQUIREMENTS.md` split (state.yaml + briefing + Mnemonic already own that); replacing `simple-execution` (it stays the no-subagent-harness fallback, per its line 22).
    - **Acceptance:** TBD (part (a) attaches to `subagent-execution`'s Decompose & Gate + Red Flags; part (b) to the wave transition in `subagent-execution` / `parallel-execution`; part (c) to the Final QA Gate boundary — a new `verification` subagent vs a `skillgrid:qa` mode; how the verifier's gap list maps to ticket IDs; whether (b) runs on `trivial`/`small` fast-track or only `standard`/`max` per the effort-budget table).
    - **Acceptance scenario:** pending `acceptance.feature`

## Implementation Decisions

- **Runtime (locked 2026-10-02):** JS scripts only. Steal BMAD’s logic (JSON stdout including errors, atomic write, `--dry-run`, tests beside the script, “agent decides *which*; script owns everything after”). Implement as `scripts/*.mjs` or `.agents/skills/_shared/scripts/*.mjs`, matching `state-drift-check.mjs` / `skill-size-budget.mjs`. Not Python/`uv`. Not a new `skillgrid` Go subcommand for these helpers (`deterministic-boundary.md` already names `.mjs`).
- **Modules to build/modify:** hooks + planning/lifecycle skills that today do the repeating half in prose; new `.mjs` helpers + tests (state, coverage, **validation-row coverage**, rules-drift path-claims, SessionStart JSON shapes); fail-open file-coupling hook + committed coupling map; SessionStart router inject (fix path; Cursor = router + prime); `reflect` (or a named invoke) for reactive opportunity-scan; `wait-what` (skill or mode); blueprint + review deep-module gate (reuse `codebase-design.md`); `writing-blueprints` / `slicing` grow a required Validation matrix. Not a new store.
- **Interfaces:** each helper prints one JSON object on stdout; exit 0 = pass, 1 = findings/fail, 2 = usage. Writes use temp + replace. `--dry-run` for mutating scripts.
- **Data flow:** disk artifacts remain source of truth; Mnemonic mirrors as an index (existing dual-write rule).
- **Error handling:** new hooks fail open (ADR-0021 posture). A state script that cannot validate must not invent phase.
- **Dependencies:** no new dependency without an ADR. Use the existing `yaml` package where YAML is parsed.

## Testing Decisions

Prior art for the locked JS contract: `scripts/test-state-drift.mjs`, `scripts/test-skill-size-budget.mjs`, `scripts/test-state-lock.mjs`. New helpers get a sibling `scripts/test-<name>.mjs`. SessionStart bootstrap gets a shape test (Cursor `additional_context` only / Claude nested only / SDK `additionalContext` only).

**Validation planning (req 19) — starter matrix for *this* change** (rows fill when the first slice is cut; empty cells are the gap this req exists to close):

| Req | Happy | Edge | Failure | Check | Falsifier |
|-----|-------|------|---------|-------|-----------|
| 24a filesystem-as-truth | worker claims done; artifact on disk + commit in git log → marked complete | worker claims done; commit present, artifact missing → not done, fix round | completion signal lost (no return message) → spot-check decides, never waits | `subagent-execution` Decompose & Gate spot-check (script or rule, TBD per req 8) | task marked complete on return message alone |
| 8 script contract | helper JSON stdout, exit 0/1/2 | `--dry-run` writes nothing | missing args → exit 2 | `node scripts/test-<name>.mjs` | test file absent or asserts prose |
| 4 Nyquist | each in-scope req names a command | wave-0 scaffold only | req with no command | coverage `.mjs` vs briefing IDs | slicing passes with a blank command |
| 13 bootstrap | SessionStart injects router | missing file → INFO degrade | wrong JSON shape (double-inject) | `scripts/test-session-start-shape.mjs` | Cursor session has prime only |
| 19 this matrix | every locked req has a row | follow-ups (12, 14) marked out-of-scope | locked req with no row | same coverage `.mjs` | `tasks.md` exists, Validation empty |
| 21 pushback | pile of ≥2 shippable outcomes → stop + one recommended wedge | human picks hold / cut / split; pick logged | agent writes `blueprint.md` on a fat briefing | interview log + blueprint absence | blueprint exists, interview log has no pick |

A green helper suite is not the verdict for this change. The fresh check is: a planner that drops a validation row cannot pass slicing, and a “let’s build X” session still hits `brainstorming` (req 13) when that slice lands.

## Impact on Global Docs

- `.skillgrid/artifacts/00-prd.md`: None this draft (skills/pipeline behavior is out of PRD scope per H4 unless interview expands engine surface).
- `.skillgrid/ASSUMPTIONS.md`: None this draft. A state-CLI or router decision that is hard to reverse may need ADR-0024+ after interview.
- `.skillgrid/ARCHITECTURE.md`: None this draft.
- `.skillgrid/artifacts/06-research-findings.md`: Durable distillation of the GSD, BMAD, colemedin-skills, Superpowers, and Matt comparisons (written with this draft).

## Clarity Report

Interview not started. Clarity gate has not run.

| Dimension           | Score | Min  | Status | Notes |
|---------------------|-------|------|--------|-------|
| Goal Clarity        | —     | 0.75 | ⚠      | First slice chosen (21 + 19 + 8); reqs 22–23 added as queued, not first-slice |
| Boundary Clarity    | —     | 0.70 | ⚠      | Catalogs / dark factory / Python hooks / Drill / skip-interview rejected; attach points open |
| Constraint Clarity  | —     | 0.65 | ⚠      | Serial + no-second-store + JS-scripts + Cole + bootstrap + Matt locked; harness events and first slice are not |
| Acceptance Criteria | —     | 0.70 | ⚠      | Reqs 9–11, 13, 15–16, 19, 21 in-scope; attach/acceptance still TBD |
| **Clarity**         | —     | ≤0.20| ⚠      | Do not write `blueprint.md` until interview exits |

**Interview log:**

| Round | Question summary | Decision locked |
|-------|------------------|-----------------|
| 0     | Findings captured | 2026-10-02 from gsd-core, BMAD, colemedin-skills, superpowers, mattpocock-skills, and gstack |
| 1     | Script runtime?   | **JS scripts; steal BMAD logic.** Not Python/`uv`. Not a new Go CLI for these helpers. |
| 2     | Scripts vs guessing? | **Scripts over guessing.** Same input → same output is a `.mjs` run, not skill prose the model re-derives. Documented in `docs/user-guide/99-logics.md`. |
| 3     | Cole on the plan? | **Yes.** Rules-file drift, declared file coupling, and reactive opportunity-scan are in-scope (reqs 9–11). Ablation is a named follow-up (req 12), not dropped. Do not import the 34-skill catalog. |
| 4     | Superpowers on the plan? | **Yes.** Harness bootstrap is in-scope (req 13). Skill-behavior TDD is a named follow-up (req 14), not dropped. Do not import the 14-skill catalog or Drill. |
| 5     | Matt on the plan? | **Yes.** `wait-what` and the deep-module gate are in-scope (reqs 15–16). Do not import wayfinder, skip-interview `to-spec`, `CONTEXT.md`, TS dep-cruiser, or a second handoff. |
| 6     | Technical validation planning lacking? | **Yes — lock it.** Req 19: a Validation matrix before implement (happy/edge/failure, check, falsifier). Must-Haves + Nyquist are not enough. Not gstack `/plan-eng-review`. |
| 7     | Karpathy-loop project-context skill? | **Optional, not decided.** Req 20. Skip / map-only skill / map + post-ship reconcile. Existing files stay the store. No dump SKILL.md. |
| 8     | gstack’s real advantage? | **Scope pushback.** When the user wants too many things, stop and cut (Hold/Reduce). Locked as req 21. Not freeze/careful, not a CEO persona, not Expansion-as-default. |
| 9     | Which wedge is the first slice? | **First slice:** cut — reqs 21, 19, 8. Scope pushback + required Validation matrix + tested `.mjs` contract (`scope-pushback-check.mjs`, `validation-coverage.mjs`). Queue the rest. |
| 10    | Enforce ARCHITECTURE.md and the code index? | **Added as proposed reqs 22 and 23, queued behind the first slice** (req 21: new outcomes after the cut are queued, not merged in). Enforcement points: ARCHITECTURE scaffold + drift script; ADR-0021 policy hook `guide`/`warn` for shell search. |
| 11    | What can Skillgrid learn from test-book (book production pipeline)? | **Six borrow patterns captured** (2026-10-04): evaluator-independence grade (→ `qa` + `requesting-code-review`), `manifest.yaml` spine (→ req 7), `evals.json` (→ req 14), packet schema (→ `subagent-execution` + `parallel-code-review`), checkpoint-then-choice recovery (→ `resume`), `allowed-tools` frontmatter (→ req 23). No new req; all attach to existing reqs or follow-ups. Not first-slice. |
| 12    | Should Skillgrid run everything in subagents (GSD gsd-core `gsd-execute-phase` model)? | **One req 24 (2026-10-07):** (a) filesystem-as-truth completion (spot-check artifacts + git log; return message alone is never the signal) lands in the first slice; (b) post-wave build+test gate and (c) goal verifier + gap-closure loop are queued behind it. **Rejected:** wave fan-out (waves sequential, plans-in-wave parallel) and worktree-by-default — both fight the serial lock and the "worktree-by-default parallelism" out-of-scope line. `simple-execution` stays the no-subagent-harness fallback; the default split is out of scope. |

## Open Questions & Assumptions

- **Locked (round 9):** first slice = **21 + 19 + 8**. Stop piling; require a Validation matrix; extract repeating bookkeeping to a tested `.mjs`. Queue 1–7, 9–18, 20, 22, 23 as follow-up changes.
- **Locked (round 12):** req 24 is one requirement with three parts. Part (a) filesystem-as-truth completion is **in the first slice**; parts (b) post-wave gate and (c) goal verifier are queued. Wave fan-out and worktree-by-default are rejected (serial lock). `simple-execution` stays the no-subagent-harness fallback.
- **Locked:** Scripts over guessing. Repeating procedures are JS `.mjs`; the agent does not re-derive them. Steal BMAD’s contract. Not Python/`uv`. Not a new `skillgrid` Go surface for these jobs. Guide: `docs/user-guide/99-logics.md`.
- **Locked:** Cole meta-layer is on this plan (reqs 9–11). Catalog / dark factory / prime-as-dump / Python hooks stay out. Ablation is req 12 (follow-up).
- **Locked:** Superpowers bootstrap is on this plan (req 13). Catalog / Drill / 14-harness tree stay out. Skill-behavior TDD is req 14 (follow-up).
- **Locked:** Matt `wait-what` and deep-module gate are on this plan (reqs 15–16). Wayfinder / skip-interview / one-file CONTEXT / TS dep-cruiser stay out.
- **Locked:** Technical validation planning is on this plan (req 19). A change that cannot say how it will be proven cannot pass slicing. Not a gstack eng-review persona.
- **Locked:** Scope pushback is on this plan (req 21). A pile of independently shippable outcomes cannot proceed to blueprint until the human picks one wedge. Default Hold/Reduce. This briefing is over-scope until the first-slice cut.
- **Question:** Req 21 attach — hard stop in `using-skillgrid` / brainstorming, or a clarity-gate fail until the cut is logged?
- **Question:** Context-monitor signal and thresholds per harness (Cursor / OpenCode / Kilo)?
- **Question:** Do two-stage routers help on Cursor’s flat loader, or only shorter descriptions?
- **Question:** Nyquist attaches to `acceptance.feature`, `#### Gates`, or both? (Req 19 may fold Nyquist into the Validation matrix.)
- **Question:** Req 19 lives as `## Validation` on `blueprint.md` or a sibling `validation.md`?
- **Question:** Is the skill graph a committed file or generated from skill frontmatter?
- **Question:** Does correct-course live as a `resume` mode or a new skill?
- **Question:** May `briefing.md` be regenerated from a memlog/Mnemonic, or must it stay hand-authored?
- **Question:** Does rules-drift attach to `qa`, `requesting-code-review`, or a pre-merge hook?
- **Question:** Is file coupling a first-slice hook or a follow-up after state/plan-check scripts?
- **Question:** Reactive opportunity-scan in `reflect` vs a separate skill?
- **Question:** Superpowers bootstrap (req 13) — first slice with routers, or a later harness-install ticket? Cursor: one hook (router + prime) or two?
- **Question:** `wait-what` (req 15) — new skill vs a `resume` / interviewing mode?
- **Question:** Deep-module gate (req 16) — `writing-blueprints`, `requesting-code-review`, or both?
- **Question:** gstack freeze + careful (reqs 17–18) — on the plan? Not locked. One hook vs two; first slice vs follow-up?
- **Question:** Project-context skill (req 20) — skip, map-only skill, or map + post-ship reconcile? Not decided. Not a dump of the project into a SKILL.md. (Req 22 overlaps it: the post-ship reconcile of `ARCHITECTURE.md` is req 22 (b).)
- **Question:** ARCHITECTURE.md (req 22) — does a missing or stale file WARN in QA or block ship? Is the scaffold built from the code index or `git ls-files`? Is it one change with req 9 rules-drift (both check path claims)?
- **Question:** Code-index enforcement (req 23) — `guide` only, `warn` on repeat, or `block` when the index is healthy? Repo `.skillgrid/policy.yaml` or an installed default? Can Cursor native `Grep` / `Glob` be covered at all?
- **Assumption:** Serial constraint holds — this change stays queued behind `mnemonic-project-init`.
- **Assumption:** Mnemonic remains the memory/code store; GSD intel/learnings files are not imported.
- **Question (test-book borrows, round 11):** Evaluator-independence grade (borrow 1) — section in the QA report + `state.yaml` `progress` line, or a field in `state.yaml` only? Grade-C self-eval: diagnostic-only advisory, or a hard `CONCERNS` that blocks `PASS` until a Grade-A re-run?
- **Question (test-book borrows, round 11):** `manifest.yaml` (borrow 2) — one per change under `.skillgrid/specs/<change>/`, or repo-level pipeline manifest + per-change overrides? Does it replace `sdd-structure.md` or complement it (shape vs semantics)?
- **Question (test-book borrows, round 11):** `evals.json` (borrow 3) — does the QA gate run them, or a separate `skill-eval` invoke? Only for `qa` / `acceptance-test-authoring` / `research`, or all skills?
- **Question (test-book borrows, round 11):** Packet schema (borrow 4) — `_shared/rules/subagent-packet.md` convention, or frontmatter on the dispatching skill? Does `parallel-code-review` get the same schema?
- **Question (test-book borrows, round 11):** Checkpoint-then-choice (borrow 5) — checkpoint in `state.yaml` `progress` + Mnemonic, or a `RUN_REPORT.md` line? Does it apply to `trivial` / `small` fast-track or only `standard` / `max`?
- **Question (req 24, round 12):** Part (a) — one-line rule in the execution skills, or a tested `.mjs` helper per req 8 (spot-check: artifact existence + `git log --grep=<task-id>` → JSON verdict)? The `[skillgrid-context]` block names the task ID — confirm it is always present in worker commits.
- **Question (req 24, round 12):** Part (b) — gate runs `testing.runner` + `commands.build` only, or also lint? `trivial`/`small` fast-track or `standard`/`max` only (effort-budget table)?
- **Question (req 24, round 12):** Part (c) — new `verification` subagent dispatched by the execution skills, or a mode inside `skillgrid:qa`? How do its named gaps map to Backlog ticket IDs (new tickets vs a correct-course briefing patch, req 6)?

## Decisions (ADR)

None. No decision from this draft cleared the ADR bar.

## Terms

None yet. If the interview keeps “Nyquist”, “plan-check”, or “context monitor” as Skillgrid terms, add them to `02-technical-terms.md` then — do not redefine them here.
