# Mnemonic Artifact Convention (shared)

NOTE: Critical Mnemonic calls (`mem_search`, `mem_save`, `mem_get_observation`) are inlined directly in each skill's SKILL.md. This document is supplementary reference — sub-agents do NOT need to read it to function.

Applies to every Skillgrid SDD skill. Memory mode is `hybrid` always: the same artifact goes to both Mnemonic and the filesystem (openspec / docs). Mnemonic is recovery working memory, not the audit trail.

## Mnemonic Tool Mapping

Mnemonic (skillgrid) differs from Engram — adapt calls to this:

| Engram convention | Mnemonic equivalent |
|---|---|
| `mem_save(..., project: X)` | no `project` param — `scope: "project"`, name goes in `topic_key` |
| `capture_prompt: false` | **not a parameter** — omit; never pass unknown fields |
| `mem_update(id, ...)` | upsert via same `topic_key`; or `mem_update` only if tool exposed |
| `ENGRAM_PROJECT` / auto-detect | Mnemonic scopes to the workspace; pass `session_id` from `mem_session_start` |
| `mem_review` lifecycle | `mem_session_set_title` for legibility in the dashboard; no lifecycle gating |

Session setup (once per agent session, before first save):

```
sid = mem_session_start(title: "sdd/{project}/{change or phase}")
```

Reuse `sid` for every `mem_save` in that session. `mem_save` requires `session_id` — a save without it fails.

## Naming Rules

ALL SDD artifacts persisted to Mnemonic MUST follow this deterministic naming:

```
title:     sdd/{change-name}/{artifact-type}
topic_key: sdd/{change-name}/{artifact-type}
type:      architecture
scope:     project
session_id: {active sdd session}
```

Project-level facts (no change scope) use:

```
title/topic_key: sdd/{project}/{fact}   (issue_tracker, testing-capabilities, tech_stack)
title/topic_key: sdd/{project}/project_name
title/topic_key: sdd-init/{project}     (init-time full project context)
```

`title` and `topic_key` must be identical — exact-match recovery depends on it.

### Artifact Types

| Artifact Type | Produced By | `type` |
|---|---|---|
| `explore` | sdd-explore | architecture |
| `proposal` | sdd-propose | architecture |
| `design` | sdd-design | architecture |
| `spec` | sdd-spec | architecture |
| `tasks` | sdd-tasks | architecture |
| `issue-creation` | sdd-issue-creation | config |
| `apply-progress` | sdd-apply (one per batch) | architecture |
| `verify-report` | sdd-verify | architecture |
| `archive-report` | sdd-archive (lineage: all obs IDs) | architecture |
| `state` | orchestrator (DAG state for recovery) | architecture |
| `tech_stack` | sdd-init | config |
| `issue_tracker` | sdd-init | config |
| `testing-capabilities` | sdd-init | config |
| `skill-registry` | sdd-init (topic `skill-registry`, global project scope) | config |

### State Artifact

```
mem_save(
  title: "sdd/{change-name}/state",
  topic_key: "sdd/{change-name}/state",
  type: "architecture",
  scope: "project",
  session_id: {sid},
  content: "change: {change-name}\nphase: {last-phase}\nartifact_store: hybrid\nartifacts:\n  proposal: true\n  specs: true\n  design: false\n  tasks: false\ntasks_progress:\n  completed: []\n  pending: []\nlast_updated: {ISO date}"
)
```

Recovery: `mem_search("sdd/{change-name}/state")` → `mem_get_observation(id)` → parse → restore state.

## Recovery Protocol (2 steps)

```
Step 1: mem_search(query: "sdd/{change-name}/{artifact-type}") → truncated preview + ID
Step 2: mem_get_observation(id: {observation-id}) → complete content
```

Search previews are always truncated; `mem_get_observation` is the only way to get full content. When retrieving multiple artifacts, group all searches first, then all retrievals:

```
STEP A — SEARCH (get IDs only):
  mem_search(query: "sdd/{change-name}/proposal") → save ID
  mem_search(query: "sdd/{change-name}/spec") → save ID
  mem_search(query: "sdd/{change-name}/design") → save ID

STEP B — RETRIEVE FULL CONTENT (mandatory):
  mem_get_observation(id: {proposal_id})
  mem_get_observation(id: {spec_id})
  mem_get_observation(id: {design_id})
```

Loading project context:

```
mem_search(query: "sdd-init/{project}") → get ID
mem_get_observation(id) → full project context
```

Browsing all artifacts for a change: `mem_search(query: "sdd/{change-name}/")`.

At session start, before asking the user "what were we doing": `mem_context` → if empty, `mem_search` with phase keywords. Never ask what you can search.

## Writing Artifacts

Standard write:

```
mem_save(
  title: "sdd/{change-name}/{artifact-type}",
  topic_key: "sdd/{change-name}/{artifact-type}",
  type: "architecture",
  scope: "project",
  session_id: {sid},
  content: "{full markdown content}"
)
```

Concrete example — saving a proposal for `add-dark-mode`:

```
mem_save(
  title: "sdd/add-dark-mode/proposal",
  topic_key: "sdd/add-dark-mode/proposal",
  type: "architecture",
  scope: "project",
  session_id: "ses-abc123",
  content: "## Proposal\n\nAdd dark mode toggle..."
)
```

Upserts: same `topic_key` + `scope` → UPDATE (overwrite), not INSERT. Previous content is lost — Mnemonic is working memory, not an audit trail. For iteration history, the filesystem copy (openspec / docs) is the source of truth. Reuse `topic_key` for evolving topics instead of creating near-duplicates; never mix different topics under one key.

## Session Close Protocol

Before ending a phase, session, or saying "done":

1. `mem_session_summary(session_id: {sid}, summary: "## Goal … ## Accomplished … ## Next Steps … ## Relevant Files …")` — required structure.
2. `mem_session_end(session_id: {sid}, summary: "one-line outcome")`.

After compaction / resumed session: FIRST call `mem_session_summary` with the compacted content, then `mem_context` to recover prior context.

## Why This Convention

- Deterministic `title` == `topic_key` → recovery works by exact match
- `sdd/` prefix → namespaces all SDD artifacts per change
- Two-step recovery → `mem_get_observation` is the only full-content path
- Lineage → `archive-report` content lists all observation IDs for complete traceability
- Hybrid → filesystem survives Mnemonic wipes; Mnemonic survives branch switches and /clear
