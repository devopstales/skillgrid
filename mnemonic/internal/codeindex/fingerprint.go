package codeindex

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// fingerprintVersion is the extractor stamp that keys the fingerprint. A
// change to the extractor (a new version) invalidates every stored
// fingerprint, forcing a full re-walk on the next query.
const fingerprintVersion = "v1"

// ExtractorStamp returns the current extractor stamp. The fingerprint is keyed
// by this value (a stamp/version change invalidates the stored fingerprint,
// forcing a re-walk).
func ExtractorStamp() string { return fingerprintVersion }

// fingerprintRecord is one file's (size, mtime) entry in the stored
// fingerprint.
type fingerprintRecord struct {
	RelPath string
	Size    int64
	MtimeNs int64
}

// StoreFingerprint records the current extractor stamp in the project store so
// the next query's stat-walk can detect a stamp change (which invalidates the
// fingerprint). The per-file (size, mtime) fingerprint is the files table
// itself (the indexer's single-transaction (size, mtime) + content-hash
// guard), so there is nothing else to persist here.
func StoreFingerprint(db *sql.DB, _ string, _ Config, stamp string) error {
	if _, err := db.Exec(`
		INSERT INTO index_meta (key, schema_version) VALUES ('fingerprint_stamp', 1)
		ON CONFLICT(key) DO UPDATE SET schema_version = excluded.schema_version`,
		"fingerprint_stamp"); err != nil {
		return err
	}
	// The stamp value is stored in the schema_version column (TEXT-tolerant:
	// it is compared as a string, not used numerically).
	if _, err := db.Exec(`
		UPDATE index_meta SET schema_version = 1 WHERE key = 'fingerprint_stamp'`); err != nil {
		return err
	}
	return upsertStampValue(db, stamp)
}

// upsertStampValue stores the extractor stamp string (index_meta has no value
// column; we keep a dedicated side table created lazily).
func upsertStampValue(db *sql.DB, stamp string) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS fingerprint_meta (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	)`); err != nil {
		return err
	}
	_, err := db.Exec(`
		INSERT INTO fingerprint_meta (key, value) VALUES ('stamp', ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		stamp)
	return err
}

// fingerprintDigest computes a stable digest of the fingerprint + stamp (the
// stamp is mixed in so a stamp change changes the digest).
func fingerprintDigest(rel map[string]fingerprintRecord, stamp string) string {
	keys := make([]string, 0, len(rel))
	for k := range rel {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	h := sha256.New()
	for _, k := range keys {
		rec := rel[k]
		_, _ = fmt.Fprintf(h, "%s\t%d\t%d\n", k, rec.Size, rec.MtimeNs)
	}
	_, _ = h.Write([]byte(stamp))
	return hex.EncodeToString(h.Sum(nil))
}

// FingerprintDrift walks the index root with a (size, mtime) stat-walk (~3ms;
// no file content is read) and compares against the last index's stored
// fingerprint (the files table's mtime_ns/size). It returns the set of changed
// (added/modified/deleted) source files. A missing/changed extractor stamp
// forces a full re-walk (every file is reported as changed).
//
// The walk is structural-only: it reads file metadata (size, mtime) and
// nothing else, so it never touches the embedder leg.
func FingerprintDrift(root string, cfg Config, stamp string) (map[string]struct{}, error) {
	db, err := openProjectDBForRoot(root)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	return FingerprintDriftDB(db, root, cfg, stamp)
}

type storedFingerprint struct {
	stamp   string
	records map[string]fingerprintRecord
}

// FingerprintDriftDB walks the index root with a (size, mtime) stat-walk and
// compares against the stored fingerprint read from db (the project store's
// files table + the fingerprint stamp). This is the store-scoped form used by
// the query path (and tests); FingerprintDrift resolves db from the CWD.
func FingerprintDriftDB(db *sql.DB, root string, cfg Config, stamp string) (map[string]struct{}, error) {
	stored := loadStoredFingerprintDB(db)
	current := map[string]fingerprintRecord{}
	if err := walkFingerprint(root, cfg, func(rec fingerprintRecord) {
		current[rec.RelPath] = rec
	}); err != nil {
		return nil, err
	}
	changed := map[string]struct{}{}
	// Stamp mismatch → full re-walk (every current file is "changed").
	if stored.stamp != stamp {
		for p := range current {
			changed[p] = struct{}{}
		}
		return changed, nil
	}
	for p, c := range current {
		s, ok := stored.records[p]
		if !ok {
			changed[p] = struct{}{}
			continue
		}
		if c.Size != s.Size || c.MtimeNs != s.MtimeNs {
			changed[p] = struct{}{}
		}
	}
	for p := range stored.records {
		if _, ok := current[p]; !ok {
			changed[p] = struct{}{}
		}
	}
	return changed, nil
}

// loadStoredFingerprintDB reads the last index's (size, mtime) fingerprint from
// the files table (the indexer's source of truth) plus the extractor stamp.
func loadStoredFingerprintDB(db *sql.DB) storedFingerprint {
	out := storedFingerprint{stamp: "", records: map[string]fingerprintRecord{}}
	var s string
	if err := db.QueryRow(`SELECT value FROM fingerprint_meta WHERE key = 'stamp'`).Scan(&s); err != nil {
		return out // no stored stamp → full re-walk (stamp mismatch)
	}
	out.stamp = s
	rows, err := db.Query(`SELECT path, mtime_ns, size FROM files`)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var rec fingerprintRecord
		var path string
		if err := rows.Scan(&path, &rec.MtimeNs, &rec.Size); err != nil {
			continue
		}
		rec.RelPath = path
		out.records[filepath.ToSlash(path)] = rec
	}
	return out
}

// walkFingerprint stats every source file under root (metadata only) and calls
// fn with each record. It honors the 005 include/exclude filters.
func walkFingerprint(root string, cfg Config, fn func(fingerprintRecord)) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			if rel != "." && shouldSkipDir(rel, cfg.Exclude) {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if matchesAny(cfg.Exclude, rel) {
			return nil
		}
		if len(cfg.Include) > 0 && !matchesAny(cfg.Include, rel) {
			return nil
		}
		fn(fingerprintRecord{RelPath: rel, Size: info.Size(), MtimeNs: info.ModTime().UnixNano()})
		return nil
	})
}

// openProjectDBForRoot opens a read-only *sql.DB for the project store that
// indexes root (best-effort: the first store in the data dir). Returns nil db
// on error.
func openProjectDBForRoot(_ string) (*sql.DB, error) {
	dataDir := dataDirForFingerprint()
	db, err := sql.Open("sqlite", firstSQLiteIn(dataDir))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	return db, nil
}

// dataDirForFingerprint resolves the mnemonic data dir (env or default).
func dataDirForFingerprint() string {
	if v := os.Getenv("SKILLGRID_MNEMONIC_DATA_DIR"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".skillgrid", "mnemonic")
}

// firstSQLiteIn returns the first *.sqlite file in dir ("" when none).
func firstSQLiteIn(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	var found []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".sqlite" {
			found = append(found, e.Name())
		}
	}
	sort.Strings(found)
	if len(found) == 0 {
		return ""
	}
	return filepath.Join(dir, found[0])
}
