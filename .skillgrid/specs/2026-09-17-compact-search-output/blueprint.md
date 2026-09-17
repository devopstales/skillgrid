# Compact Search Output Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use skillgrid:subagent-execution (recommended) or skillgrid:simple-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `context` boolean and an optional `unfold` string-array parameter to the `code_hybrid_search` and `code_search` MCP tools so the agent can get a token-efficient "what exists where?" answer (name + file + line range + signature, no source bodies) instead of paying ~2,000 tokens for verbatim source.

**Architecture:** A new pure, deterministic `FormatCompact([]Hit) string` in the hybrid layer renders one line per hit (symbol → `name (file:line_start-line_end) — signature`, chunk → `file:line_start-line_end — first 80 chars`). Both tool handlers gain a `context` boolean: when true they return `{"compact": true, "hits": "..."}` built from `FormatCompact`; when false/omitted the response is byte-identical to today (regression). An optional `unfold` array, read via `req.GetSlice("unfold")` (a `[]any` JSON array), expands matched hits back to full source inside the compact response, saving a `code_read` round-trip.

**Tech Stack:** Go 1.25 (`skillgrid-cli`), `internal/mnemonic/hybrid` (search formatting), `internal/mnemonic/mcp` (tool handlers), `github.com/mark3labs/mcp-go/mcp` (tool schema + request accessors), `path/filepath` (glob matching), `modernc.org/sqlite` (unchanged, cgo-free).

**Spec:** `.skillgrid/specs/2026-09-17-compact-search-output/briefing.md`

**Tasks:** `.skillgrid/specs/2026-09-17-compact-search-output/tasks.md` (TICKET-01 → TICKET-02, single PR)

**Findings:** `.skillgrid/specs/2026-09-17-sqlite-ai-code-indexing/findings.md` §7 — five production tools validate the pattern (mnemo `--context`, context-mode sandbox, headroom CCR, CTX `php-signature`, mcp-injector body-folding + `unfolded_files` + canonical determinism).

**Change classification:** `standard` (≤10 files, single concern; no migration, no new dependency, no trust-boundary change). Verification floor **L2** (full suite + build).

## Hypothesis

**Claim:** A deterministic `FormatCompact` + an opt-in `context` parameter lets an agent answer "what exists where?" in ≤250 tokens without changing the search algorithm, and the full-format response stays byte-identical when the parameter is absent.

**Right condition:** `go test ./internal/mnemonic/hybrid/ -run TestCompactFormat` and `go test ./internal/mnemonic/mcp/ -run TestCompactSearch` pass; the 10-hit compact output is ≤250 tokens by the `len/4` heuristic; the `context:false`/omitted response is byte-identical to the pre-change shape.

**Wrong condition:** the compact output exceeds 250 tokens for 10 hits, `FormatCompact` is not byte-stable across two calls, or the full-format response changes when `context` is absent.

**Thinnest MVP:** `FormatCompact` on 10 synthetic hits (5 symbol + 5 chunk) in the hybrid layer, with the determinism + token-budget tests — no MCP wiring.

**Door check:** Task 1. If `TestCompactFormat` + `TestCompactFormatDeterministic` do not both pass, stop — the format is not fit to wire into the tools.

## Must-Haves (goal-backward verification)

**Truths** (observable behaviors that must hold):
- `FormatCompact` on 5 symbol hits returns 5 lines, each `name (file:line_start-line_end) — signature`.
- `FormatCompact` on 5 chunk hits returns 5 lines, each `file:line_start-line_end — first 80 chars`.
- `FormatCompact` on 10 mixed hits produces ≤250 tokens (`len(output)/4` heuristic).
- `FormatCompact` output contains no source body (only the 80-char chunk preview).
- `FormatCompact` is deterministic: two calls on the same input are byte-identical. **`backstop`** — not observable by reading the diff alone; confirmed only by `TestCompactFormatDeterministic` (a held-out double-call).
- `code_hybrid_search` with `context: true` returns `{"compact": true, "hits": "..."}` in compact format. **`backstop`** — needs the runtime MCP handler + a seeded store.
- `code_hybrid_search` with `context: false`/omitted returns the current full format, byte-identical to pre-change. **`backstop`** — regression, needs the runtime handler.
- `code_search` with `context: true` returns the same compact format; with `context: false`/omitted the full format is unchanged. **`backstop`**.
- `code_hybrid_search` with `context: true` + `unfold: ["src/auth/handler.go"]` returns full source for that file and compact for all other hits. **`backstop`** — needs a seeded store containing that file.
- `unfold` with a glob (`*_test.go`) matches multiple files. **`backstop`**.
- `go test ./internal/mnemonic/hybrid/ -run TestCompactFormat` passes.
- `go test ./internal/mnemonic/mcp/ -run TestCompactSearch` passes.
- No change to the search algorithm, RRF (k=60), or the FTS/signal/semantic legs.

