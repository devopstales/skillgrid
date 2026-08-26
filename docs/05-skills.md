# Skills

Skills are behaviors agents load per task. Installed via `config.d/skills.yaml` and `config.d/skills/` — see [03-config-reference](03-config-reference.md). Plugins: [07-plugins](07-plugins.md). Rollout order: [superpowers/README](superpowers/README.md).

---

## Proposed end-state skill list

**35 skills** in the default bundle after full rollout (Phases 1–4). Grouped by when they apply in a session.

### Bootstrap & memory

| Skill | Role |
|-------|------|
| `using-superpowers` | Find and invoke skills at session start |
| `brainstorming` | Creative work before code |
| `engram-memory` | Always-active persistent memory (`mem_save`, `mem_search`, …) |
| `engram-memory-protocol` | When to save, search, and close sessions |

### Spec, IDD & BDD

| Skill | Role |
|-------|------|
| `idd-workflow` | Proposal → design → plan → promote under `docs/superpowers/` |
| `bdd-workflow` | Gherkin in `docs/superpowers/specs/*-design.md` |
| `gherkin-authoring` | Gherkin quality |
| `acceptance-test-authoring` | Extract `.feature`, cucumber runner |
| `bdd-git-discipline` | Git rules for spec vs code zones |
| `bdd-zone-check` | Zone-guard companion (`docs/` vs application code) |
| `glossary` | Domain terms in `docs/glossary/` |
| `architectural-decision-records` | ADRs in `docs/superpowers/adr/` |
| `c4-diagrams` | Architecture diagrams |
| `grilling` | Adversarial spec review before apply |

### Plan & execute

| Skill | Role |
|-------|------|
| `writing-plans` | Implementation plans from specs |
| `executing-plans` | Plan execution with checkpoints |
| `subagent-driven-development` | Parallel task execution |
| `dispatching-parallel-agents` | Independent parallel work |
| `using-git-worktrees` | Isolated feature branches |

### Implement & test

| Skill | Role |
|-------|------|
| `test-driven-development` | Strict TDD — sole owner for unit tests |
| `karpathy-guidelines` | Surgical changes, simplicity |
| `karpathy-coder` | Coding style |
| `systematic-debugging` | Bug investigation |

### Verify, review & ship

| Skill | Role |
|-------|------|
| `verification-before-completion` | Evidence before claiming done |
| `webapp-testing` | Playwright smoke after UI changes |
| `requesting-code-review` | Pre-merge review |
| `receiving-code-review` | Respond to review feedback |
| `finishing-a-development-branch` | Merge, PR, cleanup |

### Research, browser & UI

| Skill | Role |
|-------|------|
| `exa-search` | Web search |
| `mcp-deepwiki` | Repo wiki helper |
| `playwright-cli` | Browser automation CLI |
| `playwright-best-practices` | E2E patterns |
| `agent-browser` | Browser agent automation |
| `impeccable` | UI design: `DESIGN.md`, shape, audit, polish |

### Meta

| Skill | Role |
|-------|------|
| `writing-skills` | Author and test new skills |

---

### Rollout status

| Status | Skills |
|--------|--------|
| **Installed** | All superpowers (14), `engram-memory`, `engram-memory-protocol`, `exa-search`, `karpathy-coder`, `mcp-deepwiki`, `playwright-cli`, `playwright-best-practices`, `agent-browser`, partial local pack (`bdd-workflow`, `gherkin-authoring`, `acceptance-test-authoring`, `bdd-git-discipline`, `glossary`, `grilling`, `webapp-testing`, `karpathy-guidelines`) |
| **Planned** | `idd-workflow`, `bdd-zone-check`, `architectural-decision-records`, `c4-diagrams`, `impeccable` |
| **Remove** | `engram-testing-coverage` (duplicate TDD owner) |

---

### Opt-in (not in default list)

| Skill | When |
|-------|------|
| `engram-docs-alignment` | Doc-heavy repos |
| `engram-backlog-triage` | GitHub maintainer triage |

---

### Not in end list

| Skill | Why |
|-------|-----|
| `engram-testing-coverage` | Duplicates `test-driven-development` |
| `engram-sdd-flow` | Duplicates `idd-workflow` + `writing-plans` |
| `engram-branch-pr`, `engram-issue-creation`, `engram-commit-hygiene` | Engram GitHub-only workflow |
| `engram-pr-review-deep` | Duplicates `requesting-code-review` |
| Other Engram product skills | Wrong scope (Engram repo internals, TUI, htmx dashboard) |
| OpenSpec skills | Not used |
| `taste-skill` | Overlaps `impeccable` |

---

## Install mechanics

```bash
~/.skillgrid/npm/node_modules/.bin/skills add <repo> --agent <agent> -g -s <skill> -y
```

Skill add failures **warn and continue**. Re-run `skillgrid install` to reconcile after `skills.yaml` changes.

---

## Related

- [02-usage](02-usage.md) — workflow command order
- [Engram skills catalog](https://github.com/Gentleman-Programming/engram/blob/main/skills/catalog.md) — full upstream list (most excluded)

*Last updated: 2026-08-26.*
