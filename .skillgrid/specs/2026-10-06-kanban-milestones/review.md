# Review — 2026-10-06-kanban-milestones

> Change: `.skillgrid/specs/2026-10-06-kanban-milestones/` (moves to `.skillgrid/archive/2026-10-06-kanban-milestones/` at ship)
> Generated: 2026-10-06T21:20:00Z (requesting-code-review)
> Diff range: `44eb8a4d..69bbbe62` (the 4 kanban commits: 267bfffb, b5de2639, 50a27ad1, 22d7031b)
> Rigor tier: T2 (per `_shared/planning/rigor-tiers.md`) — three-axis single-pass
> Independence: Grade B per axis — see `## Independence`
>
> The durable audit record of the code review. **requesting-code-review** writes this into
> the spec folder, next to `report.md` (which owns the QA gate). It is committed in the spec
> zone before code-zone work continues. **ship** archives it with the folder; **reflect**
> cites it for lineage. The three review axes are reported **side by side, never merged** —
> that separation is the point.
>
> NOTE on paths: the 4 kanban commits land on the OLD module layout (`skillgrid-cli/internal/mnemonic/...`); a later unrelated restructure commit moved them to `mnemonic/internal/...`. Findings below cite the CURRENT path.

## What Important means

> The explicit threshold that separates a finding worth fixing now from one to park.

- **Critical** — must fix before merge. Security, data loss, broken functionality, a
  scenario whose code path does not match the Given/When/Then or has no covering test. Blocks `ship`.
- **Important** — should fix before merge. Architecture drift from a term or in-force ADR, a
  hard standard breach, a missing or partial requirement, a behavior the spec never asked for (scope creep). Unfixed Important issues do not proceed.
- **Minor** — nice to have. Baseline smells, style, optimization, doc polish. Capped and summarized.

## Cap the nits

- **Max 5 Minor findings listed per axis.** Beyond that, collapse the rest into a count.
- A Minor finding that is actually a Critical dressed as polish gets re-labeled, not capped.

## Do not report

- Anything tooling already enforces (gofmt, tsc, lint).
- Informational context with no required action — `FYI` at most, one line, does not count against any cap.

## Passes

> Three axes, reported side by side. Never merged into one verdict.

### Standards

- **Worst issue (within this axis):** none remaining (the 3 Important findings were fixed in `a8b7edb6`)
- **Findings:** 0 Critical / 3 Important (all fixed) / 7 Minor (capped to 5 listed)
- **Verdict:** met

### Spec

- **Worst issue (within this axis):** none remaining (the 3 Important findings were fixed in `be0ee139` + `40ff278` + `3f65a21`)
- **Findings:** 0 Critical / 3 Important (all fixed) / 2 Minor
- **Verdict:** met

### Security

- **Worst issue (within this axis):** none — all candidates triaged away
- **Findings:** 0 Critical / 0 High / 0 Medium / 0 Low / 0 Info
- **Verdict:** secure

## Findings

### Critical

(none)

### Important (all fixed)

- [standards] `server_tracker_stream.go` vs `backlog.go:39` — `milestonesDir` was hardcoded in the SSE handler while the reader derived it from `backlogAdapter.milestonesDir()` — **FIXED** (`a8b7edb6`): the handler now derives both watched dirs from package-level constants that mirror the reader, single source of truth.
- [standards] `backlog.go` `Milestones()` — the per-file `os.ReadFile` skip silently dropped unreadable files — **FIXED** (`a8b7edb6`): written justification added (partial board > no board; the `ReadDir`-level error is what 502s).
- [standards] `server_tracker_stream.go` — `trackerStreamHub` / `newTrackerStreamHub` (+ `add`/`remove`/`watchDir`) were dead code — **FIXED** (`a8b7edb6`): deleted, stale comment removed.
- [spec] `ListView.tsx` + `TaskDetail.tsx` — TASK-047 AC#3 had no render test (the QA report mis-mapped it to `BoardView.test.tsx`) — **FIXED** (`40ff278`): title-resolved + raw-ID-fallback render tests added for both components.
- [spec] `server_tracker.go:60` — TASK-047 AC#5 (GET /tracker/milestones) had no HTTP round-trip test — **FIXED** (`be0ee139`): `httptest` round-trip added asserting the `{"milestones":[...],"provider":"backlogmd"}` shape, id-sorted, plus the missing-dir degrade (200 + empty array).
- [spec] `BoardView.test.tsx` — TASK-049 scope item 2 ("DnD still works within milestone rows") had no test — **FIXED** (`3f65a21`): a real pointer drag against the `SortableContext` card drives the drop → `onMove`; a plain click still opens the task instead of moving.

