---
name: sdd-onboard
description: "Bootstrap Skillgrid SDD in safe order (map → init → agent-context → constraints → domain) for greenfield or brownfield repos. Detects project_name, tech_stack, testing_capabilities, issue_tracker (AGENTS → config → Mnemonic → git → project files), validates with the user, writes the docs/skillgrid skeleton and AGENTS skillgrid block. Use when the repo is uninitialized, the user says onboard/bootstrap/init SDD, or use-skillgrid routes to onboard."
license: MIT
metadata:
  author: devopstales
  version: "2.0"
  part-of: skillgrid
  family: sdd
---

# SDD Onboard

Stage orchestrator for Skillgrid bootstrap. Run helpers in **safe order**. Does **not** implement product changes, write `proposal.md`, or start propose/apply.

Layout: [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md).

## Hard Rules

- **Safe order** only — never skip ahead to propose/spec/apply from this skill.
- Greenfield: skip map. Brownfield (existing source): map first when useful.
- Stop with a summary + next step. Do not freestyle a product feature here.
- **Prompt-driven** — detect → present → validate with the user → then write. Never guess.
- Source precedence for `project_name`, `tech_stack`, `testing_capabilities`, `issue_tracker`: **AGENTS.md/CLAUDE.md/GEMINI.md → docs/skillgrid/config.yaml → Mnemonic → git → project files**. First source that answers wins; later sources fill gaps.
- Default issue tracker is **Backlog.md** unless remote/config says otherwise.
- `skill-registry.md` is **optional** — never an init gate. Skip it or generate on demand.
- Memory is hybrid: filesystem + Mnemonic.
- Agent config targets: `AGENTS.md` / `CLAUDE.md` / `GEMINI.md`. Primary gets the full block; others get a one-line pointer. See `../_shared/agent-config/`.
- If `docs/skillgrid/` already exists, report what is there and ask before updating.
- Use git only for detection — do not `git init`.

## Workflow

```
[ ] 1. Classify greenfield vs brownfield
[ ] 2. (Brownfield) map — references/map-codebase.md; optional; skip if user declines
[ ] 3. Init — detect facts, validate with user, write skeleton + persistence
[ ] 4. agent-context — references/agent-context.md; AGENTS.md sentinel
[ ] 5. constraints — references/constraints.md; quality bar into config.yaml rules.*
[ ] 6. domain — references/domain.md; glossary stubs under docs/skillgrid/glossary/
[ ] 7. Summary + next (propose or idle)
```

### 1. Classify

- **Greenfield** — little/no application source (empty repo, docs-only, or user says new app).
- **Brownfield** — meaningful existing code to navigate.

Ask once if unclear. Record the choice in the summary.

### 2. Map (brownfield only)

Follow `references/map-codebase.md`. User may skip. Primary navigation remains Mnemonic `code_*` — the map is narrative, not a second index.

### 3. Init

**3a. Detect existing context.** In order, record which source answered each fact:

- Root `AGENTS.md` / `CLAUDE.md` / `GEMINI.md` (including `<!-- skillgrid-sdd:start/end -->`)
- `docs/skillgrid/config.yaml` and existing layout
- Mnemonic: `mem_context`, then `mem_search` for `sdd-init/{project}`, `sdd/{project}/issue_tracker`, `sdd/{project}/testing-capabilities`
- `git remote -v` / `.git/config` if `.git` exists
- Prior `docs/skillgrid/agents/issue-tracker.md` (registry is optional history only)

**3b. Detect stack and testing.** Per [references/init-details.md](references/init-details.md): manifests, CI, test runner/layers, coverage, linter, type checker, formatter. Record exact commands.

**3c. Resolve project name and tracker.**

| Option | id | Signal |
|---|---|---|
| **Backlog.md** (default) | `backlogmd` | no remote, or preference |
| GitHub | `gh` | `github.com` remote |
| GitLab | `glab` | GitLab remote |
| Jira | `jira` | `jira config` resolves, or user names instance + **project key** |

