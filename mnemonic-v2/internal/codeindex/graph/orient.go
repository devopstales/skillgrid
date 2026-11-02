package graph

import (
	"context"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// OrientUnitKind is the kind of unit to orient on.
type OrientUnitKind string

const (
	UnitSymbol  OrientUnitKind = "symbol"
	UnitPackage OrientUnitKind = "package"
	UnitFile    OrientUnitKind = "file"
)

// OrientUnit is the unit to orient on (spec 004 §8).
type OrientUnit struct {
	Kind OrientUnitKind
	ID   int64
	Path string
}

// OrientResult is the navigation map (spec 004 §8).
type OrientResult struct {
	Meta     string
	TOC      []store.SymbolRow
	Outbound []OrientEdge
	Depth    int
}

// OrientEdge is one outbound edge in the orient map.
type OrientEdge struct {
	Kind           string
	Confidence     string
	Line           int
	ToID           int64
	ToName         string
	TargetExpanded bool
}

// Orient returns the navigation map for a unit (spec 004 §8).
// The navigation entry point (the ki outline pattern): a compact, bounded view
// of the codebase that the agent can navigate.
func Orient(ctx context.Context, st store.Store, unit OrientUnit, depth int) (OrientResult, error) {
	if depth <= 0 {
		depth = 1
	}

	res := OrientResult{Depth: depth}

	switch unit.Kind {
	case UnitPackage:
		// Orient on a package: return the package's symbol TOC + outbound edges
		return orientPackage(ctx, st, unit.Path, depth)
	case UnitFile:
		// Orient on a file: return the file's symbol TOC + import edges
		return orientFile(ctx, st, unit.Path, depth)
	case UnitSymbol:
		// Orient on a symbol: return the symbol's metadata + signature + file TOC + outbound edges
		return orientSymbol(ctx, st, unit.ID, depth)
	default:
		return res, nil
	}
}

// orientPackage orients on a package: returns the package's symbol TOC + outbound edges.
func orientPackage(ctx context.Context, st store.Store, path string, depth int) (OrientResult, error) {
	res := OrientResult{Depth: depth, Meta: "package:" + path}

	// Get all files in the package
	files, err := st.FilesByLanguage(ctx, "go")
	if err != nil {
		return res, err
	}

	// Filter to files in this package
	var pkgFiles []store.FileRow
	for _, f := range files {
		if topLevelPackagePath(f.Path) == path {
			pkgFiles = append(pkgFiles, f)
		}
	}

	// Get all symbols in the package (the TOC)
	for _, f := range pkgFiles {
		syms, err := st.SymbolsByFile(ctx, f.ID)
		if err != nil {
			continue
		}
		res.TOC = append(res.TOC, syms...)
	}

	// Sort the TOC by file path, then start_line
	sortTOC(res.TOC, st)

	// Get outbound edges from all symbols in the package
	seen := map[int64]bool{}
	for _, sym := range res.TOC {
		if seen[sym.ID] {
			continue
		}
		seen[sym.ID] = true
		edges, err := st.EdgesByFrom(ctx, sym.ID, []string{"calls", "imports", "references"})
		if err != nil {
			continue
		}
		for _, e := range edges {
			var toID int64
			if e.ToID != nil {
				toID = *e.ToID
			} else if e.ToName != "" {
				// Try to resolve the name-only edge
				cands, err := st.SymbolByName(ctx, e.ToName)
				if err == nil && len(cands) == 1 {
					toID = cands[0].ID
				}
			}
			res.Outbound = append(res.Outbound, OrientEdge{
				Kind:           e.Kind,
				Confidence:     e.Confidence,
				Line:           e.Line,
				ToID:           toID,
				ToName:         e.ToName,
				TargetExpanded: false, // depth=1 does not expand targets
			})
		}
	}

	// Record the trail with the TOC + outbound target ids.
	RecordTrail(ctx, st, "code_orient", "package:"+path, "", orientTrailIDs(res)...)

	return res, nil
}

// orientFile orients on a file: returns the file's symbol TOC + import edges.
func orientFile(ctx context.Context, st store.Store, path string, depth int) (OrientResult, error) {
	res := OrientResult{Depth: depth, Meta: "file:" + path}

	file, err := st.FileByPath(ctx, path)
	if err != nil {
		return res, err
	}

	// Get the file's symbol TOC
	syms, err := st.SymbolsByFile(ctx, file.ID)
	if err != nil {
		return res, err
	}
	res.TOC = syms

	// Get outbound edges (imports) from all symbols in the file
	seen := map[int64]bool{}
	for _, sym := range syms {
		if seen[sym.ID] {
			continue
		}
		seen[sym.ID] = true
		edges, err := st.EdgesByFrom(ctx, sym.ID, []string{"imports"})
		if err != nil {
			continue
		}
		for _, e := range edges {
			var toID int64
			if e.ToID != nil {
				toID = *e.ToID
			} else if e.ToName != "" {
				cands, err := st.SymbolByName(ctx, e.ToName)
				if err == nil && len(cands) == 1 {
					toID = cands[0].ID
				}
			}
			res.Outbound = append(res.Outbound, OrientEdge{
				Kind:           e.Kind,
				Confidence:     e.Confidence,
				Line:           e.Line,
				ToID:           toID,
				ToName:         e.ToName,
				TargetExpanded: false,
			})
		}
	}

	// Record the trail with the TOC + outbound target ids.
	RecordTrail(ctx, st, "code_orient", "file:"+path, "", orientTrailIDs(res)...)

	return res, nil
}

// orientSymbol orients on a symbol: returns the symbol's metadata + signature + file TOC + outbound edges.
func orientSymbol(ctx context.Context, st store.Store, symID int64, depth int) (OrientResult, error) {
	res := OrientResult{Depth: depth}

	sym, err := st.SymbolByID(ctx, symID)
	if err != nil {
		return res, err
	}

	res.Meta = "symbol:" + sym.Name

	// Get the file TOC (all symbols in the containing file)
	if sym.FileID != 0 {
		syms, err := st.SymbolsByFile(ctx, sym.FileID)
		if err == nil {
			res.TOC = syms
		}
	}

	// Get outbound edges from this symbol
	edges, err := st.EdgesByFrom(ctx, symID, []string{"calls", "imports", "references"})
	if err != nil {
		return res, err
	}
	for _, e := range edges {
		var toID int64
		if e.ToID != nil {
			toID = *e.ToID
		} else if e.ToName != "" {
			cands, err := st.SymbolByName(ctx, e.ToName)
			if err == nil && len(cands) == 1 {
				toID = cands[0].ID
			}
		}
		res.Outbound = append(res.Outbound, OrientEdge{
			Kind:           e.Kind,
			Confidence:     e.Confidence,
			Line:           e.Line,
			ToID:           toID,
			ToName:         e.ToName,
			TargetExpanded: false,
		})
	}

	// Record the trail with the TOC + outbound target ids.
	RecordTrail(ctx, st, "code_orient", "symbol:"+sym.Name, "", orientTrailIDs(res)...)

	return res, nil
}

// orientTrailIDs collects the TOC symbol ids + outbound edge target ids for
// the trail (the audit answer to "which rows did this orient read touch?").
func orientTrailIDs(res OrientResult) []int64 {
	seen := map[int64]bool{}
	var ids []int64
	add := func(id int64) {
		if id != 0 && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	for _, s := range res.TOC {
		add(s.ID)
	}
	for _, e := range res.Outbound {
		add(e.ToID)
	}
	return ids
}

// sortTOC sorts the TOC by file path, then start_line.
func sortTOC(toc []store.SymbolRow, st store.Store) {
	// For now, just sort by start_line (the TOC is already in file order)
	for i := 1; i < len(toc); i++ {
		for j := i; j > 0 && toc[j].StartLine < toc[j-1].StartLine; j-- {
			toc[j], toc[j-1] = toc[j-1], toc[j]
		}
	}
}
