---
name: onboarding
description: Use when setting up Skillgrid for a new project. Detects stack, testing, and tracker; writes .skillgrid/config.yaml and an AGENTS.md block. Run once per project.
license: MIT
disable-model-invocation: true
metadata:
  author: devopstales
  version: "1.1"
  part-of: skillgrid
  based_on: skillgrid-v2:sdd-onboard + mattpocock-skills:setup-matt-pocock-skills + BMAD:bmad-project-context
---

# Onboarding

Detect project facts, confirm with the user, write `.skillgrid/config.yaml` and an AGENTS.md block. Run once per project — subsequent runs merge, not overwrite.

**The real job is inherited code.** The typical onboarding target is not a shiny new project — it is a codebase somebody else wrote, years old, with conventions to reverse-engineer and no documentation (or outdated documentation). Onboard it by reading what's actually there, not what a project "like this" would look like. Plan the next slice **on top of reality**, against the constraints the existing system already imposes — the data it has, the patterns it follows. **Reuse instead of regenerate.**

**Announce at start:** "I'm using the skillgrid:onboarding skill to configure Skillgrid for this project."

## When to Use

- First time running any Skillgrid skill in a new project (no `.skillgrid/config.yaml` exists)
- User says "set up Skillgrid", "init Skillgrid", "onboard this project"
- Project stack changed significantly (new language, new test runner)

**When NOT to use:** the project already has a valid `.skillgrid/config.yaml` and nothing about the stack changed — onboarding re-detects and confirms, it's not a per-task step. (Re-running to pick up a stack change IS a use, via merge mode.)

## Keep the AGENTS.md block lean

The `## Skillgrid` block is injected into the agent's context on **every** request, so bloat in it poisons *all* prompts. Treat it as a **navigation spine** — it points at where things live — not a knowledge dump. Source: the "configure your AI so it doesn't suck" discipline (WebDev Simplified), folded in here.

Apply these when rendering the block and whenever the user asks to extend the AGENTS.md:

1. **Keep it small.** Every line here ships with every prompt. If a section isn't needed for *most* requests, it does not belong inline.
2. **Split section-specific conventions out** into the files the block already references (`.skillgrid/artifacts/`, the skills, `docs/user-guide/`) and *link* to them, instead of inlining CSS/API/TS/delivery rules. A one-line "reference →" beats a ten-line dump.
3. **No "getting started / setup" section.** Onboarding, install, and first-run steps are not what the agent file is for — the agent already has the repo.
4. **Grow only on an observed failure (observe → fix).** Add a line to the block only when the agent *actually* did something wrong that this file can prevent. Don't preemptively document mistakes it hasn't made — you don't know which it will make. Notice a problem → record it in the right place → it stops repeating.
5. **Reference, don't write.** A standards line names ONE canonical file (or skill) in the pattern `**<Topic>** — reference `<path>` for detailed <topic>.` — never restate the standard's content in the block. The canonical `## Skillgrid` block carries the four reference lines (coding, testing, commits/verification, memory/code-index/web-cache) by default; a project with no shared-standards file for a topic skips that line rather than inventing content.
6. **High-level layout only.** A one-line "what's here" map is fine; never point at individual files or churning paths — they go stale and mislead the agent.
7. **Start minimal.** On a greenfield project, a couple of sentences (what the project is, the package manager if non-standard) + the references is the right size. Never "read the repo and generate a giant AGENTS.md" — that yields verbose, hard-to-use files.
8. **One source of truth across platforms.** `AGENTS.md` is the source of truth. `CLAUDE.md` / `GEMINI.md` get a **one-line pointer** (Skillgrid's choice over a symlink, which breaks under git/CI on some systems). Never two full blocks — they drift.
9. **Nested AGENTS.md for monorepos/areas.** Area-specific rules (a payments module, a UI kit, a `apps/api` vs `apps/web` split) live in a nested `<area>/AGENTS.md`; the agent loads root + the nested file for the folder it's working in. The root block stays global and lean.

**The preamble is separate from the sentinel block.** The rules above govern the `## Skillgrid` **sentinel block** — the lean navigation spine that ships with every prompt. The **preamble** (the `# Standards` region: `Environment & Tooling`, `Key Directories`, `Security & Escalation Boundaries`, `Dependency Policies`, `Architecture Constraints`, `Definition of Done`, `Engineering Standards`) is the one place AGENTS.md states *repo-specific facts* inline, because they're exactly what every request needs and there's no better canonical file for them. The "reference, don't write" rule applies to shared standards (testing, memory, code-indexing, SDD) — those stay reference pointers to the installed rule files; it does not apply to the repo-specific preamble sections. Render the preamble by filling `.skillgrid/config.yaml`'s `agents:` block and running `skillgrid init` (which writes `templates/agents-preamble.md`'s structure) — never by hand-editing the region.