Resolution: existing config → git remote match → jira CLI → **Backlog.md**. Ambiguous tracker or facts → call **`questioning`**. Persist choice to Mnemonic `sdd/{project}/issue_tracker`.

**3d. Validate with the user (blocking).** Short confirmations, one fact at a time: project name → stack → testing → tracker → agent config target → artifact plan. Adjust on corrections, then write.

Artifact plan must include: `config.yaml`, `agents/` stubs (`issue-tracker.md`, `triage-labels.md`), **`glossary/` stubs** (`business.md`, `technical.md` — sibling of `agents/`), `changes/`, `archive/`, AGENTS skillgrid block. Registry: ask whether to generate now or skip.

**3e. Write skeleton + persistence.**

1. **SDD skeleton** — `docs/skillgrid/config.yaml`, `agents/` stubs, `glossary/` stubs, `changes/`, `archive/`. Formats in [references/init-details.md](references/init-details.md).
2. **Agent config** — render [`../_shared/agent-config/block.md`](../_shared/agent-config/block.md) via [`../_shared/agent-config/README.md`](../_shared/agent-config/README.md) (sentinel upsert).
3. **Issue tracker doc** — seed from `../_shared/issue-tracker/` (`backlogmd` | `github` | `gitlab` | `jira`). Triage: `../_shared/triage-labels.md`.
4. **Skill registry (optional)** — only if user asked: `node scripts/extract_skills.js --root <project-root>` (from this skill dir). Never block on missing registry.
5. **Mnemonic** — session start; save `sdd-init/{project}`, `…/project_name`, `sdd/{project}/tech_stack`, `issue_tracker`, `testing-capabilities` (and registry only if generated).
6. **Backlog.md** — when selected: `backlog init "<name>" --integration-mode cli --backlog-dir .backlog --config-location folder --zero-padded-ids 3`; verify `backlog status`.

### 4. Agent-context

Follow `references/agent-context.md`. Refresh even if init already wrote a block — ensure the v4 workflow line.

### 5. Constraints

Follow `references/constraints.md`. Quality bar into `config.yaml` `rules.*`.

### 6. Domain

Follow `references/domain.md`. Glossary stubs under `docs/skillgrid/glossary/`.

If `docs/skillgrid/config.yaml` already exists mid-run, ask before overwriting; prefer refresh of missing pieces only.

### 7. Stop — summary + next

Return:

- Classification (greenfield/brownfield)
- Init envelope: `status` · `project` · `tech_stack` · `testing_capabilities` · `issue_tracker` · `artifacts` · `validations` · `risks` (shape in [references/init-details.md](references/init-details.md))
- Artifacts created/updated (paths)
- What was skipped
- **Next:** if the user already stated a change → suggest `sdd-propose` (via `use-skillgrid`); else **idle** until they name work

Do not auto-enter propose without a stated change.

## Gotchas

- Glossary lives at `docs/skillgrid/glossary/` — **not** under `agents/`.
- Initialized? = `config.yaml` + AGENTS sentinel — not CONTEXT.md / CONSTRAINTS.md / registry.
- Gentleman/GSD "onboard" that teaches by shipping a full cycle is **out of scope** — Skillgrid onboard only bootstraps context.
- Registry (`skill-registry.md`) is optional — never block onboard on it.
- Mnemonic topics: `sdd-init/{project}` vs `sdd/{project}/…` — misspell the project segment and later phases search into the void.
- `config.yaml` `context:` must stay under 10 lines.
- Backlog storage is `.backlog/tasks/`, not repo-root `backlog/`.
- Do not re-scan a fact AGENTS/config already answered.
- After onboard, product work starts at **propose**, not explore-as-top-level-stage.

## References

- [references/init-details.md](references/init-details.md) — detection checklists, optional registry scan, skeleton, envelope
- [references/map-codebase.md](references/map-codebase.md) · [references/agent-context.md](references/agent-context.md) · [references/constraints.md](references/constraints.md) · [references/domain.md](references/domain.md)
- [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md)
- [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md)
- `../_shared/issue-tracker/` · `../_shared/triage-labels.md` · `../_shared/agent-config/`
