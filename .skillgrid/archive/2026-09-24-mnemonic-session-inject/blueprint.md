# Session Context Injection Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use skillgrid:subagent-execution (recommended) or skillgrid:simple-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Status:** PROPOSED

**Tier:** T2

**Build shape:** Tracer thread (one end-to-end path — events → summary → retrieval → injection → MCP tool — works first, then thickens with the on-demand tool and privacy filtering)

**Goal:** Build the two-layer session context injection mechanism: (1) auto-prepend a slim, token-capped L1 summary of prior-session state on resume, and (2) an on-demand `mem_inject_session` MCP tool for deeper hybrid (BM25 + semantic) retrieval.

**Architecture:** Reuses the existing `session_events` capture layer (migration 040) and the existing FTS5 + in-memory cosine hybrid search (`hybrid/` package). Adds a `session_inject` package that: (a) distills a prior session's events into a deterministic, privacy-filtered L1 summary; (b) retrieves the relevant slice via hybrid search (BM25 floor + vector leg when embedder active, RRF-fused, degrades to BM25-only when no embedder); (c) renders the selected slice as a compact context block with per-item token-cost estimates. The MCP tool `mem_inject_session` wraps (b)+(c) as an on-demand read path. The auto-prepend layer is a function the session-resume path calls (not an MCP tool — it fires implicitly at resume).

**Tech Stack:** Go 1.22+, `modernc.org/sqlite` (existing), FTS5 (existing), in-memory cosine (`hybrid/vectorcache.go`), MCP Go (`mark3labs/mcp-go`), `mark3labs/mcp-go/mcp` tool definitions.

**Spec:** `.skillgrid/artifacts/07-mnemonic-tool-surface.md` (Session-inject plan section) + `.skillgrid/artifacts/05-locked-constraints.md` (4 locked session-inject constraints).

**Findings:** `.skillgrid/specs/2026-09-24-mnemonic-vector-db/findings.md` (Prototype 001 — G in-SQL latency, viant deferred; in-memory cosine remains the hot path).

## Terms

- **Session-inject** — the two-layer mechanism (auto-prepend L1 on resume + on-demand tool) defined in `07-mnemonic-tool-surface.md`. Not a new term in the glossary; the 07 doc is the source.
- **L1 summary** — the slim, token-capped, deterministic summary of a prior session's state (events distilled to signatures/summaries, not raw bodies). See 07 doc design step 4.
- **L2 details** — the full, untruncated content of an individual item (observation, event payload), retrievable on demand via the existing `mem_get_observation` / `session_changes` tools. Not a new term; the progressive-disclosure pattern from claude-mem.

## Hypothesis

**Claim:** A two-layer injection (slim auto-prepend on resume + on-demand hybrid retrieval) reduces the agent's "cold start" confusion on resume without flooding the context window, and the hybrid selection (BM25 + semantic, RRF-fused) delivers materially better recall than BM25-only when an embedder is active.

**Right condition:** On resume, the injected L1 summary contains the prior session's key decisions, errors, and file changes (the "what happened" the agent needs to pick up) and is ≤ 800 tokens. The on-demand `mem_inject_session` tool returns relevant prior-session state for a query, with the vector leg contributing to the ranking (not just BM25).

**Wrong condition:** The L1 summary is either empty (no prior-session state captured) or > 2000 tokens (context flood), OR the hybrid selection returns the same ranking as BM25-only in all test cases (vector leg contributes nothing).

**Thinnest MVP:** Task 1 (L1 summary distillation) + Task 2 (auto-prepend on resume) — the auto-prepend layer alone, with a fixed token cap and no hybrid retrieval. This proves the "slim summary on resume" value before the on-demand tool exists.

**Door check:** Task 1 — if the L1 summary distillation produces empty output for a session with ≥ 10 events, the capture layer is not feeding enough signal and the blueprint is invalidated (the capture layer needs to land first).

## Must-Haves (goal-backward verification)

**Truths** (observable behaviors that must hold):
- `session_inject.DistillSummary(ctx, sessionID, maxTokens)` returns a non-empty, deterministic (byte-identical across two calls on the same store) L1 summary for a session with ≥ 10 events, ≤ `maxTokens` tokens.
- `session_inject.DistillSummary` excludes sensitive events (`is_sensitive=1`) from the output.
- `session_inject.DistillSummary` excludes `private`-tagged observations from the output.
- The auto-prepend path (called at session resume) prepends the L1 summary to the context when a prior session exists for the project; returns empty for a fresh session (no prior session).
- `mem_inject_session` MCP tool returns a ranked list of prior-session items (observations + event summaries) for a query, scoped to the current project by default.
- `mem_inject_session` with `all_projects: true` returns items across projects.
- `mem_inject_session` with no embedder active degrades to BM25-only (no error, no vector leg) and returns BM25-ranked results.
- `mem_inject_session` with an embedder active returns RRF-fused (BM25 + semantic) rankings that differ from BM25-only for at least one test query.
- The injected context block includes a per-item token-cost estimate.
- **backstop:** The auto-prepend L1 summary on resume is ≤ 800 tokens for a session with 50+ events (requires a runtime test with a realistic event volume — the diff alone cannot confirm the token cap is effective at scale).

**Artifacts** (files that must exist with real implementation, not stubs):
- [`skillgrid-cli/internal/mnemonic/session_inject/summary.go` — L1 summary distillation: reads session_events + observations, produces a deterministic, privacy-filtered, token-capped summary]
- [`skillgrid-cli/internal/mnemonic/session_inject/retrieve.go` — hybrid retrieval: BM25 (existing FTS5) + semantic (existing in-memory cosine), RRF-fused, degrades to BM25-only when no embedder]
- [`skillgrid-cli/internal/mnemonic/session_inject/render.go` — context block rendering: formats selected items into a compact, token-cost-annotated context block]
- [`skillgrid-cli/internal/mnemonic/session_inject/privacy.go` — privacy filtering: excludes secrets, full local paths, `private`-tagged content]
- [`skillgrid-cli/internal/mnemonic/mcp/tools_session_inject.go` — `mem_inject_session` MCP tool registration + handler]
- [`skillgrid-cli/internal/mnemonic/session_inject/summary_test.go` — tests for distillation determinism, token cap, privacy exclusion]
- [`skillgrid-cli/internal/mnemonic/session_inject/retrieve_test.go` — tests for hybrid ranking, BM25-only degradation, project scoping]
- [`skillgrid-cli/internal/mnemonic/session_inject/render_test.go` — tests for context block format, token-cost annotation]
- [`.skillgrid/specs/2026-09-24-mnemonic-session-inject/acceptance.feature` — BDD scenarios for the truths above]

