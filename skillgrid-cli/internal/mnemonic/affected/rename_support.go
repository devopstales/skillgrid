package affected

import (
	"context"
	"database/sql"
	"os"
	"sort"
)

// resolveFilter is the 005 disambiguation filter (file/uid/kind) used by
// rename; it mirrors graph.ResolveFilter so the affected package stays
// independent of the graph package (no import cycle).
type resolveFilter struct {
	File string
	UID  string
	Kind string
}

// resolution is a symbol resolution outcome (mirrors graph.Resolution).
type resolution struct {
	Matches   []sym
	Ambiguous bool
	NotFound  bool
	Target    sym
}

// resolveSymbol resolves a symbol by name with an optional file/uid/kind
// narrow. Ambiguous (multiple matches) is never a silent pick: all matches
// are returned for ranking.
func resolveSymbol(ctx context.Context, db *sql.DB, name string, filter resolveFilter) (resolution, error) {
	var res resolution
	rows, err := db.QueryContext(ctx, `
		SELECT s.id, s.name, s.kind, s.uid, f.path
		FROM symbols s JOIN files f ON f.id = s.file_id
		WHERE s.name = ?
		ORDER BY s.id`, name)
	if err != nil {
		return res, err
	}
	defer rows.Close()
	for rows.Next() {
		var s sym
		var kind string
		if err := rows.Scan(&s.ID, &s.Name, &kind, &s.UID, &s.Path); err != nil {
			return res, err
		}
		s.Kind = kind
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

// rankCandidates orders ambiguous candidates: most-connected first (hubs are
// the likelier intent), then path for determinism.
func rankCandidates(ctx context.Context, db *sql.DB, matches []sym) []sym {
	type scored struct {
		sym sym
		deg int
	}
	list := make([]scored, 0, len(matches))
	for _, m := range matches {
		deg := 0
		_ = db.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM edges
			WHERE (from_id = ? AND to_id IS NOT NULL) OR to_id = ?`, m.ID, m.ID).Scan(&deg)
		list = append(list, scored{sym: m, deg: deg})
	}
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].deg != list[j].deg {
			return list[i].deg > list[j].deg
		}
		if list[i].sym.Path != list[j].sym.Path {
			return list[i].sym.Path < list[j].sym.Path
		}
		return list[i].sym.UID < list[j].sym.UID
	})
	out := make([]sym, len(list))
	for i, s := range list {
		out[i] = s.sym
	}
	return out
}

// symbolDTO renders a resolved symbol for the plan.
func symbolDTO(s sym) map[string]any {
	return map[string]any{"name": s.Name, "uid": s.UID, "path": s.Path}
}

// readFileInto / writeFileFromDB are thin file I/O helpers for the apply
// path. The indexed path is repo-relative; it is resolved against the
// process CWD (the CLI/MCP run in the repo root).
func readFileInto(db *sql.DB, path string, out *[]byte) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	*out = b
	return nil
}

func writeFileFromDB(db *sql.DB, path string, content []byte) error {
	return os.WriteFile(path, content, 0o644)
}
