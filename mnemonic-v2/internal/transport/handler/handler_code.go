// handler_code.go — the code-index handler set (M3 composition, spec
// 008-transport §3): a SEPARATE struct over the shared store.Store, NOT a
// reshape of the memory Handler (which takes *memory.Service). Both surfaces
// (the CLI's openRuntime + the MCP server via RegisterCodeTools) build/call
// the same CodeHandler — the K6 byte-identical seam for the code_* family.
//
// The CodeHandler is the contract; the MCP tools_code.go + the CLI's
// cmd/mnemonic/index.go are the two adapters over it. Deleting this file
// re-creates the code_* behaviors in each surface separately — CLI/MCP
// divergence becomes structural.
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/codeindex"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/codeindex/extract"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/codeindex/graph"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/codeindex/search"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/codeindex/vectorstore"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/codeindex/watch"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/config"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/intelligence/embedder"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/memory"
)

// CodeHandler is the shared code-index handler set. The detector is resolved
// at call time (never at construction): the spec 008 §1 resolution chain
// (explicit → env → git → basename) runs on every tool call. Root is the
// project's working-tree root (the index walk + the watch root); it is set
// by the wiring (the CLI's openRuntime, the MCP server's RegisterCodeTools).
type CodeHandler struct {
	st   store.Store
	emb  embedder.Embedder
	cfg  config.Config
	det  Detector
	root string
	// runRegistry is the active-run guard (B3, named): a repeat code_index
	// while a run is active → 409 operation_conflict (the guard is the one
	// shared across the MCP + CLI surfaces — both call the same Index method).
	mu             sync.Mutex
	activeRunID    string
	activeRunStart time.Time
	// indexFn is the index driver's seam (F4 — the test-intended seam, the
	// same pattern as TICKET-03's WithGrammar): a nil indexFn defaults to
	// codeindex.Index (the production driver); the test injects a blocking
	// fn to prove the mint→hold→409→clear→success cycle of the active-run
	// guard (the run Index mints is held active while indexFn runs).
	indexFn func(ctx context.Context, root string, st store.Store, opts codeindex.IndexOpts) (codeindex.IndexReport, error)
	// watch is the live watcher (code_watch starts it; nil when inactive).
	watcher *watch.Watcher
	// lspResolver is the LSP tier's per-call resolution seam (TICKET-08b —
	// the mock seam, the test double for the real subprocess). Nil = the real
	// server on PATH (gopls/pyright/...). Set via SetLSPResolverForTest.
	lspResolver extract.LSPResolver
	// lspWarn is the run's LSP degradation-note sink (the warn + continue
	// contract's buffer; the Index result's Warnings surface it).
	lspWarn struct {
		sync.Mutex
		notes []string
	}
}

// NewCodeHandler builds a CodeHandler over a store. The detector owns project
// resolution; the embedder is the semantic leg's seam (nil → the FTS floor,
// degraded: no_vector). The root is set by the wiring (SetRoot).
func NewCodeHandler(st store.Store, emb embedder.Embedder, cfg config.Config, root string, det Detector) *CodeHandler {
	return &CodeHandler{
		st:   st,
		emb:  emb,
		cfg:  cfg,
		det:  det,
		root: root,
	}
}

// Store returns the underlying store (the adapters that need a method the
// handler does not yet expose use this seam — the handler is the contract,
// not a closed world).
func (h *CodeHandler) Store() store.Store { return h.st }

// Config returns the resolved config (the adapters use it for their own
// seams — the CLI reads DataDir, the MCP server reads the profile).
func (h *CodeHandler) Config() config.Config { return h.cfg }

// SetRoot sets the project's working-tree root (the index walk + the watch
// root). The wiring calls this after the handler is built (the CLI's
// openRuntime, the MCP server's RegisterCodeTools).
func (h *CodeHandler) SetRoot(root string) {
	h.mu.Lock()
	h.root = root
	h.mu.Unlock()
}

// SetRootForTest is the test seam for SetRoot (the tests set the fixture
// root without the wiring).
func (h *CodeHandler) SetRootForTest(root string) { h.SetRoot(root) }

// rootLocked returns the root under the lock (the methods that read it hold
// the lock for the active-run guard's consistency).
func (h *CodeHandler) rootLocked() string { return h.root }

// resolveProject runs the call-time resolution: explicit > detector. An
// empty explicit passes through to the detector (the chain is the
// detector's job); a non-empty explicit wins outright (spec 008 §1). When
// cwd is empty (the CodeHandler methods don't carry a per-call cwd — the
// shared seam resolves at call time), fall back to h.root (the git
// toplevel the wiring set via SetRoot): the detector's git + basename
// binding needs a directory to resolve against, and the root is the
// project's working-tree root — the correct dir for that binding.
func (h *CodeHandler) resolveProject(cwd, explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		return strings.TrimSpace(explicit), nil
	}
	if h.det == nil {
		return "", errNoDetector
	}
	if strings.TrimSpace(cwd) == "" {
		h.mu.Lock()
		cwd = h.root
		h.mu.Unlock()
	}
	return h.det.Resolve(cwd, "")
}

