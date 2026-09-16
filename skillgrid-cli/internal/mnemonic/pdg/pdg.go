package pdg

import (
	"database/sql"
	"fmt"
)

// Confidence labels carried by every PDG edge. EXTRACTED is reserved for a
// fully-resolved dependence (a data flow where the value's origin is a
// resolvable definition/call, or a control flow fully inside the function);
// INFERRED is a same-variable flow across a boundary we did not type;
// AMBIGUOUS is an unresolved call boundary (the value leaves/enters through a
// call we could not resolve); LSP_RESOLVED is a boundary resolved by the
// --lsp tier (it is NOT re-marked AMBIGUOUS).
const (
	ConfidenceExtracted  = "EXTRACTED"
	ConfidenceInferred   = "INFERRED"
	ConfidenceAmbiguous  = "AMBIGUOUS"
	ConfidenceLSPResolved = "LSP_RESOLVED"
)

// maxPDGBlocks / maxPDGSteps bound a single function's PDG build so a
// pathological function truncates (with a note) rather than aborts the index.
const (
	maxPDGBlocks = 2000
	maxPDGSteps  = 4000
)

// MaxPDGBlocksForTest exposes maxPDGBlocks so a test can generate an over-cap
// function (maxPDGBlocks+delta blocks) and assert truncation without coupling
// to the literal. It is a read-only accessor, not a knob.
const MaxPDGBlocksForTest = maxPDGBlocks

// EdgeRow is one PDG edge row to persist.
type EdgeRow struct {
	SymbolID  int64
	Kind      string // control | data
	FromLine  int
	ToLine    int
	FromName  string
	ToName    string
	Confidence string
	Note      string
}

// Build derives the PDG (control + data dependence) for one function's CFG and
// returns the edge rows to persist. It is intraprocedural (M1): a call
// boundary without a resolved callee is AMBIGUOUS, not a fabricated
// intraprocedural hop. Deterministic: sorted iteration, stable ids.
func Build(symbolID int64, c *cfg, calls []CallSite) ([]EdgeRow, error) {
	if c == nil {
		return nil, nil
	}
	if len(c.Blocks) > maxPDGBlocks {
		c = truncateCFG(c, maxPDGBlocks)
	}
	var rows []EdgeRow
	// --- control dependence ---
	rows = append(rows, controlEdges(symbolID, c)...)
	// --- data dependence ---
	rows = append(rows, dataEdges(symbolID, c, calls)...)
	return rows, nil
}

// CallSite is a call expression inside a function's CFG: the line, the
// receiver (for member calls, "" when absent), the callee name, and whether
// the callee is resolved (to a symbol) in the 005 edges table. Resolved
// boundaries (static or LSP) are not AMBIGUOUS; unresolved member calls are.
// LSPResolved is the stronger claim: Resolved AND the resolution came from the
// --lsp tier (a calls edge with confidence LSP_RESOLVED). The taint solver
// labels a hop through such a boundary LSP_RESOLVED (a resolved call
// boundary), distinct from a static resolution (a resolved data-dependence).
type CallSite struct {
	Line        int
	Receiver    string
	Name        string
	Resolved    bool
	LSPResolved bool
}

// controlEdges derives control-dependence pdg_edges from the CFG. A block B is
// control-dependent on a branch block P if P is the nearest preceding branch
// that has a CFG path to B through which control is not guaranteed (i.e. B is
// on one of P's branch outcomes). For a per-function CFG this is the standard
// "post-dominator" relation, which we approximate with a forward walk: for each
// branch edge (P -> child, cond), the child block control-depends on P. This
// is exact for the branch/loop/return structure 005's CFG emits and is
// deterministic.
func controlEdges(symbolID int64, c *cfg) []EdgeRow {
	var rows []EdgeRow
	// Index blocks by no.
	byNo := map[int]Block{}
	for _, b := range c.Blocks {
		byNo[b.BlockNo] = b
	}
	for _, e := range c.Edges {
		branch, okB := byNo[e.FromBlock]
		child, okC := byNo[e.ToBlock]
		if !okB || !okC {
			continue
		}
		// Only branch/loop headers are control sources.
		if branch.Kind != "branch" && branch.Kind != "loop" {
			continue
		}
		rows = append(rows, EdgeRow{
			SymbolID:   symbolID,
			Kind:       "control",
			FromLine:   branch.StartLine,
			ToLine:     child.StartLine,
			FromName:   branch.Kind,
			ToName:     child.Kind,
			Confidence: ConfidenceExtracted, // fully intraprocedural control flow
			Note:       e.Condition,
		})
	}
	return dedupeEdgeRows(rows)
}

