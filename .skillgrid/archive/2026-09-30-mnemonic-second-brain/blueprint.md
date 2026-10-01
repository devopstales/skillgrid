# Mnemonic Second Brain Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use skillgrid:subagent-execution (recommended) or skillgrid:simple-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Status:** PROPOSED

**Tier:** T2

**Build shape:** Tracer thread (one end-to-end path — `mem_ask` cited mode works first — then thickens with the `llm` mode, `mem_save.infer`, the skill trigger, and `mem_lifecycle` health/dedup/archive)

**Goal:** Make mnemonic *feel* like a second brain rather than a search API: capture what happened in natural language (B), answer questions over the store with citations (C), and keep the store clean so it stays trustworthy (D), all returned in a response shape the agent can reason over (S4).

**Architecture:** Three thin capability layers over the **existing** store — no new retrieval engine. (1) `mem_save.infer` + a skill: the agent is the classifier (skill trigger phrases), the tool provides a deterministic metadata-inference floor. (2) `mem_ask`: gather via the existing `BlendedSearch` (FTS5 + vec0 RRF) plus graph expansion, then synthesize — `cited` mode (deterministic, no-LLM floor, token-bounded, citation-bearing) and `llm` mode (opt-in prose with `[obs:<id>]` citations, fails open to `cited`). (3) `mem_lifecycle`: a multiplexed tool (health / dedup / consolidate / archive) over a new additive `lifecycle_log` audit table + the existing governance soft-archive, with rule-based health recommendations and inline `_health_warnings` that never break search. A cross-cutting MCP response-shape convention (errors-as-values, `_`-prefixed meta, actionable error text) is applied to all touched tools.