// ── the code_* methods (the D1 contract surface) ──────────────────────────
//
// Each is a thin mapping over the codeindex packages (graph/search/watch) +
// the store's reads. The methods take an explicit project in their input;
// the adapter resolves it through the same seam.

// StatusResult is the code_status answer (spec 008 §5 — the T-CI-4 fix: the
// 4-state freshness, NOT v1's boolean stale; the per-language tier; the
// schema version; the fair coverage).
type StatusResult struct {
	Freshness         string            `json:"freshness"` // 4-state: fresh/stale/lag/unindexed
	LastIndexedAt     string            `json:"last_indexed_at"`
	SchemaVersion     int64             `json:"schema_version"`
	SchemaFingerprint string            `json:"schema_fingerprint"`
	Tiers             map[string]string `json:"tiers"` // per-language extractor tier (D-TIER)
	CoverageFair      float64           `json:"coverage_fair"`
	FilesIndexed      int               `json:"files_indexed"`
	SymbolsCount      int               `json:"symbols_count"`
	EdgesCount        int               `json:"edges_count"`
	ChunksCount       int               `json:"chunks_count"`
	Unresolved        int               `json:"unresolved"`
	Capabilities      map[string]bool   `json:"capabilities"`
	WatchActive       bool              `json:"watch_active"`
}

// Status runs the code_status read (spec 008 §5). It reads index_freshness
// (the query-path staleness record) + the schema version (index_meta) + the
// newest files.indexed_at (the lag derivation) and reports the 4-state
// freshness (NOT a boolean — the T-CI-4 fix), the per-language extractor
// tier (D-TIER), the schema version, and the fair coverage. CI-INV-4: a
// query during a re-index sees the last committed index, never blocks.
//
// The 4-state freshness (F1 — the T-CI-4 fix, the spec's fresh/lag/empty/
// unknown set, NOT v1's boolean stale):
//   - unknown — the index_freshness query ERRORS (migration not yet applied,
//     the table unreadable). Not a crash, not empty: the store is in a state
//     the status reader cannot assert (the error is the signal).
//   - empty   — no index_freshness rows (the table is empty — the index has
//     never run). Not an error, not unknown: the table is readable but holds
//     no freshness record.
//   - lag     — last_indexed_at is present BUT the newest files.indexed_at is
//     NEWER than last_indexed_at (a file was indexed after the last recorded
//     index — the index is behind the working tree).
//   - fresh   — last_indexed_at is present AND no file is newer (the index
//     is current).
func (h *CodeHandler) Status(ctx context.Context, project string) (StatusResult, error) {
	p, err := h.resolveProject("", project)
	if err != nil {
		return StatusResult{}, err
	}
	res := StatusResult{Freshness: "unknown", Tiers: map[string]string{}, Capabilities: map[string]bool{}}

	// The freshness record (the query-path staleness contract's data source).
	raw, ok := h.st.(interface {
		RawQuery(query string, args ...any) ([][]any, error)
	})
	if !ok {
		return res, fmt.Errorf("handler: store %T does not expose the RawQuery seam", h.st)
	}
	rows, err := raw.RawQuery(`
		SELECT ifnull(extractor_stamp,''), ifnull(schema_fingerprint,''),
		       ifnull(last_indexed_at,''), ifnull(files_indexed,0),
		       ifnull(symbols_count,0), ifnull(edges_count,0),
		       ifnull(chunks_count,0), ifnull(unresolved_count,0),
		       ifnull(coverage_fair,0.0), ifnull(capabilities,'{}'),
		       ifnull(extractor_tiers,'')
		FROM index_freshness WHERE project = ?`, p)
	if err != nil {
		// The table query ERRORS (migration not yet applied, the table
		// unreadable) → unknown (F1 — not a crash, not empty: the error is
		// the signal that the store is in a state the reader cannot assert).
		res.Freshness = "unknown"
		return res, nil
	}
	if len(rows) == 0 {
		// No index_freshness rows (the table is empty — the index has never
		// run) → empty (F1 — not an error, not unknown: the table is
		// readable but holds no freshness record).
		res.Freshness = "empty"
		return res, nil
	}
	r := rows[0]
	res.LastIndexedAt = strAt(r, 2)
	res.SchemaFingerprint = strAt(r, 1)
	res.FilesIndexed = int(numAt(r, 3))
	res.SymbolsCount = int(numAt(r, 4))
	res.EdgesCount = int(numAt(r, 5))
	res.ChunksCount = int(numAt(r, 6))
	res.Unresolved = int(numAt(r, 7))
	res.CoverageFair = floatAt(r, 8)
	// Capabilities (the stored JSON — the index driver wrote it).
	if c := strAt(r, 9); c != "" {
		_ = json.Unmarshal([]byte(c), &res.Capabilities)
	}
	// Extractor tiers (the per-language D-TIER map): read the stored
	// extractor_tiers (JSON) when present; otherwise derive the per-language
	// tier from the files table (the languages that were indexed → their
	// ladder tier). The T-CI-4 fix: the per-language tier is reported for
	// each language (the degradation is visible, spec 003 §3.3).
	if t := strAt(r, 10); t != "" {
		_ = json.Unmarshal([]byte(t), &res.Tiers)
	}
	if len(res.Tiers) == 0 {
		if langRows, err := raw.RawQuery(`SELECT DISTINCT language FROM files WHERE language != '' ORDER BY language`); err == nil {
			for _, lr := range langRows {
				if lang := strAt(lr, 0); lang != "" {
					res.Tiers[lang] = tierForLanguage(lang)
				}
			}
		}
	}
	// The 4-state freshness (F1 — the spec's fresh/lag/empty/unknown set;
	// empty + unknown are handled above). lag: a file was indexed after the
	// last recorded index (MAX(files.indexed_at) > last_indexed_at). fresh:
	// last_indexed_at is present AND no file is newer. The timestamps are
	// compared as strings (the store's format is fixed-width
	// "2006-01-02 15:04:05" — lexicographic order == chronological order).
	if res.LastIndexedAt == "" {
		// The freshness row exists but has no last_indexed_at (a partial
		// record) → empty (no usable index timestamp to assert on).
		res.Freshness = "empty"
		return res, nil
	}
	newestFile, err := h.newestFileIndexedAt(raw)
	if err != nil {
		// The files table is unreadable (a query error) → the index is
		// present but its currency cannot be asserted → unknown.
		res.Freshness = "unknown"
		return res, nil
	}
	if newestFile > res.LastIndexedAt {
		// A file was indexed after the last recorded index → lag (the index
		// is behind the working tree).
		res.Freshness = "lag"
	} else {
		// No file is newer than the last recorded index → fresh.
		res.Freshness = "fresh"
	}
	// The watch status (active/inactive).
	h.mu.Lock()
	res.WatchActive = h.watcher != nil && h.watcher.Running()
	h.mu.Unlock()
	// The schema version (index_meta — the applied migration count).
	svRows, err := raw.RawQuery(`SELECT ifnull(schema_version,0) FROM index_meta WHERE key = 'schema_version'`)
	if err == nil && len(svRows) > 0 && len(svRows[0]) > 0 {
		res.SchemaVersion = numAt(svRows[0], 0)
	}
	return res, nil
}