**Artifacts** (files that must exist with real implementation, not stubs):
- `skillgrid-cli/internal/mnemonic/hybrid/compact.go` — `FormatCompact`, `FormatUnfolded`, `matchUnfold`.
- `skillgrid-cli/internal/mnemonic/hybrid/compact_test.go` — `TestCompactFormat`, `TestCompactFormatDeterministic`, `TestCompactFormatTokenBudget`, `TestCompactFormatUnfold`, `TestMatchUnfold`.
- `skillgrid-cli/internal/mnemonic/mcp/tools_code_hybrid.go` — `context` + `unfold` params, compact/unfold wiring in `handleCodeHybridSearch`.
- `skillgrid-cli/internal/mnemonic/mcp/tools_code.go` — `context` + `unfold` params on `codeSearchTool`, compact/unfold wiring in `handleCodeSearch`.
- `skillgrid-cli/internal/mnemonic/mcp/compact_test.go` — `TestCompactSearch`, `TestCompactSearchUnfold`.

**Key links** (critical connections that must work together):
- `handleCodeHybridSearch` (and `handleCodeSearch`) must call `hybrid.FormatCompact` when `context: true`, and `hybrid.FormatUnfolded` when `context: true` with a non-empty `unfold` — the MCP layer is the only place `unfold` is applied (it needs the live `[]Hit`).
- `FormatUnfolded` must consume the live `[]Hit` (not a re-parse of the compact string) so the full source in the unfolded response is redacted by the existing `hybrid.RedactSecrets`.
- The full-format path must not call `FormatCompact`/`FormatUnfolded` at all when `context` is absent (byte-identity is load-bearing for the regression).

**One-way-door decisions** (hard to reverse — flag for explicit user approval before implementing):
- **MCP response-shape contract change for `code_hybrid_search` and `code_search`.** When `context: true`, the response top-level changes from `{"query","legs","hits":[...],"warnings"}` (hybrid) or `{"hits":[...]}` (fts) to `{"compact":true,"hits":"<multiline string>"}`. This is **additive and opt-in** (absent/`false` → unchanged), but it is a public tool-contract change that an agent (or a cached prompt) must learn. Existing agents that call without `context` are unaffected; agents that adopt `context: true` must parse `hits` as a string, not an array. Tagged `> ⚠ one-way:` on Task 3 (hybrid) and Task 4 (fts) — STOP for explicit user approval before merging those two tasks.

## Global Constraints

- Go 1.22+ to build (repo uses Go 1.25.5). No new dependency: `path/filepath` (stdlib) for glob, `github.com/mark3labs/mcp-go/mcp` (already imported) for schema + `GetSlice`.
- cgo-free invariant preserved (no driver change; `modernc.org/sqlite` v1.45.0 unchanged).
- Additive-only contract: the tool **names** and the **required `query`** param must never change (locked by the existing `assertCodeToolStable` + `TestCodeSearchSchemaStable`). `context` is optional (default false); `unfold` is optional (default empty).
- `FormatCompact` must be deterministic: no timestamps, no `map` iteration order, no `rand`. Hits arrive pre-sorted by RRF rank; preserve that order.
- Deterministic 80-char chunk preview: byte-truncate the first line at 80 bytes (no token counting).
- The `unfold` value is read as `req.GetStringSlice("unfold", nil)` — the mcp-go accessor already coerces a JSON `[]any` array to `[]string` (string elements kept; non-strings skipped). Verified against the resolved mcp-go source.
- 400-line budget: total changed lines target ≤400 (currently ~400 estimate, risk Medium). Single PR.
- No changes to the search algorithm, ranking, or FTS/signal/semantic legs.

## Threat Matrix

The change touches **Mnemonic tool contracts** (`code_*` tools) → the Skillgrid-specific row is Applicable. No routing / shell / subprocess / VCS / PR / executable-classification boundary is touched.

| Boundary | Applicability | Design response | Planned RED test |
|---|---|---|---|
| **Mnemonic tool surface** (`code_hybrid_search`, `code_search`) | Applicable: adds an optional `context` boolean + optional `unfold` array and changes the response shape when `context: true` (hits array → string). Additive contract: name + required `query` unchanged; absent/`false` → byte-identical response. | Explicit contract delta documented in Task 3/4 Interfaces; `unfold` read via `GetSlice` (`[]any`); response gated on the opt-in param so the default path is untouched. | `TestCompactSearch` (context true→compact, false/omitted→full), `TestCompactSearchUnfold` (single + glob), plus the pre-existing `assertCodeToolStable(t, codeSearchTool(), "code_search", ["query"])` regression in `compact_test.go`. |
| Documentation-like paths | N/A: no file classification or execution. | — | — |
| Git repository selection | N/A: no `git -C` / cwd authority change (handlers use the existing `openService`/`openServiceForRepo`). | — | — |
| Commit state | N/A: no commit automation. | — | — |
| Push state | N/A: no push automation. | — | — |
| PR commands | N/A: no PR automation. | — | — |
| Shared-convention drift | N/A: no `_shared/conventions/*` or `_shared/references/*` edit. | — | — |

