// handler_projectwiki.go — the shared handler set for the ProjectWiki tool family (spec
// 008-transport §3, the 011 openwiki port): the 12 projectwiki_* tools.
//
// Like Handler, ProjectWikiHandler is the thin mapping every surface calls (the MCP
// server's ProjectWiki profile, the CLI's projectwiki group, the later HTTP mirror) — the K6
// byte-identical seam. Each method resolves the project at call time
// (explicit → detector → the config's already-resolved name), maps the
// request onto the projectwiki engine, and returns the engine's result shape (the
// adapters wrap it in the envelope).
//
// Roots: the engine needs two roots (006 §2 — a page's anchor paths are
// working-tree-relative, so the pin-check runs against the repo root, not
// the vault root). The adapters set them via SetRoots: vaultRoot = the
// sentinel vault dir (.skillgrid/projectwiki), repoRoot = the git worktree
// root the anchors resolve against (the CLI re-roots per call; the MCP
// server sets them once at runtime open).
package handler

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/devopstales/skillgrid/mnemonic/internal/core/config"
	"github.com/devopstales/skillgrid/mnemonic/internal/core/store"
	"github.com/devopstales/skillgrid/mnemonic/internal/projectwiki"
)

// ProjectWikiHandler is the shared handler set for the ProjectWiki tools (008-transport §3).
type ProjectWikiHandler struct {
	st  store.Store
	cfg config.Config
	det projectDetector

	mu        sync.RWMutex
	vaultRoot string
	repoRoot  string
}

// NewProjectWikiHandler builds a ProjectWikiHandler over a store. The detector is resolved
// at call time (never at construction): the resolution chain runs on every
// tool call, so a process that changes directories between calls
// re-resolves correctly.
func NewProjectWikiHandler(st store.Store, cfg config.Config, det projectDetector) *ProjectWikiHandler {
	return &ProjectWikiHandler{st: st, cfg: cfg, det: det}
}

// SetRoots sets the vault + repo roots (006 §2).
func (h *ProjectWikiHandler) SetRoots(vaultRoot, repoRoot string) {
	h.mu.Lock()
	h.vaultRoot, h.repoRoot = vaultRoot, repoRoot
	h.mu.Unlock()
}

// Roots returns the current vault + repo roots.
func (h *ProjectWikiHandler) Roots() (string, string) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.vaultRoot, h.repoRoot
}

// resolveProject runs the call-time resolution chain (008-transport §4:
// explicit → detector → config's already-resolved project). Same semantics
// as Handler.resolveProject (a detector failure falls back to
// cfg.Project.Name; an empty result with no detector is errNoDetector).
func (h *ProjectWikiHandler) resolveProject(cwd, explicit string) (string, error) {
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
		return fmt.Errorf("projectwiki: project mismatch: resolved %q, expected %q", resolved, expected)
	}
	return nil
}

// ── request / result shapes (the D1 tool-contract surface) ─────────────────
//
// The batch shapes are the handler's wire values (the adapters pass them to
// the envelope untouched — a handler-level rename does not ripple into the
// two surfaces).

// QueryRequest is the projectwiki_query input (008 §3 stage 1).
type QueryRequest struct {
	Query        string
	Type         string
	Limit        int
	MatchMode    string
	GroundedOnly bool
}

// QueryBatch is the projectwiki_query output: the ranked hits + count,
// plus the spec 005 §6 result metadata — thin with its mem_lookup
// fallback hint when the result is thin, and degraded: stale_index
// when the index lags the vault.
type QueryBatch struct {
	Results  []projectwiki.QueryResult `json:"results"`
	Count    int                       `json:"count"`
	Thin     bool                      `json:"thin,omitempty"`
	Fallback string                    `json:"fallback,omitempty"`
	Degraded string                    `json:"degraded,omitempty"`
}

// SearchBatch is the projectwiki_search output (008 §3 raw FTS power path).
type SearchBatch struct {
	Results []projectwiki.SearchResult `json:"results"`
	Count   int                        `json:"count"`
}