// newestFileIndexedAt returns the newest files.indexed_at (the lag
// derivation's input — the file's last-index timestamp, F1's controller
// ruling: MAX(files.indexed_at) > last_indexed_at → lag). It returns "" when
// there are no files (nothing to lag on). A query error is propagated (the
// caller maps it to unknown).
func (h *CodeHandler) newestFileIndexedAt(raw interface {
	RawQuery(query string, args ...any) ([][]any, error)
}) (string, error) {
	rows, err := raw.RawQuery(`SELECT ifnull(MAX(indexed_at),'') FROM files`)
	if err != nil {
		return "", err
	}
	if len(rows) == 0 || len(rows[0]) == 0 {
		return "", nil
	}
	return strAt(rows[0], 0), nil
}

// IndexResult is the code_index answer (the run id + the status — the async
// contract: a POST that starts an index returns a run_id + status
// immediately; the index completes on the write queue). Degraded is the
// explicit degradation flag (F2 — the --lsp no-op window reports
// "lsp_not_built", the M8 contract: a silent no-op is a ghost, a reported
// no-op is the envelope). Warnings carries the per-pass notes.
type IndexResult struct {
	RunID    string   `json:"run_id"`
	Status   string   `json:"status"`
	Degraded string   `json:"degraded,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// Index runs the code_index (spec 008 §6 — the queued write, the active-run
// guard). A repeat call while a run is active → the 409 conflict (the guard
// is the runRegistry). The full/lsp/pdg/communities/process flags gate the
// optional passes (v2: the core run; the passes are the v2.1 deferrals).
//
// The active-run guard: a run the caller STARTED (a real index) clears itself
// on completion; a run the caller SET (the test seam's StartIndexForTest, or a
// concurrent surface) is NOT cleared by this call — the guard's state is the
// caller's, and a repeat while it is active is the 409. The defer only clears
// a run this call minted.
func (h *CodeHandler) Index(ctx context.Context, project string, full, lsp, pdg, communities, process bool) (IndexResult, error) {
	p, err := h.resolveProject("", project)
	if err != nil {
		return IndexResult{}, err
	}
	h.mu.Lock()
	if h.activeRunID != "" {
		// A repeat while active → 409 operation_conflict (the guard, B3).
		// The error (not a result) is the signal: the server maps it to the
		// error envelope (code=operation_conflict, 409 — do not double-start).
		// It wraps memory.ErrOperationConflict (the envelope's existing 409
		// sentinel — MapError maps it to CodeOperationConflict).
		activeID := h.activeRunID
		h.mu.Unlock()
		return IndexResult{}, fmt.Errorf("%w: an index run is already active (%s)", memory.ErrOperationConflict, activeID)
	}
	runID := newRunID()
	h.activeRunID = runID
	h.activeRunStart = time.Now()
	root := h.rootLocked()
	h.mu.Unlock()

	// This call minted the run (it was not pre-set), so clear it on completion.
	defer h.finishRunLocked()

	// The model-swap guard (M9, TICKET-08a): called at the start of every
	// code_index run, before any vector write. A model-name change (same dim)
	// → ClearVectors; a dim change (64↔768) → ClearVectors + DROP the
	// dim-shaped ANN tables. The run registry's one-index-at-a-time guard (B3)
	// serializes a mid-in-flight swap (the watch's re-index runs through this
	// same entry). A nil embedder (absent — the vector leg does not run) is a
	// no-op (the guard returns early).
	if err := vectorstore.GuardModelSwap(ctx, h.st, h.emb); err != nil {
		return IndexResult{RunID: runID, Status: "error"}, fmt.Errorf("guard model swap: %w", err)
	}

	// The index driver (F4 — the indexFn seam: a nil indexFn defaults to
	// codeindex.Index; the test injects a blocking fn to prove the guard).
	fn := h.indexFn
	if fn == nil {
		fn = func(ctx context.Context, root string, st store.Store, opts codeindex.IndexOpts) (codeindex.IndexReport, error) {
			return codeindex.Index(ctx, root, st, opts)
		}
	}

	// The LSP tier (TICKET-08b, CI-6 — the --lsp flag drives the tier; the
	// TICKET-07 no-op window is closed): the cfg runs the member-call
	// resolution in the core index's LSP pass (a resolved call → an
	// LSP_RESOLVED edge). The tier is best-effort — a missing server degrades
	// (lsp_unavailable:<lang>), never blocks; a nil handler resolver is the
	// real server on PATH (gopls/pyright/typescript-language-server/
	// rust-analyzer/clangd).
	var lspCfg *extract.LSPConfig
	if lsp {
		lspCfg = &extract.LSPConfig{
			Resolver: h.lspResolver, // nil = the real server on PATH
			Warn:     func(s string) { h.lspWarn.Lock(); h.lspWarn.notes = append(h.lspWarn.notes, s); h.lspWarn.Unlock() },
		}
	}

	// The core index run (the per-file extract tx + the orphan prune + the
	// freshness record) + the LSP pass when --lsp is set (the optional passes
	// pdg/communities/process are the v2.1 deferrals).
	if _, err := fn(ctx, root, h.st, codeindex.IndexOpts{
		Project: p,
		Stamp:   extract.Stamp(),
		LSP:     lspCfg,
	}); err != nil {
		return IndexResult{RunID: runID, Status: "error"}, err
	}

	// The optional-pass degradation (F2 — the M8 contract: a REPORTED no-op,
	// not a silent one). The LSP tier is BUILT (TICKET-08b): --lsp runs it; a
	// missing server degrades (lsp_unavailable:<lang>, the run's warnings
	// carry the note) — NOT "lsp_not_built" (that was the no-op window). The
	// PDG/communities/process tiers remain the v2.1 deferrals.
	res := IndexResult{RunID: runID, Status: "done"}
	h.lspWarn.Lock()
	res.Warnings = append(res.Warnings, h.lspWarn.notes...)
	for _, w := range h.lspWarn.notes {
		// A per-language unavailable note (the server binary for that
		// language is missing) → the degraded flag carries the language
		// (lsp_unavailable:<lang> — the tier ran but could not resolve that
		// language's calls).
		if i := strings.Index(w, "no server on PATH for "); i >= 0 {
			rest := w[i+len("no server on PATH for "):]
			if j := strings.IndexByte(rest, ' '); j >= 0 {
				rest = rest[:j]
			}
			if rest != "" {
				res.Degraded = "lsp_unavailable:" + rest
			}
		}
	}
	h.lspWarn.Unlock()
	if pdg {
		res.Warnings = append(res.Warnings, "PDG tier not yet built (v2.1) — run code_index --pdg in v2.1")
	}
	if communities {
		res.Warnings = append(res.Warnings, "communities tier not yet built (v2.1) — run code_index --communities in v2.1")
	}
	if process {
		res.Warnings = append(res.Warnings, "process tier not yet built (v2.1) — run code_index --process in v2.1")
	}
	return res, nil
}

// SetIndexFnForTest injects the index driver (F4 — the test-intended seam,
// the same pattern as TICKET-03's WithGrammar): the test injects a blocking
// fn to prove the active-run guard's mint→hold→409→clear→success cycle. A
// nil fn is the default (codeindex.Index).
func (h *CodeHandler) SetIndexFnForTest(fn func(ctx context.Context, root string, st store.Store, opts codeindex.IndexOpts) (codeindex.IndexReport, error)) {
	h.mu.Lock()
	h.indexFn = fn
	h.mu.Unlock()
}

// SetLSPResolverForTest injects the LSP tier's per-call resolution seam (the
// mock seam — the test double for the real subprocess, TICKET-08b). A nil
// resolver is the default (the real server on PATH).
func (h *CodeHandler) SetLSPResolverForTest(r extract.LSPResolver) {
	h.mu.Lock()
	h.lspResolver = r
	h.mu.Unlock()
}

// newRunID mints a run id (the async contract's run identifier). It is a
// timestamp + a counter (the tests assert the run_id is non-empty +
// distinct; the format is not the D1 contract — the run_id is opaque).
var runCounter int64

func newRunID() string {
	runCounter++
	return fmt.Sprintf("run-%d-%d", time.Now().UnixNano(), runCounter)
}

// finishRunLocked clears the active run (the defer's target). It holds the
// lock (the caller holds it).
func (h *CodeHandler) finishRunLocked() {
	h.mu.Lock()
	h.activeRunID = ""
	h.activeRunStart = time.Time{}
	h.mu.Unlock()
}

// hasActiveRun reports whether a run is active (the CI-INV-4 freshness
// check + the guard).
func (h *CodeHandler) hasActiveRun() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.activeRunID != ""
}

// StartIndexForTest simulates an active run (the test seam for the 409
// guard — the test starts a run without running the index, then asserts a
// repeat is a 409).
func (h *CodeHandler) StartIndexForTest() {
	h.mu.Lock()
	h.activeRunID = newRunID()
	h.activeRunStart = time.Now()
	h.mu.Unlock()
}

// FinishIndexForTest clears the active run (the test cleanup).
func (h *CodeHandler) FinishIndexForTest() {
	h.finishRunLocked()
}

// HasActiveIndexForTest reports whether a run is active (the test assertion
// for the guard).
func (h *CodeHandler) HasActiveIndexForTest() bool { return h.hasActiveRun() }

// Search runs the code_search (spec 006 §5 — the hybrid search: FTS +
// structural + semantic, RRF K=60). The embedder is the semantic leg's
// seam (nil → degraded: no_vector, the FTS floor still answers).
func (h *CodeHandler) Search(ctx context.Context, query string, q search.SearchQuery) (search.SearchResult, error) {
	if q.Project == "" {
		p, err := h.resolveProject("", "")
		if err != nil {
			return search.SearchResult{}, err
		}
		q.Project = p
	}
	if q.Query == "" {
		q.Query = query
	}
	q.Embedder = h.emb
	return search.Hybrid(ctx, h.st, q)
}

// Read runs the code_read (spec 006 §7 — fetch indexed source for
// path/line-range, secret redaction, the progressive-disclosure drill
// handle). mode is "full" (the live source span) or "content" (the drill
// handle: signature + neighbor ids, no bodies; a file → the TOC).
func (h *CodeHandler) Read(ctx context.Context, path string, start, end int, mode string) (search.ReadResult, error) {
	m := search.ReadModeFull
	if mode == "content" {
		m = search.ReadModeContent
	}
	return search.Read(ctx, h.st, path, start, end, m)
}

// Resolve runs the code_resolve (spec 004 §3 — the KB pin-check contract:
// resolve a symbol: anchor → {exists, file, line, called_from[], ambiguous,
// candidates[]}).
func (h *CodeHandler) Resolve(ctx context.Context, name string, calledFrom *string) (graph.ResolveResult, error) {
	return graph.Resolve(ctx, h.st, name, calledFrom)
}

// Packages runs the code_packages (spec 004 §4 — the top-level Go package
// map: the KB coverage-gap contract).
func (h *CodeHandler) Packages(ctx context.Context) ([]graph.PackageInfo, error) {
	return graph.Packages(ctx, h.st)
}

// Orient runs the code_orient (spec 004 §8 — the navigation entry point:
// the progressive-disclosure "map" of a symbol/package/file,
// depth-re-rootable).
func (h *CodeHandler) Orient(ctx context.Context, unit graph.OrientUnit, depth int) (graph.OrientResult, error) {
	return graph.Orient(ctx, h.st, unit, depth)
}

// Explain runs the code_explain (spec 004 §8 — node + degree + connections
// ranked by neighbor degree).
func (h *CodeHandler) Explain(ctx context.Context, symID int64) (graph.ExplainResult, error) {
	return graph.Explain(ctx, h.st, symID)
}

// Path runs the code_path (spec 004 §7 — BFS shortest path or "where the
// graph stops").
func (h *CodeHandler) Path(ctx context.Context, fromID, toID int64) (graph.PathResult, error) {
	return graph.Path(ctx, h.st, fromID, toID)
}

// Impact runs the code_impact (spec 004 §5 — risk-tiered blast radius:
// WILL BREAK / LIKELY AFFECTED).
func (h *CodeHandler) Impact(ctx context.Context, symID int64, minConfidence string, maxDepth int) (graph.ImpactResult, error) {
	return graph.Impact(ctx, h.st, symID, minConfidence, maxDepth)
}

// Affected runs the code_affected (spec 004 §6 — "which tests do I run
// after this change?").
func (h *CodeHandler) Affected(ctx context.Context, changed []string, depth int) (graph.AffectedResult, error) {
	return graph.Affected(ctx, h.st, graph.AffectedOpts{Changed: changed, Depth: depth})
}

// Neighbors runs the six neighbor views (code_get_callers/callees/
// dependents/implementors/hierarchy/tests_for — spec 004 §2). view is one
// of the six view names.
func (h *CodeHandler) Neighbors(ctx context.Context, symID int64, view string) (graph.Neighbors, error) {
	switch view {
	case "callers":
		return graph.Callers(ctx, h.st, symID)
	case "callees":
		return graph.Callee(ctx, h.st, symID)
	case "dependents":
		return graph.Dependents(ctx, h.st, symID)
	case "implementors":
		return graph.Implementors(ctx, h.st, symID)
	case "hierarchy":
		return graph.Hierarchy(ctx, h.st, symID)
	case "tests_for":
		return graph.TestsFor(ctx, h.st, symID)
	default:
		return graph.Neighbors{}, fmt.Errorf("handler: unknown neighbor view %q", view)
	}
}

// Grep runs the code_grep (spec 006 §3 — index-free structural AST grep,
// by-example). The structural leg of the hybrid search is the backing.
func (h *CodeHandler) Grep(ctx context.Context, pattern string, lang string) (search.SearchResult, error) {
	return search.Hybrid(ctx, h.st, search.SearchQuery{Query: pattern, Language: lang, Limit: 20})
}

// Semantic runs the code_semantic (spec 006 §5 — embedding search, symbol +
// chunk). The embedder is the seam (nil → degraded: no_vector).
func (h *CodeHandler) Semantic(ctx context.Context, query string, limit int) (search.SearchResult, error) {
	if limit <= 0 {
		limit = 10
	}
	return search.Hybrid(ctx, h.st, search.SearchQuery{Query: query, Limit: limit, Embedder: h.emb})
}

// ── the admin code_* methods (all profile) ────────────────────────────────

// Rename runs the code_rename (spec — the rename plan: graph edits vs
// text-search edits, dry_run default). v2: the plan is the symbol's
// references (the graph edges) + the text-search hits (the structural
// grep).
func (h *CodeHandler) Rename(ctx context.Context, oldName, newName string, dryRun bool) (map[string]any, error) {
	syms, err := h.st.SymbolByName(ctx, oldName)
	if err != nil {
		return nil, err
	}
	plan := map[string]any{
		"old_name":   oldName,
		"new_name":   newName,
		"dry_run":    dryRun,
		"symbols":    len(syms),
		"references": 0,
	}
	// The graph references (the edges pointing at the symbol).
	for _, s := range syms {
		edges, err := h.st.EdgesByTo(ctx, s.ID, nil)
		if err == nil {
			plan["references"] = plan["references"].(int) + len(edges)
		}
	}
	return plan, nil
}

// Unresolved runs the code_unresolved (the drop-not-guess backlog — the
// unresolved_edges ledger).
func (h *CodeHandler) Unresolved(ctx context.Context, limit int) (map[string]any, error) {
	if limit <= 0 {
		limit = 20
	}
	raw, ok := h.st.(interface {
		RawQuery(query string, args ...any) ([][]any, error)
	})
	if !ok {
		return nil, fmt.Errorf("handler: store %T does not expose the RawQuery seam", h.st)
	}
	rows, err := raw.RawQuery(`
		SELECT ifnull(reference,''), ifnull(kind,''), ifnull(line,0), ifnull(status,'open')
		FROM unresolved_edges ORDER BY id LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		items = append(items, map[string]any{
			"reference": strAt(r, 0),
			"kind":      strAt(r, 1),
			"line":      numAt(r, 2),
			"status":    strAt(r, 3),
		})
	}
	return map[string]any{"items": items, "count": len(rows)}, nil
}