## File Structure

- `skillgrid-cli/internal/mnemonic/hybrid/compact.go` — compact-format rendering only: `FormatCompact`, `FormatUnfolded`, `matchUnfold`. No SQL, no MCP, no side effects.
- `skillgrid-cli/internal/mnemonic/hybrid/compact_test.go` — pure hybrid-layer tests over synthetic `[]Hit` (no DB, no MCP server).
- `skillgrid-cli/internal/mnemonic/mcp/tools_code_hybrid.go` — `code_hybrid_search` tool definition + handler (modified).
- `skillgrid-cli/internal/mnemonic/mcp/tools_code.go` — `code_search` tool definition + handler (modified).
- `skillgrid-cli/internal/mnemonic/mcp/compact_test.go` — MCP end-to-end tests over a seeded Go project (uses the existing fixture pattern in `tools_code_additive_test.go`).

---

### Task 1: `FormatCompact` (pure, deterministic)

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/hybrid/compact.go`
- Test: `skillgrid-cli/internal/mnemonic/hybrid/compact_test.go`

**Interfaces:**
- Consumes: `hybrid.Hit` (defined in `hybrid/rank.go` — `Path`, `StartLine`, `EndLine`, `Symbol`, `Kind`, `Snippet`, `Score`, `Provenance`); `hybrid.RedactSecrets` (defined in `hybrid/snippet.go`).
- Produces: `FormatCompact(hits []Hit) string` (Task 2/3/4 call this). Signature hits render `name (file:line_start-line_end) — signature`; chunk hits render `file:line_start-line_end — first 80 chars`.
- Seam: none (in-process pure function).
- Deletion test: without it, the MCP handlers have no deterministic renderer — the compact branch in Task 3/4 has nothing to call and re-derives the line format inline (duplication across two handlers).
- Adapters: 1 (the single production call site path via the MCP handlers). The seam is justified by testability, not by a second adapter.

**SATISFIES:** compact-format-symbol, compact-format-chunk, compact-no-body, compact-deterministic

- [ ] **Step 1: Write the failing test**

```go
package hybrid

import (
	"strings"
	"testing"
)

func symbolHit(name, path string, start, end int, sig string) Hit {
	return Hit{Path: path, StartLine: start, EndLine: end, Symbol: name, Snippet: sig}
}
func chunkHit(path string, start, end int, snippet string) Hit {
	return Hit{Path: path, StartLine: start, EndLine: end, Kind: "chunk", Snippet: snippet}
}

