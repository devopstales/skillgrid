# Intent: 006 — Structured Session Handoff (Session Relay)

## Classification
**Architectural** — new **Session Relay** subsystem (schema, MCP, `.cleave/` FS, CLI, optional watchdog).

## Business Problem
Context fill / session end breaks continuity. Need Cleave-style **Session Handoff** so agents resume without re-deriving state (scoped slice of plan 06 — not the mega-plan).

## Target Users & Situations
- **Agent** — MCP `session_handoff` / `session_resume` when context is tight or work spans sessions.
- **Operator** — CLI handoff/resume/status; local `.cleave/` inspect.
- Urgency: Medium–High after **003** paths; orthogonal to **004**/**005**.

## Business Rules
- **Session Relay** only — not **Fact Memory** / **Agent Skills** (**004**), not **003** **Tiered Storage** / `semantic_search` / `mnemonic_commit` / trail CLI, not **002**/**005**, not OpenCode plugin.
- Additive SQL: `session_handoffs`, `session_archives`; Go MCP + SQLite + FS.
- Writes `.skillgrid/.cleave/` (`PROGRESS.md`, `KNOWLEDGE.md`, `NEXT_PROMPT.md`); **gitignored** by default (like L0).
- Soft-after **003**: reuse L0 paths; no full tiered storage or `mnemonic_commit`.
- Triggers: explicit MCP + CLI; optional context-limit watchdog in a later **Step** of this **Change**.
- Thin `knowledge_compact`: refresh `.cleave/KNOWLEDGE.md` only — no **Fact Memory** dependency.
- Complements Engram `mem_session_*`; does not replace them. Do not stretch **Long-term Memory**.

## Success Criteria (UAT-level)
- [ ] `session_handoff` writes three `.cleave/` files and records `session_handoffs`.
- [ ] `session_resume` returns a next-session prompt (optional archive).
- [ ] `session_status` reports session stats (cost/context/handoff count).
- [ ] Thin `knowledge_compact` refreshes `KNOWLEDGE.md` without facts.
- [ ] CLI: `skillgrid session handoff|resume|status`.
- [ ] Optional watchdog triggers handoff past a configured limit.
- [ ] `.cleave/` gitignored by default; `go test ./...` passes for touched packages.

## Scope

### In Scope
- Schema: `session_handoffs`, `session_archives`.
- MCP: `session_handoff`, `session_resume`, `session_status`; thin `knowledge_compact`.
- FS: `.skillgrid/.cleave/` + L0 session workspace as needed.
- Optional handoff watchdog; CLI `skillgrid session handoff|resume|status`.

### Out of Scope
- **004** facts/skills; **003** tiered/semantic/commit/trail cores.
- **002** identity; **005** code intelligence; OpenCode plugin; plan-06 rewrite.

## Step Blueprint
- `01-relay-schema`: Migrations for handoffs/archives.
- `02-handoff-resume`: `.cleave/` + MCP handoff/resume; L0 paths as needed.
- `03-status-compact`: `session_status` + thin `knowledge_compact`.
- `04-session-cli`: `skillgrid session handoff|resume|status`.
- `05-handoff-watchdog`: Optional context-limit watchdog → `session_handoff`.

## Affected Areas
| Area | Impact | Description |
|------|--------|-------------|
| `…/mnemonic/store/migrations/` | New | handoffs, archives |
| `…/mnemonic/` (relay/cleave) | New | Handoff FS + compact |
| `…/mnemonic/mcp/` | Modified | session_* (+ compact) |
| `skillgrid-cli/` CLI | Modified | `session` subcommands |
| `.gitignore` / templates | Modified | Ignore `.cleave/` |

## Risks
| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Scope creep into plan-06 | High | Hard Out of Scope |
| Over-coupling to **003** | Med | Soft-after; L0 only |
| Watchdog false triggers | Med | Flag; threshold; last Step |
| Thin compact vs expectations | Med | Document: compounding file only |

## Rollback Plan
Drop new migrations, tools, CLI, watchdog, `.cleave/` helpers; leave **003**/**004**.

## Dependencies
- Soft-after **003** (L0 paths). Source: `docs/plan/06-structured-session-handof.md`. Approved 2026-09-04.