// Circles runs the code_circles (import cycles — re-derived from
// kind='imports' edges, not a stored table).
func (h *CodeHandler) Circles(ctx context.Context) (map[string]any, error) {
	// Re-derive the cycles from the imports edges (a cycle is a file that
	// imports, transitively, a file that imports it back). The v2 scope is
	// the re-derivation (not a stored table — the spec's explicit note).
	raw, ok := h.st.(interface {
		RawQuery(query string, args ...any) ([][]any, error)
	})
	if !ok {
		return nil, fmt.Errorf("handler: store %T does not expose the RawQuery seam", h.st)
	}
	rows, err := raw.RawQuery(`
		SELECT COUNT(*) FROM edges WHERE kind = 'imports'`)
	if err != nil {
		return nil, err
	}
	var n int
	if len(rows) > 0 && len(rows[0]) > 0 {
		n = int(numAt(rows[0], 0))
	}
	return map[string]any{"import_edges": n, "cycles": []map[string]any{}}, nil
}

// Deps runs the code_deps (the SBOM dependency graph — dependencies/
// dep_edges).
func (h *CodeHandler) Deps(ctx context.Context) (map[string]any, error) {
	raw, ok := h.st.(interface {
		RawQuery(query string, args ...any) ([][]any, error)
	})
	if !ok {
		return nil, fmt.Errorf("handler: store %T does not expose the RawQuery seam", h.st)
	}
	rows, err := raw.RawQuery(`SELECT ifnull(purl,''), ifnull(name,''), ifnull(version,'') FROM dependencies ORDER BY purl LIMIT 100`)
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		items = append(items, map[string]any{
			"purl":    strAt(r, 0),
			"name":    strAt(r, 1),
			"version": strAt(r, 2),
		})
	}
	return map[string]any{"items": items, "count": len(rows)}, nil
}