// ValidateBatch is the projectwiki_validate output (008 §3 / 006 §5): the per-page
// checks + the strict verdict (OK = no drift; the CLI maps strict && !OK
// to exit 1, the agent reads ok/drifted).
type ValidateBatch struct {
	Results []*projectwiki.ValidateResult `json:"results"`
	Checked int                           `json:"checked"`
	Drifted int                           `json:"drifted"`
	Strict  bool                          `json:"strict"`
	OK      bool                          `json:"ok"`
}

// LintBatch is the projectwiki_lint output (008 §3 / 007-lint): the deterministic
// findings + the safe-fix count.
type LintBatch struct {
	Issues []projectwiki.LintIssue `json:"issues"`
	Count  int                     `json:"count"`
	Fixed  int                     `json:"fixed,omitempty"`
}

// ── the 12 methods ─────────────────────────────────────────────────────────
//
// Agent profile (8): Query, Read, Search, Generate, Validate, Lint, Status,
// CiteSource. All profile (+4): Reindex, Index, Bootstrap, RebuildMeta.

// Query runs projectwiki_query's stage-1 deep retrieval (008 §3): ranked links +
// 1-line snippets over projectwiki_pages_fts (full hydration is stage 2, projectwiki_read).
func (h *ProjectWikiHandler) Query(ctx context.Context, cwd, explicit string, r QueryRequest) (*QueryBatch, error) {
	p, err := h.resolveProject(cwd, explicit)
	if err != nil {
		return nil, err
	}
	f := projectwiki.QueryFilters{Project: p, Type: r.Type, Limit: r.Limit, MatchMode: r.MatchMode, GroundedOnly: r.GroundedOnly}
	res, err := projectwiki.Query(ctx, h.st, r.Query, f)
	if err != nil {
		return nil, err
	}
	root, _ := h.Roots()
	thin, fallback, degraded, err := projectwiki.AssessQuery(ctx, h.st, root, p, len(res))
	if err != nil {
		return nil, err
	}
	return &QueryBatch{Results: res, Count: len(res), Thin: thin, Fallback: fallback, Degraded: degraded}, nil
}

// Read runs projectwiki_read's stage-2 hydration (008 §3): the full page (or one
// "##" section) + backlinks + the pin-check state for grounded pages.
func (h *ProjectWikiHandler) Read(ctx context.Context, cwd, relPath, section, explicit string, includeBacklinks bool) (*projectwiki.ReadResult, error) {
	p, err := h.resolveProject(cwd, explicit)
	if err != nil {
		return nil, err
	}
	root, repo := h.Roots()
	return projectwiki.Read(ctx, h.st, root, repo, relPath, p, section, includeBacklinks)
}

// Search runs projectwiki_search (008 §3): the raw FTS over projectwiki_pages_fts (the power
// path — more rows than projectwiki_query's default, longer snippets).
func (h *ProjectWikiHandler) Search(ctx context.Context, cwd, query, explicit string, limit int) (*SearchBatch, error) {
	p, err := h.resolveProject(cwd, explicit)
	if err != nil {
		return nil, err
	}
	res, err := projectwiki.Search(ctx, h.st, query, p, limit)
	if err != nil {
		return nil, err
	}
	return &SearchBatch{Results: res, Count: len(res)}, nil
}

// Generate runs projectwiki_generate (008 §3 — the ProjectWiki's "save" equivalent). It is a
// write tool (the ProjectWiki write queue serializes it). expectedProject is the
// mandatory ownership assertion (008 §4): the resolved project must match
// it. explicit is the surface's project selector (the CLI passes its
// --project so the resolution matches its own; the MCP passes ""). full →
// a full re-derive; the default is incremental (entity-targeted from the
// latest save).
func (h *ProjectWikiHandler) Generate(ctx context.Context, cwd, explicit, expectedProject string, full bool) (*projectwiki.GenerateResult, error) {
	p, err := h.resolveProject(cwd, explicit)
	if err != nil {
		return nil, err
	}
	if err := assertOwnership(p, expectedProject); err != nil {
		return nil, err
	}
	facts, err := projectwiki.AllFacts(ctx, h.st, p)
	if err != nil {
		return nil, err
	}
	root, repo := h.Roots()
	return projectwiki.Generate(ctx, h.st, root, repo, p, full, nil, facts)
}

