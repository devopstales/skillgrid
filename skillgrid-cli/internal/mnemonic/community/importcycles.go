package community

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"
)

// ImportCycle is one dependency loop discovered in the file import subgraph:
// the file paths in traversal order (the start file repeats at the end to
// close the loop) and the distinct-file count. A cycle of node_count 2 is the
// classic A imports B imports A; node_count 1 is a self-import.
type ImportCycle struct {
	Cycle     []string `json:"cycle"`
	NodeCount int      `json:"node_count"`
}

// FileEdge is one directed import edge between two files.
type FileEdge struct {
	From, To string
}

// FindImportCycles detects simple directed cycles in the file import subgraph
// built from the 005 edges table (kind='imports', resolved to file paths).
//
// This is a graphify report signal: dependency loops (A→B→A) are a structural
// smell the index surfaces to an agent rather than letting them silently
// accumulate. The detection is over FILES (not symbols) because an import is a
// file-level relationship — a symbol in A imports a symbol in B, which means
// file A depends on file B. We restrict to kind='imports' (not calls) because
// a call cycle is normal and expected (mutual recursion); an IMPORT cycle is
// the rarer, higher-signal smell.
//
// Algorithm: color-based DFS (white/gray/black) over the file import graph.
// A back edge to a GRAY (on-stack) node closes a cycle; the stack slice from
// that node to the top is the cycle. Each cycle is canonicalized (rotated so
// the lexicographically-smallest file is first) and de-duplicated so a cycle
// reached from multiple entry points is reported once. Self-loops (A imports
// A) are reported as a single-file cycle. The result is deterministic (files
// are sorted, adjacency lists are sorted) so an unchanged import graph yields
// the same cycle set across index runs.
//
// Advisory, never load-bearing: callers (the index pass) warn-and-continue on
// error. A large import graph is fine — the DFS is O(V+E) and the gray-stack
// bound keeps per-cycle work small.
func FindImportCycles(db *sql.DB) ([]ImportCycle, error) {
	edges, err := importFileEdges(db)
	if err != nil {
		return nil, err
	}
	return DetectCycles(edges)
}