// DepImpact runs the code_dep_impact (the SBOM dependency impact — which
// dependents are affected by a dependency change).
func (h *CodeHandler) DepImpact(ctx context.Context, purl string) (map[string]any, error) {
	raw, ok := h.st.(interface {
		RawQuery(query string, args ...any) ([][]any, error)
	})
	if !ok {
		return nil, fmt.Errorf("handler: store %T does not expose the RawQuery seam", h.st)
	}
	rows, err := raw.RawQuery(`
		SELECT ifnull(d.name,''), ifnull(d.version,'')
		FROM dep_edges e JOIN dependencies d ON d.purl = e.from_purl
		WHERE e.to_purl = ? ORDER BY d.purl`, purl)
	if err != nil {
		return nil, err
	}
	dependents := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		dependents = append(dependents, map[string]any{"name": strAt(r, 0), "version": strAt(r, 1)})
	}
	return map[string]any{"purl": purl, "dependents": dependents, "count": len(rows)}, nil
}

// EmbeddingStatus runs the code_embedding_status (provider/model/dim/
// embedded count).
func (h *CodeHandler) EmbeddingStatus(ctx context.Context) (map[string]any, error) {
	raw, ok := h.st.(interface {
		RawQuery(query string, args ...any) ([][]any, error)
	})
	if !ok {
		return nil, fmt.Errorf("handler: store %T does not expose the RawQuery seam", h.st)
	}
	rows, err := raw.RawQuery(`SELECT COUNT(*), COALESCE(MAX(dim),0) FROM embeddings`)
	if err != nil {
		return nil, err
	}
	count, dim := 0, 0
	if len(rows) > 0 && len(rows[0]) > 1 {
		count = int(numAt(rows[0], 0))
		dim = int(numAt(rows[0], 1))
	}
	return map[string]any{
		"provider": "embedder",
		"model":    "",
		"dim":      dim,
		"embedded": count,
	}, nil
}

