# Skill registry index at `.skillgrid/projectwiki/skill-registry.md`

---
status: accepted
date: 2026-11-25
supersedes: none
---

## Context and Problem Statement

The 2026-11-25 study of `../gentle-ai/` (tracked for the envelope port in
`change-specs/2026-11-23-pipeline-envelope/proposal.md`) identified the skill
registry pattern: a disk index of skill name, full description, scope, and the
exact `SKILL.md` path; the orchestrator passes subagents the **exact paths**
and they read the full skill before work — "delegators pass paths, not digested
rules" (gentle-ai `docs/skill-registry.md`). Skillgrid adopts the pattern for
its own skill family (subagent handoffs by reference, never by paraphrase).

Gentle-ai writes its index to `.atl/skill-registry.md`. That path is a
decision for this repo: where the index file lives.

## Considered Options

- **`.atl/skill-registry.md`** (gentle-ai's location verbatim): `.atl/` is a
  managed directory owned by the gentle-ai binary (its `install`/`sync`
  project into it, with its own gitignore handling). Reusing it here would
  import another product's managed root for an artifact skillgrid's tooling
  owns.
- **`.skillgrid/projectwiki/skill-registry.md`**: durable project knowledge
  already roots in `.skillgrid/projectwiki/` (`conventions` in
  `.skillgrid/config.yaml`; the vault layout). The registry is a catalog of
  the skill family — project knowledge, not per-change state
  (`change-specs/`) and not scratch (`work/`).
- **No index file** (harness-native discovery only): status quo; subagent
  handoffs stay name-based and depend on the harness resolving the skill.

## Decision Outcome

Chosen option: **`.skillgrid/projectwiki/skill-registry.md`** — the skill
registry index (name, full description, scope, exact `SKILL.md` path) is a
derived file in the projectwiki vault. The path is fixed now so the generator
(later change) and the skill prose agree on one location.

### Consequences

- Good: one managed root (`.skillgrid/`) — the registry sits beside
  `backlog/`, `decisions/`, `glossary/` instead of opening a second root.
- Good: no coupling to gentle-ai's `.atl/` managed-directory convention.
- Bad: the file is generated, not hand-maintained; the vault's "every page
  gets an index row" convention applies once the generator lands (the row is
  added in that change, not now).
- Out of scope here: the refresh/generator (byte-compare on re-scan, like
  gentle-ai's `skill-registry refresh`) and the handoff contract (parent lists
  exact `SKILL.md` + `references/...` paths; subagent reads them first). Both
  belong to the implementing change.

## Not adopted

- **`.atl/skill-registry.md`** — another product's managed directory; skillgrid
  already keeps every durable artifact under `.skillgrid/`.

## Related

- Supersedes: none.
- Depends on: the projectwiki vault layout (`config.yaml` `conventions`).
- Consumed by: the skill registry generator change; subagent handoff contract
  in `execute`/`slicing` skill prose.
- Context: `change-specs/2026-11-23-pipeline-envelope/proposal.md` (the
  gentle-ai study), 2026-11-25 gentle-ai study (registry learning item).
