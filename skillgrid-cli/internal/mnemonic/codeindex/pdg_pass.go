package codeindex

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/pdg"
)

// pdgPass runs the opt-in per-function CFG + PDG pass over the indexed graph,
// on the pass DB (a fresh *sql.DB opened after the 005 tx commits — the store's
// single-connection pool cannot share that tx). It is gated on the --pdg flag;
// without it it never runs (the cfg/pdg tables stay empty). Advisory, never
// load-bearing: a per-function failure warns and continues.
func (idx *Indexer) pdgPass(ctx context.Context, passDB *sql.DB, scanned []ScannedFile) error {
	if tableMissing(passDB, "cfg_blocks") {
		return nil // pre-016 store — nothing to do
	}
	// Group files by path for source lookup.
	fileByPath := map[string][]byte{}
	for _, f := range scanned {
		fileByPath[f.Path] = f.Contents
	}
	// Load the function/method symbols (intradependent per function).
	syms, err := pdgFunctions(passDB)
	if err != nil {
		return err
	}
	if len(syms) == 0 {
		return nil
	}
	// Resolve call sites (member calls + their LSP_RESOLVED status) from 005's
	// edges table. The LSP tier (when run first) has already written
	// LSP_RESOLVED edges, so a member call with one is Resolved.
	callsBySymbol, err := pdgCallSites(passDB)
	if err != nil {
		return err
	}
	for _, s := range syms {
		if err := ctx.Err(); err != nil {
			return err
		}
		relPath, err := fileIDToPath(passDB, s.FileID)
		if err != nil {
			continue
		}
		src, ok := fileByPath[relPath]
		if !ok {
			// File not scanned this run (unchanged): read from disk.
			if b, rerr := os.ReadFile(filepath.Join(scanRoot, relPath)); rerr == nil {
				src = b
			} else {
				continue
			}
		}
		c, err := pdg.BuildCFG(src, s.Language, s.Name, s.StartLine)
		if err != nil {
			// Malformed function CFG: skip + continue (01.8).
			fmt.Fprintf(os.Stderr, "warn: pdg cfg %s (%s): %v\n", s.Name, relPath, err)
			continue
		}
		if c == nil {
			continue // body not locatable — skip
		}
		calls := callsBySymbol[s.ID]
		rows, err := pdg.Build(s.ID, c, calls)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: pdg build %s: %v\n", s.Name, err)
			continue
		}
		if err := pdg.PersistBlocks(passDB, s.ID, c); err != nil {
			fmt.Fprintf(os.Stderr, "warn: pdg persist blocks %s: %v\n", s.Name, err)
			continue
		}
		if err := pdg.Persist(passDB, rows); err != nil {
			fmt.Fprintf(os.Stderr, "warn: pdg persist %s: %v\n", s.Name, err)
		}
		// Taint (011 step 02): the opt-in source->sink solver runs AFTER the
		// PDG derivation (it consumes the just-built dataDeps + call
		// boundaries). Additive: it only writes taint_findings (a non---pdg
		// index never reaches this point, so the table stays empty and the
		// 005/008/010 graph is byte-for-byte unchanged).
		ti := taintInput(s.ID, rows, calls)
		findings := pdg.Taint(ti, pdg.DefaultTaintConfig())
		if err := pdg.PersistTaint(passDB, s.ID, findings); err != nil {
			fmt.Fprintf(os.Stderr, "warn: taint persist %s: %v\n", s.Name, err)
		}
	}
	return nil
}

// taintInput assembles the solver's input for one function from its PDG rows
// (data-dependence edges only) and call sites. Deterministic: dataDeps are
// sorted by (from_line, to_line, from_name).
func taintInput(symbolID int64, rows []pdg.EdgeRow, calls []pdg.CallSite) pdg.TaintInput {
	in := pdg.TaintInput{SymbolID: symbolID, ByLine: map[int]*pdg.CallInfo{}}
	for i := range calls {
		cs := &calls[i]
		in.ByLine[cs.Line] = &pdg.CallInfo{
			Name: cs.Name, Resolved: cs.Resolved, LSPResolved: cs.LSPResolved,
		}
	}
	var deps []struct {
		fromLine, toLine int
		fromName, toName, conf, note string
	}
	for _, r := range rows {
		if r.Kind != "data" {
			continue
		}
		deps = append(deps, struct {
			fromLine, toLine int
			fromName, toName, conf, note string
		}{r.FromLine, r.ToLine, r.FromName, r.ToName, r.Confidence, r.Note})
	}
	sort.Slice(deps, func(i, j int) bool {
		if deps[i].fromLine != deps[j].fromLine {
			return deps[i].fromLine < deps[j].fromLine
		}
		if deps[i].toLine != deps[j].toLine {
			return deps[i].toLine < deps[j].toLine
		}
		return deps[i].fromName < deps[j].fromName
	})
	for _, d := range deps {
		in.DataDeps = append(in.DataDeps, pdg.DataDep{
			FromLine: d.fromLine, ToLine: d.toLine, FromName: d.fromName,
			ToName: d.toName, Confidence: d.conf, Note: d.note,
		})
	}
	return in
}

// scanRoot is the root the pdg pass reads unchanged files from. It is set by
// Run before the pass; for a cold index it is the scan root, for an incremental
// one the scanned files are already in memory (fileByPath) so this is a
// fallback.
var scanRoot string

// pdgFunctions loads the function/method symbols that a CFG can be built for.
type pdgFunction struct {
	ID        int64
	FileID    int64
	Name      string
	Language  string
	StartLine int
	EndLine   int
}