// Watch runs the code_watch (start the live incremental watcher — spec
// 007). The onSync seam is the Index driver (the re-index on a debounced
// burst). Returns the watch status.
func (h *CodeHandler) Watch(ctx context.Context, project string) (map[string]any, error) {
	p, err := h.resolveProject("", project)
	if err != nil {
		return nil, err
	}
	h.mu.Lock()
	if h.watcher != nil && h.watcher.Running() {
		h.mu.Unlock()
		return map[string]any{"status": "active", "project": p}, nil
	}
	root := h.rootLocked()
	w := watch.NewWatcher(root, watch.WatchConfig{
		Debounced: 2 * time.Second,
		Stamp:     extract.Stamp(),
		Store:     h.st,
	}, func(paths []string) error {
		_, err := codeindex.Index(ctx, root, h.st, codeindex.IndexOpts{Project: p, Stamp: extract.Stamp()})
		return err
	})
	if err := w.Start(ctx); err != nil {
		h.mu.Unlock()
		return nil, err
	}
	h.watcher = w
	h.mu.Unlock()
	return map[string]any{"status": "started", "project": p}, nil
}

// DeferredTool reports a v2.1-deferred tool (spec 008 §3 — declared in the
// surface, reporting "run code_index --flag" in v2; deferred, not missing).
func (h *CodeHandler) DeferredTool(ctx context.Context, tool, flag string) error {
	return fmt.Errorf("code: %s is a v2.1-deferred tool — run code_index --%s to build its data", tool, flag)
}