These principles govern *what goes in the block*. The block's exact shape and the idempotent upsert rule stay authoritative in `_shared/agent-config/block.md`. The preamble's exact shape stays authoritative in `templates/agents-preamble.md`.

## The Process

The full step-by-step procedure lives in `references/process.md` — load it when actually running onboarding. The spine here names the decision and the handoff; the reference holds the steps.

```
Step 1    Detect     — read project, detect facts (never guess); git worktree guard
Step 1.5  Enroll     — brownfield only: mark existing capabilities, reverse-engineer conventions
Step 1.7  Generate   — fill .skillgrid/ARCHITECTURE.md <detect> markers with verified facts
Step 2    Confirm    — present each detected fact, confirm one at a time
Step 3    Write      — config.yaml, state.yaml, durable-knowledge zone, AGENTS.md block (idempotent upsert), tracker bootstrap, commit, skillgrid init
Step 4    Verify     — show written artifacts; confirm no leftover <detect> markers
```

Key invariants the spine must keep (details in `references/process.md`):
- **Detect, never guess.** Source precedence: AGENTS.md/CLAUDE.md → `.skillgrid/config.yaml` → Mnemonic → git remote → project files.
- **TDD and BDD are non-negotiable.** Always `tdd: true`, `bdd.enabled: true`.
- **AGENTS.md block is an idempotent upsert** between `<!-- skillgrid:start -->` / `<!-- skillgrid:end -->` — running onboarding twice never duplicates it.
- **One source of truth across platforms.** Full block in `AGENTS.md`; one-line pointer in `CLAUDE.md` (never a symlink).
- **`skillgrid init` is the deterministic finish** — it renders the AGENTS.md preamble and scaffolds `ARCHITECTURE.md`; onboarding only fills the `<detect>` markers.

## Merge Mode (Re-running)

If `.skillgrid/config.yaml` already exists:
1. Read the existing config
2. Re-detect facts (stack may have changed)
3. Show what changed: "Detected changes: test runner changed from X to Y. Update?"
4. Merge: update changed values, keep user-customized `rules:` sections
5. Re-write AGENTS.md block (idempotent upsert) — `{rules_block}` is owned by AGENTS.md's own `### Rules` section (edit there); the block re-renders from that section, not from `ASSUMPTIONS.md`
6. Reconcile `state.yaml` + the durable-knowledge zone: fill missing keys in `state.yaml` (never reset `pipeline`/`progress`); ensure `ASSUMPTIONS.md` / `ARCHITECTURE.md` exist; if `ARCHITECTURE.md` was scaffolded but still has unfilled `<detect>` markers (a stack change or first-pass gap), re-run Step 1.7 to fill them — never clobber the filled sections, only replace `<detect>` markers with newly detected facts; create any missing `artifacts/` stubs; run the brownfield migration (architecture into the root file; ADR bodies out to `artifacts/04-adr-*.md` with a path row left in `ASSUMPTIONS.md`; `00-prd.md` stays in `artifacts/`; locked constraints and glossary as in the migration bullet) if the project was onboarded before this layout

## What This Does NOT Do

- Does not install the Skillgrid CLI or skills — that's a machine-level concern
- Does not create `.skillgrid/specs/` or `.skillgrid/sdd/` directories — they're created on first use
- Does not write `.skillgrid/config.user.yaml` — there is one config file, committed
- Does not reset `state.yaml` `pipeline`/`progress` on re-run — those are the live project state, not onboarding outputs

## Common Rationalizations

