package graph

import (
	"context"
	"database/sql"
)

// CoverageLang is the measured fair-coverage share for one language: the
// fraction of that language's symbol-bearing files that have at least one
// resolved cross-file dependent (a symbol in the file is the target of an
// edge from another file). Measured from the edges table, not asserted.
type CoverageLang struct {
	Language     string  `json:"language"`
	Files        int     `json:"files"`
	CoveredFiles int     `json:"covered_files"`
	Share        float64 `json:"share"`
}

// FairCoverage measures per-language fair coverage from the edges table. A
// symbol-bearing file is "covered" when at least one of its symbols is the
// target of a resolved cross-file edge (to_id set, target in another file).
// Languages with no symbol-bearing files are omitted. An empty index returns
// an empty (non-nil) map.
func FairCoverage(ctx context.Context, db *sql.DB) (map[string]CoverageLang, error) {
	// covered: file ids that have >=1 resolved cross-file dependent.
	covered := map[int64]bool{}
	covRows, err := db.QueryContext(ctx, `
		SELECT DISTINCT tf.file_id
		FROM edges e
		INNER JOIN symbols tf ON tf.id = e.to_id
		INNER JOIN symbols fs ON fs.id = e.from_id
		WHERE e.to_id IS NOT NULL
		  AND fs.file_id <> tf.file_id`)
	if err != nil {
		return map[string]CoverageLang{}, err
	}
	for covRows.Next() {
		var fid int64
		if err := covRows.Scan(&fid); err != nil {
			covRows.Close()
			return map[string]CoverageLang{}, err
		}
		covered[fid] = true
	}
	if err := covRows.Err(); err != nil {
		covRows.Close()
		return map[string]CoverageLang{}, err
	}
	covRows.Close()

	// Per-language file counts, and which of those files are covered.
	rows, err := db.QueryContext(ctx, `
		SELECT f.language, f.id
		FROM files f
		INNER JOIN symbols s ON s.file_id = f.id
		GROUP BY f.language, f.id
		ORDER BY f.language, f.id`)
	if err != nil {
		return map[string]CoverageLang{}, err
	}
	defer rows.Close()
	type agg struct{ files, covered int }
	byLang := map[string]*agg{}
	out := map[string]CoverageLang{}
	for rows.Next() {
		var lang string
		var fid int64
		if err := rows.Scan(&lang, &fid); err != nil {
			return out, err
		}
		if byLang[lang] == nil {
			byLang[lang] = &agg{}
		}
		byLang[lang].files++
		if covered[fid] {
			byLang[lang].covered++
		}
	}
	if err := rows.Err(); err != nil {
		return out, err
	}
	for lang, a := range byLang {
		share := 0.0
		if a.files > 0 {
			share = float64(a.covered) / float64(a.files)
		}
		out[lang] = CoverageLang{
			Language:     lang,
			Files:        a.files,
			CoveredFiles: a.covered,
			Share:        share,
		}
	}
	return out, nil
}
