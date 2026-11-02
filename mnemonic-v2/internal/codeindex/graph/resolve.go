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
		// A read that failed is still a degraded retrieval worth auditing
		// (INV-5): record the trail before returning.
		RecordTrail(ctx, st, "code_resolve", name, "symbol_by_name: "+err.Error())
		return res, err
	}

	// calledFrom file-hint disambiguation (v1 resolve.go:53-55, spec 004 §12):
	// a name + a file hint that disambiguates to exactly one candidate is NOT
	// ambiguous. Match the hint against each candidate's file path (SymbolRow.
	// FileID + FileByID). If the hint matches no candidate, fall through to the
	// full ambiguous list (an unsatisfied filter is not a silent drop).
	if calledFrom != nil && *calledFrom != "" {
		hint := *calledFrom
		var filtered []store.SymbolRow
		fileByID := map[int64]store.FileRow{}
		for _, s := range syms {
			if s.FileID == 0 {
				continue
			}
			f, ok := fileByID[s.FileID]
			if !ok {
				f, err = st.FileByID(ctx, s.FileID)
				if err != nil {
					continue
				}
				fileByID[s.FileID] = f
			}
			if f.Path == hint {
				filtered = append(filtered, s)
			}
		}
		if len(filtered) > 0 {
			syms = filtered
		}
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

	// Record the trail with the resolved/candidate ids.
	var trailIDs []int64
	if res.Exists {
		trailIDs = append(trailIDs, syms[0].ID)
	} else if res.Ambiguous {
		for _, c := range res.Candidates {
			trailIDs = append(trailIDs, c.ID)
		}
	}
	RecordTrail(ctx, st, "code_resolve", name, "", trailIDs...)

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