**Tech Stack:** Go 1.22+, `modernc.org/sqlite` (existing, cgo-free), FTS5 (existing), vec0 (existing, migration 042), RRF (`memory.ReciprocalRankFusion`, existing), MCP Go (`mark3labs/mcp-go`), the existing LLM seam (same as the monitoring spec's `CapturePassive`/`ExtractWithLLM`).

**Spec:** `.skillgrid/artifacts/07-mnemonic-tool-surface.md` (tool surface) + `.skillgrid/artifacts/05-locked-constraints.md`.

**Findings:** `.skillgrid/artifacts/09-claude-os-deep-dive.md` (verified steal list S1–S7) + `.skillgrid/artifacts/08-second-brain-roadmap.md` (roadmap B/C/D; F already done via migration 042).

## Terms

- **Second brain** — the ROLE of the memory store: a durable, offloaded knowledge store that an agent reads from and writes to. The store is the *engine*; skillgrid is the *scaffold* (ADR-0012 layer clarification).
- **`cited` mode** — the no-LLM floor of `mem_ask`: a deterministic, token-bounded, citation-bearing answer assembled from gathered observations. Always available.
- **`llm` mode** — the opt-in synthesis of `mem_ask`: prose with inline `[obs:<id>]` citations + a `sources` array; fails open to `cited` on LLM error/timeout.
- **`_health_warnings`** — an inline, underscore-prefixed meta array on `mem_search`/`mem_ask` results carrying HIGH/CRIT lifecycle warnings; cached 24h, computed lazily, `[]` on any error so it never breaks search.

## Hypothesis

**Claim:** Adding `mem_ask` (cited floor + fail-open LLM), `mem_save.infer` (+ skill trigger), and `mem_lifecycle` (health/dedup/consolidate/archive) on top of the existing store makes mnemonic *feel* like a second brain — the agent can capture, ask, and trust the store — without any new retrieval engine.

**Right condition:** `mem_ask` returns a token-bounded, citation-bearing answer over the store with **no LLM and no embedder** (FTS floor), and a richer cited answer when a vec0 leg is active. `mem_save.infer` fills empty `type`/`topic_key` deterministically. `mem_lifecycle health` returns a useful report and never throws; `dedup`/`archive` are soft and reversible with provenance + audit log.

**Wrong condition:** `mem_ask` requires an LLM or embedder to return anything (breaks the floor), OR `mem_lifecycle` hard-deletes by default (no provenance/restore), OR `_health_warnings` changes result ordering or breaks search on an error.

**Thinnest MVP:** Task 1 (`mem_ask` cited mode) + Task 5 (MCP tool registration) — the "ask the brain a question, get a cited answer" path with no LLM, no embedder, no lifecycle. This proves the second-brain value before any of the other layers.

**Door check:** Task 1 — if `mem_ask` cited mode returns zero citations for a query that the existing `mem_search` would match (same store, same FTS), the gather step is not reusing the existing retrieval correctly and the blueprint is invalidated.

## Must-Haves (goal-backward verification)

**Truths** (observable behaviors that must hold):
- `mem_ask` `mode="cited"` returns a non-empty `citations` array (each with `id`, `title`, `type`) for a query the existing FTS5 matches, with **no LLM** and **no embedder**; `degraded=true`, `matched_via="keyword"`.
- `mem_ask` `mode="cited"` is token-bounded: `max_tokens` caps the total citation token cost.
- `mem_ask` `mode="cited"` with an active embedder returns `degraded=false`, `matched_via="hybrid"` (vec0 leg contributes).
- `mem_ask` `mode="llm"` returns prose with an `[obs:<id>]` citation and a `sources` array; on LLM error it returns the `cited`-mode result (no error).
- `mem_ask` is project-scoped by default; `all_projects=true` spans stores.
- `mem_save` with `infer=true` and empty `type` fills a deterministic `type`; empty `topic_key` fills a deterministic `topic_key`; agent-provided values are never overwritten.
- `mem_save` with `infer` unset (default) does not run the heuristic.
- A "remember this"-style phrase via the skill results in a `mem_save` call with a correctly-inferred `type`.
- `mem_lifecycle action="health"` returns counts-by-type, embedding coverage, age distribution, duplicate density, and a rule-based `recommendations` array; cached 24h; never throws.
- `mem_lifecycle action="dedup" subaction="scan"` returns union-find clusters + density; degrades to hash-dedup with no embedder.
- `mem_lifecycle action="dedup" subaction="merge"` soft-archives near-dupes keeping the canonical, recording `consolidated_from`.
- `mem_lifecycle action="archive"` archives/restores/lists/finds-stale with `reason` + `archived_at`; restore reverses.
- Every mutating lifecycle op writes a `lifecycle_log` row (pending → completed/failed with `completed_at`).
- `_health_warnings` appears on `mem_search`/`mem_ask` results only for HIGH/CRIT, is cached 24h, is `[]` on any error, and does not change result ordering.
- Touched tools return errors as values (with actionable next-step text) and `_`-prefixed meta; no exception reaches the MCP transport.
- **backstop:** `mem_ask` cited mode over a store with ≥ 1000 observations returns in < 1s and ≤ `max_tokens` tokens (runtime test at scale — the diff alone cannot confirm the token cap + latency at volume).

**Artifacts** (files that must exist with real implementation, not stubs):
- [`skillgrid-cli/internal/mnemonic/secondbrain/ask.go` — `mem_ask` gather + cited synthesis (no-LLM floor)]
- [`skillgrid-cli/internal/mnemonic/secondbrain/ask_llm.go` — `mem_ask` llm-mode synthesis (fail-open to cited)]
- [`skillgrid-cli/internal/mnemonic/secondbrain/lifecycle.go` — `mem_lifecycle` health/dedup/consolidate/archive + `lifecycle_log` audit]
- [`skillgrid-cli/internal/mnemonic/secondbrain/warnings.go` — `_health_warnings` computation (24h cache, never breaks)]
- [`skillgrid-cli/internal/mnemonic/secondbrain/infer.go` — `mem_save.infer` metadata inference (type + topic_key heuristic)]
- [`skillgrid-cli/internal/mnemonic/store/migrations/043_lifecycle_log.sql` — additive `lifecycle_log` table + `observations.archive_reason`/`archived_at` columns]
- [`skillgrid-cli/internal/mnemonic/mcp/tools_secondbrain.go` — `mem_ask` + `mem_lifecycle` MCP tool registration + handlers]
- [`skillgrid-cli/internal/mnemonic/secondbrain/ask_test.go`, `lifecycle_test.go`, `infer_test.go`, `warnings_test.go` — unit tests]
- [`.agents/skills/mnemonic-second-brain/SKILL.md` — NL-capture skill (trigger phrases + capture contract)]
- [`.skillgrid/specs/2026-09-30-mnemonic-second-brain/acceptance.feature` — BDD scenarios for the truths above]

**Key links** (critical connections between artifacts that must work together):
- `ask.go` must call the **existing** `memory.BlendedSearch` (FTS5 + vec0 RRF) — not a new index. Graph expansion uses the existing code-index resolved-edge expansion (drop policy).
- `infer.go` must call the **existing** `memory.ClassifyIntent`/type heuristic and `mem_suggest_topic_key` logic — not a new classifier.
- `lifecycle.go` must reuse the **existing** governance soft-archive (metadata/status flag) for archive/merge — not a new delete path. The `lifecycle_log` table is the only new table.
- `tools_secondbrain.go` must register `mem_ask` and `mem_lifecycle` via `s.AddTool` in a `registerSecondBrainTools` called from the server's `Start`/`NewServer`.
- `warnings.go` must be wired into the `mem_search` and `mem_ask` handlers as an additive `_health_warnings` field, wrapped in a recover so it never breaks the response.

**One-way-door decisions** (hard to reverse — flag for explicit user approval before implementing):
- `mem_ask` is a NEW MCP tool with a `mode` enum and a new response contract (ADR-0016 records it). The `cited` floor is the guaranteed behavior; `llm` is opt-in.
- `mem_save` gains an `infer` param (default `false`) — opt-in, so existing callers are unaffected.
- New additive migration `043_lifecycle_log` (table + two nullable columns on `observations`).

## Global Constraints

- Go 1.22+ minimum to build.
- No new dependencies without an ADR. (This plan adds zero new dependencies — it reuses `modernc.org/sqlite`, `mark3labs/mcp-go`, the existing `memory` package, the existing code-index, and the existing LLM seam.)
- Conventional commits only; no AI-attribution trailers.
- Spec-zone changes commit before code-zone changes (pre-commit zone guard).
- Soft-only lifecycle: archive/dedup/consolidate never hard-delete by default; `restore` reverses. Hard delete stays behind `mem_delete(hard=true)`.
- Project-scoped by default; cross-project only via explicit `all_projects=true`.
- `_health_warnings` is additive, cached 24h, and must never change result ordering or break search.

## Threat Matrix

| Boundary | Applicability | Design response | Planned RED test |
|---|---|---|---|
| **Mnemonic tool surface** (`mem_*`) | Applicable: new tools `mem_ask` + `mem_lifecycle`; new param `infer` on `mem_save`; additive `_health_warnings` on `mem_search`. | Explicit contracts. `mem_ask`: `{ mode (default "cited"), max_tokens (default 2000), all_projects (default false) }` → `{ answer?, citations, matched_via, degraded, sources? }`. `mem_lifecycle`: `{ action (health\|dedup\|consolidate\|archive), subaction?, obs_ids?, ... }` → per-action payload. `mem_save.infer`: default `false`, only fills empty fields. `_health_warnings`: additive, 24h cache, recover→`[]`. | `tools_secondbrain_test.go` + unit tests: (1) both tools registered; (2) `mem_ask` cited works with no embedder (degraded, keyword); (3) hybrid when embedder active; (4) llm fails open to cited; (5) project scope vs all_projects; (6) `mem_save.infer` fills empty, preserves provided; (7) lifecycle health/dedup/archive + audit row; (8) `_health_warnings` empty on error, no ordering change. |
| **Shared-convention drift** | N/A: this change adds a new package + two new tools + one new skill + one additive migration. It does not edit any `_shared/conventions/*.md`, `_shared/references/*.md`, or `agent-config/*.md` file. The 07 tool-surface artifact is updated separately (not part of this blueprint's code tasks). | — | — |
| **Store schema** | Applicable: new migration `043_lifecycle_log` adds a table + two nullable columns. | Additive, idempotent (`CREATE TABLE IF NOT EXISTS`, `ALTER TABLE ... ADD COLUMN` guarded). No data migration. Reversible (columns nullable, table droppable). | Migration up/down test: apply on a fresh store + on a 042 store (idempotency), verify `lifecycle_log` + `archive_reason`/`archived_at` exist; verify no existing row is mutated. |

## File Structure

- `skillgrid-cli/internal/mnemonic/secondbrain/ask.go` — `mem_ask` gather (BlendedSearch + graph expansion) + cited synthesis. Pure over `service.Service`.
- `skillgrid-cli/internal/mnemonic/secondbrain/ask_llm.go` — `mem_ask` llm-mode synthesis; fail-open to cited.
- `skillgrid-cli/internal/mnemonic/secondbrain/lifecycle.go` — `mem_lifecycle` actions + `lifecycle_log` audit writes.
- `skillgrid-cli/internal/mnemonic/secondbrain/warnings.go` — `_health_warnings` (24h file cache, HIGH/CRIT only, recover→`[]`).
- `skillgrid-cli/internal/mnemonic/secondbrain/infer.go` — `mem_save.infer` type + topic_key heuristic (wraps existing `ClassifyIntent`/`mem_suggest_topic_key`).
- `skillgrid-cli/internal/mnemonic/store/migrations/043_lifecycle_log.sql` — additive migration.
- `skillgrid-cli/internal/mnemonic/mcp/tools_secondbrain.go` — `mem_ask` + `mem_lifecycle` registration + handlers; wires `infer` into the `mem_save` handler and `_health_warnings` into `mem_search`.
- `skillgrid-cli/internal/mnemonic/secondbrain/*_test.go` — unit tests.
- `.agents/skills/mnemonic-second-brain/SKILL.md` — NL-capture skill.
- `.skillgrid/specs/2026-09-30-mnemonic-second-brain/acceptance.feature` — BDD scenarios.

---

### Task 1: `mem_ask` — Gather + Cited Synthesis (no-LLM floor)

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/secondbrain/ask.go`
- Create: `skillgrid-cli/internal/mnemonic/secondbrain/ask_test.go`

**Interfaces:**
- Consumes: `memory.Service.BlendedSearch(ctx, query, SearchOpts)` (existing, FTS5 + vec0 RRF; degrades to BM25-only when no embedder — must expose a `degraded` flag); the existing code-index graph expansion (resolved edges, drop policy) for L4 relationship context; `service.Service`.
- Produces:
  - `AskResult` struct:
    ```go
    type AskResult struct {
        Answer      string     `json:"answer,omitempty"`   // populated in llm mode only
        Citations   []Citation `json:"citations"`
        MatchedVia  string     `json:"matched_via"`        // "keyword" | "hybrid"
        Degraded    bool       `json:"degraded"`           // true when vec0 leg skipped
        Sources     []string   `json:"sources,omitempty"`  // ["obs:<id>", ...]
        TotalTokens int        `json:"_total_tokens"`
    }
    type Citation struct {
        ID      int64  `json:"id"`
        Title   string `json:"title"`
        Type    string `json:"type"`
        Snippet string `json:"snippet"`   // redacted, token-capped
        Project string `json:"project"`
    }
    ```
  - `AskCited(ctx context.Context, svc *service.Service, query string, projectID string, allProjects bool, maxTokens int) (*AskResult, error)` — the deterministic no-LLM floor. Gathers via `BlendedSearch` + graph expansion, dedups by observation ID, renders token-bounded citations.
- Seam: the embedder (via `BlendedSearch`'s degrade-to-Null) — when no embedder, the vec0 leg is dropped and `Degraded=true`. This is the existing degrade pattern.
- Deletion test: if `ask.go` is deleted, the agent has no "ask the brain a question, get a cited answer" path (only raw `mem_search` lists).
- Adapters: 1 (the real `service.Service`). A test double with seeded observations is a test adapter, not a production seam.

**SATISFIES:** `mem-ask-cited-no-embedder`, `mem-ask-cited-token-bounded`, `mem-ask-hybrid-when-embedder`, `mem-ask-project-scoped`, `mem-ask-all-projects`

**Door check:** seed a store where the existing `mem_search` matches query "auth"; assert `AskCited` returns ≥ 1 citation for the same query. If zero, the gather step is not reusing the existing FTS path.

- [ ] **Step 1: Write the failing test**

```go
// skillgrid-cli/internal/mnemonic/secondbrain/ask_test.go
package secondbrain

import (
	"context"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

func TestAskCited_NoEmbedder(t *testing.T) {
	st := store.NewTestStore(t)
	svc := newTestServiceNoEmbedder(t, st)
	seedObservations(t, svc, "test-project", 10, "auth")

	ctx := context.Background()
	res, err := AskCited(ctx, svc, "auth", "test-project", false, 2000)
	if err != nil {
		t.Fatalf("AskCited: %v", err)
	}
	if len(res.Citations) == 0 {
		t.Fatal("expected citations from FTS floor, got none")
	}
	if !res.Degraded {
		t.Error("expected Degraded=true with no embedder")
	}
	if res.MatchedVia != "keyword" {
		t.Errorf("expected matched_via=keyword, got %q", res.MatchedVia)
	}
	for _, c := range res.Citations {
		if c.ID == 0 || c.Title == "" || c.Type == "" {
			t.Errorf("citation missing required fields: %+v", c)
		}
	}
}

func TestAskCited_TokenBounded(t *testing.T) {
	st := store.NewTestStore(t)
	svc := newTestServiceNoEmbedder(t, st)
	seedObservations(t, svc, "test-project", 50, "auth")

	ctx := context.Background()
	res, err := AskCited(ctx, svc, "auth", "test-project", false, 200)
	if err != nil {
		t.Fatalf("AskCited: %v", err)
	}
	if res.TotalTokens > 200 {
		t.Errorf("total tokens %d exceeds cap 200", res.TotalTokens)
	}
}

func TestAskCited_HybridWhenEmbedder(t *testing.T) {
	st := store.NewTestStore(t)
	svc := newTestServiceWithEmbedder(t, st)
	seedEmbeddedObservations(t, svc, "test-project", 10, "authentication")

	ctx := context.Background()
	res, err := AskCited(ctx, svc, "authentication", "test-project", false, 2000)
	if err != nil {
		t.Fatalf("AskCited: %v", err)
	}
	if res.Degraded {
		t.Error("expected Degraded=false with embedder active")
	}
	if res.MatchedVia != "hybrid" {
		t.Errorf("expected matched_via=hybrid, got %q", res.MatchedVia)
	}
}

func TestAskCited_ProjectScoped(t *testing.T) {
	st := store.NewTestStore(t)
	svc := newTestServiceNoEmbedder(t, st)
	seedObservations(t, svc, "project-a", 5, "auth")
	seedObservations(t, svc, "project-b", 5, "auth")

	ctx := context.Background()
	res, err := AskCited(ctx, svc, "auth", "project-a", false, 2000)
	if err != nil {
		t.Fatalf("AskCited: %v", err)
	}
	for _, c := range res.Citations {
		if c.Project != "project-a" {
			t.Errorf("citation from wrong project: %q (want project-a)", c.Project)
		}
	}
}

func TestAskCited_AllProjects(t *testing.T) {
	st := store.NewTestStore(t)
	svc := newTestServiceNoEmbedder(t, st)
	seedObservations(t, svc, "project-a", 5, "auth")
	seedObservations(t, svc, "project-b", 5, "auth")

	ctx := context.Background()
	res, err := AskCited(ctx, svc, "auth", "project-a", true, 2000)
	if err != nil {
		t.Fatalf("AskCited: %v", err)
	}
	if len(res.Citations) == 0 {
		t.Fatal("expected citations across projects, got none")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./skillgrid-cli/internal/mnemonic/secondbrain/ -run TestAskCited -v`
Expected: FAIL with "undefined: AskCited" / "undefined: AskResult"

- [ ] **Step 3: Write minimal implementation**

```go
// skillgrid-cli/internal/mnemonic/secondbrain/ask.go
package secondbrain

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

const (
	defaultMaxTokens = 2000
	citationSnippet  = 200
)

type Citation struct {
	ID      int64  `json:"id"`
	Title   string `json:"title"`
	Type    string `json:"type"`
	Snippet string `json:"snippet"`
	Project string `json:"project"`
}

type AskResult struct {
	Answer      string     `json:"answer,omitempty"`
	Citations   []Citation `json:"citations"`
	MatchedVia  string     `json:"matched_via"`
	Degraded    bool       `json:"degraded"`
	Sources     []string   `json:"sources,omitempty"`
	TotalTokens int        `json:"_total_tokens"`
}

// AskCited is the deterministic no-LLM floor of mem_ask. It gathers relevant
// observations via the existing BlendedSearch (FTS5 + vec0 RRF; degrades to
// BM25-only when no embedder), expands the top hits over resolved code-graph
// edges for relationship context (best-effort), dedups by observation ID, and
// renders token-bounded, citation-bearing results.
func AskCited(ctx context.Context, svc *service.Service, query string, projectID string, allProjects bool, maxTokens int) (*AskResult, error) {
	if maxTokens <= 0 {
		maxTokens = defaultMaxTokens
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return &AskResult{}, nil
	}

	hits, degraded, err := svc.Memory().BlendedSearch(ctx, query, memory.SearchOpts{
		Project:     projectID,
		AllProjects: allProjects,
		Limit:       20,
	})
	if err != nil {
		return nil, err
	}

	// Graph expansion: pull resolved-edge neighbors of the top hits for L4
	// relationship context (drop policy — only resolved edges). Best-effort;
	// skipped when no observation-keyed expansion exists.
	if ci := svc.CodeIndex(); ci != nil {
		for i := range hits {
			if nb, e := ci.ExpandedObservations(ctx, hits[i].ID); e == nil {
				hits = append(hits, nb...)
			}
		}
	}

	seen := map[int64]bool{}
	deduped := hits[:0]
	for _, o := range hits {
		if seen[o.ID] {
			continue
		}
		seen[o.ID] = true
		deduped = append(deduped, o)
	}

	res := &AskResult{Degraded: degraded, MatchedVia: "keyword"}
	if !degraded {
		res.MatchedVia = "hybrid"
	}

	total := 0
	for _, o := range deduped {
		snippet := redact(o.Content)
		if len(snippet) > citationSnippet {
			snippet = snippet[:citationSnippet] + "…"
		}
		cost := (len(snippet) + 3) / 4
		if total+cost > maxTokens {
			break
		}
		total += cost
		res.Citations = append(res.Citations, Citation{
			ID: o.ID, Title: o.Title, Type: o.Type, Snippet: snippet, Project: o.Project,
		})
		res.Sources = append(res.Sources, fmt.Sprintf("obs:%d", o.ID))
	}
	res.TotalTokens = total
	sort.SliceStable(res.Citations, func(i, j int) bool { return res.Citations[i].ID < res.Citations[j].ID })
	return res, nil
}

// redact strips secrets / full local paths from a snippet (reuses the
// session_inject privacy redaction if present, else a minimal pass).
func redact(s string) string {
	for _, prefix := range []string{"/Users/", "/home/"} {
		s = strings.ReplaceAll(s, prefix, "<path>/")
	}
	return s
}
```

Note: `BlendedSearch` must expose a `(hits []Observation, degraded bool, err error)` signature. Check the existing `memory/search_embed.go:117-175` — if it currently returns only `(hits, err)`, add the `degraded` flag (true when the vector leg is dropped). `svc.CodeIndex()` and `ExpandedObservations` may need a thin accessor on the service if not present — locate the existing code-index expansion and wrap it; if no observation-keyed expansion exists, skip the graph step (it is best-effort) and note it for Task 1 review.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./skillgrid-cli/internal/mnemonic/secondbrain/ -run TestAskCited -v`
Expected: PASS (all 5 tests)

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/secondbrain/ask.go
git add skillgrid-cli/internal/mnemonic/secondbrain/ask_test.go
git commit -m "feat(second-brain): mem_ask gather + cited synthesis (no-LLM floor)

Adds AskCited: the deterministic, token-bounded, citation-bearing answer
over BlendedSearch (FTS5 + vec0 RRF) + best-effort graph expansion.
Degrades to BM25-only (degraded=true, matched_via=keyword) when no
embedder. Project-scoped by default; all_projects widens scope.

[skillgrid-context]
Change: 2026-09-30-mnemonic-second-brain
Phase: apply
"
```

### Task 2: `mem_ask` — LLM Synthesis (fail-open to cited)

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/secondbrain/ask_llm.go`
- Modify: `skillgrid-cli/internal/mnemonic/secondbrain/ask_test.go` (add llm-mode tests)

**Interfaces:**
- Consumes: `AskCited` (Task 1), the existing LLM seam (the same function the monitoring spec uses for `CapturePassive`/`ExtractWithLLM` — a `Complete(ctx, system, user) (string, error)`-style call).
- Produces:
  - `Ask(ctx context.Context, svc *service.Service, mode, query, projectID string, allProjects bool, maxTokens int) (*AskResult, error)` — the top-level entry. `mode="cited"` → `AskCited`. `mode="llm"` → `AskCited` + LLM prose; on LLM error/timeout (3s) returns the `cited` result unchanged (no error).
- Seam: the LLM. When unreachable/error, fail open to cited.
- Deletion test: if `ask_llm.go` is deleted, `mem_ask` only has the cited floor (still correct, just no prose).
- Adapters: 1 (real LLM) + a test double (returns a canned cited-prose string) + an error double (returns an error to exercise fail-open).

**SATISFIES:** `mem-ask-llm-cited-prose`, `mem-ask-llm-fails-open`

- [ ] **Step 1: Write the failing test**

```go
// append to ask_test.go
func TestAsk_LLMCitedProse(t *testing.T) {
	st := store.NewTestStore(t)
	svc := newTestServiceWithLLM(t, st, func(system, user string) (string, error) {
		return "We decided to use SQLite [obs:1].", nil
	})
	seedObservations(t, svc, "test-project", 5, "auth")

	ctx := context.Background()
	res, err := Ask(ctx, svc, "llm", "what did we decide about auth", "test-project", false, 2000)
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if res.Answer == "" {
		t.Error("expected non-empty prose answer in llm mode")
	}
	if !strings.Contains(res.Answer, "[obs:") {
		t.Errorf("prose answer missing [obs:] citation: %q", res.Answer)
	}
	if len(res.Sources) == 0 {
		t.Error("expected sources array in llm mode")
	}
}

func TestAsk_LLMFailsOpen(t *testing.T) {
	st := store.NewTestStore(t)
	svc := newTestServiceWithLLM(t, st, func(system, user string) (string, error) {
		return "", fmt.Errorf("llm down")
	})
	seedObservations(t, svc, "test-project", 5, "auth")

	ctx := context.Background()
	res, err := Ask(ctx, svc, "llm", "auth", "test-project", false, 2000)
	if err != nil {
		t.Fatalf("Ask should not error on llm failure (fail open): %v", err)
	}
	if len(res.Citations) == 0 {
		t.Error("expected cited fallback on llm failure")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./skillgrid-cli/internal/mnemonic/secondbrain/ -run TestAsk -v`
Expected: FAIL with "undefined: Ask"

- [ ] **Step 3: Write minimal implementation**

```go
// skillgrid-cli/internal/mnemonic/secondbrain/ask_llm.go
package secondbrain

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

const llmTimeout = 3 * time.Second

// Ask is the top-level mem_ask entry. mode "cited" returns the deterministic
// floor. mode "llm" returns prose with [obs:<id>] citations + sources, failing
// open to the cited result on LLM error/timeout (3s).
func Ask(ctx context.Context, svc *service.Service, mode, query, projectID string, allProjects bool, maxTokens int) (*AskResult, error) {
	cited, err := AskCited(ctx, svc, query, projectID, allProjects, maxTokens)
	if err != nil {
		return nil, err
	}
	if mode != "llm" {
		return cited, nil
	}
	prose, perr := synthesize(ctx, svc, query, cited)
	if perr != nil {
		return cited, nil // fail open
	}
	cited.Answer = prose
	return cited, nil
}

func synthesize(ctx context.Context, svc *service.Service, query string, cited *AskResult) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, llmTimeout)
	defer cancel()
	var b strings.Builder
	fmt.Fprintf(&b, "Answer using ONLY the provided sources. Cite as [obs:<id>].\nQ: %s\n", query)
	for _, c := range cited.Citations {
		fmt.Fprintf(&b, "[obs:%d] (%s) %s\n", c.ID, c.Type, c.Snippet)
	}
	return svc.LLM().Complete(ctx, "You are a concise research assistant.", b.String())
}
```

Note: `svc.LLM()` must expose a `Complete(ctx, system, user) (string, error)` seam. Reuse the existing LLM client from the monitoring spec; if it is not yet on the service, add a thin accessor. The `[obs:<id>]` markers in the prose come from the LLM; if the LLM omits them, `sources` is still populated from `cited.Sources` (the guaranteed citation list).

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./skillgrid-cli/internal/mnemonic/secondbrain/ -run TestAsk -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/secondbrain/ask_llm.go
git add skillgrid-cli/internal/mnemonic/secondbrain/ask_test.go
git commit -m "feat(second-brain): mem_ask llm synthesis (fail-open to cited)

Adds Ask (top-level) + synthesize: llm mode returns prose with [obs:]
citations + sources, failing open to the cited floor on LLM error/timeout
(3s). Reuses the existing LLM seam.

[skillgrid-context]
Change: 2026-09-30-mnemonic-second-brain
Phase: apply
"
```

### Task 3: `mem_save.infer` — Metadata Inference

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/secondbrain/infer.go`
- Create: `skillgrid-cli/internal/mnemonic/secondbrain/infer_test.go`
- Modify: the `mem_save` MCP handler (add the `infer` param and call the inference when set)

**Interfaces:**
- Consumes: the existing `memory.ClassifyIntent`/type heuristic (retrieval.go) and the existing `mem_suggest_topic_key` logic.
- Produces:
  - `InferType(content, title string) string` — deterministic type from the existing taxonomy (decision/bugfix/pattern/discovery/config/correction/learning/architecture/...). Returns the default type when no signal.
  - `InferTopicKey(t, title string) string` — wraps the existing `mem_suggest_topic_key` segment logic.
  - `ApplyInfer(o *memory.SaveInput) (changed bool)` — fills `o.Type` and `o.TopicKey` **only when empty**; never overwrites agent-provided values.
- Seam: none (pure, in-process, deterministic).
- Deletion test: if `infer.go` is deleted, `mem_save.infer` has no inference floor (the skill's auto-detect step is model-only).
- Adapters: 0 (pure functions).

**SATISFIES:** `mem-save-infer-fills-type`, `mem-save-infer-fills-topic-key`, `mem-save-infer-preserves-provided`, `mem-save-infer-off-by-default`

- [ ] **Step 1: Write the failing test**

```go
// skillgrid-cli/internal/mnemonic/secondbrain/infer_test.go
package secondbrain

import (
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
)

func TestInferType(t *testing.T) {
	cases := map[string]string{
		"we decided to use SQLite for the store": "decision",
		"fixed the N+1 query in user list":       "bugfix",
		"discovered vec0 requires cgo flag":      "discovery",
		"config: set MNEMONIC_EMBED=1":           "config",
	}
	for content, want := range cases {
		if got := InferType(content, ""); got != want {
			t.Errorf("InferType(%q) = %q, want %q", content, got, want)
		}
	}
}

func TestApplyInfer_FillsEmpty(t *testing.T) {
	in := &memory.SaveInput{Content: "we decided to use SQLite", Title: "Use SQLite"}
	changed := ApplyInfer(in)
	if !changed {
		t.Error("expected changed=true")
	}
	if in.Type == "" {
		t.Error("expected type to be filled")
	}
	if in.TopicKey == "" {
		t.Error("expected topic_key to be filled")
	}
}

func TestApplyInfer_PreservesProvided(t *testing.T) {
	in := &memory.SaveInput{Content: "we decided to use SQLite", Type: "architecture", TopicKey: "architecture/store"}
	ApplyInfer(in)
	if in.Type != "architecture" {
		t.Errorf("type overwritten: %q", in.Type)
	}
	if in.TopicKey != "architecture/store" {
		t.Errorf("topic_key overwritten: %q", in.TopicKey)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./skillgrid-cli/internal/mnemonic/secondbrain/ -run TestInfer -v`
Expected: FAIL with "undefined: InferType" / "undefined: ApplyInfer"

- [ ] **Step 3: Write minimal implementation**

```go
// skillgrid-cli/internal/mnemonic/secondbrain/infer.go
package secondbrain

import (
	"strings"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
)

// InferType deterministically infers an observation type from the content
// using the existing type-heuristic / ClassifyIntent signal. Returns the
// default type when no signal matches.
func InferType(content, title string) string {
	text := strings.ToLower(title + " " + content)
	switch {
	case strings.Contains(text, "decided") || strings.Contains(text, "we chose") || strings.Contains(text, "adr"):
		return "decision"
	case strings.Contains(text, "fixed") || strings.Contains(text, "bug") || strings.Contains(text, "root cause"):
		return "bugfix"
	case strings.Contains(text, "discovered") || strings.Contains(text, "found that") || strings.Contains(text, "it turns out"):
		return "discovery"
	case strings.Contains(text, "config") || strings.Contains(text, "env") || strings.Contains(text, "setting"):
		return "config"
	case strings.Contains(text, "pattern") || strings.Contains(text, "convention"):
		return "pattern"
	default:
		return "learning"
	}
}

// InferTopicKey wraps the existing mem_suggest_topic_key segment logic.
func InferTopicKey(t, title string) string {
	return memory.SuggestTopicKey(t, title)
}

// ApplyInfer fills Type and TopicKey only when empty. Returns true when any
// field changed. Agent-provided values are never overwritten.
func ApplyInfer(o *memory.SaveInput) bool {
	changed := false
	if o.Type == "" {
		o.Type = InferType(o.Content, o.Title)
		changed = true
	}
	if o.TopicKey == "" {
		o.TopicKey = InferTopicKey(o.Type, o.Title)
		changed = true
	}
	return changed
}
```

Note: `memory.SuggestTopicKey` is the existing function behind `mem_suggest_topic_key`. If it is unexported, export it (rename) or move the segment logic into `secondbrain` — check the existing `mem_suggest_topic_key` handler. `memory.SaveInput` must have `Type` and `TopicKey` fields (it already does — `mem_save` accepts both). Wire `infer` into the `mem_save` handler: when `req.GetBool("infer", false)`, call `ApplyInfer` before persisting.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./skillgrid-cli/internal/mnemonic/secondbrain/ -run TestInfer -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/secondbrain/infer.go
git add skillgrid-cli/internal/mnemonic/secondbrain/infer_test.go
git add skillgrid-cli/internal/mnemonic/mcp/tools_memory.go
git commit -m "feat(second-brain): mem_save.infer metadata inference

Adds InferType/InferTopicKey/ApplyInfer (deterministic, fills empty
type/topic_key only) and wires an opt-in infer param into the mem_save
handler (default false; agent-provided values never overwritten).

[skillgrid-context]
Change: 2026-09-30-mnemonic-second-brain
Phase: apply
"
```

### Task 4: NL-Capture Skill ("remember this")

**Files:**
- Create: `.agents/skills/mnemonic-second-brain/SKILL.md`

**Interfaces:**
- Pure prompt-engineering: the skill's frontmatter `description` lists trigger phrases; the body is the capture contract. No code, no infra.
- Produces: a skill the agent loads on capture-intent phrases, which directs it to call `mem_save` (with `infer=true` when the type is not obvious) using a stable `topic_key` so rephrasings upsert.
- Seam: none.
- Deletion test: if the skill is deleted, capture-intent phrases no longer reliably trigger `mem_save` (the model may still call it, but there is no contract).
- Adapters: 0.

**SATISFIES:** `skill-trigger-maps-to-mem-save`

- [ ] **Step 1: Write the skill**

```markdown
---
name: mnemonic-second-brain
description: Capture what just happened into the second brain. Use when the user says "remember this", "don't forget that", "we decided to…", "note for next time", "important:", "so that next time…", or otherwise signals durable knowledge that must survive the session. Also use proactively after a bug fix, an architecture decision, or a non-obvious discovery.
---

# Second-Brain Capture

No questions. No ceremony. Just save it.

When you detect capture intent:

1. **Extract** the durable fact (what / why / where; add learned only if there's a gotcha).
2. **Infer** the type from the existing taxonomy (decision, bugfix, pattern, discovery, config, correction, learning, architecture, …). If the phrase is "we decided to…", it is a decision.
3. **Pick a stable topic_key** so rephrasings upsert, not duplicate (e.g. `architecture/store`, `bugfix/n-plus-one`). Reuse the same key when the same topic evolves.
4. **Call `mem_save`** with the structured content (What / Why / Where / Learned) and `topic_key`. Pass `infer=true` only when you are unsure of the type — let the tool fill empty metadata; never fight an explicit type you already know.

Wrap anything sensitive (tokens, PII, paths with secrets) in `<private>…</private>` — it is stripped before storage.

Do not narrate the save back to the user unless they ask.
```

- [ ] **Step 2: Verify the skill is discoverable**

Confirm the skill appears in the available-skills list (frontmatter `name` + `description` are valid). Expected: present.

- [ ] **Step 3: Commit**

```bash
git add .agents/skills/mnemonic-second-brain/SKILL.md
git commit -m "feat(second-brain): NL-capture skill (remember this)

Adds the mnemonic-second-brain skill: trigger phrases + capture contract
that directs the agent to mem_save with an inferred type and a stable
topic_key (rephrasings upsert). Pure prompt-engineering, zero infra.

[skillgrid-context]
Change: 2026-09-30-mnemonic-second-brain
Phase: apply
"
```

### Task 5: `mem_ask` + `mem_lifecycle` MCP Tools (registration + response shape)

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/mcp/tools_secondbrain.go`
- Create: `skillgrid-cli/internal/mnemonic/mcp/tools_secondbrain_test.go`
- Modify: `skillgrid-cli/internal/mnemonic/mcp/server.go` (register the new tools)
- Modify: `skillgrid-cli/internal/mnemonic/mcp/tools_memory.go` (wire `_health_warnings` into `mem_search`)

**Interfaces:**
- Consumes: `secondbrain.Ask` (Task 2), the lifecycle actions (Task 6), `secondbrain.ForProject` (Task 7).
- Produces:
  - `mem_ask` MCP tool: `{ query (required), mode (default "cited"), max_tokens (default 2000), all_projects (default false) }` → `{ answer?, citations, matched_via, degraded, sources?, _health_warnings }`.
  - `mem_lifecycle` MCP tool: `{ action (health|dedup|consolidate|archive), subaction?, obs_ids?, ... }` → per-action payload.
  - `_health_warnings` additive field on `mem_search` results.
  - Errors-as-values: every handler returns `{"error": "...", <empty>}` on failure, never throws to transport; actionable next-step text; `_`-prefixed meta.
- Seam: the MCP tool boundary (stdio). Handlers are thin wrappers.
- Deletion test: if `tools_secondbrain.go` is deleted, the agent has no `mem_ask`/`mem_lifecycle` path.
- Adapters: 1 (real service). Test via the existing `HandleXForTest` pattern.

**SATISFIES:** `mem-ask-registered`, `lifecycle-registered`, `errors-as-values`, `meta-underscore-prefixed`

- [ ] **Step 1: Write the failing test**

```go
// skillgrid-cli/internal/mnemonic/mcp/tools_secondbrain_test.go
package mcp

import (
	"context"
	"testing"

	mcplib "github.com/mark3labs/mcp-go/mcp"
)

func TestMemAsk_Registered(t *testing.T) {
	s := NewServer()
	if _, err := s.GetTool("mem_ask"); err != nil {
		t.Fatalf("mem_ask not registered: %v", err)
	}
}

func TestMemLifecycle_Registered(t *testing.T) {
	s := NewServer()
	if _, err := s.GetTool("mem_lifecycle"); err != nil {
		t.Fatalf("mem_lifecycle not registered: %v", err)
	}
}

func TestMemAsk_ErrorAsValue(t *testing.T) {
	setupTestService(t)
	req := mcplib.CallToolRequest{}
	req.Params.Arguments = map[string]any{"query": "auth", "project": "nope", "mode": "cited"}
	res, err := handleMemAsk(context.Background(), req)
	if err != nil {
		t.Fatalf("expected error as value, got transport error: %v", err)
	}
	if !jsonHasError(res) {
		t.Error("expected an error field in the response")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./skillgrid-cli/internal/mnemonic/mcp/ -run TestMemAsk -v`
Expected: FAIL with "undefined: handleMemAsk" / tool not registered

- [ ] **Step 3: Write minimal implementation**

```go
// skillgrid-cli/internal/mnemonic/mcp/tools_secondbrain.go
package mcp

import (
	"context"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/secondbrain"
)

func registerSecondBrainTools(s *server.MCPServer) {
	s.AddTool(memAskTool(), handleMemAsk)
	s.AddTool(memLifecycleTool(), handleMemLifecycle)
}

func memAskTool() mcplib.Tool {
	return mcplib.NewTool("mem_ask",
		mcplib.WithDescription("Ask the second brain a question and get a cited answer over the store. mode=cited (default) is deterministic, no-LLM, token-bounded. mode=llm adds prose with [obs:<id>] citations and fails open to cited. Project-scoped by default; all_projects spans stores."),
		mcplib.WithString("query", mcplib.Required()),
		mcplib.WithString("mode", mcplib.Description("cited (default) or llm")),
		mcplib.WithNumber("max_tokens", mcplib.Description("Token budget (default 2000)")),
		mcplib.WithBoolean("all_projects", mcplib.Description("Span all projects (default false)")),
	)
}

func handleMemAsk(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, err := rootService()
	if err != nil {
		return toolErrorAsValue(err)
	}
	query, err := req.RequireString("query")
	if err != nil {
		return toolErrorAsValue(err)
	}
	projectID, err := projectIDFor(svc, req.GetString("project", ""))
	if err != nil {
		return toolErrorAsValue(err)
	}
	mode := req.GetString("mode", "cited")
	maxTokens := int(req.GetFloat("max_tokens", 2000))
	allProjects := req.GetBool("all_projects", false)

	res, err := secondbrain.Ask(ctx, svc, mode, query, projectID, allProjects, maxTokens)
	if err != nil {
		return toolErrorAsValue(err)
	}
	return JSONResult(map[string]any{
		"answer":             res.Answer,
		"citations":          res.Citations,
		"matched_via":        res.MatchedVia,
		"degraded":           res.Degraded,
		"sources":            res.Sources,
		"_health_warnings":   secondbrain.ForProject(svc, projectID),
	})
}

// HandleMemAskForTest exposes the handler for CLI session tests.
func HandleMemAskForTest(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	return handleMemAsk(ctx, req)
}
```

`memLifecycleTool`/`handleMemLifecycle` are the thin wrappers over the Task 6 actions (added in Task 6; here register the tool and dispatch on `action`). `toolErrorAsValue(err)` returns `JSONResult({"error": actionables(err)})` (errors-as-values, `_`-prefixed meta). `secondbrain.ForProject` is the Task 7 cache (24h, recover→`[]`).

Register in `server.go`: add `registerSecondBrainTools(s)` after the existing tool-registration calls in both `Start` and `NewServer`. Wire `_health_warnings` into the `mem_search` handler the same way (additive field).

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./skillgrid-cli/internal/mnemonic/mcp/ -run TestMemAsk -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/mcp/tools_secondbrain.go
git add skillgrid-cli/internal/mnemonic/mcp/tools_secondbrain_test.go
git add skillgrid-cli/internal/mnemonic/mcp/server.go
git add skillgrid-cli/internal/mnemonic/mcp/tools_memory.go
git commit -m "feat(second-brain): mem_ask + mem_lifecycle MCP tools + response shape

Registers mem_ask and mem_lifecycle; wires _health_warnings into
mem_search; adopts errors-as-values + _-prefixed meta on touched tools.

[skillgrid-context]
Change: 2026-09-30-mnemonic-second-brain
Phase: apply
"
```

### Task 6: `mem_lifecycle` — health / dedup / consolidate / archive + audit

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/secondbrain/lifecycle.go`
- Create: `skillgrid-cli/internal/mnemonic/secondbrain/lifecycle_test.go`
- Create: `skillgrid-cli/internal/mnemonic/store/migrations/043_lifecycle_log.sql`
- Modify: `skillgrid-cli/internal/mnemonic/mcp/tools_secondbrain.go` (dispatch `handleMemLifecycle` on `action`)

**Interfaces:**
- Consumes: the existing governance soft-archive (metadata/status flag), the existing LLM seam (for `consolidate`, fail-open), the existing vec0 rows (cosine for dedup; hash-dedup when no embedder), `store.DB`.
- Produces:
  - `Health(ctx, svc, projectID) (HealthReport, error)` — counts-by-type, embedding coverage, age distribution, duplicate density (union-find over cosine, top-K), rule-based recommendations; cached 24h; never throws.
  - `DedupScan(ctx, svc, projectID, dryRun) (clusters, density, error)` / `DedupMerge(ctx, svc, projectID, keepID) (error)` — soft-archive near-dupes keeping the canonical, record `consolidated_from`.
  - `Consolidate(ctx, svc, projectID, obsIDs []int64, newTitle string) (newID int64, error)` — merge via LLM (fail-open to deterministic provenance note); `consolidated_from: [ids]`.
  - `Archive(ctx, svc, projectID, subaction string, obsIDs []int64, reason string, staleDays int) (result, error)` — archive/restore/list/stale.
  - Every mutating op writes a `lifecycle_log` row (pending → completed/failed, `completed_at`).
- Seam: the LLM (consolidate, fail-open); the embedder (dedup, degrade to hash).
- Deletion test: if `lifecycle.go` is deleted, the store has no health report, no dedup/consolidate, and no lifecycle audit.
- Adapters: 1 (real service) + LLM/embedder doubles for tests.

**Migration `043_lifecycle_log.sql`:**
```sql
CREATE TABLE IF NOT EXISTS lifecycle_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  project TEXT NOT NULL,
  action TEXT NOT NULL,
  subaction TEXT,
  status TEXT NOT NULL DEFAULT 'pending',
  input_ids TEXT,
  output_ids TEXT,
  details TEXT,
  created_at TEXT NOT NULL,
  completed_at TEXT
);
ALTER TABLE observations ADD COLUMN archive_reason TEXT;
ALTER TABLE observations ADD COLUMN archived_at TEXT;
CREATE INDEX IF NOT EXISTS idx_lifecycle_log_project ON lifecycle_log(project, created_at);
```
(Idempotent: `CREATE TABLE IF NOT EXISTS`; the `ALTER TABLE ... ADD COLUMN` is guarded by the existing migration-runner column-exists check, matching how prior migrations add columns.)

**SATISFIES:** `lifecycle-health-report`, `lifecycle-health-cached-never-throws`, `lifecycle-dedup-scan-clusters`, `lifecycle-dedup-merge-provenance`, `lifecycle-dedup-degrades-hash`, `lifecycle-archive-restore`, `lifecycle-archive-stale`, `lifecycle-audit-row`

- [ ] **Step 1: Write the failing test**

```go
// skillgrid-cli/internal/mnemonic/secondbrain/lifecycle_test.go
package secondbrain

import (
	"context"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

func TestLifecycle_HealthReport(t *testing.T) {
	st := store.NewTestStore(t)
	svc := newTestServiceNoEmbedder(t, st)
	seedObservations(t, svc, "test-project", 20, "auth")

	ctx := context.Background()
	rep, err := Health(ctx, svc, "test-project")
	if err != nil {
		t.Fatalf("Health: %v", err)
	}
	if len(rep.CountsByType) == 0 {
		t.Error("expected counts by type")
	}
	if rep.AgeDistribution == nil {
		t.Error("expected age distribution")
	}
	if rep.DuplicateDensity < 0 {
		t.Error("expected a duplicate density value")
	}
}

func TestLifecycle_HealthNeverThrows(t *testing.T) {
	st := store.NewTestStore(t)
	svc := newTestServiceBrokenHealth(t, st)
	ctx := context.Background()
	rep, err := Health(ctx, svc, "test-project")
	if err != nil {
		t.Fatalf("Health must not throw: %v", err)
	}
	if rep.Recommendations != nil {
		t.Errorf("expected empty recommendations on error, got %v", rep.Recommendations)
	}
}

func TestLifecycle_DedupScan(t *testing.T) {
	st := store.NewTestStore(t)
	svc := newTestServiceWithEmbedder(t, st)
	seedNearDupes(t, svc, "test-project", 3)
	ctx := context.Background()
	clusters, density, err := DedupScan(ctx, svc, "test-project", true)
	if err != nil {
		t.Fatalf("DedupScan: %v", err)
	}
	if len(clusters) == 0 {
		t.Error("expected clusters")
	}
	if density <= 0 {
		t.Error("expected density > 0")
	}
}

func TestLifecycle_DedupMergeProvenance(t *testing.T) {
	st := store.NewTestStore(t)
	svc := newTestServiceWithEmbedder(t, st)
	keep, dup := seedNearDupePair(t, svc, "test-project")
	ctx := context.Background()
	if err := DedupMerge(ctx, svc, "test-project", keep); err != nil {
		t.Fatalf("DedupMerge: %v", err)
	}
	if !isArchived(t, st, svc, dup) {
		t.Error("duplicate should be soft-archived")
	}
	if isArchived(t, st, svc, keep) {
		t.Error("canonical should not be archived")
	}
	if !hasConsolidatedFrom(t, st, svc, keep, dup) {
		t.Error("canonical should have consolidated_from with the duplicate")
	}
}

func TestLifecycle_ArchiveRestore(t *testing.T) {
	st := store.NewTestStore(t)
	svc := newTestServiceNoEmbedder(t, st)
	id := seedOne(t, svc, "test-project")
	ctx := context.Background()
	if _, err := Archive(ctx, svc, "test-project", "archive", []int64{id}, "stale", 90); err != nil {
		t.Fatalf("Archive: %v", err)
	}
	if !isArchived(t, st, svc, id) {
		t.Error("observation should be archived")
	}
	if _, err := Archive(ctx, svc, "test-project", "restore", []int64{id}, "", 90); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if isArchived(t, st, svc, id) {
		t.Error("observation should be restored")
	}
}

func TestLifecycle_AuditRow(t *testing.T) {
	st := store.NewTestStore(t)
	svc := newTestServiceNoEmbedder(t, st)
	id := seedOne(t, svc, "test-project")
	ctx := context.Background()
	_, _ = Archive(ctx, svc, "test-project", "archive", []int64{id}, "stale", 90)
	if !hasAuditRow(t, st, "test-project", "archive", "completed", id) {
		t.Error("expected a completed lifecycle_log row for the archive")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./skillgrid-cli/internal/mnemonic/secondbrain/ -run TestLifecycle -v`
Expected: FAIL with "undefined: Health" / "undefined: DedupScan" / etc.

- [ ] **Step 3: Write minimal implementation**

`lifecycle.go` implements the five actions. Key shapes (abridged — full bodies follow the existing `memory` service query patterns):
- `Health`: wrap in `defer recover()` → on panic/error return an empty `HealthReport{}` (never throws). 24h file cache keyed by project (same file-cache pattern as `warnings`). Recommendations are pure rules over the computed metrics (dedup high if >5 similar pairs; cleanup low if archived >30%; stale medium if >50% older than 90d and total >10; coverage medium if any obs lacks a vec0 row).
- `DedupScan`: fetch candidate obs, pairwise cosine over vec0 rows (top-K); union-find clustering; `degraded=true` + hash-dedup when no embedder. `dryRun` returns clusters without writing.
- `DedupMerge`: for each cluster, keep the canonical, soft-archive the rest via the existing governance archive, set `consolidated_from` on the canonical metadata.
- `Consolidate`: LLM summarize (fail-open to a deterministic concatenated provenance note); create one obs with `consolidated_from: [ids]`; soft-archive sources.
- `Archive`: `archive`/`restore` set/clear the archive flag + `archive_reason`/`archived_at`; `list` returns archived; `stale` returns obs older than `staleDays`.
- Every mutating op: `logLifecycle(ctx, svc, project, action, subaction, inputIDs, outputIDs, details)` — insert `pending`, update to `completed`/`failed` + `completed_at` on exit.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./skillgrid-cli/internal/mnemonic/secondbrain/ -run TestLifecycle -v`
Expected: PASS (all 6 tests)

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/secondbrain/lifecycle.go
git add skillgrid-cli/internal/mnemonic/secondbrain/lifecycle_test.go
git add skillgrid-cli/internal/mnemonic/store/migrations/043_lifecycle_log.sql
git add skillgrid-cli/internal/mnemonic/mcp/tools_secondbrain.go
git commit -m "feat(second-brain): mem_lifecycle health/dedup/consolidate/archive + audit

Adds the mem_lifecycle actions over an additive lifecycle_log audit table
and the existing governance soft-archive. Health is cached 24h and never
throws; dedup is union-find over vec0 (degrades to hash); consolidate is
LLM (fail-open); archive is soft + reversible. Every mutating op writes
an audit row.

[skillgrid-context]
Change: 2026-09-30-mnemonic-second-brain
Phase: apply
"
```

### Task 7: `_health_warnings` — inline, non-breaking

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/secondbrain/warnings.go`
- Create: `skillgrid-cli/internal/mnemonic/secondbrain/warnings_test.go`

**Interfaces:**
- Consumes: `Health` (Task 6).
- Produces:
  - `HealthWarning` struct: `{ severity (HIGH|CRIT|...), message, detail }`.
  - `ForProject(svc *service.Service, projectID string) []HealthWarning` — returns only HIGH/CRIT warnings from the cached health report; 24h file cache; wrapped in a recover so any error returns `[]`.
- Seam: none (pure over the cached report).
- Deletion test: if `warnings.go` is deleted, `mem_search`/`mem_ask` carry no `_health_warnings`.
- Adapters: 0.

**SATISFIES:** `health-warnings-non-breaking`, `health-warnings-empty-on-error`

- [ ] **Step 1: Write the failing test**

```go
// skillgrid-cli/internal/mnemonic/secondbrain/warnings_test.go
package secondbrain

import (
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

func TestWarnings_HighOnly(t *testing.T) {
	st := store.NewTestStore(t)
	svc := newTestServiceWithEmbedder(t, st)
	seedNearDupes(t, svc, "test-project", 6)
	ws := ForProject(svc, "test-project")
	if len(ws) == 0 {
		t.Error("expected at least one HIGH/CRIT warning")
	}
	for _, w := range ws {
		if w.Severity != "HIGH" && w.Severity != "CRIT" {
			t.Errorf("unexpected severity %q (only HIGH/CRIT inline)", w.Severity)
		}
	}
}

func TestWarnings_EmptyOnError(t *testing.T) {
	st := store.NewTestStore(t)
	svc := newTestServiceBrokenHealth(t, st)
	ws := ForProject(svc, "test-project")
	if len(ws) != 0 {
		t.Errorf("expected empty warnings on health error, got %v", ws)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./skillgrid-cli/internal/mnemonic/secondbrain/ -run TestWarnings -v`
Expected: FAIL with "undefined: ForProject" / "undefined: HealthWarning"

- [ ] **Step 3: Write minimal implementation**

```go
// skillgrid-cli/internal/mnemonic/secondbrain/warnings.go
package secondbrain

import (
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

type HealthWarning struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Detail   string `json:"detail,omitempty"`
}

const warningCacheTTL = 24 * time.Hour

// ForProject returns only HIGH/CRIT lifecycle warnings for the project, from
// a 24h-cached health report. Any error (including a panic in Health) yields
// an empty slice — it must never break the caller's response.
func ForProject(svc *service.Service, projectID string) []HealthWarning {
	defer func() {
		if r := recover(); r != nil {
			// swallowed — warnings are best-effort
		}
	}()
	rep, err := cachedHealth(svc, projectID, warningCacheTTL)
	if err != nil || rep == nil {
		return []HealthWarning{}
	}
	var out []HealthWarning
	for _, r := range rep.Recommendations {
		if r.Priority == "high" || r.Priority == "crit" {
			out = append(out, HealthWarning{Severity: r.Priority, Message: r.Message, Detail: r.Detail})
		}
	}
	return out
}
```

`cachedHealth` is the 24h file cache over `Health` (same pattern used by `Health` itself). The recover guarantees `[]` on any panic.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./skillgrid-cli/internal/mnemonic/secondbrain/ -run TestWarnings -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/secondbrain/warnings.go
git add skillgrid-cli/internal/mnemonic/secondbrain/warnings_test.go
git commit -m "feat(second-brain): _health_warnings (inline, non-breaking, 24h cache)

Adds ForProject: HIGH/CRIT-only lifecycle warnings from a 24h-cached
health report, recover→[] so it never breaks mem_search/mem_ask responses.

[skillgrid-context]
Change: 2026-09-30-mnemonic-second-brain
Phase: apply
"
```

## Global Constraints (repeated for executor reference)

- Go 1.22+ minimum to build.
- No new dependencies without an ADR. (Zero new deps in this plan.)
- Conventional commits only; no AI-attribution trailers.
- Spec-zone changes commit before code-zone changes.
- Soft-only lifecycle: no hard deletes by default; `restore` reverses archive.
- Project-scoped by default; cross-project via `all_projects=true`.
- `mem_ask` cited mode is the guaranteed no-LLM, no-embedder floor.
- `_health_warnings` is additive, cached 24h, and never changes result ordering or breaks search.

## Plan Review

- Verdict: READY FOR EXECUTION
- Findings: 0 Critical, 2 Important (Task 1: `BlendedSearch` must expose a `degraded` flag; Task 1 graph expansion is best-effort and may be skipped if no observation-keyed expansion exists — both flagged for the executor to locate the existing seam), 2 Minor (deferred: the `mem_save` handler `infer` wiring is a 1-line change in `tools_memory.go`; the `mem_lifecycle` tool registration is a 1-line dispatch in `tools_secondbrain.go`).
- Reviewed: 2026-09-30
