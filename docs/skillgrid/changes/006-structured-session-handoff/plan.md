# Plan: 006-structured-session-handoff — Structured Session Handoff (Session Relay)

## Technical Approach
Ship Cleave-style **Session Relay** only: additive SQL, `.skillgrid/.cleave/` FS, MCP `session_*` + thin `knowledge_compact`, CLI `skillgrid session …`, optional flag-gated watchdog. Soft-after **003** L0 paths (`.skillgrid/workspace/sessions/{id}/`) without **Tiered Storage**, `semantic_search`, `mnemonic_commit`, or trail CLI. Orthogonal to **004**/**005**; leave `009_*`/`010_*`/`011_*` for 001/003/004 — this **Change** uses `012_*`. Complements Engram `mem_session_*`; does not replace them or stretch **Long-term Memory**. Source: `docs/plan/06-structured-session-handof.md` §2.4 / §3.3.

## Architecture Decisions

### Decision: Deep Session Relay Module
**Module / Interface / Seam / Adapter / Depth**: `relay`; Handoff/Resume/Status/Compact; SQL+FS **Seam**; SQLite + cleave FS (+ optional L0); deep.
**Choice**: One `relay` package owns schema writes and `.cleave/` files; MCP/CLI are thin callers.
**Alternatives considered**: Logic inside MCP handlers; fold into `memory.Service`.
**Rationale**: Deletion test — callers stay shallow; tests hit `relay` without MCP framing.

### Decision: Soft L0 paths, hard `.cleave/`
**Module / Interface / Seam / Adapter / Depth**: cleave FS; WriteBundle/ReadBundle; path **Seam**; `.cleave/` required, L0 optional; deep.
**Choice**: Always write `PROGRESS.md` / `KNOWLEDGE.md` / `NEXT_PROMPT.md` under `.skillgrid/.cleave/`; optionally mirror scratch under L0 when present.
**Alternatives considered**: L0-only; hard-require **003** tables.
**Rationale**: Approved soft-after **003**; relay works before tiered core lands.

### Decision: Thin knowledge_compact
**Module / Interface / Seam / Adapter / Depth**: Compact; RefreshKnowledge; compounding **Seam**; FS-only adapter; deep for callers.
**Choice**: Refresh `.cleave/KNOWLEDGE.md` from handoff inputs/session notes only — no **Fact Memory**.
**Alternatives considered**: Full plan-06 compact (facts + log roll-off); stub no-op.
**Rationale**: Locked Q2; documents compounding file only.

### Decision: Watchdog behind same Handoff Interface
**Module / Interface / Seam / Adapter / Depth**: Watchdog; Check→Handoff; threshold **Seam**; off-by-default adapter; deep.
**Choice**: Optional context-limit watchdog calls the same `Handoff` path; env/flag + threshold.
**Alternatives considered**: Always-on; separate MCP tool.
**Rationale**: Q1 explicit+watchdog; false-trigger risk → last **Step**, default off.

### Decision: Migration slot `012_*`
**Choice**: `012_session_relay.sql` for `session_handoffs` + `session_archives`.
**Alternatives considered**: `009` (collides 001); `011` (004).
**Rationale**: Matches 003/004 reservation chain; additive, idempotent open.

## Data Flow

```
session_handoff ──▶ relay.Handoff ──▶ .cleave/{PROGRESS,KNOWLEDGE,NEXT_PROMPT}.md
                              └──▶ session_handoffs row (+ optional L0 scratch)
session_resume ──▶ read .cleave + row ──▶ next-session prompt (+ optional archive)
session_status ──▶ handoff count + last cost/context (caller stats when supplied)
knowledge_compact ──▶ rewrite KNOWLEDGE.md only
watchdog (opt) ──▶ threshold ──▶ same Handoff
CLI session * ──▶ same relay Module
```

## Impacted Files Map
| File | Action | Step | Description |
|------|--------|------|-------------|
| `skillgrid-cli/internal/mnemonic/store/migrations/012_session_relay.sql` | Create | 01 | `session_handoffs`, `session_archives` |
| `skillgrid-cli/internal/mnemonic/relay/relay.go` | Create | 02 | Handoff/Resume **Module** + SQL |
| `skillgrid-cli/internal/mnemonic/relay/cleave.go` | Create | 02 | `.cleave/` FS; soft L0 paths |
| `skillgrid-cli/internal/mnemonic/mcp/tools_session_handoff.go` | Create | 02 | `session_handoff`, `session_resume` + registrar hook |
| `skillgrid-cli/internal/mnemonic/mcp/server.go` | Modify | 02 | Call `registerSessionTools` |
| `.gitignore` | Modify | 02 | Ignore `.skillgrid/.cleave/` |
| `skillgrid-cli/internal/mnemonic/relay/status.go` | Create | 03 | Status aggregation |
| `skillgrid-cli/internal/mnemonic/relay/compact.go` | Create | 03 | Thin `KNOWLEDGE.md` refresh |
| `skillgrid-cli/internal/mnemonic/mcp/tools_session_status.go` | Create | 03 | `session_status`, `knowledge_compact` via registrar hook |
| `skillgrid-cli/cmd/skillgrid/session.go` | Create | 04 | `session handoff\|resume\|status` |
| `skillgrid-cli/cmd/skillgrid/main.go` | Modify | 04 | Dispatch `session` |
| `skillgrid-cli/internal/mnemonic/relay/watchdog.go` | Create | 05 | Flag-gated context-limit → Handoff |
| `skillgrid-cli/internal/mnemonic/mcp/server_test.go` | Modify | 03 | Expect new session_* (+ compact) tools |

## Step WHAT

### Step 01-relay-schema — What it delivers
- As operator: opening a store creates handoff/archive tables without rewriting sessions/observations.
- Given an existing DB through `008_*`, migrate leaves prior rows intact.
- Edge: re-open is idempotent; no duplicate tables.

### Step 02-handoff-resume — What it delivers
- As agent: `session_handoff` writes the three `.cleave/` files and records a **Session Handoff** row.
- As agent: `session_resume` returns a next-session prompt from the latest (or named) handoff; optional archive when requested.
- Edge: missing `.cleave/` or unknown id → clear error; fail closed (no orphan row without files).

### Step 03-status-compact — What it delivers
- As agent: `session_status` reports handoff count and last known cost/context for the session.
- As agent: thin `knowledge_compact` refreshes `KNOWLEDGE.md` without **Fact Memory**.
- Edge: no handoffs yet → zero counts / empty knowledge file, not a crash.

### Step 04-session-cli — What it delivers
- As operator: `skillgrid session handoff|resume|status` mirrors MCP outcomes on the same project store.
- Edge: bad flags / missing id → non-zero exit with stderr message.

### Step 05-handoff-watchdog — What it delivers
- As operator: with watchdog enabled and usage past threshold, a **Session Handoff** is triggered via the same path as explicit handoff.
- Edge: disabled/default → never auto-handoff; below threshold → no-op.

## Interfaces / Contracts

```go
type Relay interface {
  Handoff(ctx context.Context, in HandoffInput) (HandoffResult, error)
  Resume(ctx context.Context, in ResumeInput) (ResumeResult, error)
  Status(ctx context.Context, sessionID string) (StatusResult, error)
  CompactKnowledge(ctx context.Context, in CompactInput) error
}
// session_handoff → {handoff_id, paths:{progress,knowledge,next_prompt}}
// session_resume → {prompt, handoff_id, archive_id?}
// session_status → {handoff_count, context_usage_percent?, cost_usd?}
// knowledge_compact → {knowledge_path}
// Registrar hook: step 03 appends tools without re-editing server.go
```

## Mnemonic Integration
New tools: `session_handoff`, `session_resume`, `session_status`, `knowledge_compact`. Existing `mem_*` / `code_*` / `web_*` unchanged. Topic: `sdd/006-structured-session-handoff/plan`.

## Threat Matrix

| Boundary | Applicability | Design response | Planned RED tests | Owner step |
|---|---|---|---|---|
| Documentation-like paths | N/A: writes markdown under `.cleave/` only; no executable classification | — | — | — |
| Git repository selection | N/A: `.gitignore` edit only; no `git -C` / cwd authority | — | — | — |
| Commit state | N/A: no commit automation | — | — | — |
| Push state | N/A: no push | — | — | — |
| PR commands | N/A: no PR CLI | — | — | — |
| **Mnemonic tool surface** | Applicable — new session_*/compact tools | Additive; JSON-only; `mem_*` still registered | New tools present; `mem_save` still works; compact does not require facts | 02, 03 |
| **Shared-convention drift** | N/A: no `_shared/conventions/*` edits in this plan | — | — | — |

## Migration / Rollout
Additive `012_session_relay.sql`. Soft-after **003** paths; no hard SQL dep on `010_*`. Watchdog off by default (`SKILLGRID_HANDOFF_WATCHDOG` + threshold). Rollback: drop `012` tables/tools/CLI/watchdog/`.cleave/` helpers.

## Open Questions
- [x] Tracker: `TASK-002` created via `backlog task create` (`.backlog/tasks/task-002 - FEATURE-SDD-plan-for-006-structured-session-handoff-mnemonic.md`).
- [ ] Watchdog usage signal (client `%` vs token estimate) — pick in step 05; **Interface** takes a fraction either way.
