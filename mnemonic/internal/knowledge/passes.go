package knowledge

import (
	"context"
	"database/sql"
	"path/filepath"
)

// FileInput is one scanned file for the knowledge pass: its index-relative
// path and its contents.
type FileInput struct {
	Path     string
	Contents []byte
}

// PassResult summarizes one knowledge pass run.
type PassResult struct {
	Docs   int
	Configs int
	SQL    int
}

// RunPasses runs the three knowledge extractors (doc + config + SQL) over the
// scanned files, persisting doc_nodes / config_nodes / sql_schema_nodes and
// their references / configures / reads / writes edges via st (the indexer's
// open transaction in production, so the pass is in the SAME tx as the 005
// extraction and a single rollback undoes it). Each extractor is non-fatal: a
// malformed file yields zero rows (the index continues, the bad part is
// skipped, the rest is indexed).
func RunPasses(ctx context.Context, st *Store, files []FileInput) (PassResult, error) {
	var res PassResult
	for _, f := range files {
		if err := ctx.Err(); err != nil {
			return res, err
		}
		switch {
		case isMarkdown(f.Path):
			doc := ExtractDoc(f.Path, f.Contents)
			if n, err := st.SaveDoc(ctx, f.Path, doc); err != nil {
				return res, err
			} else {
				res.Docs++
				_ = n
			}
		case isConfig(f.Path):
			cfg := ExtractConfig(f.Path, f.Contents)
			if n, err := st.SaveConfig(ctx, f.Path, cfg); err != nil {
				return res, err
			} else {
				res.Configs++
				_ = n
			}
		case isSQL(f.Path, f.Contents):
			sq := ExtractSQL(f.Path, f.Contents)
			if n, err := st.SaveSchema(ctx, f.Path, sq); err != nil {
				return res, err
			} else {
				res.SQL++
				_ = n
			}
		}
	}
	return res, nil
}

// isSQL reports whether a file should be parsed for SQL DDL/DML. Only .sql
// files are parsed: scanning every file for SQL keywords (SELECT / INSERT /
// UPDATE / DELETE FROM) produced false positives on Go code or prose that
// merely contains those words, yielding spurious table references. A .sql file
// is always parsed (even with no DDL/DML, so it is a known schema source);
// non-.sql files are not parsed for SQL at all.
func isSQL(path string, contents []byte) bool {
	return filepath.Ext(path) == ".sql" || filepath.Ext(path) == ".SQL"
}

var _ = sql.ErrNoRows