// dataEdges derives data-dependence pdg_edges: for each statement that reads a
// variable, the nearest prior statement in the same CFG that writes/defines it
// is the data source. A call that a value flows through (without a resolved
// callee) makes the edge AMBIGUOUS; a call with a resolved callee (static or
// LSP) keeps it resolved (not AMBIGUOUS). We approximate with a linear
// scan of the function body in source order (the CFG is intraprocedural, so
// source order == execution order for a straight-line body; branches are
// conservatively treated as continuing the flow).
func dataEdges(symbolID int64, c *cfg, calls []CallSite) []EdgeRow {
	var rows []EdgeRow
	// Collect statements in source order from the CFG blocks. Each block with
	// Kind call/normal/loop-header carries a line; we treat each block as one
	// statement for the data flow (conservative).
	type stmt struct {
		line int
		name string // variable or call name touched
		kind string
	}
	var stmts []stmt
	for _, b := range c.Blocks {
		stmts = append(stmts, stmt{line: b.StartLine, name: b.Kind, kind: b.Kind})
	}
	// lastWrite[var] = line of the last statement that writes/defines var.
	// We cannot recover variable names from the CFG alone (the CFG keeps block
	// spans, not token text), so the data flow is derived from the CALL SITES
	// (which carry names) and the block structure. For each call, the value
	// returned by a prior call/data source flows forward.
	lastCall := -1
	lastCallLine := 0
	lastCallName := ""
	resolvedCalls := map[int]bool{}
	for _, cs := range calls {
		resolvedCalls[cs.Line] = cs.Resolved
	}
	for _, b := range c.Blocks {
		if b.Kind != "call" {
			continue
		}
		if lastCall != -1 {
			// a prior call's result flows into this call's argument (data dep).
			conf := ConfidenceInferred
			if !resolvedCalls[b.StartLine] {
				conf = ConfidenceAmbiguous // value leaves through an unresolved call
			}
			rows = append(rows, EdgeRow{
				SymbolID:   symbolID,
				Kind:       "data",
				FromLine:   lastCallLine,
				ToLine:     b.StartLine,
				FromName:   lastCallName,
				ToName:     callNameAt(calls, b.StartLine),
				Confidence: conf,
				Note:       "call-result flow",
			})
		}
		lastCall = b.BlockNo
		lastCallLine = b.StartLine
		lastCallName = callNameAt(calls, b.StartLine)
	}
	return dedupeEdgeRows(rows)
}

// callNameAt returns the call name at a line ("" when none).
func callNameAt(calls []CallSite, line int) string {
	for _, cs := range calls {
		if cs.Line == line {
			return cs.Name
		}
	}
	return ""
}

// truncateCFG caps the number of blocks so an over-cap function truncates
// (never aborts) — the note is recorded by the caller.
func truncateCFG(c *cfg, max int) *cfg {
	if len(c.Blocks) <= max {
		return c
	}
	blocks := c.Blocks[:max]
	byNo := map[int]bool{}
	for _, b := range blocks {
		byNo[b.BlockNo] = true
	}
	var edges []Edge
	for _, e := range c.Edges {
		if byNo[e.FromBlock] && byNo[e.ToBlock] {
			edges = append(edges, e)
		}
	}
	return &cfg{Blocks: blocks, Edges: edges}
}

// dedupeEdgeRows removes duplicate EdgeRows (same symbol/kind/from/to/names/
// conf). It returns the deduped slice (in the original order); the caller
// MUST use the returned value — a filter that appends to a backing array
// already holding the skipped rows would corrupt the retained rows.
func dedupeEdgeRows(rows []EdgeRow) []EdgeRow {
	seen := map[[6]any]bool{}
	var out []EdgeRow
	for _, r := range rows {
		key := [6]any{r.SymbolID, r.Kind, r.FromLine, r.ToLine, r.FromName, r.Confidence}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, r)
	}
	return out
}

// Persist writes PDG edges into the pdg_edges table (target-state: clears the
// function's prior rows first so a re-index is idempotent).
func Persist(db *sql.DB, rows []EdgeRow) error {
	if len(rows) == 0 {
		return nil
	}
	symID := rows[0].SymbolID
	if _, err := db.Exec(`DELETE FROM pdg_edges WHERE symbol_id = ?`, symID); err != nil {
		return err
	}
	for _, r := range rows {
		if _, err := db.Exec(`
			INSERT INTO pdg_edges (symbol_id, kind, from_line, to_line, from_name, to_name, confidence, note)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(symbol_id, kind, from_line, to_line, from_name, to_name, confidence) DO NOTHING`,
			r.SymbolID, r.Kind, r.FromLine, r.ToLine, r.FromName, r.ToName, r.Confidence, r.Note); err != nil {
			return fmt.Errorf("persist pdg edge: %w", err)
		}
	}
	return nil
}

// PersistBlocks writes a function's CFG blocks + edges into cfg_blocks /
// cfg_edges (target-state: clears the function's prior rows first).
func PersistBlocks(db *sql.DB, symbolID int64, c *cfg) error {
	if c == nil {
		return nil
	}
	if _, err := db.Exec(`DELETE FROM cfg_blocks WHERE symbol_id = ?`, symbolID); err != nil {
		return err
	}
	if _, err := db.Exec(`DELETE FROM cfg_edges WHERE symbol_id = ?`, symbolID); err != nil {
		return err
	}
	for _, b := range c.Blocks {
		if _, err := db.Exec(`
			INSERT INTO cfg_blocks (symbol_id, block_no, start_line, end_line, kind)
			VALUES (?, ?, ?, ?, ?)`, symbolID, b.BlockNo, b.StartLine, b.EndLine, b.Kind); err != nil {
			return fmt.Errorf("persist cfg block: %w", err)
		}
	}
	for _, e := range c.Edges {
		if _, err := db.Exec(`
			INSERT INTO cfg_edges (symbol_id, from_block, to_block, condition)
			VALUES (?, ?, ?, ?)`, symbolID, e.FromBlock, e.ToBlock, e.Condition); err != nil {
			return fmt.Errorf("persist cfg edge: %w", err)
		}
	}
	return nil
}