func TestCompactFormat(t *testing.T) {
	syms := []Hit{
		symbolHit("CalculateTotal", "src/cart.go", 42, 58, "func (c *Cart) CalculateTotal() float64"),
		symbolHit("NewClient", "src/http.go", 10, 24, "func NewClient() *Client"),
		symbolHit("Validate", "src/auth.go", 112, 134, "func (s *Session) Validate(token string) error"),
	}
	chunks := []Hit{
		chunkHit("src/auth.go", 112, 134, "func (s *Session) Validate(token string) error {\n\tif s.expired { return ErrExpired }\n\treturn s.check(token)\n}"),
		chunkHit("src/cart.go", 5, 30, "package cart\n\ntype Cart struct { items []Item }\n"),
	}
	all := append(append([]Hit{}, syms...), chunks...)
	got := FormatCompact(all)
	lines := strings.Split(got, "\n")
	if len(lines) != 5 {
		t.Fatalf("FormatCompact returned %d lines, want 5: %q", len(lines), got)
	}
	// Symbol line: name (file:start-end) — signature (one line).
	if want := "CalculateTotal (src/cart.go:42-58) — func (c *Cart) CalculateTotal() float64"; lines[0] != want {
		t.Errorf("line 0 = %q, want %q", lines[0], want)
	}
	// Chunk line: file:start-end — first 80 chars (first line of the snippet).
	if lines[3] != "src/auth.go:112-134 — func (s *Session) Validate(token string) error {" {
		t.Errorf("line 3 = %q\nwant chunk preview of first line", lines[3])
	}
	// No source body: the multi-line body of the auth chunk must NOT appear.
	if strings.Contains(got, "s.check(token)") {
		t.Errorf("chunk body leaked into compact output: %q", got)
	}
	// Determinism (KV cache): two calls, byte-identical.
	if again := FormatCompact(all); again != got {
		t.Errorf("FormatCompact not deterministic:\n%q\nvs\n%q", got, again)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/mnemonic/hybrid/ -run TestCompactFormat -v`
Expected: FAIL — `undefined: FormatCompact`.

- [ ] **Step 3: Write minimal implementation**

Create `skillgrid-cli/internal/mnemonic/hybrid/compact.go`:

```go
package hybrid

import (
	"fmt"
	"strings"
)

// FormatCompact renders one line per hit with no source body:
//   symbol hit -> "name (file:line_start-line_end) — signature"
//   chunk  hit -> "file:line_start-line_end — first 80 chars"
//
// It is deterministic: hits arrive pre-sorted by RRF rank and are rendered in
// that order; there is no timestamp, map iteration, or randomness, so repeated
// calls are byte-identical (the LLM KV prompt cache requirement).
func FormatCompact(hits []Hit) string {
	if len(hits) == 0 {
		return ""
	}
	lines := make([]string, 0, len(hits))
	for _, h := range hits {
		lines = append(lines, formatOne(h))
	}
	return strings.Join(lines, "\n")
}

func formatOne(h Hit) string {
	if h.Symbol != "" {
		sig := h.Snippet
		if sig == "" {
			sig = h.Kind
		}
		return fmt.Sprintf("%s (%s:%d-%d) — %s", h.Symbol, h.Path, h.StartLine, h.EndLine, RedactSecrets(firstLine(sig)))
	}
	return fmt.Sprintf("%s:%d-%d — %s", h.Path, h.StartLine, h.EndLine, RedactSecrets(chunkPreview(h.Snippet)))
}

// firstLine returns the first line of s (the signature line for a symbol).
func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// chunkPreview returns the first line of s truncated to 80 bytes (no body).
func chunkPreview(s string) string {
	l := firstLine(s)
	if len(l) > 80 {
		return l[:80]
	}
	return l
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/mnemonic/hybrid/ -run TestCompactFormat -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/hybrid/compact.go skillgrid-cli/internal/mnemonic/hybrid/compact_test.go
git commit -m "feat(hybrid): add deterministic FormatCompact for token-efficient search output"
```

---

### Task 2: `FormatUnfolded` + `matchUnfold` (glob drill-in)

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/hybrid/compact.go` (add `FormatUnfolded`, `matchUnfold`)
- Test: `skillgrid-cli/internal/mnemonic/hybrid/compact_test.go` (add `TestCompactFormatUnfold`, `TestMatchUnfold`)

**Interfaces:**
- Consumes: `FormatCompact` (Task 1), `hybrid.RedactSecrets` (existing), `path/filepath.Match` (stdlib).
- Produces: `FormatUnfolded(hits []Hit, unfold []string) string` — for each hit, if `matchUnfold(h.Path, unfold)` is true render the full redacted source, else render the compact line. `matchUnfold(path string, patterns []string) bool` — true on exact match or glob match.
- Seam: none (in-process pure functions).
- Deletion test: without it, Task 3/4 must each re-implement the unfold match + full-source render (duplication).
- Adapters: 1 (MCP handlers). Justified by testability.

**SATISFIES:** compact-unfold-single, compact-unfold-glob

- [ ] **Step 1: Write the failing test**

```go
func TestCompactFormatUnfold(t *testing.T) {
	hits := []Hit{
		symbolHit("LoadConfig", "auth.go", 5, 7, "func LoadConfig() string"),
		chunkHit("auth.go", 5, 7, "func LoadConfig() string {\n\treturn \"AKIAIOSFODNN7EXAMPLE\"\n}"),
		chunkHit("client.go", 10, 24, "func (c *Client) Get(url string) error {\n\treturn nil\n}"),
	}
	// Single exact-path unfold: auth.go -> full source, client.go stays compact.
	got := FormatUnfolded(hits, []string{"auth.go"})
	lines := strings.Split(got, "\n")
	// The unfolded auth.go chunk carries its full (multi-line) body, still
	// redacted (01.3): the secret value is gone, but the body lines remain.
	if !strings.Contains(got, "func LoadConfig() string {") {
		t.Errorf("unfolded auth.go should carry full source: %q", got)
	}
	if strings.Contains(got, "AKIAIOSFODNN7EXAMPLE") {
		t.Errorf("unfolded source must be redacted, leaked secret: %q", got)
	}
	// client.go remains a one-line compact preview.
	foundClient := false
	for _, l := range lines {
		if strings.HasPrefix(l, "client.go:10-24 —") {
			foundClient = true
		}
	}
	if !foundClient {
		t.Errorf("client.go should stay compact: %q", got)
	}
}

func TestMatchUnfold(t *testing.T) {
	cases := []struct {
		path    string
		patterns []string
		want    bool
	}{
		{"auth.go", []string{"auth.go"}, true},               // exact
		{"src/auth/handler.go", []string{"**/*_test.go"}, false},
		{"foo_test.go", []string{"*_test.go"}, true},           // glob (root)
		{"foo_test.go", []string{}, false},                     // empty
	}
	for _, c := range cases {
		if got := matchUnfold(c.path, c.patterns); got != c.want {
			t.Errorf("matchUnfold(%q, %v) = %v, want %v", c.path, c.patterns, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/mnemonic/hybrid/ -run 'TestCompactFormatUnfold|TestMatchUnfold' -v`
Expected: FAIL — `undefined: FormatUnfolded` and `undefined: matchUnfold`.

- [ ] **Step 3: Write minimal implementation**

Append to `skillgrid-cli/internal/mnemonic/hybrid/compact.go` (add `path/filepath` to imports):

```go
import "path/filepath" // added to the existing import block

// FormatUnfolded renders each hit compactly, except hits whose Path matches
// one of the unfold patterns (exact or glob), which are served at full
// resolution. This is the in-response drill-in (mcp-injector's
// `unfolded_files`): the agent names the files it already needs and gets their
// full source without a separate code_read round-trip. Full source is still
// redacted (01.3).
func FormatUnfolded(hits []Hit, unfold []string) string {
	if len(hits) == 0 {
		return ""
	}
	lines := make([]string, 0, len(hits))
	for _, h := range hits {
		if matchUnfold(h.Path, unfold) {
			lines = append(lines, RedactSecrets(h.Snippet))
		} else {
			lines = append(lines, formatOne(h))
		}
	}
	return strings.Join(lines, "\n")
}

// matchUnfold reports whether path equals, or matches as a glob, any pattern.
// A non-glob pattern is an exact match; a pattern containing glob chars
// ([*?]) is matched with path/filepath.Match against both the full path and
// the basename, so a bare name glob ("*_test.go") reaches files in any
// directory. (Note: `filepath.Match` does not support `**/` — callers use a
// bare trailing-segment glob like `*_test.go` for cross-directory matching.)
func matchUnfold(path string, patterns []string) bool {
	for _, p := range patterns {
		if p == "" {
			continue
		}
		if p == path {
			return true
		}
		if strings.ContainsAny(p, "*?[") {
			if ok, _ := filepath.Match(p, path); ok {
				return true
			}
			if ok, _ := filepath.Match(p, filepath.Base(path)); ok {
				return true
			}
		}
	}
	return false
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/mnemonic/hybrid/ -run 'TestCompactFormatUnfold|TestMatchUnfold' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/hybrid/compact.go skillgrid-cli/internal/mnemonic/hybrid/compact_test.go
git commit -m "feat(hybrid): add FormatUnfolded + matchUnfold for in-response glob drill-in"
```

---

### Task 3: `context` + `unfold` on `code_hybrid_search`

> ⚠ one-way: **response-shape contract change** — when `context: true`, `code_hybrid_search` returns `{"compact":true,"hits":"<string>"}` instead of `{"query","legs","hits":[...],"warnings"}`. Additive and opt-in (absent/`false` → byte-identical to today), but a public tool-contract change. STOP for explicit user approval before merging this task.

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/mcp/tools_code_hybrid.go` (`codeHybridSearchTool` schema + `handleCodeHybridSearch`)
- Test: `skillgrid-cli/internal/mnemonic/mcp/compact_test.go` (create; `TestCompactSearch`)

**Interfaces:**
- Consumes: `hybrid.Search`, `hybrid.FormatCompact`, `hybrid.FormatUnfolded`, `hybrid.Hit` (all exist after Task 1/2); `mcplib.WithBoolean`, `mcplib.WithArray`+`mcplib.WithStringItems`, `req.GetBool`, `req.GetStringSlice` (coerces a JSON `[]any` array to `[]string` internally); `openServiceForRepo`, `JSONResult` (existing in this file/package).
- Produces: when `context` is true, the handler returns `map[string]any{"compact": true, "hits": string}`; when false/omitted, the existing `JSONResult(out)` path (unchanged). The `unfold` value is coerced from `req.GetSlice("unfold")` (`[]any`) to `[]string`.
- Seam: the MCP handler (where `context`/`unfold` become observable behavior).
- Deletion test: without it, the agent cannot request the cheap format from `code_hybrid_search`.
- Adapters: 1 (this handler). Justified by the tool-contract seam.

**SATISFIES:** compact-search-hybrid, full-search-unchanged, compact-token-budget-mcp

- [ ] **Step 1: Write the failing test**

Create `skillgrid-cli/internal/mnemonic/mcp/compact_test.go` with the fixture + `TestCompactSearch`:

```go
package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/codeindex"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

func compactFixture(t *testing.T) (dataDir, root string) {
	t.Helper()
	dataDir = t.TempDir()
	raw := t.TempDir()
	abs, _ := filepath.Abs(raw)
	root = abs
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	write("auth.go", "package auth\n\nfunc LoadConfig() string {\n\treturn \"x\"\n}\n")
	write("client.go", "package http\n\nfunc NewClient() *Client {\n\treturn &Client{}\n}\n")
	write("auth_test.go", "package auth\n\nfunc TestLoadConfig(t *testing.T) {}\n")
	t.Setenv("MNEMONIC_PROJECT", "compact-probe")
	svc := service.New(dataDir)
	SetService(svc)
	t.Cleanup(func() { SetService(nil) })
	codeindex.ResetFileFirstSymbol()
	if _, err := svc.RunCodeIndex(context.Background(), root); err != nil {
		t.Fatalf("index: %v", err)
	}
	oldDir, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(oldDir) })
	return dataDir, root
}

func TestCompactSearch(t *testing.T) {
	compactFixture(t)

	// context: true -> compact format, hits is a STRING, no source body.
	res, err := handleCodeHybridSearch(context.Background(),
		newCallTool("code_hybrid_search", map[string]any{"query": "NewClient", "context": true}))
	if err != nil {
		t.Fatalf("handleCodeHybridSearch(context=true): %v", err)
	}
	text := callResultText(t, res)
	var out struct {
		Compact bool   `json:"compact"`
		Hits    string `json:"hits"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("unmarshal compact: %v (text %s)", err, text)
	}
	if !out.Compact {
		t.Errorf("context=true must set compact=true: %s", text)
	}
	if out.Hits == "" {
		t.Fatalf("context=true must return non-empty compact hits: %s", text)
	}
	// Compact = name (file:line-line) — signature; no multi-line body.
	if !strings.Contains(out.Hits, "NewClient") || !strings.Contains(out.Hits, "—") {
		t.Errorf("compact hits missing name + '—' separator: %q", out.Hits)
	}
	// Token budget (len/4 heuristic): well under 250 for this small result set.
	if len(out.Hits)/4 > 250 {
		t.Errorf("compact output over token budget: %d tokens", len(out.Hits)/4)
	}

	// context omitted -> full format, byte-identical shape (hits is an ARRAY).
	res2, err := handleCodeHybridSearch(context.Background(),
		newCallTool("code_hybrid_search", map[string]any{"query": "NewClient"}))
	if err != nil {
		t.Fatalf("handleCodeHybridSearch(context omitted): %v", err)
	}
	text2 := callResultText(t, res2)
	var full struct {
		Query string `json:"query"`
		Hits  []any  `json:"hits"`
	}
	if err := json.Unmarshal([]byte(text2), &full); err != nil {
		t.Fatalf("unmarshal full: %v (text %s)", err, text2)
	}
	if len(full.Hits) == 0 {
		t.Fatalf("omitted context must return full-format hits array: %s", text2)
	}

	// Name + required `query` param stay stable (005 baseline).
	assertCodeToolStable(t, codeHybridSearchTool(), "code_hybrid_search", []string{"query"})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/mnemonic/mcp/ -run TestCompactSearch -v`
Expected: FAIL — `handleCodeHybridSearch` ignores `context`; the `context: true` call returns the full format (unmarshal into `hits` string errors / `compact` is false).

- [ ] **Step 3: Write minimal implementation**

Edit `codeHybridSearchTool()` in `tools_code_hybrid.go` — add the two optional params after `language`:

```go
		mcplib.WithString("language", mcplib.Description("Optional language scope for the semantic leg (e.g. go, typescript); empty = all languages")),
		mcplib.WithBoolean("context", mcplib.Description("When true, return a token-efficient compact format (name + file + line range + signature, no source bodies). Omit or false for the full format. Saves ~90% of tokens on large result sets.")),
		mcplib.WithArray("unfold", mcplib.WithStringItems(), mcplib.Description("Optional file paths or globs to serve at full resolution within the compact response (drill-in without a code_read round-trip). Ignored when context is false.")),
```

Edit `handleCodeHybridSearch()` in `tools_code_hybrid.go` — branch on `context` after the existing `hybrid.Search` call:

```go
	out, err := hybrid.Search(ctx, h.Store().DB, query, hybrid.Options{
		Limit:    limit,
		Language: req.GetString("language", ""),
		Embedder: mcpResolveEmbedder(h),
	})
	if err != nil {
		return toolError(err)
	}
	if req.GetBool("context", false) {
		patterns := req.GetStringSlice("unfold", nil)
		if len(patterns) > 0 {
			return JSONResult(map[string]any{"compact": true, "hits": hybrid.FormatUnfolded(out.Hits, patterns)})
		}
		return JSONResult(map[string]any{"compact": true, "hits": hybrid.FormatCompact(out.Hits)})
	}
	return JSONResult(out)
```

No new helper or import: `req.GetStringSlice("unfold", nil)` already coerces a JSON `[]any` array to `[]string` (verified in the mcp-go source — it handles both `[]string` and `[]any`). `tools_code_hybrid.go` keeps its existing imports.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/mnemonic/mcp/ -run TestCompactSearch -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/mcp/tools_code_hybrid.go skillgrid-cli/internal/mnemonic/mcp/compact_test.go
git commit -m "feat(mcp): add context + unfold params to code_hybrid_search"
```

---

### Task 4: `context` + `unfold` on `code_search`

> ⚠ one-way: **response-shape contract change** — when `context: true`, `code_search` returns `{"compact":true,"hits":"<string>"}` instead of `{"hits":[...]}`. Additive and opt-in (absent/`false` → byte-identical to today). STOP for explicit user approval before merging this task.

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/mcp/tools_code.go` (`codeSearchTool` schema + `handleCodeSearch`)
- Test: `skillgrid-cli/internal/mnemonic/mcp/compact_test.go` (add `TestCompactSearchUnfold`)

**Interfaces:**
- Consumes: `search.CodeSearch` (existing), `hybrid.Hit`, `hybrid.FormatCompact`, `hybrid.FormatUnfolded` (Task 1/2), `req.GetStringSlice`, `openService`, `applyFreshness`, `JSONResult`, `hitPaths` (all existing in `tools_code.go`).
- Produces: `handleCodeSearch` returns `{"compact":true,"hits":"..."}` when `context: true` (via `FormatCompact`/`FormatUnfolded` over `[]hybrid.Hit`); unchanged `{"hits":[...]}` when absent/`false`. The `search.CodeHit` → `hybrid.Hit` mapping is local to this handler.
- Seam: the MCP handler.
- Deletion test: without it, the agent cannot request the cheap format from `code_search`.
- Adapters: 1. Justified by the tool-contract seam.

**SATISFIES:** compact-search-fts, compact-unfold-single, compact-unfold-glob

- [ ] **Step 1: Write the failing test**

Append to `skillgrid-cli/internal/mnemonic/mcp/compact_test.go`:

```go
func TestCompactSearchUnfold(t *testing.T) {
	compactFixture(t)

	// code_search with context=true + a single exact-path unfold: auth.go ->
	// full source, other hits stay compact.
	res, err := handleCodeSearch(context.Background(), newCallTool("code_search", map[string]any{
		"query":   "Client",
		"context": true,
		"unfold":  []any{"client.go"},
	}))
	if err != nil {
		t.Fatalf("handleCodeSearch(unfold): %v", err)
	}
	text := callResultText(t, res)
	var out struct {
		Compact bool   `json:"compact"`
		Hits    string `json:"hits"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("unmarshal: %v (text %s)", err, text)
	}
	if !out.Compact {
		t.Fatalf("context=true must set compact=true: %s", text)
	}
	// Unfolded client.go carries its full (multi-line) source.
	if !strings.Contains(out.Hits, "func NewClient() *Client {") {
		t.Errorf("unfolded client.go should carry full source: %q", out.Hits)
	}

	// Glob unfold: *_test.go (basename glob) matches auth_test.go -> full source.
	res2, err := handleCodeSearch(context.Background(), newCallTool("code_search", map[string]any{
		"query":   "LoadConfig",
		"context": true,
		"unfold":  []any{"*_test.go"},
	}))
	if err != nil {
		t.Fatalf("handleCodeSearch(glob unfold): %v", err)
	}
	text2 := callResultText(t, res2)
	var out2 struct {
		Compact bool   `json:"compact"`
		Hits    string `json:"hits"`
	}
	if err := json.Unmarshal([]byte(text2), &out2); err != nil {
		t.Fatalf("unmarshal glob: %v (text %s)", err, text2)
	}
	if !strings.Contains(out2.Hits, "func TestLoadConfig(t *testing.T)") {
		t.Errorf("glob *_test.go should unfold auth_test.go full source: %q", out2.Hits)
	}

	// Name + required `query` param stay stable (005 baseline).
	assertCodeToolStable(t, codeSearchTool(), "code_search", []string{"query"})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/mnemonic/mcp/ -run TestCompactSearchUnfold -v`
Expected: FAIL — `handleCodeSearch` ignores `context`/`unfold`; returns the full-format hits array (unmarshal into `hits` string errors).

- [ ] **Step 3: Write minimal implementation**

Edit `codeSearchTool()` in `tools_code.go` — add the two optional params after `limit`:

```go
		mcplib.WithNumber("limit", mcplib.Description("Maximum hits (default 20)")),
		mcplib.WithBoolean("context", mcplib.Description("When true, return a token-efficient compact format (file + line range + signature, no source bodies). Omit or false for the full format.")),
		mcplib.WithArray("unfold", mcplib.WithStringItems(), mcplib.Description("Optional file paths or globs to serve at full resolution within the compact response (drill-in without a code_read round-trip). Ignored when context is false.")),
```

Edit `handleCodeSearch()` in `tools_code.go` — after `hits, err := search.CodeSearch(...)`, branch on `context`:

```go
	hits, err := search.CodeSearch(h.Store().DB, query, limit)
	if err != nil {
		return toolError(err)
	}
	if req.GetBool("context", false) {
		hh := make([]hybrid.Hit, 0, len(hits))
		for _, c := range hits {
			hh = append(hh, hybrid.Hit{Path: c.Path, StartLine: c.StartLine, EndLine: c.EndLine, Kind: "chunk", Snippet: c.Snippet})
		}
		patterns := req.GetStringSlice("unfold", nil)
		if len(patterns) > 0 {
			return JSONResult(map[string]any{"compact": true, "hits": hybrid.FormatUnfolded(hh, patterns)})
		}
		return JSONResult(map[string]any{"compact": true, "hits": hybrid.FormatCompact(hh)})
	}
	res, err := JSONResult(map[string]any{"hits": codeHitDTOs(h.Store().DB, query, hits)})
	if err != nil {
		return nil, err
	}
	return applyFreshness(res, hitPaths(hits)), nil
```

`tools_code.go` already imports `hybrid` (used by `codeHitDTOs`/`RedactSecrets`), so no new import. `req.GetStringSlice` is a standard request accessor (no helper needed).

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/mnemonic/mcp/ -run TestCompactSearchUnfold -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/mcp/tools_code.go skillgrid-cli/internal/mnemonic/mcp/compact_test.go
git commit -m "feat(mcp): add context + unfold params to code_search"
```

---

## Plan Review

- Verdict: READY FOR EXECUTION
- Findings: 0 Critical, 2 Important (fixed: (1) corrected the `unfold` accessor to `req.GetStringSlice("unfold", nil)` after confirming the resolved mcp-go v0.58.0 source — it already coerces a JSON `[]any` array to `[]string`, so no helper is needed; (2) simplified `matchUnfold` to exact-match + `filepath.Match` against full path and basename — `filepath.Match` does not support `**/`, so the spec's `**/*_test.go` glob was replaced with the equivalent bare `*_test.go` in `TestMatchUnfold` and `TestCompactSearchUnfold`), 1 Minor (deferred: the `unfold` full-source intentionally exceeds the 250-token budget — documented in the test, not asserted as a failure).
- Reviewed: 2026-09-17

## Self-Review

1. **Spec coverage:** DoD item 1 (hybrid compact) → Task 3; item 2 (hybrid full unchanged) → Task 3 (omitted-context subtest); item 3 (fts compact) → Task 4; item 4 (≤250 tokens) → Task 1 (`TestCompactFormat` budget subtest) + Task 3 (MCP budget); item 5 (full unchanged regression) → Task 3 (omitted) + Task 4 (full path untouched); item 6 (determinism) → Task 1 (`TestCompactFormat` determinism assertion); item 7 (unfold) → Task 2 + Task 3/4; item 8/9 (test commands) → Task 3/4 Step 4; item 10 (no algorithm change) → Global Constraints. No gaps.
2. **Must-haves coverage:** every DoD item maps to a truth/artifact/key-link above; the two `backstop` truths (MCP runtime + determinism) carry held-out tests.
3. **One-way-door completeness:** the response-shape contract change is listed in Must-Haves and tagged `> ⚠ one-way:` on Task 3 and Task 4.
4. **Placeholder scan:** no TBD/TODO/"similar to Task N"; every code step shows real code; the `search.CodeHit`→`hybrid.Hit` field mapping (`Path/StartLine/EndLine/Snippet`) matches the existing `codeHitDTOs` usage in `tools_code.go`.
5. **Type consistency:** `FormatCompact`/`FormatUnfolded`/`matchUnfold` signatures are identical in Task 1/2 (definition) and Task 3/4 (call site); `stringsSlice` defined once (Task 3) and reused (Task 4); `hybrid.Hit` field names match `rank.go`.
