# based on skillgrid-v2:_shared/conventions/mnemonic-memory.md

# Mnemonic Artifact Convention

NOTE: Critical Mnemonic calls (`mem_search`, `mem_save`, `mem_get_observation`) are inlined directly in each skill's SKILL.md. This document is supplementary reference — sub-agents do NOT need to read it to function.

Applies to every Skillgrid skill. Memory mode is `hybrid` always: the same artifact goes to both Mnemonic and the filesystem (`.skillgrid/specs/YYYY-MM-DD-<topic>/`). Mnemonic is recovery working memory, not the audit trail.

## Mnemonic Tool Mapping

Mnemonic (skillgrid) local conventions:

Session setup (once per agent session, before first save):

```
sid = mem_session_start(title: "skillgrid/{YYYY-MM-DD-<topic>}/{phase}")
```

Reuse `sid` for every `mem_save` in that session. `mem_save` requires `session_id` — a save without it fails.

## Naming Rules

ALL Skillgrid artifacts persisted to Mnemonic MUST follow this deterministic naming:

```
title:     skillgrid/{YYYY-MM-DD-<topic>}/{artifact-type}
topic_key: skillgrid/{YYYY-MM-DD-<topic>}/{artifact-type}
type:      architecture
scope:     project
session_id: {active skillgrid session}
```

where `{YYYY-MM-DD-<topic>}` is the date-prefixed topic identifier (e.g. `2026-09-13-oauth-login`), created by `brainstorming`.

Project-level facts (no change scope) use:

```
title/topic_key: skillgrid/{project}/{fact}   (issue_tracker, testing-capabilities, tech_stack)
title/topic_key: skillgrid/{project}/project_name
title/topic_key: skillgrid-init/{project}     (init-time full project context)
```

`title` and `topic_key` must be identical — exact-match recovery depends on it.

### Artifact Types

| Artifact Type | Produced By | `type` |
|---|---|---|
| `briefing` | brainstorming | architecture |
| `blueprint` | writing-blueprints | architecture |
| `tasks` | slicing | architecture |
| `spec` (per `acceptance.feature`, concatenated) | slicing | architecture |
| `ticketing` | ticketing | config |
| `findings` (research/spike/sketch evidence) | research / spike / sketch | architecture |
| `execution-progress` (cumulative, per change) | simple-execution / subagent-execution (one per batch) | architecture |
| `qa-report` (test plan + verdict + evidence) | qa | architecture |
| `ship` (integration + move context) | ship | architecture |
| `report` (final close: integration + retro + lineage) | reflect | learning |
| `changelog` (topic reservations + archives) | brainstorming / (archive phase) | config |
| `tech_stack` | onboarding | config |
| `issue_tracker` | onboarding | config |
| `testing-capabilities` | onboarding | config |
| `skill-registry` | onboarding (topic `skill-registry`, global project scope) | config |

### Briefing Artifact (planning-phase state)

The in-repo `state.md` is gone; the planning-phase position is mirrored under the `briefing` slot. The spec-zone artifacts (which files exist) are the primary phase signal; this observation is a fallback index.

```
mem_save(
  title: "skillgrid/{YYYY-MM-DD-<topic>}/briefing",
  topic_key: "skillgrid/{YYYY-MM-DD-<topic>}/briefing",
  type: "architecture",
  scope: "project",
  session_id: {sid},
  content: "change: <YYYY-MM-DD-<topic>>\nphase: {last-phase}\nartifact_store: hybrid\nartifacts:\n  briefing: true\n  blueprint: true\n  tasks: true\n  spec: true\n  review: false\nsteps_progress:\n  01-migration: {n}/{n} [x]\n  02-api: {n}/{n}\nlast_updated: {ISO date}"
)
```

Recovery: `mem_search("skillgrid/{YYYY-MM-DD-<topic>}/briefing")` → `mem_get_observation(id)` → parse → restore planning state. For execution-phase position, use the `execution-progress` slot + `checkpoint.json`.

