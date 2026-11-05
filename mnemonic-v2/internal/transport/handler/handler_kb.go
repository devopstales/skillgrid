// handler_kb.go — the shared handler set for the KB tool family (spec
// 008-transport §3, the 011 openwiki port): the 12 kb_* tools.
//
// Like Handler, KbHandler is the thin mapping every surface calls (the MCP
// server's KB profile, the CLI's kb group, the later HTTP mirror) — the K6
// byte-identical seam. Each method resolves the project at call time
// (explicit → detector → the config's already-resolved name), maps the
// request onto the kb engine, and returns the engine's result shape (the
// adapters wrap it in the envelope).
//
// Roots: the engine needs two roots (006 §2 — a page's anchor paths are
// working-tree-relative, so the pin-check runs against the repo root, not
// the vault root). The adapters set them via SetRoots: vaultRoot = the
// sentinel vault dir (.skillgrid/knowledge), repoRoot = the git worktree
// root the anchors resolve against (the CLI re-roots per call; the MCP
// server sets them once at runtime open).
package handler

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/config"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/kb"
)

// KbHandler is the shared handler set for the KB tools (008-transport §3).
type KbHandler struct {
	st  store.Store
	cfg config.Config
	det projectDetector

	mu        sync.RWMutex
	vaultRoot string
	repoRoot  string
}

// NewKbHandler builds a KbHandler over a store. The detector is resolved
// at call time (never at construction): the resolution chain runs on every
// tool call, so a process that changes directories between calls
// re-resolves correctly.
func NewKbHandler(st store.Store, cfg config.Config, det projectDetector) *KbHandler {
	return &KbHandler{st: st, cfg: cfg, det: det}
}

// SetRoots sets the vault + repo roots (006 §2).
func (h *KbHandler) SetRoots(vaultRoot, repoRoot string) {
	h.mu.Lock()
	h.vaultRoot, h.repoRoot = vaultRoot, repoRoot
	h.mu.Unlock()
}

// Roots returns the current vault + repo roots.
func (h *KbHandler) Roots() (string, string) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.vaultRoot, h.repoRoot
}

// resolveProject runs the call-time resolution chain (008-transport §4:
// explicit → detector → config's already-resolved project). Same semantics
// as Handler.resolveProject (a detector failure falls back to
// cfg.Project.Name; an empty result with no detector is errNoDetector).
func (h *KbHandler) resolveProject(cwd, explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		return strings.TrimSpace(explicit), nil
	}
	if h.det != nil {
		name, err := h.det.Resolve(cwd, "")
		if err == nil {
			return name, nil
		}
		if h.cfg.Project.Name != "" {
			return h.cfg.Project.Name, nil
		}
		return "", err
	}
	if h.cfg.Project.Name != "" {
		return h.cfg.Project.Name, nil
	}
	return "", errNoDetector
}

// assertOwnership enforces the expected_project ownership assertion
// (008-transport §4: the write tools take a mandatory expected_project).
// An empty expectedProject passes (the CLI may omit it); a non-empty one
// must match the resolved project.
func assertOwnership(resolved, expected string) error {
	if strings.TrimSpace(expected) != "" && resolved != expected {
		return fmt.Errorf("kb: project mismatch: resolved %q, expected %q", resolved, expected)
	}
	return nil
}

// ── request / result shapes (the D1 tool-contract surface) ─────────────────
//
// The batch shapes are the handler's wire values (the adapters pass them to
// the envelope untouched — a handler-level rename does not ripple into the
// two surfaces).

// QueryRequest is the kb_query input (008 §3 stage 1).
type QueryRequest struct {
	Query        string
	Type         string
	Limit        int
	MatchMode    string
	GroundedOnly bool
}

// QueryBatch is the kb_query output: the ranked hits + count.
type QueryBatch struct {
	Results []kb.QueryResult `json:"results"`
	Count   int              `json:"count"`
}

// SearchBatch is the kb_search output (008 §3 raw FTS power path).
type SearchBatch struct {
	Results []kb.SearchResult `json:"results"`
	Count   int               `json:"count"`
}

// ValidateBatch is the kb_validate output (008 §3 / 006 §5): the per-page
// checks + the strict verdict (OK = no drift; the CLI maps strict && !OK
// to exit 1, the agent reads ok/drifted).
type ValidateBatch struct {
	Results []*kb.ValidateResult `json:"results"`
	Checked int                  `json:"checked"`
	Drifted int                  `json:"drifted"`
	Strict  bool                 `json:"strict"`
	OK      bool                 `json:"ok"`
}

