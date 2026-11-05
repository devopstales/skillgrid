package kb

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// ValidateResult is the outcome of a single pin-check.
type ValidateResult struct {
	RelPath string
	OK      bool
	Reason  string
}

// Validate runs the pin-check for one page (011 deliverable 8: file-based —
// the page's own frontmatter anchors/claims are checked against the live
// code at repoRoot (anchor paths are working-tree-relative, 006 §2), per 006 §3). The stored-baseline fast path is kept for the
// no-op proof: a baseline current for the page's source fingerprint and live
// bytes is itself proof (a baseline is only ever recorded for a page that
// passed its pin-check), so the re-check is skipped.
//
// Reasons: not_found, bad_frontmatter, baseline_current, not_grounded,
// fresh, stale, unresolved.
func Validate(ctx context.Context, st store.Store, vaultRoot, repoRoot, relPath, project string) (*ValidateResult, error) {
	res := &ValidateResult{RelPath: relPath, OK: false, Reason: "not_found"}
	data, err := os.ReadFile(filepath.Join(vaultRoot, filepath.FromSlash(relPath)))
	if err != nil {
		return res, nil
	}
	fm, _, err := SplitFrontmatter(string(data))
	if err != nil {
		res.Reason = "bad_frontmatter"
		return res, nil
	}

	// Fast path: a current baseline proves this page (003 §5.2: a page earns
	// a baseline only when its markdown and its pin-check are mutually
	// consistent; Current() re-proves both against the live source).
	fp, err := SourceFingerprint(ctx, st, project, repoRoot)
	if err != nil {
		return nil, fmt.Errorf("kb: validate %s: %w", relPath, err)
	}
	baselines, err := Baselines(ctx, st, project)
	if err != nil {
		return nil, err
	}
	if b, ok := baselines[relPath]; ok && b.Current(fp, hashBody(string(data))) {
		res.OK = true
		res.Reason = "baseline_current"
		return res, nil
	}

	// Non-grounded pages carry no pins to check (006 §5: validate scopes to
	// grounded pages).
	if !IsGrounded(fm.Type) {
		res.OK = true
		res.Reason = "not_grounded"
		return res, nil
	}

	pc, err := ValidatePage(ctx, st, repoRoot, relPath, fm)
	if err != nil {
		return nil, err
	}
	res.OK = pc.Status == "fresh"
	res.Reason = pc.Status
	return res, nil
}

// GroundedPages lists the vault's grounded pages (sorted rel paths) — the
// default scope for kb_validate (006 §5: validate checks all grounded pages).
func GroundedPages(root string) ([]string, error) {
	pages, err := collectPages(root)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, p := range pages {
		fm, _, err := SplitFrontmatter(p.Body)
		if err != nil {
			continue
		}
		if IsGrounded(fm.Type) {
			out = append(out, p.RelPath)
		}
	}
	sort.Strings(out)
	return out, nil
}
