package kb

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// CiteResult is one external-source citation: the source page ref + the
// section content that was cited.
type CiteResult struct {
	Source  string
	Section string
	Content string
}

// CiteSource resolves a source reference to the cited section content.
// Re-fetch granularity (section-level) is the anchor used here.
func CiteSource(ctx context.Context, st store.Store, project, sourceRef string) (*CiteResult, error) {
	res := &CiteResult{Source: sourceRef}
	rows, err := st.Query(ctx,
		`SELECT id, body FROM kb_pages WHERE project = ? AND rel_path = ? AND deleted_at IS NULL`,
		project, sourceRef)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return res, nil
	}
	res.Content = strVal(rows[0], 1)
	return res, nil
}

// CiteExternalResult is the kb_cite_source outcome (008 §3: record an
// external citation).
type CiteExternalResult struct {
	Source           string `json:"source"` // "external:<name>"
	RelPath          string `json:"rel_path"`
	Version          string `json:"version,omitempty"`
	SourceURL        string `json:"source_url,omitempty"`
	SourceCommit     string `json:"source_commit,omitempty"`
	BuildFingerprint string `json:"build_fingerprint"`
	AlreadyCited     bool   `json:"already_cited,omitempty"`
}

// CiteExternal records an external citation on a page's `sources:` (008 §3
// kb_cite_source): `external:<name>` + version + source_url + a computed
// build_fingerprint (the deterministic staleness token, 001 §5.4). It does
// not fetch or store the body (KB-8: sources are cited, not stored). The
// page file is the truth (KB-INV-1) — the frontmatter is re-emitted, the
// page is re-indexed, and the INV-5 page-write event is emitted. Idempotent
// per (name, version): re-citing the same reference is a no-op.
func CiteExternal(ctx context.Context, st store.Store, vaultRoot, project, relPath, name, version, sourceURL, sourceCommit string) (*CiteExternalResult, error) {
	if name == "" {
		return nil, fmt.Errorf("kb: cite-source: an external name is required")
	}
	path := filepath.Join(vaultRoot, filepath.FromSlash(relPath))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("kb: cite-source %s: page not found", relPath)
	}
	fm, body, err := SplitFrontmatter(string(data))
	if err != nil {
		return nil, fmt.Errorf("kb: cite-source %s: %w", relPath, err)
	}
	fp := citeFingerprint(name, version, sourceURL)
	res := &CiteExternalResult{
		Source: "external:" + name, RelPath: relPath, Version: version,
		SourceURL: sourceURL, SourceCommit: sourceCommit, BuildFingerprint: fp,
	}
	for _, s := range fm.Sources {
		if s.External == name && s.Version == version {
			res.AlreadyCited = true
			return res, nil
		}
	}
	fm.Sources = append(fm.Sources, Source{
		External: name, Version: version, SourceURL: sourceURL,
		SourceCommit: sourceCommit, BuildFingerprint: fp,
	})
	fmBlock, err := EmitFrontmatter(fm)
	if err != nil {
		return nil, err
	}
	newData := fmBlock + body
	if err := os.WriteFile(path, []byte(newData), 0o644); err != nil {
		return nil, fmt.Errorf("kb: cite-source %s: %w", relPath, err)
	}
	if _, err := Index(ctx, st, vaultRoot, project); err != nil {
		return nil, err
	}
	if err := emitPageEvent(ctx, st, project, relPath, hashBody(newData)); err != nil {
		return nil, err
	}
	return res, nil
}

// citeFingerprint is the external citation's staleness token (001 §5.4):
// a deterministic hash of what the citation refers to — name, version, URL.
func citeFingerprint(name, version, sourceURL string) string {
	h := sha256.Sum256([]byte(name + "\x00" + version + "\x00" + sourceURL))
	return "sha256:" + hex.EncodeToString(h[:])
}