// LintBatch is the kb_lint output (008 §3 / 007-lint): the deterministic
// findings + the safe-fix count.
type LintBatch struct {
	Issues []kb.LintIssue `json:"issues"`
	Count  int            `json:"count"`
	Fixed  int            `json:"fixed,omitempty"`
}

// ── the 12 methods ─────────────────────────────────────────────────────────
//
// Agent profile (8): Query, Read, Search, Generate, Validate, Lint, Status,
// CiteSource. All profile (+4): Reindex, Index, Bootstrap, RebuildMeta.

// Query runs kb_query's stage-1 deep retrieval (008 §3): ranked links +
// 1-line snippets over kb_pages_fts (full hydration is stage 2, kb_read).
func (h *KbHandler) Query(ctx context.Context, cwd, explicit string, r QueryRequest) (*QueryBatch, error) {
	p, err := h.resolveProject(cwd, explicit)
	if err != nil {
		return nil, err
	}
	f := kb.QueryFilters{Project: p, Type: r.Type, Limit: r.Limit, MatchMode: r.MatchMode, GroundedOnly: r.GroundedOnly}
	res, err := kb.Query(ctx, h.st, r.Query, f)
	if err != nil {
		return nil, err
	}
	return &QueryBatch{Results: res, Count: len(res)}, nil
}

// Read runs kb_read's stage-2 hydration (008 §3): the full page (or one
// "##" section) + backlinks + the pin-check state for grounded pages.
func (h *KbHandler) Read(ctx context.Context, cwd, relPath, section, explicit string, includeBacklinks bool) (*kb.ReadResult, error) {
	p, err := h.resolveProject(cwd, explicit)
	if err != nil {
		return nil, err
	}
	root, repo := h.Roots()
	return kb.Read(ctx, h.st, root, repo, relPath, p, section, includeBacklinks)
}

// Search runs kb_search (008 §3): the raw FTS over kb_pages_fts (the power
// path — more rows than kb_query's default, longer snippets).
func (h *KbHandler) Search(ctx context.Context, cwd, query, explicit string, limit int) (*SearchBatch, error) {
	p, err := h.resolveProject(cwd, explicit)
	if err != nil {
		return nil, err
	}
	res, err := kb.Search(ctx, h.st, query, p, limit)
	if err != nil {
		return nil, err
	}
	return &SearchBatch{Results: res, Count: len(res)}, nil
}

// Generate runs kb_generate (008 §3 — the KB's "save" equivalent). It is a
// write tool (the KB write queue serializes it). expectedProject is the
// mandatory ownership assertion (008 §4): the resolved project must match
// it. explicit is the surface's project selector (the CLI passes its
// --project so the resolution matches its own; the MCP passes ""). full →
// a full re-derive; the default is incremental (entity-targeted from the
// latest save).
func (h *KbHandler) Generate(ctx context.Context, cwd, explicit, expectedProject string, full bool) (*kb.GenerateResult, error) {
	p, err := h.resolveProject(cwd, explicit)
	if err != nil {
		return nil, err
	}
	if err := assertOwnership(p, expectedProject); err != nil {
		return nil, err
	}
	facts, err := kb.AllFacts(ctx, h.st, p)
	if err != nil {
		return nil, err
	}
	root, repo := h.Roots()
	return kb.Generate(ctx, h.st, root, repo, p, full, nil, facts)
}

// Validate runs kb_validate (008 §3 / 006 §5): the pin-check for one or
// all grounded pages against the live code at the repo root. An empty
// relPaths scopes to every grounded page (006 §5's default). strict is
// carried through the result (never an error — the agent reads
// ok/drifted, the CLI decides the exit code).
func (h *KbHandler) Validate(ctx context.Context, cwd, explicit string, relPaths []string, strict bool) (*ValidateBatch, error) {
	p, err := h.resolveProject(cwd, explicit)
	if err != nil {
		return nil, err
	}
	root, repo := h.Roots()
	if len(relPaths) == 0 {
		rels, err := kb.GroundedPages(root)
		if err != nil {
			return nil, err
		}
		relPaths = rels
	}
	res := &ValidateBatch{Results: make([]*kb.ValidateResult, 0, len(relPaths)), Strict: strict}
	for _, rel := range relPaths {
		v, err := kb.Validate(ctx, h.st, root, repo, rel, p)
		if err != nil {
			return nil, err
		}
		if !v.OK {
			res.Drifted++
		}
		res.Results = append(res.Results, v)
	}
	res.Checked = len(res.Results)
	res.OK = res.Drifted == 0
	return res, nil
}