### Minor

- [standards] `backlog.go:203-233` + `282-316` — Duplicated Code: the dir-list→filter-.md→read→parse→skip-empty-id→sort loop is in two places (`loadTasks` + `Milestones`) — extract `forEachFrontmatterFile` if a third consumer appears.
- [standards] `backlog.go:289` — `errBadOutput{CLI: "", ...}` reads oddly for a file-based provider (the message says "CLI output invalid: read ...") — Mysterious Name (mild); a dedicated `errReadDir` or a comment that `CLI` is intentionally empty.
- [standards] `server_tracker_stream.go:135` — `strings.HasPrefix(evt.Name, milestonesDir)` matches an absolute fsnotify path against a relative dir — correct today (test proves it) but depends on undocumented fsnotify CWD-relative behavior; classify by `filepath.Dir(evt.Name)` equality instead.
- [standards] `BoardView.tsx`/`TaskCard.tsx`/`ListView.tsx`/`TaskDetail.tsx`/`epicTree.ts` — `milestoneTitles` plumbed as `Record<string,string>` into five components — Data Clump / Primitive Obsession (mild, idiomatic for a UI); wrap in a small type only if the lookup grows.
- [standards] `backlog_test.go` — no regression test for the `Milestones()` error path (non-IsNotExist read error → 502) — a refactor that drops the `IsNotExist` special-case wouldn't be caught.
- (2 further Minor items collapsed — SSE `time.Sleep` attach heuristic matches pre-existing pattern; `fetchMilestones` couples the board to a new hard dependency, FYI)
- [spec] `epicTree.ts` — milestone groups now sorted by resolved title, not ID; the spec says "sorted alphabetically, unassigned last" without specifying the key — defensible but an unspecified behavior change.
- [spec] commit `22d7031b` — carries only `[skillgrid-context] task-049` despite half its content closing task-048's W018 — attribution gap.

## Independence

> The no-leaks proof. Each axis records its independence grade.

| Axis | Grade | Rationale |
|------|-------|-----------|
| Standards | B | fresh subagent, no shared history; same model family as the implementer |
| Spec | B | fresh subagent, no shared history; same model family as the implementer |
| Security | B | fresh subagent, no shared history; same model family as the implementer |

**Reading the grade:** **A** = fresh subagent, no shared history (independent). **B** = fresh
context but same model family / toolchain as the implementer (independent, weaker). **C** =
inline self-review or the implementer's verdict leaked in — **diagnostic only**.

All three were dispatched as fresh reviewer subagents that saw only the diff + the named
standards/spec/security sources — no session narrative, no implementer rationale, no prior
verdict leaked. Grade B because the subagents share the implementer's model family/toolchain.

## Verdict

> The overall review verdict is the **floor** across the three axes (per `_shared/verification/floor.md`): the weaker axis caps the whole review. Never average the three axes into a single "overall."

- **Standards:** met
- **Spec:** met
- **Security:** secure
- **Floor (decides):** met
- **Worst issue (across all three axes):** none remaining — all 6 Important findings fixed
- **Unfixed Important count (must be 0 to proceed):** 0