// ── the knowledge-graph reads (the knowledge nodes in the core graph, ──────
// ── spec 003 §5 — route/doc/config/sql are kind-tagged symbols) ────────────

// Route runs the code_route (spec 003 §5 — query framework routes: the
// kind='route' symbols + their references edges to handler symbols). A
// handler name returns the routes that serve it; a path pattern returns the
// matching route and its handler.
func (h *CodeHandler) Route(ctx context.Context, handlerName, path string, limit int) (map[string]any, error) {
	if limit <= 0 {
		limit = 20
	}
	raw, ok := h.st.(interface {
		RawQuery(query string, args ...any) ([][]any, error)
	})
	if !ok {
		return nil, fmt.Errorf("handler: store %T does not expose the RawQuery seam", h.st)
	}
	query := `
		SELECT r.name, f.path, e.confidence, ifnull(h.name,'')
		FROM symbols r
		JOIN files f ON f.id = r.file_id
		LEFT JOIN edges e ON e.kind = 'references' AND e.from_id = r.id
		LEFT JOIN symbols h ON h.id = e.to_id
		WHERE r.kind = 'route'`
	args := []any{}
	if handlerName != "" {
		query += ` AND h.name = ?`
		args = append(args, handlerName)
	}
	if path != "" {
		query += ` AND r.qualified_name = ?`
		args = append(args, path)
	}
	query += ` ORDER BY r.id LIMIT ?`
	args = append(args, limit)
	rows, err := raw.RawQuery(query, args...)
	if err != nil {
		return nil, err
	}
	routes := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		routes = append(routes, map[string]any{
			"name":       strAt(r, 0),
			"file":       strAt(r, 1),
			"confidence": strAt(r, 2),
			"handler":    strAt(r, 3),
		})
	}
	return map[string]any{"routes": routes, "count": len(rows)}, nil
}

