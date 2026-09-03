---
name: sdd-init
description: Initialize the Skillgrid SDD workspace — detect project_name, tech_stack, testing_capabilities, and issue_tracker from AGENTS.md/CLAUDE.md/GEMINI.md, docs/skillgrid/config.yaml, Mnemonic, git remote, and project files; validate with the user; then build the skill registry, agent config block (AGENTS.md/CLAUDE.md/GEMINI.md), the docs/skillgrid/ skeleton, Mnemonic observations, and Backlog.md if selected. Use when the user says sdd init, initialize sdd, or sets up SDD in a new project.
license: MIT
metadata:
  author: devopstales
  version: "2.0"
  part-of: skillgrid
---

# SDD Init

First phase of the Skillgrid SDD workflow: `init → explore → propose → design → tasks → spec → apply → verify → archive`. Bootstrap the SDD context so every later phase has known project facts and a working issue tracker.

Prompt-driven skill: explore, present findings, validate with the user, then write. Never guess — detect the real stack.

## Hard Rules

- Detect before writing. Every fact (project name, stack, testing, tracker) must come from a detected source or an explicit user answer.
- Source precedence for `project_name`, `tech_stack`, `testing_capabilities`, `issue_tracker`: **AGENTS.md/CLAUDE.md/GEMINI.md → docs/skillgrid/config.yaml → Mnemonic → git repo → project files**. First source that answers wins; later sources fill gaps.
- Memory is `hybrid` always: persist to both Mnemonic and filesystem.
- Agent config targets are `AGENTS.md`, `CLAUDE.md`, and `GEMINI.md`. Never create a second root config file — edit the primary that exists. If none exists, ask the user which to create (default suggestion `AGENTS.md`). Selection + payload rules live in `../_shared/agent-config/`.
- If `docs/skillgrid/` already exists, report what is there and ask before updating it.
- Use git only to detect `project_name` and `issue_tracker`; do not force a git work tree or run `git init`.
- Confirm the full findings summary with the user before writing any artifact.
- Default issue tracker is **Backlog.md** unless the git remote or prior config says otherwise.

## Workflow

```
[ ] 1. Detect existing context (AGENTS.md/CLAUDE.md/GEMINI.md, docs/skillgrid/config.yaml, Mnemonic)
[ ] 2. Detect stack and testing capabilities from project files
[ ] 3. Resolve project name and issue tracker from git remote + user
[ ] 4. Validate findings with user
[ ] 5. Initialize persistence (registry, agent config, docs/skillgrid/, Mnemonic, optional Backlog.md)
[ ] 6. Return the initialization envelope
```

### 1. Detect existing context

Check, in order, and record which source answered each fact:

- `AGENTS.md`, `CLAUDE.md`, or `GEMINI.md` at project root — search `project_name`, `tech_stack`, `testing_capabilities`, `issue_tracker` (including an `## Agent skills` block or `<!-- skillgrid-sdd:start/end -->` markers).
- `docs/skillgrid/config.yaml` — same four facts; note existing `docs/skillgrid/` layout (changes/, archive/, NNN-slug folders).
- Mnemonic — `mem_search` for `sdd/{project}/issue_tracker`, `sdd/{project}/testing-capabilities`, `sdd-init/{project}`. Use `mem_context` first for recent sessions.
- `git remote -v` / `.git/config` — is this a git repo? Which host (GitHub, GitLab, other)? Gives `project_name` candidate and tracker candidate.
- `docs/agents/skill-registry.md` and `docs/agents/issue-tracker.md` — prior sdd-init output.

### 2. Detect stack and testing capabilities

Inspect per the checklist in [references/init-details.md](references/init-details.md): stack manifests (`package.json`, `go.mod`, `pyproject.toml`, `Cargo.toml`, `requirements.txt`), CI config, lint/test/formatter config. Detect test runner, test layers (unit/integration/E2E), coverage tool, linter, type checker, formatter. Record exact commands.

### 3. Resolve project name and issue tracker

Use git only for detection, not for creating a work tree. Run `git remote -v` / `.git/config` if a `.git` directory exists — this gives `project_name` candidate and tracker candidate (GitHub/GitLab). If no `.git` directory exists, skip git detection entirely and ask the user for `project_name` and tracker preference.

Four tracker options — propose one, the user confirms:

| Option | id | Signal / requirement |
|---|---|---|
| **Backlog.md** (default) | `backlogmd` | no remote, or user preference |
| GitHub | `gh` | git remote on `github.com` |
| GitLab | `glab` | git remote on `gitlab.com` / self-hosted |
| Jira | `jira` | existing `jira init` config (`jira config` resolves), or user points to an instance + project key |

