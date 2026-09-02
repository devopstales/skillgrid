---
name: _shared
description: Shared Skillgrid SDD references consumed by sdd-* and issue-creation skills (issue tracker conventions, triage labels). Not invokable.
disable-model-invocation: true
user-invocable: false
---

## Purpose

This directory stores shared reference documents consumed by real SDD skills. Do not invoke it as a skill.

- `issue-tracker/` — per-tracker CLI convention templates written to `docs/agents/issue-tracker.md` by `sdd-init` and consumed by `issue-creation`:
  - [issue-tracker/backlogmd.md](issue-tracker/backlogmd.md) — Backlog.md (default tracker)
  - [issue-tracker/github.md](issue-tracker/github.md) — GitHub (`gh`)
  - [issue-tracker/gitlab.md](issue-tracker/gitlab.md) — GitLab (`glab`)
  - [issue-tracker/jira.md](issue-tracker/jira.md) — Jira (jira-cli)
- `agent-config/` — agent config block family written to `AGENTS.md` / `CLAUDE.md` / `GEMINI.md` by `sdd-init`:
  - [agent-config/README.md](agent-config/README.md) — target decision matrix + multi-platform rules (which file gets the full block)
  - [agent-config/block.md](agent-config/block.md) — the canonical `## Agent skills` payload + idempotent upsert sentinels (single source of truth)
  - [agent-config/agents.md](agent-config/agents.md) / [agent-config/claude.md](agent-config/claude.md) / [agent-config/gemini.md](agent-config/gemini.md) — per-target placement rules
- `conventions/` — shared contract documents every SDD skill must honor:
   - [conventions/sdd-structure.md](conventions/sdd-structure.md) — the `docs/skillgrid/` directory layout (changes/archive, NNN-slug, per-step files), artifact paths, phase order, and `config.yaml` reference.
   - [conventions/mnemonic-memory.md](conventions/mnemonic-memory.md) — naming, write, recovery, and session-close rules for all Mnemonic memory saves (the common memory config for every sdd-* skill).
  - [conventions/mnemonic-code-indexing.md](conventions/mnemonic-code-indexing.md) — the Mnemonic code-indexing ladder (`code_status` → `code_index` → `code_search` → `code_read`), config, and gotchas shared by every code-exploring skill (full schemas in the `mnemonic-code-index` skill).
  - [conventions/commits.md](conventions/commits.md) — commit message contract (conventional commits, no AI trailers, issue-tracker close token, multi-commit batches) shared by `sdd-apply`, `sdd-verify`, and any skill that commits.
- `triage-labels.md` — the five canonical triage roles shared across trackers.