// Lint runs kb_lint (008 §3 / 007-lint): deterministic health over the
// index (missing titles, empty bodies, stale index rows). fix auto-repairs
// the safe findings (missing_title backfill) and re-lints. The index syncs
// first — lint is store-level (it reads kb_pages, never the vault), so the
// findings reflect the current vault only after the index caught up.
func (h *KbHandler) Lint(ctx context.Context, cwd, explicit string, fix bool) (*LintBatch, error) {
	p, err := h.resolveProject(cwd, explicit)
	if err != nil {
		return nil, err
	}
	root, _ := h.Roots()
	if _, err := kb.Index(ctx, h.st, root, p); err != nil {
		return nil, err
	}
	issues, err := kb.Lint(ctx, h.st, p)
	if err != nil {
		return nil, err
	}
	fixed := 0
	if fix {
		var rels []string
		for _, iss := range issues {
			if iss.Code == "missing_title" {
				rels = append(rels, iss.RelPath)
			}
		}
		if len(rels) > 0 {
			fixed, err = kb.Fix(ctx, h.st, p, rels)
			if err != nil {
				return nil, err
			}
			issues, err = kb.Lint(ctx, h.st, p)
			if err != nil {
				return nil, err
			}
		}
	}
	return &LintBatch{Issues: issues, Count: len(issues), Fixed: fixed}, nil
}

// Status runs kb_status (008 §3): the vault's derived state (indexed page
// count, the run phase/id, baseline + event counts). A read-only view —
// agent-profile because it never writes.
func (h *KbHandler) Status(ctx context.Context, cwd, explicit string) (*kb.StatusResult, error) {
	p, err := h.resolveProject(cwd, explicit)
	if err != nil {
		return nil, err
	}
	return kb.Status(ctx, h.st, p)
}

// CiteSource runs kb_cite_source (008 §3): records an external citation on
// a page's sources (external:<name> + version + source_url; the build
// fingerprint is computed). A write tool (the KB write queue;
// expected_project mandatory). It does not fetch/store the body.
func (h *KbHandler) CiteSource(ctx context.Context, cwd, explicit, expectedProject, relPath, name, version, sourceURL, sourceCommit string) (*kb.CiteExternalResult, error) {
	p, err := h.resolveProject(cwd, explicit)
	if err != nil {
		return nil, err
	}
	if err := assertOwnership(p, expectedProject); err != nil {
		return nil, err
	}
	root, _ := h.Roots()
	return kb.CiteExternal(ctx, h.st, root, p, relPath, name, version, sourceURL, sourceCommit)
}

// Reindex runs kb_reindex (008 §3, admin): the full rebuild of
// kb_pages/kb_pages_fts/kb_claims from the markdown. Idempotent.
func (h *KbHandler) Reindex(ctx context.Context, cwd, explicit string) (*kb.IndexResult, error) {
	p, err := h.resolveProject(cwd, explicit)
	if err != nil {
		return nil, err
	}
	root, _ := h.Roots()
	return kb.Reindex(ctx, h.st, root, p)
}

// Index runs kb_index (008 §3, admin): the incremental reindex (changed
// pages only; advances the checkpoint).
func (h *KbHandler) Index(ctx context.Context, cwd, explicit string) (*kb.IndexResult, error) {
	p, err := h.resolveProject(cwd, explicit)
	if err != nil {
		return nil, err
	}
	root, _ := h.Roots()
	return kb.Index(ctx, h.st, root, p)
}

// Bootstrap runs kb_bootstrap (008 §3, admin): scaffolds a new vault (the
// sentinel config.json, KNOWLEDGE_SCHEMA.md, templates, the page folders).
// Idempotent no-op on an existing vault.
func (h *KbHandler) Bootstrap(ctx context.Context, cwd, explicit string) (*kb.Scaffold, error) {
	p, err := h.resolveProject(cwd, explicit)
	if err != nil {
		return nil, err
	}
	root, _ := h.Roots()
	return kb.Bootstrap(ctx, h.st, root, kb.Options{Project: p})
}

// RebuildMeta runs kb_rebuild_meta (008 §3, admin): the derived metadata
// that is NOT the FTS index — the kb_run_state checkpoint + the backlink
// in-degree cache.
func (h *KbHandler) RebuildMeta(ctx context.Context, cwd, explicit string) (*kb.RebuildMetaResult, error) {
	p, err := h.resolveProject(cwd, explicit)
	if err != nil {
		return nil, err
	}
	root, repo := h.Roots()
	return kb.RebuildMeta(ctx, h.st, root, repo, p)
}