Resolution order: existing config (step 1 sources) → git remote match (GitHub/GitLab) → `jira` CLI present with a resolvable project key → **Backlog.md** default. For Jira, capture the **project key** (from `jira config` or the user) — it is mandatory for every `jira` command and for the tracker-line in the agent config block. Record the choice — it persists to Mnemonic `sdd/{project}/issue_tracker`.

### 4. Validate with the user

Present a single summary table: project name, tech stack, testing capability table, issue tracker, and the list of artifacts that will be created/updated. One confirmation, then write. Adjust per user corrections and re-detect only the corrected facts.

### 5. Initialize persistence

Create/update the artifacts:
1. **Skill registry** at `docs/agents/skill-registry.md` — scan and index installed skills per the scan rules in [references/init-details.md](references/init-details.md). The registry is an index (paths + triggers), not a summary.
2. **Agent config** — render the canonical `## Agent skills` block from [`../_shared/agent-config/block.md`](../_shared/agent-config/block.md) and write it per the target decision matrix in [`../_shared/agent-config/README.md`](../_shared/agent-config/README.md): primary = existing `AGENTS.md` → `CLAUDE.md` → `GEMINI.md` (else ask). Use the idempotent sentinel upsert; secondary targets get a one-line pointer only. Point at `docs/agents/issue-tracker.md`.
3. **Issue tracker doc** — write `docs/agents/issue-tracker.md` from the matching seed template in `../_shared/issue-tracker/`:
   - [`../_shared/issue-tracker/backlogmd.md`](../_shared/issue-tracker/backlogmd.md) — Backlog.md (default)
   - [`../_shared/issue-tracker/github.md`](../_shared/issue-tracker/github.md) — GitHub
   - [`../_shared/issue-tracker/gitlab.md`](../_shared/issue-tracker/gitlab.md) — GitLab
   - [`../_shared/issue-tracker/jira.md`](../_shared/issue-tracker/jira.md) — Jira
   Triage role vocabulary: `../_shared/triage-labels.md`.
4. **SDD skeleton** if absent: `docs/skillgrid/config.yaml`, `docs/skillgrid/changes/`, `docs/skillgrid/archive/`. Config format in [references/init-details.md](references/init-details.md).
5. **Mnemonic observations** — per the shared memory config in [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md) (start a `mem_session_start` session first; `scope: project`):
   - `sdd-init/{project}` (type `architecture`) — detected project context.
   - `sdd-init/{project}/project_name` (type `config`) — detected project name.
   - `sdd/{project}/tech_stack` (type `config`) — detected stack.
   - `sdd/{project}/issue_tracker` (type `config`) — tracker + CLI conventions.
   - `sdd/{project}/testing-capabilities` (type `config`) — testing table.
   - `skill-registry` (type `config`) — registry index.
6. **Backlog.md** — only when selected: initialize via the backlog CLI, then scaffold support files:
   - Run: `backlog init "<project-name>" --integration-mode cli --backlog-dir .backlog --config-location folder --zero-padded-ids 3`
   - Creates `config.yml` with `backlog_directory: .backlog`, `.backlog/tasks/`, and seed docs (`docs/agents/issue-tracker.md` already written at step 3).
   - Verify with `backlog status` (no uncommitted work should exist after init).

### 6. Return the envelope

`status` · `project` · `tech_stack` · `testing_capabilities` table · `issue_tracker` · `artifacts` created/updated (paths + Mnemonic observation ids) · `validations` applied by user · `risks`/limitations · `next` step (`/sdd-explore`). Full skeleton in [references/init-details.md](references/init-details.md).

## Gotchas

- Mnemonic topics are namespaced: `sdd-init/{project}` vs `sdd/{project}/...` — misspell the project segment and later phases search into the void. Reuse `topic_key` upserts; never create near-duplicate observations.
- `docs/skillgrid/config.yaml` `context:` must stay under 10 lines — it is injected into every later phase.
- Backlog.md storage lives under `.backlog/tasks/<ID>.md` (set via `backlog_directory: .backlog` in `backlog.config.yml`), not repo root `backlog/` — the plan's tree shows both, but config is authoritative.
- Don't re-scan the world for a fact that AGENTS.md/CLAUDE.md/GEMINI.md already answers; source precedence exists to avoid double-writing conflicting facts.
- A git remote on `gitlab.com` means GitLab even if the user says "GitHub" — confirm, don't assume.

## References

- [references/init-details.md](references/init-details.md) — detection checklists, registry scan rules, SDD skeleton, Mnemonic saves, output envelope.
- [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md) — common Mnemonic memory config: naming, upserts, 2-step recovery, session protocol.
- [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md) — the SDD layout (changes/archive, NNN-slug, steps, per-step files) the skeleton seeds.
- `../_shared/issue-tracker/` + `../_shared/triage-labels.md` — tracker templates and label vocabulary consumed by `sdd-init` and `issue-creation`.
