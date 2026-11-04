package extract

import (
	"context"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// ToStoreGraph converts an extract.FileGraph (the ladder's own types) into a
// store.FileGraph (the driver's write path). The conversion is field-for-field
// (the two shapes are parallel; the extract package owns its types so the
// ladder has no store import — this adapter is the only seam).
func ToStoreGraph(g FileGraph) store.FileGraph {
	out := store.FileGraph{
		Symbols:    make([]store.Symbol, 0, len(g.Symbols)),
		Edges:      make([]store.Edge, 0, len(g.Edges)),
		Unresolved: make([]store.UnresolvedRef, 0, len(g.Unresolved)),
		Chunks:     make([]store.Chunk, 0, len(g.Chunks)),
		Oversized:    g.Oversized,
	}
	for _, s := range g.Symbols {
		out.Symbols = append(out.Symbols, store.Symbol{
			Name:          s.Name,
			QualifiedName: s.QualifiedName,
			Kind:          s.Kind,
			Language:      s.Language,
			Signature:     s.Signature,
			StartLine:     s.StartLine,
			EndLine:       s.EndLine,
			Visibility:    s.Visibility,
			IsExported:    s.IsExported,
			ReturnType:    s.ReturnType,
			ParamTypes:    s.ParamTypes,
			ContentHash:   s.ContentHash,
			UID:           s.UID,
		})
	}
	for _, e := range g.Edges {
		out.Edges = append(out.Edges, store.Edge{
			Kind:       e.Kind,
			FromID:     e.FromID,
			ToID:       e.ToID,
			ToName:     e.ToName,
			Confidence: e.Confidence,
			Line:       e.Line,
			Context:    e.Context,
		})
	}
	for _, u := range g.Unresolved {
		out.Unresolved = append(out.Unresolved, store.UnresolvedRef{
			Reference:  u.Reference,
			Kind:       u.Kind,
			Line:       u.Line,
			Candidates: u.Candidates,
		})
	}
	for _, c := range g.Chunks {
		out.Chunks = append(out.Chunks, store.Chunk{
			Kind:        c.Kind,
			StartLine:   c.StartLine,
			EndLine:     c.EndLine,
			Text:        c.Text,
			ContentHash: c.ContentHash,
		})
	}
	return out
}

// DefaultExtractFn is the codeindex.Index ExtractFn seam wired to the real
// 3-tier ladder (TICKET-03 replaces TICKET-01's test double). path is the
// repo-relative path; content is the file body; the language is detected from
// the path ("" when unmapped → regex tier). opts are the ExtractOptions (the
// WithPool seam: the circuit-breaker state lives for the run, not per call).
func DefaultExtractFn(path, content string, opts ...ExtractOption) (store.FileGraph, error) {
	g, _, err := ExtractFile(context.Background(), path, content, "", opts...)
	if err != nil {
		return ToStoreGraph(g), err
	}
	return ToStoreGraph(g), nil
}
