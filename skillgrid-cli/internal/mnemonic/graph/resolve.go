package graph

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
)

// ResolveFilter narrows symbol resolution. All fields optional; an empty
// filter is a pure name match.
type ResolveFilter struct {
	File string
	UID  string
	Kind string
}

// Resolution is a symbol resolution outcome. Exactly one of Found (len
// Matches == 1, Target set), Ambiguous (len Matches > 1), or NotFound is the
// state. Ambiguous resolution is NEVER a silent drop: Matches carries every
// candidate so callers can rank and present them.
type Resolution struct {
	Name      string
	Matches   []Symbol
	Ambiguous bool
	NotFound  bool
	// Target is set when resolution is unambiguous.
	Target Symbol
}

// Resolve resolves a symbol by name/file/uid. A name matching several
// symbols returns an Ambiguous resolution with every candidate (ranked); it
// never silently picks one. file/uid/kind narrow the match; an unsatisfied
// filter yields NotFound.
func Resolve(ctx context.Context, db *sql.DB, name string, filter ResolveFilter) (Resolution, error) {
	res := Resolution{Name: name}
	rows, err := db.QueryContext(ctx, `
		SELECT s.id, s.uid, s.name, s.qualified_name, s.kind, s.language,
		       f.path, s.start_line, s.end_line
		FROM symbols s INNER JOIN files f ON f.id = s.file_id
		WHERE s.name = ?
		ORDER BY s.id`, name)
	if err != nil {
		return res, err
	}
	defer rows.Close()
	for rows.Next() {
		var s Symbol
		if err := rows.Scan(&s.ID, &s.UID, &s.Name, &s.QualifiedName, &s.Kind,
			&s.Language, &s.Path, &s.StartLine, &s.EndLine); err != nil {
			return res, err
		}
		if filter.File != "" && s.Path != filter.File {
			continue
		}
		if filter.UID != "" && s.UID != filter.UID {
			continue
		}
		if filter.Kind != "" && s.Kind != filter.Kind {
			continue
		}
		res.Matches = append(res.Matches, s)
	}
	if err := rows.Err(); err != nil {
		return res, err
	}
	switch len(res.Matches) {
	case 0:
		res.NotFound = true
	case 1:
		res.Target = res.Matches[0]
	default:
		res.Ambiguous = true
	}
	return res, nil
}

// RankCandidates orders ambiguous candidates for presentation: most-connected
// first (hubs are the likelier intent), then most-referenced by name, then
// path/line for determinism.
func RankCandidates(ctx context.Context, db *sql.DB, matches []Symbol) ([]Symbol, error) {
	if len(matches) < 2 {
		return matches, nil
	}
	type scored struct {
		sym    Symbol
		deg    int
		refer  int
	}
	scoredList := make([]scored, 0, len(matches))
	for _, m := range matches {
		deg := 0
		_ = db.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM edges
			WHERE (from_id = ? AND to_id IS NOT NULL) OR to_id = ?`, m.ID, m.ID).Scan(&deg)
		refer := 0
		_ = db.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM edges WHERE to_id = ?`, m.ID).Scan(&refer)
		scoredList = append(scoredList, scored{sym: m, deg: deg, refer: refer})
	}
	sort.SliceStable(scoredList, func(i, j int) bool {
		if scoredList[i].deg != scoredList[j].deg {
			return scoredList[i].deg > scoredList[j].deg
		}
		if scoredList[i].refer != scoredList[j].refer {
			return scoredList[i].refer > scoredList[j].refer
		}
		if scoredList[i].sym.Path != scoredList[j].sym.Path {
			return scoredList[i].sym.Path < scoredList[j].sym.Path
		}
		return scoredList[i].sym.StartLine < scoredList[j].sym.StartLine
	})
	out := make([]Symbol, len(scoredList))
	for i, s := range scoredList {
		out[i] = s.sym
	}
	return out, nil
}

// FormatRefusal renders a refused/ambiguous name-only match for a
// where-the-graph-stops answer: "name [CONFI] at path:line".
func FormatRefusal(r RefusedMatch) string {
	if r.Path != "" {
		return fmt.Sprintf("%s [%s] at %s:%d", r.Name, r.Confidence, r.Path, r.Line)
	}
	return fmt.Sprintf("%s [%s] at line %d", r.Name, r.Confidence, r.Line)
}