// Validate runs projectwiki_validate (008 §3 / 006 §5): the pin-check for one or
// all grounded pages against the live code at the repo root. An empty
// relPaths scopes to every grounded page (006 §5's default). strict is
// carried through the result (never an error — the agent reads
// ok/drifted, the CLI decides the exit code).
func (h *ProjectWikiHandler) Validate(ctx context.Context, cwd, explicit string, relPaths []string, strict bool) (*ValidateBatch, error) {
	p, err := h.resolveProject(cwd, explicit)
	if err != nil {
		return nil, err
	}
	root, repo := h.Roots()
	if len(relPaths) == 0 {
		rels, err := projectwiki.GroundedPages(root)
		if err != nil {
			return nil, err
		}
		relPaths = rels
	}
	res := &ValidateBatch{Results: make([]*projectwiki.ValidateResult, 0, len(relPaths)), Strict: strict}
	for _, rel := range relPaths {
		v, err := projectwiki.Validate(ctx, h.st, root, repo, rel, p)
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

// Lint runs projectwiki_lint (008 §3 / 007-lint): deterministic health over the
// index (missing titles, empty bodies, stale index rows). fix auto-repairs
// the safe findings (missing_title backfill) and re-lints. The index syncs
// first — lint is store-level (it reads projectwiki_pages, never the vault), so the
// findings reflect the current vault only after the index caught up.
func (h *ProjectWikiHandler) Lint(ctx context.Context, cwd, explicit string, fix bool) (*LintBatch, error) {
	p, err := h.resolveProject(cwd, explicit)
	if err != nil {
		return nil, err
	}
	root, repo := h.Roots()
	if _, err := projectwiki.Index(ctx, h.st, root, p); err != nil {
		return nil, err
	}
	issues, err := projectwiki.Lint(ctx, h.st, repo, p)
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
			fixed, err = projectwiki.Fix(ctx, h.st, p, rels)
			if err != nil {
				return nil, err
			}
			issues, err = projectwiki.Lint(ctx, h.st, repo, p)
			if err != nil {
				return nil, err
			}
		}
	}
	return &LintBatch{Issues: issues, Count: len(issues), Fixed: fixed}, nil
}

// Status runs projectwiki_status (008 §3): the vault's derived state (indexed page
// count, the run phase/id, baseline + event counts). A read-only view —
// agent-profile because it never writes.
func (h *ProjectWikiHandler) Status(ctx context.Context, cwd, explicit string) (*projectwiki.StatusResult, error) {
	p, err := h.resolveProject(cwd, explicit)
	if err != nil {
		return nil, err
	}
	root, _ := h.Roots()
	return projectwiki.Status(ctx, h.st, root, p)
}

// CiteSource runs projectwiki_cite_source (008 §3): records an external citation on
// a page's sources (external:<name> + version + source_url; the build
// fingerprint is computed). A write tool (the ProjectWiki write queue;
// expected_project mandatory). It does not fetch/store the body.
func (h *ProjectWikiHandler) CiteSource(ctx context.Context, cwd, explicit, expectedProject, relPath, name, version, sourceURL, sourceCommit string) (*projectwiki.CiteExternalResult, error) {
	p, err := h.resolveProject(cwd, explicit)
	if err != nil {
		return nil, err
	}
	if err := assertOwnership(p, expectedProject); err != nil {
		return nil, err
	}
	root, _ := h.Roots()
	return projectwiki.CiteExternal(ctx, h.st, root, p, relPath, name, version, sourceURL, sourceCommit)
}

// Reindex runs projectwiki_reindex (008 §3, admin): the full rebuild of
// projectwiki_pages/projectwiki_pages_fts/projectwiki_claims from the markdown. Idempotent.
func (h *ProjectWikiHandler) Reindex(ctx context.Context, cwd, explicit string) (*projectwiki.IndexResult, error) {
	p, err := h.resolveProject(cwd, explicit)
	if err != nil {
		return nil, err
	}
	root, _ := h.Roots()
	return projectwiki.Reindex(ctx, h.st, root, p)
}