func pdgFunctions(db *sql.DB) ([]pdgFunction, error) {
	rows, err := db.Query(`
		SELECT s.id, s.file_id, s.name, COALESCE(s.language,'go'), s.start_line, s.end_line
		FROM symbols s
		WHERE s.kind IN ('function','method')
		ORDER BY s.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []pdgFunction
	for rows.Next() {
		var f pdgFunction
		if err := rows.Scan(&f.ID, &f.FileID, &f.Name, &f.Language, &f.StartLine, &f.EndLine); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// pdgCallSites resolves, per symbol, the call sites in its body and whether
// each is resolved (a calls edge with a non-NULL to_id, or an LSP_RESOLVED
// edge). Returns symbol id -> []pdg.CallSite.
func pdgCallSites(db *sql.DB) (map[int64][]pdg.CallSite, error) {
	// Join calls edges to their from-symbol; to_id NULL means unresolved.
	// The call name is the callee (to_name), not the from-symbol's name.
	// to_id NULL means unresolved; a non-NULL to_id is a static resolution
	// (EXTRACTED). LSP_RESOLVED edges carry a non-NULL to_id AND the
	// LSP_RESOLVED confidence (the --lsp tier wrote them).
	rows, err := db.Query(`
		SELECT e.from_id, e.line, e.to_name,
		       e.to_id IS NOT NULL, e.confidence = 'LSP_RESOLVED'
		FROM edges e
		WHERE e.kind = 'calls' AND e.to_name IS NOT NULL
		ORDER BY e.from_id, e.line`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64][]pdg.CallSite{}
	for rows.Next() {
		var symID int64
		var line int
		var name string
		var resolved, lspResolved bool
		if err := rows.Scan(&symID, &line, &name, &resolved, &lspResolved); err != nil {
			return nil, err
		}
		out[symID] = append(out[symID], pdg.CallSite{
			Line:        line,
			Name:        name,
			Resolved:    resolved,
			LSPResolved: resolved && lspResolved,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// fileIDToPath resolves a file id to its path.
func fileIDToPath(db *sql.DB, fileID int64) (string, error) {
	var path string
	err := db.QueryRow(`SELECT path FROM files WHERE id = ?`, fileID).Scan(&path)
	return path, err
}

// lspPass runs the independent opt-in language-server edge tier: it resolves
// member calls the static pass could not type and writes LSP_RESOLVED edges
// into 005's edges table. It is gated on the --lsp flag (independent of --pdg)
// and runs BEFORE the PDG pass so LSP_RESOLVED edges exist before the PDG
// derives dependences. Best-effort: an absent/failing/timing-out server warns
// and continues, leaving the static index unchanged (no partial edge set).
// See pdg/lsp.go for the external-process adapter.
func (idx *Indexer) lspPass(ctx context.Context, passDB *sql.DB, scanned []ScannedFile) error {
	if tableMissing(passDB, "edges") {
		return nil
	}
	// The resolver seam (test hook / real server) is resolved once; when a
	// hermetic resolver is set (lspResolver) it is used, otherwise the client
	// shells out to the language server on PATH.
	files := func() []pdg.LSPFile {
		var out []pdg.LSPFile
		for _, f := range scanned {
			if lang := pdg.LSPLanguageForPath(f.Path); lang != "" {
				out = append(out, pdg.LSPFile{Path: f.Path, Contents: f.Contents})
			}
		}
		return out
	}
	resolver := idx.lspResolver
	timeout := idx.lspTimeout // 0 -> adapter default (30s)
	var client *pdg.LSPClient
	if resolver != nil {
		// Hermetic resolver: construct the client directly (no PATH lookup) and
		// attach the resolver seam.
		client = pdg.NewLSPClientForTest(pdg.LSPClientOptions{Root: scanRoot, Files: files, Timeout: timeout}, resolver)
	} else {
		c, err := pdg.NewLSPClient(pdg.LSPClientOptions{
			Root:    scanRoot,
			Files:   files,
			Timeout: timeout,
		})
		if err != nil {
			// No resolvable server for the scanned languages: warn + continue,
			// static index unchanged (01.3 / 01.10).
			fmt.Fprintf(os.Stderr, "warn: lsp tier: %v (static index unchanged)\n", err)
			return nil
		}
		client = c
	}
	// Bound the round-trip by the configured timeout so BOTH the hermetic seam
	// path and the real-server path are bounded the same way (fix #1: a
	// hanging/failing server is a best-effort no-op, not an indefinite block).
	// timeout=0 keeps the caller ctx (the default hermetic path is unbounded).
	bctx, cancel := pdg.BoundedCtx(ctx, timeout)
	defer cancel()
	edges, err := client.ResolveMemberCalls(bctx)
	if err != nil {
		// Best-effort: a failing/timing-out server is a no-op (static index
		// unchanged, no partial edge set) (01.10).
		fmt.Fprintf(os.Stderr, "warn: lsp tier: %v (static index unchanged)\n", err)
		return nil
	}
	if len(edges) == 0 {
		return nil
	}
	if _, err := pdg.PersistResolvedEdges(passDB, scanRoot, edges); err != nil {
		fmt.Fprintf(os.Stderr, "warn: lsp persist: %v (static index unchanged)\n", err)
	}
	return nil
}

// lspPass is a no-op for languages without a known server; the adapter in
// pdg/lsp.go resolves the server binary on PATH and shells out via JSON-RPC.

// (helper) sort is imported for deterministic ordering in call-site maps.
var _ = sort.Strings

// (helper) strings is imported for language-detection helpers.
var _ = strings.TrimSpace