| Rationalization | Reality |
|---|---|
| "The config already looks right — skip re-detection" | Stack and test runner drift between runs. Merge mode exists exactly to re-detect and show what changed. |
| "The manifests are obvious — no need to confirm with the user" | Step 2 is explicit: present each fact, confirm one at a time. A wrong `tdd` or tracker value silently breaks every later phase. |
| "I'll bootstrap the tracker after onboarding is done" | If `ticketing.enabled: true`, the CLI must be verified during onboarding (`gh --version`, `backlog status`, etc.). A tracker that isn't verified can't be trusted later. |
| "Both AGENTS.md and CLAUDE.md exist — write the block to both" | Two full blocks are two sources that drift. Write full to `AGENTS.md`, one-line pointer in `CLAUDE.md`. |
| "I'll add a bunch of conventions to the block up front so the agent doesn't get anything wrong" | The block ships with every prompt. Grow it only on an observed failure (observe → fix); split the rest into the referenced files. See *Keep the AGENTS.md block lean*. |
| "Let me read the whole repo and write a comprehensive AGENTS.md" | "Read the repo and generate a giant AGENTS.md" yields a verbose, hard-to-use file. Start minimal (a couple sentences + references) and let it grow on evidence. |
| "I'll hand-write the ARCHITECTURE.md skeleton" | `skillgrid init` scaffolds the 17-section structure deterministically from `templates/architecture.md`. Onboarding's job is to *fill* the `<detect>` markers with verified facts — not to re-invent the skeleton. Hand-writing it drifts from the template every project. |
| "I'll pad every ARCHITECTURE.md section to make it look complete" | A section with no real content stays `<detect>` or is dropped — padding a brownfield/greenfield doc with generic prose makes it *less* trustworthy. Facts over completeness. |
| "The user said 'looks good' — skip re-reading their corrections" | "Looks good" is only valid once every detected fact has been presented and confirmed individually. |

## Red Flags

**Never:**
- Guess a test runner — detect from manifests, then confirm
- Overwrite a user-customized `rules:` section
- Skip the confirmation step — always present detected facts
- Create both AGENTS.md and CLAUDE.md — pick the first that exists, or create AGENTS.md
- Hard-code paths that the config already defines
- Inline convention/how-to content into the `## Skillgrid` block — it is a navigation spine; point at the referenced files instead. The Issue Tracker line names the one convention file for `ticketing.type` (see `block.md`); do not restate that file
- Add a "getting started / setup" section to the block, or point it at individual churning source paths

## Verification

- [ ] `.skillgrid/config.yaml` exists and parses (`yaml` load succeeds); every detected field (stack, testing, commands, ticketing, TDD, commit discipline, BDD) is filled
- [ ] `.skillgrid/state.yaml` exists, parses, and has `pipeline` + `progress` + `constraints_ref` pointing at `.skillgrid/ASSUMPTIONS.md` (existing values preserved on re-run)
- [ ] `.skillgrid/ASSUMPTIONS.md` exists with the four tiers; `.skillgrid/ARCHITECTURE.md` exists, was scaffolded by `skillgrid init` (or copied by hand), and its `<detect>` markers are filled with verified facts for everything that exists (brownfield: all 17 sections addressed; greenfield: only sections with real content filled, the rest honestly left `<detect>` or dropped); every cited `file:line` was re-verified against the code
- [ ] AGENTS.md (or CLAUDE.md) contains exactly one `<!-- skillgrid:start -->` … `<!-- skillgrid:end -->` block with the correct `{project}`, tracker/memory lines, and a `### Rules` section that is the real source of the operational rules (decisions live in `ASSUMPTIONS.md` § LOCKED, not here)
- [ ] The block is lean: no inlined conventions/how-to or "getting started" section. Issue Tracker is the exact `{tracker_line}` for `ticketing.type` (a pointer to that convention file). No pointers at churning source paths. If both exist, `CLAUDE.md` holds only a one-line pointer to `AGENTS.md`
- [ ] Detected test runner command runs and exits 0 (e.g. `go test ./...`, `npm test`, `pytest`)
- [ ] Tracker CLI verified if `ticketing.enabled: true` (`gh --version` / `glab --version` / `jira config` / `backlog status`); skipped if `false`. If `ticketing.type: backlogmd`, every `.skillgrid/artifacts/04-adr-*.md` has a `.backlog/decisions/decision-NNN` copy
- [ ] `git rev-parse --is-inside-work-tree` succeeds; onboarding artifacts are committed (`git log -1` shows `chore: add Skillgrid config + state + artifacts`)
- [ ] If Mnemonic is enabled and a non-default embedder was chosen, `.skillgrid/config.d/indexing.yaml` exists with a valid `mnemonic.embedder` block (provider matches the user's choice; base_url/model/dimension are filled for ollama/external)
- [ ] If the project is brownfield (existing feature code + history), Step 1.5 ran: existing capabilities are recorded as `existing`, conventions were reverse-engineered into the block + glossary, and the first slice is planned against the existing system (reuse, not regenerate)
