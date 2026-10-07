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

- **Worst issue (within this axis):** `milestonesDir` hardcoded in the SSE handler while the reader derives it from `backlogAdapter.dir()` — watcher and reader can silently desync (backlog.go:39 vs server_tracker_stream.go:81)
- **Findings:** 0 Critical / 3 Important / 7 Minor (capped to 5 listed)
- **Verdict:** met-with-fixes

### Spec

- **Worst issue (within this axis):** three scope items lack the tests the QA report claims — ListView/TaskDetail render (AC#3), GET /tracker/milestones endpoint (AC#5), DnD-within-milestone-rows (scope item 2)
- **Findings:** 0 Critical / 3 Important / 2 Minor
- **Verdict:** met-with-fixes

### Security

- **Worst issue (within this axis):** none — all candidates triaged away
- **Findings:** 0 Critical / 0 High / 0 Medium / 0 Low / 0 Info
- **Verdict:** secure

## Findings

### Critical

(none)

### Important

- [standards] `server_tracker_stream.go:81` vs `backlog.go:39` — `milestonesDir := ".backlog/milestones"` hardcoded in the SSE handler while `backlogAdapter.milestonesDir()` derives from `filepath.Dir(a.dir())`; the handler drops provider resolution and no longer consults the active provider — watcher and reader can point at different dirs with no signal — derive both from a single shared backlog-root helper.
- [standards] `backlog.go:296-298` — `raw, err := os.ReadFile(...); if err != nil { continue }` silently drops an unreadable milestone file (a real I/O failure, not just missing-dir) — code-standards says never swallow without a written justification; add the justification comment or surface it (log/count).
- [standards] `server_tracker_stream.go:21-54` — `trackerStreamHub` / `newTrackerStreamHub` (+ `add`/`remove`/`watchDir`) are dead code: `handleTrackerStream` uses a per-request `fsnotify.NewWatcher()` and never constructs a hub — "no dead code — delete, don't comment out"; pre-existing but in a touched file, and the doc comment is stale.
- [spec] `ListView.tsx` + `TaskDetail.tsx` — TASK-047 AC#3 (list + detail show title) code path is present and correct, but no test renders either component; the QA report maps AC#3 to `BoardView.test.tsx`, which covers neither — add render tests or correct the report's mapping.
- [spec] `server_tracker.go:60` — TASK-047 AC#5 (GET /tracker/milestones returns the list) has no HTTP-level test (no httptest round-trip of the endpoint); only the adapter `Milestones()` is unit-tested — add an httptest round-trip test for the route.
- [spec] `BoardView.test.tsx` — TASK-049 scope item 2 ("DnD still works within milestone rows") has no test simulating a drag within a milestone row; `onMove` is always a no-op `vi.fn()` in every render — the scope line is half-honored.

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

- **Standards:** met-with-fixes
- **Spec:** met-with-fixes
- **Security:** secure
- **Floor (decides):** met-with-fixes
- **Worst issue (across all three axes):** `milestonesDir` hardcoded in the SSE handler while the reader derives it from `backlogAdapter.dir()` — watcher/reader desync (standards)
- **Unfixed Important count (must be 0 to proceed):** 6 (3 standards + 3 spec)