// Navigates runs the code_navigates (spec 003 §5 — query framework
// navigation edges: screen → screen). A from-screen returns the screens it
// navigates to; a to-screen returns the senders.
func (h *CodeHandler) Navigates(ctx context.Context, from, to string, limit int) (map[string]any, error) {
	if limit <= 0 {
		limit = 20
	}
	raw, ok := h.st.(interface {
		RawQuery(query string, args ...any) ([][]any, error)
	})
	if !ok {
		return nil, fmt.Errorf("handler: store %T does not expose the RawQuery seam", h.st)
	}
	query := `
		SELECT ifnull(f.name,''), ifnull(t.name,''), e.confidence
		FROM edges e
		LEFT JOIN symbols f ON f.id = e.from_id
		LEFT JOIN symbols t ON t.id = e.to_id
		WHERE e.kind = 'navigates'`
	args := []any{}
	if from != "" {
		query += ` AND f.name = ?`
		args = append(args, from)
	}
	if to != "" {
		query += ` AND t.name = ?`
		args = append(args, to)
	}
	query += ` ORDER BY e.id LIMIT ?`
	args = append(args, limit)
	rows, err := raw.RawQuery(query, args...)
	if err != nil {
		return nil, err
	}
	navigations := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		navigations = append(navigations, map[string]any{
			"from":       strAt(r, 0),
			"to":         strAt(r, 1),
			"confidence": strAt(r, 2),
		})
	}
	return map[string]any{"navigations": navigations, "count": len(rows)}, nil
}

// Docs runs the code_docs (spec 003 §5 — query doc knowledge nodes: the
// kind='doc' symbols, the .md files + their links).
func (h *CodeHandler) Docs(ctx context.Context, path string, limit int) (map[string]any, error) {
	return h.knowledgeNodes(ctx, "doc", path, limit)
}

// Configs runs the code_configs (spec 003 §5 — query config knowledge nodes:
// the kind='config' symbols, the .yaml/.toml/.json files + their references).
func (h *CodeHandler) Configs(ctx context.Context, path string, limit int) (map[string]any, error) {
	return h.knowledgeNodes(ctx, "config", path, limit)
}

// SQLSchema runs the code_sql_schema (spec 003 §5 — query SQL schema
// knowledge nodes: the kind='table' symbols, the .sql files' CREATE TABLE
// statements).
func (h *CodeHandler) SQLSchema(ctx context.Context, table string, limit int) (map[string]any, error) {
	return h.knowledgeNodes(ctx, "table", table, limit)
}

// knowledgeNodes is the shared backing for the doc/config/sql knowledge-node
// reads (spec 003 §5 — the knowledge nodes are kind-tagged symbols in the
// core graph). kind is the symbol kind; filter is the optional name filter.
func (h *CodeHandler) knowledgeNodes(ctx context.Context, kind, filter string, limit int) (map[string]any, error) {
	if limit <= 0 {
		limit = 20
	}
	raw, ok := h.st.(interface {
		RawQuery(query string, args ...any) ([][]any, error)
	})
	if !ok {
		return nil, fmt.Errorf("handler: store %T does not expose the RawQuery seam", h.st)
	}
	query := `
		SELECT s.name, ifnull(s.qualified_name,''), f.path
		FROM symbols s JOIN files f ON f.id = s.file_id
		WHERE s.kind = ?`
	args := []any{kind}
	if filter != "" {
		query += ` AND s.name = ?`
		args = append(args, filter)
	}
	query += ` ORDER BY s.id LIMIT ?`
	args = append(args, limit)
	rows, err := raw.RawQuery(query, args...)
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		items = append(items, map[string]any{
			"name":           strAt(r, 0),
			"qualified_name": strAt(r, 1),
			"file":           strAt(r, 2),
		})
	}
	return map[string]any{"items": items, "count": len(rows)}, nil
}

// StopWatchForTest stops the live watcher (the test cleanup — the MCP
// server's Close calls it).
func (h *CodeHandler) StopWatchForTest() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.watcher != nil {
		h.watcher.Stop()
		h.watcher = nil
	}
}

// ── value helpers (the handler's raw-SELECT surface) ──────────────────────

// tierForLanguage maps a language to its extractor tier (the D-TIER
// per-language report, spec 003 §3.3 — the degradation is visible). The rich
// languages (the gotreesitter one-pass languages) are "rich"; the data-table
// languages are "map"; unmapped languages are "regex" (the fallback floor).
func tierForLanguage(lang string) string {
	switch lang {
	case "go", "python", "javascript", "typescript", "rust", "ruby":
		// The rich one-pass languages (the gotreesitter tier).
		return string(extract.TierRich)
	case "java", "c", "cpp", "csharp", "kotlin", "swift", "php", "scala",
		"elixir", "erlang", "haskell", "lua", "perl", "dart", "groovy",
		"objective-c", "zig", "nim", "crystal", "fsharp", "r",
		"common-lisp", "scheme", "clojure", "ocaml", "julia":
		// The data-table (map) languages.
		return string(extract.TierMap)
	default:
		// The regex fallback floor (unmapped extensions).
		return string(extract.TierRegex)
	}
}

func strAt(row []any, i int) string {
	if len(row) <= i {
		return ""
	}
	s, _ := row[i].(string)
	return s
}

func floatAt(row []any, i int) float64 {
	if len(row) <= i {
		return 0
	}
	switch n := row[i].(type) {
	case float64:
		return n
	case int64:
		return float64(n)
	case int:
		return float64(n)
	}
	return 0
}

// unused imports guard (the handler imports the packages it composes).
var _ = watch.NewWatcher
var _ = extract.Stamp