// Index runs projectwiki_index (008 §3, admin): the incremental reindex (changed
// pages only; advances the checkpoint).
func (h *ProjectWikiHandler) Index(ctx context.Context, cwd, explicit string) (*projectwiki.IndexResult, error) {
	p, err := h.resolveProject(cwd, explicit)
	if err != nil {
		return nil, err
	}
	root, _ := h.Roots()
	return projectwiki.Index(ctx, h.st, root, p)
}

// Bootstrap runs projectwiki_bootstrap (008 §3, admin): scaffolds a new vault (the
// sentinel config.json, PROJECTWIKI_SCHEMA.md, templates, the page folders).
// Idempotent no-op on an existing vault.
func (h *ProjectWikiHandler) Bootstrap(ctx context.Context, cwd, explicit string) (*projectwiki.Scaffold, error) {
	p, err := h.resolveProject(cwd, explicit)
	if err != nil {
		return nil, err
	}
	root, repo := h.Roots()
	sc, err := projectwiki.Bootstrap(ctx, h.st, root, projectwiki.Options{Project: p})
	if err != nil {
		return nil, err
	}
	// The repo-level wiring: write/refresh the "### ProjectWiki" block in
	// the repo root's AGENTS.md (idempotent — a no-op vault still gets the
	// block ensured). Skipped when no repo root is known (no git worktree).
	if repo != "" {
		if err := ensureAgentsBlock(repo); err != nil {
			return nil, fmt.Errorf("projectwiki bootstrap: agents.md block: %w", err)
		}
	}
	return sc, nil
}

// RebuildMeta runs projectwiki_rebuild_meta (008 §3, admin): the derived metadata
// that is NOT the FTS index — the projectwiki_run_state checkpoint + the backlink
// in-degree cache.
func (h *ProjectWikiHandler) RebuildMeta(ctx context.Context, cwd, explicit string) (*projectwiki.RebuildMetaResult, error) {
	p, err := h.resolveProject(cwd, explicit)
	if err != nil {
		return nil, err
	}
	root, repo := h.Roots()
	return projectwiki.RebuildMeta(ctx, h.st, root, repo, p)
}

// ── the AGENTS.md ProjectWiki block (projectwiki_bootstrap's repo-level wiring) ──
//
// projectwiki_bootstrap writes a marker-delimited "### ProjectWiki" block into the
// repo root's AGENTS.md (the agent's discovery path — the same pattern the
// OpenWiki run uses for its own block). Idempotent: the block is replaced
// in place on a repeat bootstrap (a refresh, never a duplicate).

const (
	agentsProjectWikiStartMarker = "<!-- project-wiki:start -->"
	agentsProjectWikiEndMarker   = "<!-- project-wiki:end -->"
	// detection-only: pre-rename blocks are found and refreshed in place
	agentsLegacyStartMarker = "<!-- knowledge-base:start -->"
	agentsLegacyEndMarker   = "<!-- knowledge-base:end -->"
)

// projectWikiAgentsBlock is the AGENTS.md block the bootstrap writes: the routing
// (read/save/verify/maintain verbs) + the two load-bearing invariants.
func projectWikiAgentsBlock() string {
	const body = "### ProjectWiki\n" +
		"\n" +
		"This project uses the skillgrid project wiki: an OKF vault at `.skillgrid/projectwiki/` + a derived FTS index, served by the `projectwiki` engine — `mnemonic projectwiki ...` CLI and the `projectwiki_*` MCP tools (the shared ProjectWikiHandler — CLI and MCP never diverge).\n" +
		"\n" +
		"- **Read**: `projectwiki_query` (stage-1 ranked retrieval) → `projectwiki_read` (full page / `##` section + pin-check state) → `projectwiki_search` (raw FTS power path).\n" +
		"- **Save**: `projectwiki_generate` — the project wiki's proactive save (run it after significant `mem_save`; incremental by default, `--full` re-derives).\n" +
		"- **Verify**: grounded pages pin live code (working-tree-relative anchors); `projectwiki validate --strict` is the CI gate (drift → exit 1).\n" +
		"- **Maintain**: `projectwiki lint [--fix]` · `projectwiki index` (incremental) / `projectwiki reindex` (full) · `projectwiki status` · `projectwiki rebuild-meta` · `projectwiki cite-source`.\n" +
		"- Vault files are truth (PW-INV-1); `index/**` + the `projectwiki_*` rows are derived — never hand-edit (PW-INV-3).\n"
	return agentsProjectWikiStartMarker + "\n" + body + agentsProjectWikiEndMarker
}