## Recovery Protocol (2 steps)

```
Step 1: mem_search(query: "skillgrid/{YYYY-MM-DD-<topic>}/{artifact-type}") → truncated preview + ID
Step 2: mem_get_observation(id: {observation-id}) → complete content
```

Search previews are always truncated; `mem_get_observation` is the only way to get full content. When retrieving multiple artifacts, group all searches first, then all retrievals:

```
STEP A — SEARCH (get IDs only):
  mem_search(query: "skillgrid/{YYYY-MM-DD-<topic>}/briefing")   → save ID
  mem_search(query: "skillgrid/{YYYY-MM-DD-<topic>}/blueprint")  → save ID
  mem_search(query: "skillgrid/{YYYY-MM-DD-<topic>}/spec")       → save ID

STEP B — RETRIEVE FULL CONTENT (mandatory):
  mem_get_observation(id: {briefing_id})
  mem_get_observation(id: {blueprint_id})
  mem_get_observation(id: {spec_id})
```

Loading project context:

```
mem_search(query: "skillgrid-init/{project}") → get ID
mem_get_observation(id) → full project context
```

Browsing all artifacts for a change: `mem_search(query: "skillgrid/{YYYY-MM-DD-<topic>}/")`.

At session start, before asking the user "what were we doing": `mem_context` → if empty, `mem_search` with phase keywords. Never ask what you can search.

## Writing Artifacts

Standard write:

```
mem_save(
  title: "skillgrid/{YYYY-MM-DD-<topic>}/{artifact-type}",
  topic_key: "skillgrid/{YYYY-MM-DD-<topic>}/{artifact-type}",
  type: "architecture",
  scope: "project",
  session_id: {sid},
  content: "{full markdown content}"
)
```

Concrete example — saving a briefing for `2026-09-13-oauth-login`:

```
mem_save(
  title: "skillgrid/2026-09-13-oauth-login/briefing",
  topic_key: "skillgrid/2026-09-13-oauth-login/briefing",
  type: "architecture",
  scope: "project",
  session_id: "ses-abc123",
  content: "## Briefing\n\nAdd OAuth login for existing users..."
)
```

Upserts: same `topic_key` + `scope` → UPDATE (overwrite), not INSERT. Previous content is lost — Mnemonic is working memory, not an audit trail. For iteration history, the filesystem copy (`.skillgrid/specs/YYYY-MM-DD-<topic>/`) is the source of truth. Reuse `topic_key` for evolving topics instead of creating near-duplicates; never mix different topics under one key.

## Session Close Protocol

Before ending a phase, session, or saying "done":

1. `mem_session_summary(session_id: {sid}, summary: "## Goal … ## Accomplished … ## Next Steps … ## Relevant Files …")` — required structure.
2. `mem_session_end(session_id: {sid}, summary: "one-line outcome")`.

After compaction / resumed session: FIRST call `mem_session_summary` with the compacted content, then `mem_context` to recover prior context.

## Operator export and visualization

Read-only observation surfaces (not Skillgrid write authority):

- `skillgrid export --project ID --out DIR` — Obsidian Markdown + viz JSON under an allowed root
- **Memory Visualization** in the `skillgrid serve` dashboard webui — browse observations; mutate/delete via viz is rejected
- **Code Graph** in the same dashboard — callers | source | callees explorers over Edges; graph viz mutate is rejected

For code retrieval (Search Intent Router, Index Freshness, Orientation Ladder), see [code-indexing.md](code-indexing.md).

## Why This Convention

- Deterministic `title` == `topic_key` → recovery works by exact match
- `skillgrid/` prefix → namespaces all Skillgrid artifacts per change
- Two-step recovery → `mem_get_observation` is the only full-content path
- Lineage → `report` content lists all observation IDs for complete traceability
- Hybrid → filesystem survives Mnemonic wipes; Mnemonic survives branch switches and /clear
