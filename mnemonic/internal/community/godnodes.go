package community

import (
	"database/sql"
	"fmt"
	"sort"
)

// GodNode is a symbol ranked by graph degree (the most-connected concepts in
// a set of symbols).
type GodNode struct {
	SymbolID int64  `json:"symbol_id"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	Degree   int    `json:"degree"`
	IsHub    bool   `json:"is_hub"`
}

// degreeOf counts the distinct symbol neighbors of id in the 005 edges table
// (forward or reverse), matching the graph package's notion of degree.
func degreeOf(db *sql.DB, id int64) int {
	var n int
	_ = db.QueryRow(`
		SELECT COUNT(DISTINCT other) FROM (
			SELECT to_id AS other FROM edges WHERE from_id = ? AND to_id IS NOT NULL
			UNION
			SELECT from_id AS other FROM edges WHERE to_id = ?
		)`, id, id).Scan(&n)
	return n
}

// RankGodNodes returns the most-connected symbols among the given ids,
// ranked by degree (descending), then name. excludeHubs suppresses utility
// super-hubs (symbols whose distinct-file reach crosses HubFileThreshold) so
// the ranking surfaces subsystem concepts rather than glue.
func RankGodNodes(db *sql.DB, symbolIDs []int64, excludeHubs bool) ([]GodNode, error) {
	if len(symbolIDs) == 0 {
		return []GodNode{}, nil
	}
	// Degree + path for every candidate in one query.
	ph := make([]any, len(symbolIDs))
	for i, id := range symbolIDs {
		ph[i] = id
	}
	query := fmt.Sprintf(`
		SELECT s.id, s.name, f.path,
			(SELECT COUNT(DISTINCT other) FROM (
				SELECT to_id AS other FROM edges WHERE from_id = s.id AND to_id IS NOT NULL
				UNION
				SELECT from_id AS other FROM edges WHERE to_id = s.id
			)) AS degree
		FROM symbols s INNER JOIN files f ON f.id = s.file_id
		WHERE s.id IN (%s)`, placeholders(len(symbolIDs)))
	rows, err := db.Query(query, ph...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GodNode
	for rows.Next() {
		var gn GodNode
		if err := rows.Scan(&gn.SymbolID, &gn.Name, &gn.Path, &gn.Degree); err != nil {
			return nil, err
		}
		out = append(out, gn)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Degree != out[j].Degree {
			return out[i].Degree > out[j].Degree
		}
		return out[i].Name < out[j].Name
	})
	if !excludeHubs {
		return out, nil
	}
	hubs := hubSymbols(db)
	kept := out[:0]
	for _, gn := range out {
		if hubs[gn.SymbolID] {
			gn.IsHub = true
			continue
		}
		kept = append(kept, gn)
	}
	return kept, nil
}

// HubFileThreshold is the number of distinct files a symbol must be
// referenced across to be considered a utility super-hub.
const HubFileThreshold = 5

// hubSymbols returns the set of symbol ids that are utility super-hubs: their
// incoming edges originate from >= HubFileThreshold distinct files (the
// source file of each edge, resolved via the source symbol's file).
func hubSymbols(db *sql.DB) map[int64]bool {
	rows, err := db.Query(`
		SELECT e.to_id, COUNT(DISTINCT srcf.id)
		FROM edges e
		JOIN symbols srcs ON srcs.id = e.from_id
		JOIN files srcf ON srcf.id = srcs.file_id
		WHERE e.to_id IS NOT NULL
		GROUP BY e.to_id
		HAVING COUNT(DISTINCT srcf.id) >= ?`, HubFileThreshold)
	if err != nil {
		return map[int64]bool{}
	}
	defer rows.Close()
	out := map[int64]bool{}
	for rows.Next() {
		var id int64
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return out
		}
		out[id] = true
	}
	return out
}

// placeholders builds "?, ?, ?" for an IN clause.
func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	s := ""
	for i := 0; i < n; i++ {
		if i > 0 {
			s += ","
		}
		s += "?"
	}
	return s
}