// ensureAgentsBlock writes (or refreshes) the ProjectWiki block in
// repoRoot/AGENTS.md. Placement: an existing block is replaced in place —
// including a pre-rename block, found by its legacy knowledge-base markers
// (the legacy markers are detection-only and never written) —; otherwise
// the block is inserted before the <!-- skillgrid:end --> marker (the end
// of the Skillgrid section) or appended to the file. A missing AGENTS.md
// is created with the block.
func ensureAgentsBlock(repoRoot string) error {
	path := filepath.Join(repoRoot, "AGENTS.md")
	block := projectWikiAgentsBlock()
	existing, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return os.WriteFile(path, []byte(block+"\n"), 0o644)
	}
	if err != nil {
		return err
	}
	s := string(existing)
	start, end, endLen := blockSpan(s, agentsProjectWikiStartMarker, agentsProjectWikiEndMarker)
	if start == -1 {
		// Pre-rename block: refresh it in place (the legacy markers are
		// detection-only — a legacy span is replaced, never duplicated).
		start, end, endLen = blockSpan(s, agentsLegacyStartMarker, agentsLegacyEndMarker)
	}
	if start != -1 {
		// Block present: replace in place (a refresh).
		s = s[:start] + block + s[end+endLen:]
		return os.WriteFile(path, []byte(s), 0o644)
	}
	if i := agentsSkillgridEnd(s); i != -1 {
		// Inside the Skillgrid section: insert before the end marker.
		s = s[:i] + block + "\n\n" + s[i:]
	} else {
		if !strings.HasSuffix(s, "\n") {
			s += "\n"
		}
		s += "\n" + block + "\n"
	}
	return os.WriteFile(path, []byte(s), 0o644)
}

// blockSpan locates a marker-delimited span in s. It returns the start
// offset, the end offset, and the length of the end marker (the markers may
// differ in length — the legacy pair is longer). Absent or mis-ordered
// markers → -1, -1, 0.
func blockSpan(s, startMarker, endMarker string) (int, int, int) {
	start := strings.Index(s, startMarker)
	if start == -1 {
		return -1, -1, 0
	}
	end := strings.Index(s, endMarker)
	if end <= start {
		return -1, -1, 0
	}
	return start, end, len(endMarker)
}

// agentsSkillgridEnd returns the offset of the standalone
// `<!-- skillgrid:end -->` line (the end of the Skillgrid section), or -1.
// The marker may be QUOTED inline in prose (this repo's AGENTS.md quotes
// both markers in backticks in the intro paragraph) — an inline occurrence
// is not the section end, so only a marker that occupies its own line
// (whitespace on either side is fine) anchors the insertion.
func agentsSkillgridEnd(s string) int {
	const marker = "<!-- skillgrid:end -->"
	for i := 0; ; {
		j := strings.Index(s[i:], marker)
		if j == -1 {
			return -1
		}
		j += i
		lineStart := strings.LastIndex(s[:j], "\n") + 1
		lineEnd := strings.Index(s[j+len(marker):], "\n")
		tail := s[j+len(marker):]
		if lineEnd != -1 {
			tail = s[j+len(marker) : j+len(marker)+lineEnd]
		}
		if strings.TrimSpace(s[lineStart:j]) == "" && strings.TrimSpace(tail) == "" {
			return j
		}
		i = j + len(marker)
	}
}
