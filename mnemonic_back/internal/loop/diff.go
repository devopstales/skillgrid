package loop

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// ImpactLine is one changed file and how many direct callers the first
// indexed symbol in that file has. Empty symbol means the file is not in
// the code index yet.
type ImpactLine struct {
	File    string
	Symbol  string
	Callers int
}

// DiffImpact maps changed files to indexed symbols and counts direct callers.
// It reuses the edges table code_impact reads. Files with no symbol are still
// listed so the diff is visible before the next index.
func DiffImpact(ctx context.Context, db *sql.DB, files []string) ([]ImpactLine, error) {
	if db == nil {
		return nil, fmt.Errorf("diff impact: no store")
	}
	out := make([]ImpactLine, 0, len(files))
	for _, file := range files {
		file = strings.TrimSpace(file)
		if file == "" {
			continue
		}
		var symID int64
		var name string
		err := db.QueryRowContext(ctx, `
			SELECT s.id, s.name
			FROM symbols s
			JOIN files f ON f.id = s.file_id
			WHERE f.path = ?
			ORDER BY s.start_line
			LIMIT 1`, file).Scan(&symID, &name)
		line := ImpactLine{File: file}
		if err == nil {
			line.Symbol = name
			var n int
			_ = db.QueryRowContext(ctx, `
				SELECT COUNT(*) FROM edges WHERE to_id = ? AND kind IN ('calls', 'call')`, symID).Scan(&n)
			line.Callers = n
		}
		out = append(out, line)
	}
	return out, nil
}

// FormatImpact renders DiffImpact rows for the prime block.
func FormatImpact(lines []ImpactLine) []string {
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		if l.Symbol == "" {
			out = append(out, l.File+" (not indexed)")
			continue
		}
		out = append(out, fmt.Sprintf("%s %s callers: %d", l.File, l.Symbol, l.Callers))
	}
	return out
}
