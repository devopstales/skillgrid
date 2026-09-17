---
id: 0001
title: Web Admin Dashboard — PRD
type: prd
created_date: '2026-09-08'
updated_date: '2026-09-17'
---
# Web Admin Dashboard — PRD

## 1. Problem

`skillgrid serve` already runs a full HTTP API (`:7438`) and embeds a minimal
vanilla data viewer. The operator pain is real: today you switch between the CLI,
MCP tools, the backlog CLI, and a browser to inspect one machine's memory, code
index, tasks, and sessions. A vanilla build-less SPA hit its ceiling — a vector
graph of thousands of nodes, rendered Mermaid docs, a drag-and-drop multi-provider
Kanban, and live SSE streams want a real component framework.

The bottleneck in the AI era is **attention, not code volume**: an operator can
read a screenful of task specs, a decision record, or a plan before any code
exists, but cannot meaningfully review 15,000 generated lines in one sitting.

## 2. Goals

- **One dashboard, one process.** `skillgrid serve` at `GET /` administers every
  surface from a single embedded Vite/React SPA — no separate frontend process,
  no external CDN (the binary works offline).
- **Review the work, not just the code.** First-class views for the SDD artifacts
  that make agent work reviewable: tasks (with acceptance criteria + Definition of
  Done), decisions (ADRs with a status lifecycle), and plans/specs — rendered,
  linked, and searchable from the browser.
- **Any tracker, one board.** A pluggable Go provider registry drives a Kanban over
  Backlog.md / GitHub / GitLab / Jira, with provider-native transitions and honest
  degraded states.
- **Read-mostly, safe mutations.** Mutations are limited to tracker status change
  and (later) memory pin/unpin + 013 governance changes — all token-gated.

## 3. Non-Goals

- MCP surface is frozen — tool names, signatures, and return shapes are unchanged;
  this change adds HTTP routes only.
- No login UI / sessions / cookies — writes are gated on the existing
  `SKILLGRID_HTTP_TOKEN` bearer.
- No CORS / remote hosting / non-`127.0.0.1` serving by default.
- No in-browser LLM/embedding inference.
- D3.js for the main vector graph (Sigma.js + graphology is the main graph).

## 4. Target User

The operator of a skillgrid-installed machine — daily driver: "what did the agents
remember / index / commit to the backlog, and can I clean it up?" — now with a
visual graph of memory, rendered docs + decisions, and a live activity stream.

## 5. Solution Overview

A Vite + React 19 + TypeScript SPA in `skillgrid-ui/` builds into
`skillgrid-cli/internal/mnemonic/http/ui/dist/` and is embedded with
`//go:embed all:ui/dist`. The Go server gains route groups:

| View | Routes | Status |
|------|--------|--------|
| Tracker Kanban | `/tracker/providers`, `/tracker/tasks`, `/tracker/stream` (SSE) | Phase 2 |
| Docs (rendered markdown + Mermaid) | `/docs/tree`, `/docs/content`, `/docs/search`, `/docs/render` | Phase 3 |
| Mnemonic graph (Sigma.js + graphology) | `/mnemonic/graph`, `/mnemonic/graph/nodes` | Phase 4 |
| Files / Memories / Sessions | `/mnemonic/files/*`, `/mnemonic/memories/*`, `/mnemonic/sessions`, `/mnemonic/audit` | Phase 5 |
| Activity / Plans / Git | `/activity/*`, `/plans/*`, `/git/*` | Phase 6 |
| Prototypes | `/prototypes/*` | Phase 7 |

### Docs viewer (the review surface)

The Docs view renders repo markdown with Mermaid across the SDD methods the repo
uses, with **existence-gated, generic roots** (see `adr-001`):

- **Skillgrid Specs** — `.skillgrid/specs/<change>/{briefing,tasks,acceptance}.md`
- **OpenSpec** — `openspec/specs`, `openspec/changes`
- **SpecKit** — `specs/<NNN-feature>/{spec,plan,tasks}.md`, `.specify/memory`
- **Superpowers** — `docs/superpowers/plans`, `docs/superpowers/specs`
- **Backlog** — `.backlog/tasks/*.md`
- **Backlog Docs & Decisions** — `.backlog/docs/**` (doc-NNN/PRD), `.backlog/decisions/**` (ADR)
- **Docs** — `docs/**` (human docs)

Schema-aware rendering: ADRs render Context/Decision/Consequences with a status
lifecycle chip (proposed → accepted → rejected → superseded); PRDs and doc-NNN
render with type chips; tasks render with acceptance-criteria and DoD checklists.

## 6. Review Checkpoints (the "why" of the task-centric design)

1. **Review the spec** — the agent decomposes an idea into tasks with descriptions,
   acceptance criteria, and milestones *before* implementation.
2. **Review the plan** — the agent writes its implementation plan into the task;
   approve or steer before code is written.
3. **Review the code** — one task = one context window = one PR; diffs stay a size
   a human can read.

Completed tasks + decisions remain in git as a permanent, legible record of what
was attempted and why.

## 7. Success Criteria

- `skillgrid serve` from any repo shows the methods that repo actually uses; absent
  methods show a clean empty state, not an error.
- An ADR renders its status lifecycle and Context/Decision/Consequences; a PRD and
  a task render with their schema-appropriate sections and chips.
- Path traversal on `/docs/content?path=...` is blocked (`..`/absolute → 400,
  unknown → 404); Mermaid + markdown are XSS-guarded (strict + sanitize + DOMPurify).
- The Kanban round-trips Backlog.md and degrades honestly for absent CLIs.
- `go test ./...` + `go vet ./...` green; SPA `tsc --noEmit` + `npm run build` green.

## 8. Open Questions

- Phase 6 "View plan progress →" deep-links into a Plans view that reuses the Docs
  renderer; the cross-link target lands in Phase 6.
- 005/008 (graph edge/community data) and 013 (memory governance) are soft
  dependencies — views render forward-compat placeholders until they land.

## References

- Change: `2026-09-08-web-admin-dashboard` (briefing + tasks + acceptance)
- Decision: `adr-001` (SDD multi-method docs viewer)
- Research: MrLesk/Backlog.md backlog schema (tasks/docs/decisions, ADR pattern),
  github/spec-kit artifact layout, obra/superpowers plan/spec layout.