**Key links** (critical connections between artifacts that must work together):
- `DistillSummary` must read from the existing `session_events` table (migration 040) and the existing `observations` table — no new tables.
- `retrieve.go` must call the existing FTS5 search path (not a new index) and the existing `hybrid/vectorcache.go` in-memory cosine for the vector leg.
- `tools_session_inject.go` must register `mem_inject_session` via `s.AddTool` in the existing `registerSessionTools` function (or a new `registerSessionInjectTools` called from `Start`/`NewServer`).
- The auto-prepend path must be callable from the session-resume code path (the CLI's `--continue` equivalent) without an MCP round-trip — it is an in-process function, not a tool.

**One-way-door decisions** (hard to reverse — flag for explicit user approval before implementing):
- None. All changes are additive (new package, new tool, new function). No migration, no existing tool contract change, no public API shape change.

## Global Constraints

- Go 1.22+ minimum to build.
- No new dependencies without an ADR. (This plan adds zero new dependencies — it reuses `modernc.org/sqlite`, `mark3labs/mcp-go`, and the existing `hybrid/` package.)
- Conventional commits only; no AI-attribution trailers.
- Spec-zone changes commit before code-zone changes (pre-commit zone guard).
- Session-inject privacy: tag-by-default — secrets, full local paths, and `private`-tagged content are auto-excluded; project-relative paths, commit SHAs, tool names, and task IDs are included by default.
- Session-inject scope: project-scoped by default; cross-project only via explicit `all_projects: true`.
- Session-inject selection: hybrid (BM25 + semantic, RRF-fused) by default; vector leg degrades to BM25-only when no embedder is active.

## Threat Matrix

| Boundary | Applicability | Design response | Planned RED test |
|---|---|---|---|
| **Mnemonic tool surface** (`mem_*`) | Applicable: new tool `mem_inject_session` with new params (`query`, `all_projects`, `max_tokens`) and a new return shape (ranked items with token-cost). | Explicit contract: `mem_inject_session` is a read-only tool; it does not modify the store. Params: `query` (required string), `all_projects` (optional bool, default false), `max_tokens` (optional int, default 2000). Return: `{ items: [{ type, id, title, snippet, token_cost, source }], total_tokens, degraded: bool }`. `degraded: true` when no embedder is active (BM25-only). | `tools_session_inject_test.go`: (1) tool is registered and callable; (2) returns ranked items for a known query; (3) `all_projects: true` widens scope; (4) no embedder → `degraded: true`, BM25-only results; (5) token-cost present on every item. |
| **Shared-convention drift** | N/A: this change does not edit any `_shared/conventions/*.md`, `_shared/references/*.md`, or `agent-config/*.md` file. It adds a new package and a new MCP tool; no convention file is the source of truth for session-inject behavior (the 07 artifact doc is, and it is updated separately, not as part of this blueprint's code tasks). | — | — |

## File Structure

- `skillgrid-cli/internal/mnemonic/session_inject/summary.go` — L1 summary distillation. Reads `session_events` + `observations` for a session, produces a deterministic, privacy-filtered, token-capped summary. Pure (no MCP, no network).
- `skillgrid-cli/internal/mnemonic/session_inject/privacy.go` — Privacy filtering. Given an event or observation, decides whether it is injectable (excludes secrets, full local paths, `private`-tagged content). Reuses `memory.isSensitivePath` logic (extracted or duplicated — see Task 1).
- `skillgrid-cli/internal/mnemonic/session_inject/retrieve.go` — Hybrid retrieval. BM25 via existing FTS5 + semantic via existing `hybrid/vectorcache.go`. RRF-fused. Degrades to BM25-only when no embedder.
- `skillgrid-cli/internal/mnemonic/session_inject/render.go` — Context block rendering. Formats selected items into a compact, token-cost-annotated context block. Pure.
- `skillgrid-cli/internal/mnemonic/session_inject/summary_test.go` — Tests for distillation.
- `skillgrid-cli/internal/mnemonic/session_inject/retrieve_test.go` — Tests for hybrid retrieval.
- `skillgrid-cli/internal/mnemonic/session_inject/render_test.go` — Tests for rendering.
- `skillgrid-cli/internal/mnemonic/mcp/tools_session_inject.go` — `mem_inject_session` MCP tool.
- `skillgrid-cli/internal/mnemonic/mcp/tools_session_inject_test.go` — MCP tool tests.
- `.skillgrid/specs/2026-09-24-mnemonic-session-inject/acceptance.feature` — BDD scenarios.

---

### Task 1: L1 Summary Distillation

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/session_inject/summary.go`
- Create: `skillgrid-cli/internal/mnemonic/session_inject/privacy.go`
- Create: `skillgrid-cli/internal/mnemonic/session_inject/summary_test.go`
- Modify: `skillgrid-cli/internal/mnemonic/memory/sensitive.go:37-61` (extract `isSensitivePath` into an exported function or a shared package — see below)

**Interfaces:**
- Consumes: `memory.Service.SessionChanges(ctx, sessionID)` (existing, returns `[]Event, from, to, err`); `memory.Service.Search(ctx, query, opts)` (existing FTS5 search over observations); `store.DB` (existing `*sql.DB`).
- Produces:
  - `DistillSummary(ctx context.Context, svc *service.Service, sessionID string, maxTokens int) (string, error)` — returns the L1 summary (a markdown string) or an error. Deterministic: two calls on the same store return byte-identical output.
  - `Injectable(event memory.Event) bool` — privacy filter for events.
  - `ObservationInjectable(obs memory.Observation) bool` — privacy filter for observations.
  - `EstimateTokens(s string) int` — token-cost estimate (chars/4 heuristic).
- Seam: none (in-process, pure).
- Deletion test: if `session_inject` is deleted, the auto-prepend and on-demand tool have no distillation/retrieval/rendering logic. The existing `session_changes` tool is unaffected (it returns raw events, not a summary).
- Adapters: 1 (the real `memory.Service`). A test adapter (mock service with seeded events/observations) is needed for tests — this is a test double, not a production adapter, so the seam is not yet justified for a port.

**SATISFIES:** `l1-summary-deterministic`, `l1-summary-token-cap`, `l1-summary-privacy-exclusion`, `l1-summary-empty-on-few-events`

**Privacy extraction note:** `memory.isSensitivePath` (sensitive.go:37) is unexported. Task 1 must either (a) export it as `memory.IsSensitivePath` and have `session_inject/privacy.go` call it, or (b) duplicate the logic in `session_inject/privacy.go`. Option (a) is preferred (DRY) — it is a trivial rename (add capital), no behavior change, no existing caller breaks (it is only called from within `memory/`).

- [ ] **Step 1: Write the failing test**

```go
// skillgrid-cli/internal/mnemonic/session_inject/summary_test.go
package session_inject

import (
	"context"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

func TestDistillSummary_Deterministic(t *testing.T) {
	// Seed a temp store with a session + 15 events.
	st := store.NewTestStore(t)
	svc := newTestService(t, st)
	sessionID := seedSessionWithEvents(t, svc, 15)

	ctx := context.Background()
	s1, err := DistillSummary(ctx, svc, sessionID, 800)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	s2, err := DistillSummary(ctx, svc, sessionID, 800)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if s1 != s2 {
		t.Errorf("summary not deterministic:\n--- call 1 ---\n%s\n--- call 2 ---\n%s", s1, s2)
	}
}

func TestDistillSummary_TokenCap(t *testing.T) {
	st := store.NewTestStore(t)
	svc := newTestService(t, st)
	sessionID := seedSessionWithEvents(t, svc, 60) // 60 events → likely > 800 tokens untrimmed

	ctx := context.Background()
	summary, err := DistillSummary(ctx, svc, sessionID, 800)
	if err != nil {
		t.Fatalf("DistillSummary: %v", err)
	}
	tokens := EstimateTokens(summary)
	if tokens > 800 {
		t.Errorf("summary exceeds token cap: %d > 800", tokens)
	}
}

func TestDistillSummary_ExcludesSensitive(t *testing.T) {
	st := store.NewTestStore(t)
	svc := newTestService(t, st)
	sessionID := seedSessionWithSensitiveEvents(t, svc, 5, 3) // 5 normal, 3 sensitive

	ctx := context.Background()
	summary, err := DistillSummary(ctx, svc, sessionID, 2000)
	if err != nil {
		t.Fatalf("DistillSummary: %v", err)
	}
	// Sensitive events touch .env / .pem paths — none should appear in the summary.
	for _, secret := range []string{".env", ".pem", ".ssh/", ".aws/"} {
		if contains(summary, secret) {
			t.Errorf("summary contains sensitive path marker %q", secret)
		}
	}
}

func TestDistillSummary_FewEvents(t *testing.T) {
	st := store.NewTestStore(t)
	svc := newTestService(t, st)
	sessionID := seedSessionWithEvents(t, svc, 2) // only 2 events

	ctx := context.Background()
	summary, err := DistillSummary(ctx, svc, sessionID, 800)
	if err != nil {
		t.Fatalf("DistillSummary: %v", err)
	}
	if summary != "" {
		t.Errorf("expected empty summary for < 10 events, got %d chars", len(summary))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./skillgrid-cli/internal/mnemonic/session_inject/ -run TestDistillSummary -v`
Expected: FAIL with "undefined: DistillSummary" / "undefined: EstimateTokens" / "undefined: Injectable"

- [ ] **Step 3: Write minimal implementation**

```go
// skillgrid-cli/internal/mnemonic/session_inject/privacy.go
package session_inject

import (
	"strings"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
)

// Injectable reports whether a session event is safe to include in the
// injected context. Sensitive events (is_sensitive=1, per memory.isSensitivePath)
// are excluded. Events with no path and no command (e.g. session_start,
// session_end) are included — they carry the commit range.
func Injectable(e memory.Event) bool {
	if e.IsSensitive {
		return false
	}
	return true
}

// ObservationInjectable reports whether an observation is safe to include.
// Observations tagged "private" (topic_key or content contains "private")
// are excluded. Full local paths (/Users/..., /home/..., C:\...) are excluded
// from the content (they are replaced with a project-relative placeholder).
// Commit SHAs, tool names, and task IDs are included by default.
func ObservationInjectable(o memory.Observation) bool {
	lower := strings.ToLower(o.Title + " " + o.Content)
	if strings.Contains(lower, "private") {
		return false
	}
	return true
}

// redactFullPaths replaces absolute local paths in s with a placeholder.
// /Users/foo/bar → <path>, /home/foo/bar → <path>, C:\foo\bar → <path>.
// Project-relative paths (src/foo.go, internal/bar.go) are left alone.
func redactFullPaths(s string) string {
	for _, prefix := range []string{"/Users/", "/home/", "C:\\\\"} {
		s = strings.ReplaceAll(s, prefix, "<path>/")
	}
	return s
}
```

```go
// skillgrid-cli/internal/mnemonic/session_inject/summary.go
package session_inject

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

const minEventsForSummary = 10

// EstimateTokens is the token-cost heuristic: 1 token ≈ 4 characters.
// This is the same heuristic used by the existing budget (memory/budget.go)
// for char-budget → token-budget conversion. It is an estimate, not exact.
func EstimateTokens(s string) int {
	if len(s) == 0 {
		return 0
	}
	return (len(s) + 3) / 4
}

// DistillSummary produces the L1 summary for a session: a deterministic,
// privacy-filtered, token-capped markdown block. It reads the session's
// events (via memory.SessionChanges) and the project's recent observations
// (via memory.Service search scoped to the session's project), filters
// sensitive/private items, and renders a compact summary.
//
// Determinism: events are processed in sequence order (already sorted by
// SessionChanges); observations are sorted by (project, id) for stable output.
// No timestamps in the output (they make the summary non-deterministic across
// calls if the store is re-read). The commit range (from_commit, to_commit)
// IS included (it is stable for a completed session).
//
// Returns "" (empty string, no error) when the session has fewer than
// minEventsForSummary events — a tiny session has no meaningful summary.
func DistillSummary(ctx context.Context, svc *service.Service, sessionID string, maxTokens int) (string, error) {
	if maxTokens <= 0 {
		maxTokens = 800
	}
	events, from, to, err := svc.Memory().SessionChanges(ctx, sessionID)
	if err != nil {
		return "", err
	}
	if len(events) < minEventsForSummary {
		return "", nil
	}

	var sb strings.Builder
	sb.WriteString("## Prior Session Summary\n\n")
	sb.WriteString(fmt.Sprintf("**Commit range:** `%s` → `%s`\n\n", shortSHA(from), shortSHA(to)))

	// Group events by action_type for a compact "what happened" section.
	byType := map[string][]memory.Event{}
	for _, e := range events {
		if !Injectable(e) {
			continue
		}
		byType[e.ActionType] = append(byType[e.ActionType], e)
	}

	// Deterministic ordering of action types.
	types := make([]string, 0, len(byType))
	for t := range byType {
		types = append(types, t)
	}
	sort.Strings(types)

	sb.WriteString("### Activity\n\n")
	for _, t := range types {
		evs := byType[t]
		sb.WriteString(fmt.Sprintf("- **%s** (%d events)\n", t, len(evs)))
		// Show the last 3 events of this type (most recent = most relevant).
		start := len(evs) - 3
		if start < 0 {
			start = 0
		}
		for _, e := range evs[start:] {
			line := renderEvent(e)
			sb.WriteString(fmt.Sprintf("  - `%s` %s\n", e.Timestamp[:10], line))
		}
	}

	// Recent observations (decisions, bugfixes, architecture) for the project.
	// Scoped to the session's project. Search with an empty query returns
	// recent items (the existing search path supports this).
	if obs, err := svc.Memory().RecentObservations(ctx, 5); err == nil {
		var injectable []memory.Observation
		for _, o := range obs {
			if ObservationInjectable(o) {
				injectable = append(injectable, o)
			}
		}
		if len(injectable) > 0 {
			sb.WriteString("\n### Key Observations\n\n")
			// Sort by ID for determinism.
			sort.Slice(injectable, func(i, j int) bool { return injectable[i].ID < injectable[j].ID })
			for _, o := range injectable {
				snippet := redactFullPaths(o.Content)
				if len(snippet) > 200 {
					snippet = snippet[:200] + "…"
				}
				sb.WriteString(fmt.Sprintf("- **%s** (%s): %s\n", o.Title, o.Type, snippet))
			}
		}
	}

	// Enforce the token cap.
	result := sb.String()
	if EstimateTokens(result) > maxTokens {
		result = truncateToTokens(result, maxTokens)
	}
	return result, nil
}

// renderEvent produces a one-line, privacy-filtered description of an event.
func renderEvent(e memory.Event) string {
	var parts []string
	if e.ToolName != "" {
		parts = append(parts, e.ToolName)
	}
	if e.Path != "" {
		parts = append(parts, e.Path)
	}
	if e.Command != "" {
		cmd := e.Command
		if len(cmd) > 60 {
			cmd = cmd[:60] + "…"
		}
		parts = append(parts, cmd)
	}
	if e.Commit != "" {
		parts = append(parts, shortSHA(e.Commit))
	}
	return redactFullPaths(strings.Join(parts, " "))
}

// shortSHA truncates a full SHA to 7 chars (the conventional short form).
func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	if sha == "" {
		return "none"
	}
	return sha
}

// truncateToTokens truncates s to fit within maxTokens (4 chars/token),
// appending a "… (N tokens truncated)" marker.
func truncateToTokens(s string, maxTokens int) string {
	maxChars := maxTokens * 4
	if len(s) <= maxChars {
		return s
	}
	kept := s[:maxChars]
	omitted := EstimateTokens(s[maxChars:])
	return kept + fmt.Sprintf("\n… (%d tokens truncated)", omitted)
}
```

Note: `RecentObservations` is a new method on `memory.Service` that Task 1 must add (a simple `SELECT ... FROM observations WHERE project=? AND deleted_at IS NULL ORDER BY created_at DESC LIMIT ?`). It is a read-only query, no new table.

Also: export `memory.IsSensitivePath` by renaming `isSensitivePath` → `IsSensitivePath` in `memory/sensitive.go:37`. Update the internal caller in the same file (sensitive.go:57 `if isSensitivePath(path)` → `if IsSensitivePath(path)`). No other file calls this function (it is package-private).

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./skillgrid-cli/internal/mnemonic/session_inject/ -run TestDistillSummary -v`
Expected: PASS (all 4 tests)

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/session_inject/ summary.go privacy.go summary_test.go
git add skillgrid-cli/internal/mnemonic/memory/sensitive.go  # IsSensitivePath export
git commit -m "feat(session-inject): L1 summary distillation with privacy filter

Adds the session_inject package with DistillSummary (deterministic,
token-capped, privacy-filtered L1 summary from session_events +
observations), Injectable/ObservationInjectable privacy filters, and
EstimateTokens (chars/4 heuristic). Exports memory.IsSensitivePath
for reuse. Returns empty for sessions with < 10 events.

[skillgrid-context]
Change: 2026-09-24-mnemonic-session-inject
Phase: apply
"
```

### Task 2: Auto-Prepend on Resume

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/session_inject/summary.go` (add `AutoPrepend` function)
- Create: `skillgrid-cli/internal/mnemonic/session_inject/autoprepend_test.go`
- Modify: the CLI's session-resume path (the `--continue` equivalent — locate the code that detects "resuming prior session" and calls `DistillSummary` + prepends the result to the context)

**Interfaces:**
- Consumes: `DistillSummary` (Task 1), `service.Service` (existing).
- Produces:
  - `AutoPrepend(ctx context.Context, svc *service.Service, projectID string, maxTokens int) (string, error)` — finds the most recent completed session for the project, calls `DistillSummary` on it, and returns the L1 summary (or "" for a fresh session / no prior session).
- Seam: none (in-process).
- Deletion test: if `AutoPrepend` is deleted, the resume path has no injection — the agent starts cold.
- Adapters: 1 (real service). Test adapter needed for tests.

**SATISFIES:** `auto-prepend-resume`, `auto-prepend-fresh-session-empty`

- [ ] **Step 1: Write the failing test**

```go
// skillgrid-cli/internal/mnemonic/session_inject/autoprepend_test.go
package session_inject

import (
	"context"
	"testing"
)

func TestAutoPrepend_Resume(t *testing.T) {
	st := store.NewTestStore(t)
	svc := newTestService(t, st)
	// Seed a completed session with 15 events for the project.
	seedCompletedSession(t, svc, "test-project", 15)

	ctx := context.Background()
	summary, err := AutoPrepend(ctx, svc, "test-project", 800)
	if err != nil {
		t.Fatalf("AutoPrepend: %v", err)
	}
	if summary == "" {
		t.Fatal("expected non-empty summary on resume, got empty")
	}
	if !contains(summary, "Prior Session Summary") {
		t.Error("summary missing 'Prior Session Summary' header")
	}
}

func TestAutoPrepend_FreshSession(t *testing.T) {
	st := store.NewTestStore(t)
	svc := newTestService(t, st)
	// No prior session for this project.

	ctx := context.Background()
	summary, err := AutoPrepend(ctx, svc, "fresh-project", 800)
	if err != nil {
		t.Fatalf("AutoPrepend: %v", err)
	}
	if summary != "" {
		t.Errorf("expected empty summary for fresh session, got %d chars", len(summary))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./skillgrid-cli/internal/mnemonic/session_inject/ -run TestAutoPrepend -v`
Expected: FAIL with "undefined: AutoPrepend"

- [ ] **Step 3: Write minimal implementation**

```go
// In skillgrid-cli/internal/mnemonic/session_inject/summary.go (append):

// AutoPrepend finds the most recent completed session for projectID and
// returns its L1 summary (via DistillSummary). It is the auto-prepend layer:
// called at session resume, not an MCP tool. Returns "" for a fresh session
// (no prior completed session) or when the prior session has too few events
// for a meaningful summary.
func AutoPrepend(ctx context.Context, svc *service.Service, projectID string, maxTokens int) (string, error) {
	sessionID, err := svc.Memory().LatestCompletedSession(ctx, projectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil // fresh session — no prior session
		}
		return "", err
	}
	return DistillSummary(ctx, svc, sessionID, maxTokens)
}
```

Add `LatestCompletedSession` to `memory.Service`:
```go
// In skillgrid-cli/internal/mnemonic/memory/ (new method, e.g. in changes.go):

// LatestCompletedSession returns the id of the most recent session with
// status='completed' for projectID. Returns sql.ErrNoRows when none exists.
func (s *Service) LatestCompletedSession(ctx context.Context, projectID string) (string, error) {
	var id string
	err := s.store.DB.QueryRowContext(ctx,
		`SELECT id FROM sessions WHERE project = ? AND status = 'completed' ORDER BY ended_at DESC, rowid DESC LIMIT 1`,
		projectID,
	).Scan(&id)
	return id, err
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./skillgrid-cli/internal/mnemonic/session_inject/ -run TestAutoPrepend -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/session_inject/summary.go
git add skillgrid-cli/internal/mnemonic/session_inject/autoprepend_test.go
git add skillgrid-cli/internal/mnemonic/memory/changes.go  # LatestCompletedSession
git commit -m "feat(session-inject): auto-prepend L1 summary on resume

Adds AutoPrepend (finds latest completed session, distills L1 summary)
and memory.LatestCompletedSession. Called at session resume; returns
empty for fresh sessions.

[skillgrid-context]
Change: 2026-09-24-mnemonic-session-inject
Phase: apply
"
```

### Task 3: Hybrid Retrieval (BM25 + Semantic, RRF-Fused)

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/session_inject/retrieve.go`
- Create: `skillgrid-cli/internal/mnemonic/session_inject/retrieve_test.go`

**Interfaces:**
- Consumes: `memory.Service.Search` (existing FTS5 search), `hybrid` package (existing in-memory cosine via `vectorcache.go`), `embedder.Embedder` (existing, `EmbedQuery` for the query vector).
- Produces:
  - `RetrieveResult` struct:
    ```go
    type RetrieveResult struct {
        Items     []InjectItem  // ranked items
        TotalTokens int         // sum of item token costs
        Degraded  bool          // true when no embedder (BM25-only)
    }
    type InjectItem struct {
        Type      string  // "observation" | "event"
        ID        int64   // observation ID or event ID
        Title     string
        Snippet   string  // redacted, token-capped
        TokenCost int
        Source    string  // "observation" | "session_event"
    }
    ```
  - `HybridRetrieve(ctx context.Context, svc *service.Service, query string, projectID string, allProjects bool, maxTokens int) (*RetrieveResult, error)` — hybrid retrieval. BM25 via FTS5 + semantic via in-memory cosine. RRF-fused. Degrades to BM25-only when no embedder.
- Seam: the embedder is the seam — when nil, the vector leg is dropped (degraded). This is the existing degrade-to-Null pattern (`embedder/select.go:37-46`).
- Deletion test: if `retrieve.go` is deleted, the on-demand tool has no retrieval logic.
- Adapters: 2 (real embedder + Null embedder when none configured). The Null embedder is the production degradation path, so the seam is justified.

**SATISFIES:** `hybrid-retrieve-bm25-only`, `hybrid-retrieve-vector-leg`, `hybrid-retrieve-project-scope`, `hybrid-retrieve-all-projects`

- [ ] **Step 1: Write the failing test**

```go
// skillgrid-cli/internal/mnemonic/session_inject/retrieve_test.go
package session_inject

import (
	"context"
	"testing"
)

func TestHybridRetrieve_BM25Only(t *testing.T) {
	st := store.NewTestStore(t)
	svc := newTestServiceNoEmbedder(t, st) // no embedder → degraded
	seedObservations(t, svc, "test-project", 10) // seed 10 observations

	ctx := context.Background()
	res, err := HybridRetrieve(ctx, svc, "auth middleware", "test-project", false, 2000)
	if err != nil {
		t.Fatalf("HybridRetrieve: %v", err)
	}
	if !res.Degraded {
		t.Error("expected Degraded=true with no embedder")
	}
	if len(res.Items) == 0 {
		t.Fatal("expected items from BM25, got none")
	}
	for _, item := range res.Items {
		if item.TokenCost <= 0 {
			t.Errorf("item %d has no token cost", item.ID)
		}
	}
}

func TestHybridRetrieve_VectorLeg(t *testing.T) {
	st := store.NewTestStore(t)
	svc := newTestServiceWithEmbedder(t, st) // embedder active
	seedEmbeddings(t, svc, "test-project", 10) // seed 10 embedded observations

	ctx := context.Background()
	res, err := HybridRetrieve(ctx, svc, "authentication flow", "test-project", false, 2000)
	if err != nil {
		t.Fatalf("HybridRetrieve: %v", err)
	}
	if res.Degraded {
		t.Error("expected Degraded=false with embedder active")
	}
	if len(res.Items) == 0 {
		t.Fatal("expected items, got none")
	}
}

func TestHybridRetrieve_ProjectScope(t *testing.T) {
	st := store.NewTestStore(t)
	svc := newTestServiceNoEmbedder(t, st)
	seedObservations(t, svc, "project-a", 5)
	seedObservations(t, svc, "project-b", 5)

	ctx := context.Background()
	res, err := HybridRetrieve(ctx, svc, "query", "project-a", false, 2000)
	if err != nil {
		t.Fatalf("HybridRetrieve: %v", err)
	}
	// All items should be from project-a only.
	for _, item := range res.Items {
		if item.Source != "observation" {
			continue
		}
		// Verify the item's observation belongs to project-a.
		// (Test helper checks the store directly.)
	}
}

func TestHybridRetrieve_AllProjects(t *testing.T) {
	st := store.NewTestStore(t)
	svc := newTestServiceNoEmbedder(t, st)
	seedObservations(t, svc, "project-a", 5)
	seedObservations(t, svc, "project-b", 5)

	ctx := context.Background()
	res, err := HybridRetrieve(ctx, svc, "query", "project-a", true, 2000)
	if err != nil {
		t.Fatalf("HybridRetrieve: %v", err)
	}
	// Items may be from both projects.
	if len(res.Items) == 0 {
		t.Fatal("expected items across projects, got none")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./skillgrid-cli/internal/mnemonic/session_inject/ -run TestHybridRetrieve -v`
Expected: FAIL with "undefined: HybridRetrieve" / "undefined: RetrieveResult"

- [ ] **Step 3: Write minimal implementation**

```go
// skillgrid-cli/internal/mnemonic/session_inject/retrieve.go
package session_inject

import (
	"context"
	"sort"
	"strings"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

const (
	rrfK        = 60 // RRF constant (standard value)
	bm25Limit   = 20 // max BM25 candidates to fuse
	vectorLimit = 20 // max vector candidates to fuse
)

// InjectItem is one ranked item in the retrieval result.
type InjectItem struct {
	Type      string `json:"type"`      // "observation" | "event"
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Snippet   string `json:"snippet"`
	TokenCost int    `json:"token_cost"`
	Source    string `json:"source"` // "observation" | "session_event"
}

// RetrieveResult is the outcome of a hybrid retrieval.
type RetrieveResult struct {
	Items       []InjectItem `json:"items"`
	TotalTokens int          `json:"total_tokens"`
	Degraded    bool         `json:"degraded"`
}

// HybridRetrieve performs hybrid retrieval over the project's observations
// (and optionally session events): BM25 (FTS5) + semantic (in-memory cosine),
// RRF-fused. Degrades to BM25-only when no embedder is active (Degraded=true).
//
// Project scoping: when allProjects is false, only observations for projectID
// are retrieved. When true, observations across all projects are in scope
// (reuses the existing all_projects flag semantics from mem_search).
func HybridRetrieve(ctx context.Context, svc *service.Service, query string, projectID string, allProjects bool, maxTokens int) (*RetrieveResult, error) {
	if maxTokens <= 0 {
		maxTokens = 2000
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return &RetrieveResult{Degraded: false}, nil
	}

	// BM25 leg: existing FTS5 search.
	bm25Hits, err := svc.Memory().Search(ctx, query, memory.SearchOpts{
		Project:     projectID,
		AllProjects: allProjects,
		Limit:       bm25Limit,
	})
	if err != nil {
		return nil, err
	}

	// Vector leg: in-memory cosine (degrades to empty when no embedder).
	vectorHits, degraded := vectorRetrieve(ctx, svc, query, projectID, allProjects)

	// RRF-fuse the two rankings.
	fused := rrfFuse(bm25Hits, vectorHits)

	// Render into InjectItems, enforcing the token cap.
	items := make([]InjectItem, 0, len(fused))
	totalTokens := 0
	for _, o := range fused {
		if !ObservationInjectable(o) {
			continue
		}
		snippet := redactFullPaths(o.Content)
		if len(snippet) > 200 {
			snippet = snippet[:200] + "…"
		}
		cost := EstimateTokens(snippet)
		if totalTokens+cost > maxTokens {
			break
		}
		totalTokens += cost
		items = append(items, InjectItem{
			Type:      "observation",
			ID:        o.ID,
			Title:     o.Title,
			Snippet:   snippet,
			TokenCost: cost,
			Source:    "observation",
		})
	}

	return &RetrieveResult{
		Items:       items,
		TotalTokens: totalTokens,
		Degraded:    degraded,
	}, nil
}

// vectorRetrieve runs the semantic leg: embed the query, cosine-score against
// stored observation embeddings (in-memory via hybrid/vectorcache.go).
// Returns empty slice + degraded=true when no embedder is active.
func vectorRetrieve(ctx context.Context, svc *service.Service, query, projectID string, allProjects bool) ([]memory.Observation, bool) {
	emb := svc.Embedder()
	if emb == nil || emb.IsNull() {
		return nil, true
	}
	qv, err := emb.EmbedQuery(ctx, query)
	if err != nil || len(qv.Data) == 0 {
		return nil, true
	}
	// Fetch candidate observations (project-scoped), score by cosine.
	obs, err := svc.Memory().RecentObservations(ctx, vectorLimit*2)
	if err != nil || len(obs) == 0 {
		return nil, true
	}
	type scored struct {
		o     memory.Observation
		score float64
	}
	var out []scored
	for _, o := range obs {
		v, err := svc.Memory().ObservationEmbedding(ctx, o.ID)
		if err != nil || len(v.Data) == 0 {
			continue
		}
		sim := memory.CosineSimilarity(qv, v)
		if sim > 0 {
			out = append(out, scored{o: o, score: sim})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].score > out[j].score })
	if len(out) > vectorLimit {
		out = out[:vectorLimit]
	}
	res := make([]memory.Observation, len(out))
	for i, s := range out {
		res[i] = s.o
	}
	return res, false
}

// rrfFuse fuses two ranked lists using Reciprocal Rank Fusion.
// Each observation's RRF score = sum(1/(rrfK + rank)) across both lists.
// Observations in only one list get a single-term score. Ties broken by ID.
func rrfFuse(bm25, vector []memory.Observation) []memory.Observation {
	scores := map[int64]float64{}
	set := map[int64]memory.Observation{}
	for rank, o := range bm25 {
		scores[o.ID] += 1.0 / float64(rrfK+rank+1)
		set[o.ID] = o
	}
	for rank, o := range vector {
		scores[o.ID] += 1.0 / float64(rrfK+rank+1)
		set[o.ID] = o
	}
	out := make([]memory.Observation, 0, len(set))
	for _, o := range set {
		out = append(out, o)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if scores[out[i].ID] != scores[out[j].ID] {
			return scores[out[i].ID] > scores[out[j].ID]
		}
		return out[i].ID < out[j].ID
	})
	return out
}
```

Note: `svc.Embedder()` and `ObservationEmbedding` may need to be exposed on the service/memory if not already. Check the existing `service.Service` for an `Embedder()` accessor and `memory.Service` for an `ObservationEmbedding(ctx, id)` method. If absent, add them (thin wrappers over the existing `dirEmb` field and `obsEmbedding` private method).

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./skillgrid-cli/internal/mnemonic/session_inject/ -run TestHybridRetrieve -v`
Expected: PASS (all 4 tests)

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/session_inject/retrieve.go
git add skillgrid-cli/internal/mnemonic/session_inject/retrieve_test.go
git commit -m "feat(session-inject): hybrid retrieval (BM25 + semantic, RRF-fused)

Adds HybridRetrieve with BM25 (FTS5) + semantic (in-memory cosine)
legs, RRF-fused. Degrades to BM25-only (Degraded=true) when no embedder.
Project-scoped by default; all_projects flag widens scope.

[skillgrid-context]
Change: 2026-09-24-mnemonic-session-inject
Phase: apply
"
```

### Task 4: Context Block Rendering

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/session_inject/render.go`
- Create: `skillgrid-cli/internal/mnemonic/session_inject/render_test.go`

**Interfaces:**
- Consumes: `RetrieveResult` (Task 3), `InjectItem` (Task 3).
- Produces:
  - `RenderContextBlock(res *RetrieveResult, projectID string) string` — formats a `RetrieveResult` into a compact, token-cost-annotated markdown context block.
- Seam: none (pure, in-process).
- Deletion test: if `render.go` is deleted, the MCP tool returns raw JSON, not a formatted context block.
- Adapters: 1 (the real result). No seam needed.

**SATISFIES:** `context-block-format`, `context-block-token-cost`

- [ ] **Step 1: Write the failing test**

```go
// skillgrid-cli/internal/mnemonic/session_inject/render_test.go
package session_inject

import (
	"testing"
)

func TestRenderContextBlock_Format(t *testing.T) {
	res := &RetrieveResult{
		Items: []InjectItem{
			{Type: "observation", ID: 1, Title: "Fixed N+1 query", Snippet: "Changed to batch...", TokenCost: 50, Source: "observation"},
			{Type: "observation", ID: 2, Title: "Auth model ADR", Snippet: "Chose JWT over...", TokenCost: 80, Source: "observation"},
		},
		TotalTokens: 130,
		Degraded:    false,
	}
	block := RenderContextBlock(res, "test-project")

	for _, want := range []string{"Session Context Injection", "Fixed N+1 query", "Auth model ADR", "50 tokens", "80 tokens", "130 tokens total"} {
		if !contains(block, want) {
			t.Errorf("context block missing %q\n--- block ---\n%s", want, block)
		}
	}
}

func TestRenderContextBlock_Degraded(t *testing.T) {
	res := &RetrieveResult{
		Items:       []InjectItem{{Type: "observation", ID: 1, Title: "T", Snippet: "S", TokenCost: 10, Source: "observation"}},
		TotalTokens: 10,
		Degraded:    true,
	}
	block := RenderContextBlock(res, "p")
	if !contains(block, "BM25-only") {
		t.Errorf("degraded context block missing 'BM25-only' marker\n--- block ---\n%s", block)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./skillgrid-cli/internal/mnemonic/session_inject/ -run TestRenderContextBlock -v`
Expected: FAIL with "undefined: RenderContextBlock"

- [ ] **Step 3: Write minimal implementation**

```go
// skillgrid-cli/internal/mnemonic/session_inject/render.go
package session_inject

import (
	"fmt"
	"strings"
)

// RenderContextBlock formats a RetrieveResult into a compact,
// token-cost-annotated markdown context block. Each item carries its
// token cost; the block ends with a total. When Degraded is true, a
// "BM25-only (no embedder)" marker is included.
func RenderContextBlock(res *RetrieveResult, projectID string) string {
	if res == nil || len(res.Items) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("## Session Context Injection\n\n")
	sb.WriteString(fmt.Sprintf("**Project:** %s\n", projectID))
	if res.Degraded {
		sb.WriteString("**Mode:** BM25-only (no embedder)\n")
	}
	sb.WriteString("\n")
	for i, item := range res.Items {
		sb.WriteString(fmt.Sprintf("%d. **%s** (%s, %d tokens)\n", i+1, item.Title, item.Source, item.TokenCost))
		sb.WriteString(fmt.Sprintf("   %s\n", item.Snippet))
	}
	sb.WriteString(fmt.Sprintf("\n**Total:** %d tokens\n", res.TotalTokens))
	return sb.String()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./skillgrid-cli/internal/mnemonic/session_inject/ -run TestRenderContextBlock -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/session_inject/render.go
git add skillgrid-cli/internal/mnemonic/session_inject/render_test.go
git commit -m "feat(session-inject): context block rendering with token costs

Adds RenderContextBlock: formats a RetrieveResult into a compact,
token-cost-annotated markdown block. Includes BM25-only marker when
degraded.

[skillgrid-context]
Change: 2026-09-24-mnemonic-session-inject
Phase: apply
"
```

### Task 5: `mem_inject_session` MCP Tool

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/mcp/tools_session_inject.go`
- Create: `skillgrid-cli/internal/mnemonic/mcp/tools_session_inject_test.go`
- Modify: `skillgrid-cli/internal/mnemonic/mcp/server.go:65-97` (register the new tool in `Start` and `NewServer`)

**Interfaces:**
- Consumes: `HybridRetrieve` (Task 3), `RenderContextBlock` (Task 4), `service.Service` (existing).
- Produces: the `mem_inject_session` MCP tool, registered on the server.
- Seam: the MCP tool boundary (stdio). The handler is a thin wrapper over `HybridRetrieve` + `RenderContextBlock`.
- Deletion test: if the tool is deleted, the agent has no on-demand injection path (only the auto-prepend layer).
- Adapters: 1 (real service). Test via the existing `HandleXForTest` pattern.

**SATISFIES:** `mcp-tool-registered`, `mcp-tool-returns-items`, `mcp-tool-all-projects`, `mcp-tool-degraded`, `mcp-tool-token-cost`

- [ ] **Step 1: Write the failing test**

```go
// skillgrid-cli/internal/mnemonic/mcp/tools_session_inject_test.go
package mcp

import (
	"context"
	"testing"

	mcplib "github.com/mark3labs/mcp-go/mcp"
)

func TestMemInjectSession_Registered(t *testing.T) {
	s := NewServer()
	_, err := s.GetTool("mem_inject_session")
	if err != nil {
		t.Fatalf("mem_inject_session not registered: %v", err)
	}
}

func TestMemInjectSession_ReturnsItems(t *testing.T) {
	// Seed a test store with observations, call the handler.
	setupTestService(t) // helper: seeds a service with test data
	req := mcplib.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"query":     "auth",
		"project":   "test-project",
		"all_projects": false,
	}
	res, err := handleMemInjectSession(context.Background(), req)
	if err != nil {
		t.Fatalf("handleMemInjectSession: %v", err)
	}
	if res == nil {
		t.Fatal("nil result")
	}
	// Parse the JSON result and check for items.
	// (Use the existing JSONResult parsing helper from testutil.)
}

func TestMemInjectSession_AllProjects(t *testing.T) {
	setupTestServiceMultiProject(t)
	req := mcplib.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"query":         "query",
		"project":       "project-a",
		"all_projects":  true,
	}
	res, err := handleMemInjectSession(context.Background(), req)
	if err != nil {
		t.Fatalf("handleMemInjectSession: %v", err)
	}
	if res == nil {
		t.Fatal("nil result")
	}
}

func TestMemInjectSession_Degraded(t *testing.T) {
	setupTestServiceNoEmbedder(t)
	req := mcplib.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"query":     "query",
		"project":   "test-project",
	}
	res, err := handleMemInjectSession(context.Background(), req)
	if err != nil {
		t.Fatalf("handleMemInjectSession: %v", err)
	}
	// Check that the result includes "degraded": true.
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./skillgrid-cli/internal/mnemonic/mcp/ -run TestMemInjectSession -v`
Expected: FAIL with "undefined: handleMemInjectSession" / tool not registered

- [ ] **Step 3: Write minimal implementation**

```go
// skillgrid-cli/internal/mnemonic/mcp/tools_session_inject.go
package mcp

import (
	"context"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/session_inject"
)

func registerSessionInjectTools(s *server.MCPServer) {
	s.AddTool(memInjectSessionTool(), handleMemInjectSession)
}

func memInjectSessionTool() mcplib.Tool {
	return mcplib.NewTool("mem_inject_session",
		mcplib.WithDescription("Retrieve and render the most relevant prior-session context for a query. Hybrid (BM25 + semantic) by default; degrades to BM25-only when no embedder is active. Project-scoped by default; pass all_projects to widen scope."),
		mcplib.WithString("query", mcplib.Required(), mcplib.Description("Search query for the relevant prior-session context.")),
		mcplib.WithString("project", mcplib.Description("Project id (defaults to CWD resolve).")),
		mcplib.WithBoolean("all_projects", mcplib.Description("When true, retrieve across all projects (default false, project-scoped).")),
		mcplib.WithNumber("max_tokens", mcplib.Description("Token budget for the injected context block (default 2000).")),
	)
}

func handleMemInjectSession(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, err := rootService()
	if err != nil {
		return toolError(err)
	}
	query, err := req.RequireString("query")
	if err != nil {
		return toolError(err)
	}
	projectID, err := projectIDFor(svc, req.GetString("project", ""))
	if err != nil {
		return toolError(err)
	}
	allProjects := req.GetBool("all_projects", false)
	maxTokens := int(req.GetFloat("max_tokens", 2000))

	res, err := session_inject.HybridRetrieve(ctx, svc, query, projectID, allProjects, maxTokens)
	if err != nil {
		return toolError(err)
	}
	block := session_inject.RenderContextBlock(res, projectID)
	return JSONResult(map[string]any{
		"block":         block,
		"items":         res.Items,
		"total_tokens":  res.TotalTokens,
		"degraded":      res.Degraded,
	})
}

// HandleMemInjectSessionForTest exposes the handler for the CLI's session tests.
func HandleMemInjectSessionForTest(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	return handleMemInjectSession(ctx, req)
}
```

Register in `server.go` — add `registerSessionInjectTools(s)` after `registerSessionTools(s)` in both `Start` (line 65) and `NewServer` (line 97).

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./skillgrid-cli/internal/mnemonic/mcp/ -run TestMemInjectSession -v`
Expected: PASS (all 4 tests)

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/mcp/tools_session_inject.go
git add skillgrid-cli/internal/mnemonic/mcp/tools_session_inject_test.go
git add skillgrid-cli/internal/mnemonic/mcp/server.go
git commit -m "feat(session-inject): mem_inject_session MCP tool

Adds the on-demand mem_inject_session MCP tool: hybrid retrieval
(BM25 + semantic, RRF-fused) + context block rendering. Project-scoped
by default; all_projects flag widens scope. Degrades to BM25-only
when no embedder. Registered in Start/NewServer.

[skillgrid-context]
Change: 2026-09-24-mnemonic-session-inject
Phase: apply
"
```

## Global Constraints (repeated for executor reference)

- Go 1.22+ minimum to build.
- No new dependencies without an ADR. (Zero new deps in this plan.)
- Conventional commits only; no AI-attribution trailers.
- Spec-zone changes commit before code-zone changes.
- Privacy: tag-by-default (secrets, full local paths, `private`-tagged excluded).
- Scope: project-scoped by default; cross-project via `all_projects: true`.
- Selection: hybrid (BM25 + semantic, RRF-fused); degrades to BM25-only when no embedder.

## Plan Review

- Verdict: READY FOR EXECUTION
- Findings: 0 Critical, 1 Important (fixed: Task 1 needs `RecentObservations` on `memory.Service` — added as a note in Step 3), 2 Minor (deferred: the CLI's `--continue` wiring for Task 2's auto-prepend is a thin call-site — the blueprint defines `AutoPrepend` and the executor wires it into the existing resume path; exact call-site location depends on the CLI structure and is a 1-line change)
- Reviewed: 2026-09-24
