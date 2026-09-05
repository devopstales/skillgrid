---
name: sdd-init
description: Bootstrap Skillgrid SDD under onboard — detect project_name, tech_stack, testing_capabilities, and issue_tracker (AGENTS → config → Mnemonic → git → project files); validate with the user; write docs/skillgrid skeleton (config.yaml, agents/ stubs, glossary/ stubs, changes/, archive/) and the AGENTS skillgrid block. Use when the user says sdd init, initialize sdd, onboard SDD, or use-skillgrid finds an uninitialized repo.
license: MIT
metadata:
  author: devopstales
  version: "4.0"
  part-of: skillgrid
---

# SDD Init

Onboard helper (v4). Not a standalone public pipeline name — `use-skillgrid` / `sdd-onboard` call this to detect facts and write the skeleton.

Prompt-driven: detect → present → validate with the user → then write. Never guess.

## Hard Rules

- Source precedence for `project_name`, `tech_stack`, `testing_capabilities`, `issue_tracker`: **AGENTS.md/CLAUDE.md/GEMINI.md → docs/skillgrid/config.yaml → Mnemonic → git → project files**. First source that answers wins; later sources fill gaps.
- Confirm the full findings summary with the user **before writing any artifact**.
- Default issue tracker is **Backlog.md** unless remote/config says otherwise.
- `skill-registry.md` is **optional** — never an init gate. Skip it or generate on demand.
- Memory is hybrid: filesystem + Mnemonic.
- Agent config targets: `AGENTS.md` / `CLAUDE.md` / `GEMINI.md`. Primary gets the full block; others get a one-line pointer. See `../_shared/agent-config/`.
- If `docs/skillgrid/` already exists, report what is there and ask before updating.
- Use git only for detection — do not `git init`.

## Workflow

```
[ ] 1. Detect existing context
[ ] 2. Detect stack and testing
[ ] 3. Resolve project name and tracker
[ ] 4. Validate with user (blocking)
[ ] 5. Write skeleton + persistence
[ ] 6. Return envelope
```

### 1. Detect existing context

In order, record which source answered each fact:

- Root `AGENTS.md` / `CLAUDE.md` / `GEMINI.md` (including `<!-- skillgrid-sdd:start/end -->`)
- `docs/skillgrid/config.yaml` and existing layout
- Mnemonic: `mem_context`, then `mem_search` for `sdd-init/{project}`, `sdd/{project}/issue_tracker`, `sdd/{project}/testing-capabilities`
- `git remote -v` / `.git/config` if `.git` exists
- Prior `docs/skillgrid/agents/issue-tracker.md` (registry is optional history only)

### 2. Detect stack and testing

Per [references/init-details.md](references/init-details.md): manifests, CI, test runner/layers, coverage, linter, type checker, formatter. Record exact commands.

### 3. Resolve project name and tracker

| Option | id | Signal |
|---|---|---|
| **Backlog.md** (default) | `backlogmd` | no remote, or preference |
| GitHub | `gh` | `github.com` remote |
| GitLab | `glab` | GitLab remote |
| Jira | `jira` | `jira config` resolves, or user names instance + **project key** |

Resolution: existing config → git remote match → jira CLI → **Backlog.md**. Ambiguous tracker or facts → call **`questioning`**. Persist choice to Mnemonic `sdd/{project}/issue_tracker`.

### 4. Validate with the user (blocking)

Short confirmations, one fact at a time: project name → stack → testing → tracker → agent config target → artifact plan. Adjust on corrections, then write.

Artifact plan must include: `config.yaml`, `agents/` stubs (`issue-tracker.md`, `triage-labels.md`), **`glossary/` stubs** (`business.md`, `technical.md` — sibling of `agents/`), `changes/`, `archive/`, AGENTS skillgrid block. Registry: ask whether to generate now or skip.

### 5. Write skeleton + persistence

1. **SDD skeleton** — `docs/skillgrid/config.yaml`, `agents/` stubs, `glossary/` stubs, `changes/`, `archive/`. Formats in [references/init-details.md](references/init-details.md).
2. **Agent config** — render [`../_shared/agent-config/block.md`](../_shared/agent-config/block.md) via [`../_shared/agent-config/README.md`](../_shared/agent-config/README.md) (sentinel upsert).
3. **Issue tracker doc** — seed from `../_shared/issue-tracker/` (`backlogmd` | `github` | `gitlab` | `jira`). Triage: `../_shared/triage-labels.md`.
4. **Skill registry (optional)** — only if user asked: `node scripts/extract_skills.js --root <project-root>`. Never block on missing registry.
5. **Mnemonic** — session start; save `sdd-init/{project}`, `…/project_name`, `sdd/{project}/tech_stack`, `issue_tracker`, `testing-capabilities` (and registry only if generated).
6. **Backlog.md** — when selected: `backlog init "<name>" --integration-mode cli --backlog-dir .backlog --config-location folder --zero-padded-ids 3`; verify `backlog status`.

### 6. Return envelope

`status` · `project` · `tech_stack` · `testing_capabilities` · `issue_tracker` · `artifacts` · `validations` · `risks` · `next` (`use-skillgrid` → propose, or idle). Full shape in [references/init-details.md](references/init-details.md).

## Gotchas

- Glossary lives at `docs/skillgrid/glossary/` — **not** under `agents/`.
- Registry is optional; initialized? = `config.yaml` + AGENTS sentinel — not registry.
- Mnemonic topics: `sdd-init/{project}` vs `sdd/{project}/…` — misspell the project segment and later phases search into the void.
- `config.yaml` `context:` must stay under 10 lines.
- Backlog storage is `.backlog/tasks/`, not repo-root `backlog/`.
- Do not re-scan a fact AGENTS/config already answered.

## References

- [references/init-details.md](references/init-details.md) — detection checklists, optional registry scan, skeleton, envelope
- [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md)
- [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md)
- `../_shared/issue-tracker/` · `../_shared/triage-labels.md` · `../_shared/agent-config/`
