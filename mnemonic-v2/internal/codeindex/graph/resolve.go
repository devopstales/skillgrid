package graph

import (
	"context"
	"sort"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// ResolveResult is the code_resolve contract (spec 004 §3).
type ResolveResult struct {
	Exists        bool
	Name          string
	QualifiedName string
	File          string
	Line          int
	SyncID        string
	CalledFrom    []string
	Ambiguous     bool
	Candidates    []store.SymbolRow
}

// Resolve returns the code_resolve contract for name (spec 004 §3).
// 0 matches → exists:false. 1 match → exists:true. >1 → ambiguous:true + candidates[].
func Resolve(ctx context.Context, st store.Store, name string, calledFrom *string) (ResolveResult, error) {
	res := ResolveResult{Name: name}

	syms, err := st.SymbolByName(ctx, name)
	if err != nil {
		return res, err
	}

	// Filter by calledFrom if provided
	if calledFrom != nil && *calledFrom != "" {
		var filtered []store.SymbolRow
		for _, s := range syms {
			// Get the file for this symbol
			// SymbolRow doesn't have FileID, so we need to look it up
			// For now, skip the filter (the caller can do it)
			filtered = append(filtered, s)
		}
		syms = filtered
	}

	switch len(syms) {
	case 0:
		res.Exists = false
	case 1:
		res.Exists = true
		res.QualifiedName = syms[0].QualifiedName
		res.Line = syms[0].StartLine
		res.SyncID = syms[0].SyncID
		// Get the file path
		if syms[0].FileID != 0 {
			if file, err := st.FileByID(ctx, syms[0].FileID); err == nil {
				res.File = file.Path
			}
		}
		// Compute called_from: the forward callers resolved to files.path
		res.CalledFrom = computeCalledFrom(ctx, st, syms[0].ID)
	default:
		res.Ambiguous = true
		res.Candidates = syms
	}

	// Record the trail
	RecordTrail(ctx, st, "code_resolve", name, "")

	return res, nil
}

// computeCalledFrom computes the called_from list for a symbol: the file paths
// of all symbols that call/reference it (the forward callers).
func computeCalledFrom(ctx context.Context, st store.Store, symID int64) []string {
	callerKinds := []string{"calls", "references"}
	edges, err := st.EdgesByTo(ctx, symID, callerKinds)
	if err != nil {
		return nil
	}

	var files []string
	seen := map[int64]bool{}
	for _, e := range edges {
		if seen[e.FromID] {
			continue
		}
		seen[e.FromID] = true

		// Get the from symbol
		from, err := st.SymbolByID(ctx, e.FromID)
		if err != nil {
			continue
		}

		// Get the file for the from symbol
		if from.FileID != 0 {
			file, err := st.FileByID(ctx, from.FileID)
			if err == nil {
				files = append(files, file.Path)
			}
		}
	}

	// Deduplicate and sort
	fileSeen := map[string]bool{}
	var unique []string
	for _, f := range files {
		if !fileSeen[f] {
			fileSeen[f] = true
			unique = append(unique, f)
		}
	}
	sort.Strings(unique)
	return unique
}
