---
name: sdd-openspec-git
id: sdd-openspec-git
category: Workflow
description: OpenSpec git gates — when proposal/apply/archive must cross main and commits are explicit
usage: "/sdd-openspec-git [context or change name]"
---

You are executing `/sdd-openspec-git` for the SDD / OpenSpec workflow.

## Skill Loading Contract

Load and follow first:
- `.agents/skills/openspec-git-discipline/SKILL.md`

## Workflow

1. Parse arguments for the active change name, phase (propose, continue, apply, verify, archive), or general git-state check.
2. Apply the skill’s gates: `git status --short`, branch vs `main`, whether proposal artifacts are committed before apply, archive only from `main` after merge.
3. If the user has not asked for commits/branches/merges, **pause and explain** — never auto-commit or merge.
4. Surface red flags from the skill (worktree-only proposal, archiving from feature branch, etc.) with the suggested wording from the skill when helpful.

## Return Contract

Return a structured result with:
- `status`: `completed | blocked | failed`
- `executive_summary`
- `detailed_report` (git facts, openspec paths checked, gate outcomes)
- `artifacts` (paths to any written notes; usually none unless user asked)
- `next_recommended` (e.g. merge to `main`, commit artifacts, run `/sdd-apply`)
- `risks`