// importFileEdges reads the 005 edges table for kind='imports' and resolves
// each edge to its from-file and to-file paths. Import edges carry the raw
// import path in target_path (they are never symbol-resolved to a to_id), so
// the target is resolved to an indexed file by prefix-matching the import
// path against indexed file paths (a Go import "github.com/devopstales/.../pkg"
// matches the indexed file "github.com/devopstales/.../pkg/x.go"). An import
// edge whose target is not in the index (external/stdlib — e.g. "context",
// "testing") is dropped: a cycle needs both endpoints present. Edges are
// de-duplicated to file pairs (many files import the same package).
func importFileEdges(db *sql.DB) ([]FileEdge, error) {
	rows, err := db.Query(`
		SELECT e.file_id, f_from.path, e.target_path, f_to.path
		FROM edges e
		JOIN files f_from ON f_from.id = e.file_id
		LEFT JOIN files f_to ON e.kind = 'imports'
			AND e.target_path IS NOT NULL AND e.target_path != ''
			AND f_to.path LIKE e.target_path || '%'
		WHERE e.kind = 'imports'
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []FileEdge
	seen := map[[2]string]bool{}
	for rows.Next() {
		var fileID int64
		var from, target, to string
		if err := rows.Scan(&fileID, &from, &target, &to); err != nil {
			return nil, err
		}
		if to == "" {
			continue // external/stdlib import — not an indexed file
		}
		key := [2]string{from, to}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, FileEdge{From: from, To: to})
	}
	return out, rows.Err()
}

// DetectCycles runs the color-DFS cycle detection over a file import edge
// list. Pure function of the edges (no DB) so it is directly unit-testable.
// Returns cycles in deterministic order (each cycle lexicographically sorted
// by its first file, then by subsequent files).
func DetectCycles(edges []FileEdge) ([]ImportCycle, error) {
	adj := map[string][]string{}
	for _, e := range edges {
		adj[e.From] = append(adj[e.From], e.To)
	}
	// Sort adjacency for deterministic traversal.
	for k := range adj {
		sort.Strings(adj[k])
	}
	var nodes []string
	for k := range adj {
		nodes = append(nodes, k)
	}
	// Include target-only files so a cycle that only reaches a file as a
	// target is still traversed (a pure-target file has no outgoing edge and
	// contributes no cycle, but this keeps the node set complete).
	for _, e := range edges {
		if _, ok := adj[e.To]; !ok {
			nodes = append(nodes, e.To)
		}
	}
	sort.Strings(nodes)

	const (
		white = 0 // unvisited
		gray  = 1 // on the current DFS stack
		black = 2 // fully explored
	)
	color := map[string]int{}
	var stack []string
	var out []ImportCycle
	canonicalSeen := map[string]bool{}

	var dfs func(u string)
	dfs = func(u string) {
		color[u] = gray
		stack = append(stack, u)
		for _, v := range adj[u] {
			switch color[v] {
			case white:
				dfs(v)
			case gray:
				// Back edge to an on-stack node: the stack from v to u is a cycle.
				start := 0
				for i, s := range stack {
					if s == v {
						start = i
						break
					}
				}
				cycle := append(append([]string{}, stack[start:]...), v)
				key := canonicalKey(cycle)
				if !canonicalSeen[key] {
					canonicalSeen[key] = true
					canon := canonicalize(cycle)
					// NodeCount is the distinct-file count: the closed cycle
					// repeats its start at the end, so subtract one.
					out = append(out, ImportCycle{Cycle: canon, NodeCount: len(canon) - 1})
				}
			}
		}
		stack = stack[:len(stack)-1]
		color[u] = black
	}

	for _, n := range nodes {
		if color[n] == white {
			dfs(n)
		}
	}
	sortCycles(out)
	return out, nil
}

// canonicalize rotates a cycle so the lexicographically-smallest file is
// first (the cycle is already closed: last element == first). This makes two
// traversals of the same loop produce the same ordering for de-duplication.
func canonicalize(cycle []string) []string {
	// cycle = [a, b, c, a]; the distinct nodes are cycle[:len-1].
	body := cycle[:len(cycle)-1]
	minIdx := 0
	for i, s := range body {
		if s < body[minIdx] {
			minIdx = i
		}
	}
	rot := append([]string{}, body[minIdx:]...)
	rot = append(rot, body[:minIdx]...)
	rot = append(rot, rot[0])
	return rot
}

// canonicalKey is a string key of the canonicalized cycle for de-duplication.
func canonicalKey(cycle []string) string {
	return strings.Join(canonicalize(cycle), "\x00")
}

// sortCycles orders the result deterministically: by node_count, then by the
// first file, then the rest.
func sortCycles(cycles []ImportCycle) {
	sort.Slice(cycles, func(a, b int) bool {
		ca, cb := cycles[a], cycles[b]
		if ca.NodeCount != cb.NodeCount {
			return ca.NodeCount < cb.NodeCount
		}
		for i := 0; i < len(ca.Cycle) && i < len(cb.Cycle); i++ {
			if ca.Cycle[i] != cb.Cycle[i] {
				return ca.Cycle[i] < cb.Cycle[i]
			}
		}
		return false
	})
}

// WriteImportCycles replaces the import_cycles table with the detected set
// (target-state: the re-detected set is the full truth, stale cycles are
// pruned). Advisory — the caller warns-and-continues on error.
func WriteImportCycles(db *sql.DB, cycles []ImportCycle) error {
	if tableMissing(db, "import_cycles") {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM import_cycles`); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for _, c := range cycles {
		if _, err := tx.Exec(`INSERT INTO import_cycles (cycle, node_count, discovered) VALUES (?, ?, ?)`,
			strings.Join(c.Cycle, ","), c.NodeCount, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// tableMissing reports whether a table is absent (pre-migration store).
func tableMissing(db *sql.DB, name string) bool {
	var n string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, name).Scan(&n)
	return err == sql.ErrNoRows
}

// RunImportCyclePass runs the full detect + write for the index pass (kept
// separate from FindImportCycles so a caller with an existing DB handle can
// invoke it in one step). Advisory.
func RunImportCyclePass(ctx context.Context, db *sql.DB) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	cycles, err := FindImportCycles(db)
	if err != nil {
		return fmt.Errorf("import cycles: %w", err)
	}
	return WriteImportCycles(db, cycles)
}
