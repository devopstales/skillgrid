package kb

import (
	"context"

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
