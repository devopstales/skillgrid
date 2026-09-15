// Package codeindex incrementally indexes source files into a project store.
package codeindex

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/community"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/embedder"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/extract"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/knowledge"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/pdg"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/process"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/route"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// MaxFileSize is the default first-class size skip threshold. Files larger
// than this are skipped (counted in stats), not an error, not a fallback.
const MaxFileSize = 500 * 1024

// maxFileSize is the current run's effective threshold (bytes).
var maxFileSize int64 = MaxFileSize

// Config controls incremental indexing behavior.
type Config struct {
	Include      []string
	Exclude      []string
	ChunkLines   int
	ChunkOverlap int
	MaxFileSize  int
	// PDG enables the opt-in per-function CFG + PDG pass (011 step 01). When
	// false (the default) the pass never runs and the cfg/pdg tables stay
	// empty — a non---pdg index is byte-for-byte the 005/008/010 graph.
	PDG bool
	// LSP enables the independent opt-in language-server edge tier (011 step
	// 01). When set it runs BEFORE the PDG pass (so LSP_RESOLVED call edges
	// exist before the PDG derives dependences) and is best-effort: an absent
	// / failing / timing-out server warns and continues, leaving the static
	// index unchanged (no partial LSP edge set).
	LSP bool
}

// Stats summarizes one indexing run.
type Stats struct {
	FilesIndexed   int `json:"files_indexed"`
	FilesSkipped   int `json:"files_skipped"`
	FilesDeleted   int `json:"files_deleted"`
	ChunksAdded    int `json:"chunks_added"`
	FilesOversized int `json:"files_oversized"`
	SymbolsAdded   int `json:"symbols_added"`
	EdgesAdded     int `json:"edges_added"`
}

// ScannedFile is a candidate file discovered under the index root.
type ScannedFile struct {
	Path     string
	MtimeNs  int64
	Size     int64
	Hash     string
	Contents []byte
}

// Scan walks root and returns files matching include/exclude globs.
func Scan(root string, include, exclude []string) ([]ScannedFile, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	var files []ScannedFile
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			if rel != "." && shouldSkipDir(rel, exclude) {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		// Note: size-based skipping is first-class in Indexer.Run (counted in
		// stats as FilesOversized), not here — so an oversized file is a skip,
		// not a silent drop, and is independently of exclude globs.
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if matchesAny(exclude, rel) {
			return nil
		}
		if len(include) > 0 && !matchesAny(include, rel) {
			return nil
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", rel, err)
		}
		sum := sha256.Sum256(contents)
		files = append(files, ScannedFile{
			Path:     rel,
			MtimeNs:  info.ModTime().UnixNano(),
			Size:     info.Size(),
			Hash:     hex.EncodeToString(sum[:]),
			Contents: contents,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func shouldSkipDir(rel string, exclude []string) bool {
	rel = filepath.ToSlash(rel)
	for _, pattern := range exclude {
		pattern = filepath.ToSlash(strings.TrimSuffix(pattern, "/**"))
		pattern = strings.TrimPrefix(pattern, "**/")
		if pattern == "" {
			continue
		}
		if rel == pattern || strings.HasPrefix(rel, pattern+"/") {
			return true
		}
		if matchesGlob(pattern, rel) || matchesGlob(pattern+"/**", rel) {
			return true
		}
	}
	return false
}

func matchesAny(patterns []string, path string) bool {
	for _, p := range patterns {
		if matchesGlob(p, path) {
			return true
		}
	}
	return false
}

func matchesGlob(pattern, path string) bool {
	pattern = filepath.ToSlash(pattern)
	path = filepath.ToSlash(path)
	if !strings.Contains(pattern, "**") {
		matched, _ := filepath.Match(pattern, path)
		return matched
	}
	re, err := globToRegexp(pattern)
	if err != nil {
		return false
	}
	return re.MatchString(path)
}

func globToRegexp(pattern string) (*regexp.Regexp, error) {
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				if i+2 < len(pattern) && pattern[i+2] == '/' {
					i += 2
					b.WriteString("(.*/)?")
				} else {
					i++
					b.WriteString(".*")
				}
			} else {
				b.WriteString("[^/]*")
			}
		case '?':
			b.WriteString("[^/]")
		case '.', '+', '(', ')', '|', '^', '$', '{', '}', '[', ']', '\\':
			b.WriteByte('\\')
			b.WriteByte(pattern[i])
		default:
			b.WriteByte(pattern[i])
		}
	}
	b.WriteString("$")
	return regexp.Compile(b.String())
}

// Chunk represents a slice of a file for FTS indexing.
type Chunk struct {
	StartLine   int
	EndLine     int
	Text        string
	ContentHash string
}

// ChunkLines splits content into overlapping windows of ~chunkLines.
func ChunkLines(content []byte, chunkLines, chunkOverlap int) []Chunk {
	if chunkLines <= 0 {
		chunkLines = 80
	}
	text := string(content)
	allLines := strings.Split(text, "\n")
	if len(allLines) == 0 {
		return nil
	}
	step := chunkLines - chunkOverlap
	if step <= 0 {
		step = chunkLines
	}
	var chunks []Chunk
	for start := 0; start < len(allLines); start += step {
		end := start + chunkLines
		if end > len(allLines) {
			end = len(allLines)
		}
		chunkText := strings.Join(allLines[start:end], "\n")
		if strings.TrimSpace(chunkText) == "" {
			if end >= len(allLines) {
				break
			}
			continue
		}
		sum := sha256.Sum256([]byte(chunkText))
		chunks = append(chunks, Chunk{
			StartLine:   start + 1,
			EndLine:     end,
			Text:        chunkText,
			ContentHash: hex.EncodeToString(sum[:]),
		})
		if end >= len(allLines) {
			break
		}
	}
	return chunks
}

// Indexer incrementally indexes source files into the store.
type Indexer struct {
	store *store.Store
	emb   embedder.Embedder
	// processLLM is the LLM labeler for the process pass (03.8 indexer hook).
	// A nil value uses the deterministic labeler stub (a clearly-marked stub,
	// not a live LLM); the key is that the process pass RUNS at index time so
	// code_processes is populated, not just in tests.
	processLLM process.LLM
	// pdgEnabled / lspEnabled are opt-in pass gates (011 step 01). They are
	// OR'd with Config.PDG / Config.LSP in Run.
	pdgEnabled bool
	lspEnabled bool
	// lspResolver is the per-call LSP resolution seam (test hook). When set,
	// the --lsp tier resolves member calls through it (hermetic) instead of
	// spawning the real server. Nil = real server on PATH.
	lspResolver pdg.ResolveMemberCall
	// lspTimeout bounds a single LSP server round-trip. 0 = the adapter's 30s
	// default. A test sets a tiny value to exercise the hang/timeout path
	// without waiting 30s.
	lspTimeout time.Duration
}

// New creates an Indexer backed by st.
func New(st *store.Store) *Indexer {
	return &Indexer{store: st}
}

// WithProcessLLM sets the LLM labeler for the process pass (03.8). A nil LLM
// (or an unset one) falls back to the deterministic labeler stub so the
// process pass still runs and labels at index time.
func (idx *Indexer) WithProcessLLM(llm process.LLM) *Indexer {
	idx.processLLM = llm
	return idx
}

// EnablePDG turns on the opt-in per-function CFG + PDG pass for the next Run.
// It is equivalent to Config.PDG=true; provided so tests and callers can set
// it without reconstructing the Config.
func (idx *Indexer) EnablePDG() *Indexer {
	idx.pdgEnabled = true
	return idx
}

// EnableLSP turns on the independent opt-in language-server edge tier for the
// next Run. It is equivalent to Config.LSP=true.
func (idx *Indexer) EnableLSP() *Indexer {
	idx.lspEnabled = true
	return idx
}

// WithLSPResolver sets the per-call LSP resolution seam (test hook). When set,
// the --lsp tier resolves member calls through it instead of spawning the real
// language server (hermetic; no gopls required).
func (idx *Indexer) WithLSPResolver(fn pdg.ResolveMemberCall) *Indexer {
	idx.lspResolver = fn
	return idx
}

// WithLSPTiming bounds a single LSP server round-trip. 0 = the adapter's 30s
// default. Tests set a tiny value to exercise the hang/timeout path fast.
func (idx *Indexer) WithLSPTiming(d time.Duration) *Indexer {
	idx.lspTimeout = d
	return idx
}

// WithEmbedder sets the optional embedder used by the eager dual-granularity
// embedding pass. A nil embedder (the default) skips the pass entirely.
func (idx *Indexer) WithEmbedder(e embedder.Embedder) *Indexer {
	idx.emb = e
	return idx
}

type existingFile struct {
	ID          int64
	MtimeNs     int64
	Size        int64
	ContentHash string
}

// Run scans root and upserts changed files; removes stale entries. The graph
// extract/prune (symbols/edges/embeddings/LSH) runs in the SAME transaction as
// the chunk sync so the content-hash + mtime guards stay single-path (no
// dual-sync drift).
func (idx *Indexer) Run(ctx context.Context, root string, cfg Config) (Stats, error) {
	var stats Stats
	if idx == nil || idx.store == nil || idx.store.DB == nil {
		return stats, fmt.Errorf("indexer not initialized")
	}
	if cfg.MaxFileSize > 0 {
		maxFileSize = int64(cfg.MaxFileSize)
	}
	scanned, err := Scan(root, cfg.Include, cfg.Exclude)
	if err != nil {
		return stats, err
	}
	existing, err := loadExistingFiles(idx.store.DB)
	if err != nil {
		return stats, err
	}
	scannedPaths := make(map[string]struct{}, len(scanned))
	targetUIDs := make(map[string]struct{})
	// skippedFileIDs holds the file IDs of unchanged files that were skipped
	// this run. Their symbols are already correct in the DB, so their UIDs
	// must be added to targetUIDs to prevent the orphan prune from deleting them.
	skippedFileIDs := make([]int64, 0, len(existing))
	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := idx.store.DB.BeginTx(ctx, nil)
	if err != nil {
		return stats, err
	}
	defer tx.Rollback()
	for _, file := range scanned {
		if err := ctx.Err(); err != nil {
			return stats, err
		}
		scannedPaths[file.Path] = struct{}{}
		if file.Size > maxFileSize {
			// First-class size skip: counted in stats, not an error, not a
			// fallback. The file is left out of the index entirely.
			stats.FilesOversized++
			continue
		}
		prev, ok := existing[file.Path]
		if ok && prev.MtimeNs == file.MtimeNs && prev.Size == file.Size && prev.ContentHash == file.Hash {
			stats.FilesSkipped++
			skippedFileIDs = append(skippedFileIDs, prev.ID)
			continue
		}
		fileID, err := upsertFile(tx, file, now)
		if err != nil {
			return stats, err
		}
		if ok {
			if _, err := tx.ExecContext(ctx, `DELETE FROM chunks WHERE file_id = ?`, fileID); err != nil {
				return stats, fmt.Errorf("delete chunks for %s: %w", file.Path, err)
			}
		}
		chunks := ChunkLines(file.Contents, cfg.ChunkLines, cfg.ChunkOverlap)
		for _, chunk := range chunks {
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO chunks (file_id, start_line, end_line, text, content_hash) VALUES (?, ?, ?, ?, ?)`,
				fileID, chunk.StartLine, chunk.EndLine, chunk.Text, chunk.ContentHash,
			); err != nil {
				return stats, fmt.Errorf("insert chunk for %s: %w", file.Path, err)
			}
			stats.ChunksAdded++
		}
		// Graph pass: extract symbols/edges/rationale for this file in the
		// same tx.
		syms, edges, rationale, err := idx.extractFile(file)
		if err != nil {
			return stats, fmt.Errorf("extract %s: %w", file.Path, err)
		}
		if n, err := writeFileGraph(tx, fileID, syms, edges, rationale); err != nil {
			return stats, err
		} else {
			stats.SymbolsAdded += n
			stats.EdgesAdded += len(edges)
		}
		// Route pass: framework routing (route nodes + references/navigates
		// edges) is extracted AFTER the 005 symbol/edge extraction, in the
		// SAME transaction, so a route node is always resolvable to the 005
		// symbols it references and a single rollback undoes both. Per-file
		// route failures are non-fatal (a malformed routing file falls back
		// to zero route rows and the index continues).
		routeUIDs, rerr := idx.extractRoutes(ctx, tx, fileID, file)
		if rerr != nil {
			fmt.Fprintf(os.Stderr, "warn: route extract %s: %v\n", file.Path, rerr)
		}
		// Record the file's target UIDs (005 + route nodes) for the end-of-tx
		// global prune so the orphan prune does not delete the route nodes.
		for _, s := range syms {
			targetUIDs[s.UID] = struct{}{}
		}
		for _, uid := range routeUIDs {
			targetUIDs[uid] = struct{}{}
		}
		stats.FilesIndexed++
	}
	// Target-state prune: a deleted file prunes its whole footprint via the
	// file_id cascade; symbols whose file still exists but was rewritten are
	// handled by the per-file writeFileGraph re-upsert above.
	for path, prev := range existing {
		if _, ok := scannedPaths[path]; ok {
			continue
		}
		if err := pruneFileFootprint(tx, prev.ID); err != nil {
			return stats, fmt.Errorf("delete file %s: %w", path, err)
		}
		stats.FilesDeleted++
	}
	// Seed the orphan-prune target set with UIDs from unchanged (skipped)
	// files. Their symbols are already correct in the DB, but their UIDs never
	// made it into targetUIDs (only re-indexed files contribute above), so
	// without this the orphan prune would delete them.
	if err := seedTargetUIDsFromSkippedFiles(tx, skippedFileIDs, targetUIDs); err != nil {
		return stats, fmt.Errorf("seed target uids: %w", err)
	}
	// Global target-state prune: any symbol whose uid is not in the declared
	// target set is an orphan (its file was deleted or its function removed).
	// Deleting it cascades to edges, embeddings, LSH buckets, rationale, and
	// FTS rows in one pass.
	if err := pruneOrphanSymbols(tx, targetUIDs); err != nil {
		return stats, fmt.Errorf("prune orphan symbols: %w", err)
	}
	// Eager dual-granularity embedding: symbol-level vectors (function/type
	// signatures) upserted in the same tx so the content-hash + mtime guard
	// stays single-path. Batched + resumable: only embeds symbols that are new
	// or whose content changed. A down embedder warns and continues — the FTS
	// floor still works.
	if idx.emb != nil {
		if err := idx.embedPass(ctx, tx); err != nil {
			fmt.Fprintf(os.Stderr, "warn: embed pass: %v\n", err)
		}
	}
	// Stale edges: an edge whose to_name no longer matches any live symbol and
	// whose to_id is null is a dangling name-only edge; drop it. (Name-only
	// edges to live symbols are kept so step 03 can resolve them.) Navigates
	// edges are excluded: their to_name is a route path (not a symbol name),
	// so the symbol-existence check does not apply.
	if _, err := tx.Exec(`
		DELETE FROM edges
		WHERE to_id IS NULL
		  AND kind != 'navigates'
		  AND NOT EXISTS (SELECT 1 FROM symbols s WHERE s.name = edges.to_name)
	`); err != nil {
		return stats, fmt.Errorf("prune dangling edges: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return stats, err
	}
	// The 005 + route extraction (above) ran in one transaction, now committed.
	// The community + process + knowledge passes run AFTER that commit, NOT in
	// the same tx: the store's single-connection *sql.DB (MaxOpenConns=1)
	// deadlocks a second connection once the committed tx's write lock is held
	// (a modernc.org/sqlite constraint), so the passes cannot share the 005
	// tx. Close the store's DB and reopen a fresh one (with foreign_keys=1 via
	// DSN, since that pragma is per-connection not persistent) so the passes
	// have a clean single-connection pool. The store's DB is replaced with the
	// pass DB so the indexer (and any caller using idx.store.DB) keeps working.
	//
	// The passes are advisory, not transactional with 005: they do not roll
	// back the committed 005 extraction, and a pass failure only warns and
	// continues (the 005 graph + FTS floor are already committed).
	dbPath := idx.store.Path()
	if err := idx.store.DB.Close(); err != nil {
		return stats, fmt.Errorf("close store db: %w", err)
	}
	passDB, err := sql.Open("sqlite", dbPath+"?_pragma=foreign_keys(1)")
	if err != nil {
		return stats, fmt.Errorf("open pass db: %w", err)
	}
	// Re-register the pool with passDB BEFORE swapping idx.store.DB: the close
	// above left the pool entry pointing at the now-closed old *sql.DB, so any
	// other live handle for this store (which cached the old DB at Open) would
	// fail with "database is closed". RebindPassDB atomically swaps the pool's
	// DB pointer to passDB (keeping the old entry's live refs) and is a no-op
	// when the store owns its DB outright (cache disabled).
	if err := idx.store.RebindPassDB(passDB); err != nil {
		return stats, fmt.Errorf("rebind pass db: %w", err)
	}
	idx.store.DB = passDB
	// Knowledge-graph passes (03.8): run the community + process + knowledge
	// passes at index time so code_processes / code_communities / code_docs /
	// code_configs / code_sql_* are populated. Each pass is advisory and
	// never load-bearing: a pass failure warns and continues.
	if err := idx.communityPass(ctx, passDB); err != nil {
		fmt.Fprintf(os.Stderr, "warn: community pass: %v\n", err)
	}
	if err := idx.processPass(ctx, passDB); err != nil {
		fmt.Fprintf(os.Stderr, "warn: process pass: %v\n", err)
	}
	if err := idx.knowledgePass(ctx, passDB, scanned); err != nil {
		fmt.Fprintf(os.Stderr, "warn: knowledge pass: %v\n", err)
	}
	// 011 step 01: opt-in LSP edge tier + per-function CFG/PDG. Each is gated
	// on its own flag (Config.PDG / Config.LSP or the EnablePDG/EnableLSP
	// hooks). The LSP tier is INDEPENDENT of PDG (it writes LSP_RESOLVED call
	// edges into 005's edges table) and, when both are set, runs FIRST so the
	// PDG pass sees the LSP_RESOLVED edges as available (not dropped). Both
	// passes are advisory (warn + continue); a failure never rolls back the
	// already-committed 005 graph.
	if idx.lspEnabled || cfg.LSP {
		if err := idx.lspPass(ctx, passDB, scanned); err != nil {
			fmt.Fprintf(os.Stderr, "warn: lsp pass: %v\n", err)
		}
	}
	// The passes read unchanged files from disk via the scan root; record it
	// (the scanned files are already in memory, so this is a fallback only).
	scanRoot = root
	if idx.pdgEnabled || cfg.PDG {
		if err := idx.pdgPass(ctx, passDB, scanned); err != nil {
			fmt.Fprintf(os.Stderr, "warn: pdg pass: %v\n", err)
		}
	}
	return stats, nil
}

// communityPass runs the 01 community-detection pass over the indexed graph,
// on the pass DB (a fresh *sql.DB opened after the 005 tx commits — the store's
// single-connection pool deadlocks once the committed tx holds the write lock,
// so the passes cannot share that tx). The community rows land in the same WAL
// database as the 005 extraction, but are committed independently: the pass is
// advisory and does not roll back the 005 extraction. Advisory, never
// load-bearing.
func (idx *Indexer) communityPass(ctx context.Context, passDB *sql.DB) error {
	if tableMissing(passDB, "communities") {
		return nil // table absent (pre-012 store) — nothing to do
	}
	_, err := community.Detect(ctx, passDB, community.Options{})
	return err
}

// processPass runs the 02 process-flow pass over the indexed graph, on the
// pass DB. Entry points are 010's: the resolved handlers of kind='route'
// symbols, plus the package main functions. It uses idx.processLLM (falling
// back to the deterministic labeler stub) so the flows are labeled at index
// time. Advisory, never load-bearing.
func (idx *Indexer) processPass(ctx context.Context, passDB *sql.DB) error {
	if tableMissing(passDB, "processes") {
		return nil // table absent (pre-014 store) — nothing to do
	}
	entries, err := idx.processEntries(passDB)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return nil
	}
	llm := idx.processLLM
	if llm == nil {
		llm = processLLMStub{}
	}
	_, err = process.Run(ctx, passDB, entries, llm, process.RunOptions{})
	return err
}

// processEntries resolves 010's entry points: the handlers served by
// kind='route' symbols (the traceable call-chain starts) and the package main
// functions. Deterministic by symbol id.
func (idx *Indexer) processEntries(db *sql.DB) ([]process.Entry, error) {
	var entries []process.Entry
	seen := map[int64]bool{}
	// 010 route handlers.
	rows, err := db.Query(`
		SELECT e.to_id FROM edges e
		JOIN symbols r ON r.id = e.from_id
		WHERE r.kind = 'route' AND e.kind = 'references' AND e.to_id IS NOT NULL
		ORDER BY e.to_id`)
	if err == nil {
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err == nil && !seen[id] {
				seen[id] = true
				entries = append(entries, process.Entry{SymbolID: id, Kind: "handler"})
			}
		}
		rows.Close()
	}
	// Package main functions.
	mrows, err := db.Query(`
		SELECT s.id FROM symbols s
		JOIN files f ON f.id = s.file_id
		WHERE s.name = 'main' AND s.kind IN ('function','method')
		ORDER BY s.id`)
	if err == nil {
		for mrows.Next() {
			var id int64
			if err := mrows.Scan(&id); err == nil && !seen[id] {
				seen[id] = true
				entries = append(entries, process.Entry{SymbolID: id, Kind: "cli-main"})
			}
		}
		mrows.Close()
	}
	return entries, nil
}

// knowledgePass runs the knowledge extractors (doc + config + SQL) over the
// scanned files, on the pass DB, persisting doc_nodes / config_nodes /
// sql_schema_nodes and their edges. The knowledge store operates directly on
// the *sql.DB (no sub-tx), so it does not deadlock the store's
// single-connection pool. Non-fatal per file (a malformed file yields zero
// rows, the index continues).
func (idx *Indexer) knowledgePass(ctx context.Context, passDB *sql.DB, scanned []ScannedFile) error {
	if tableMissing(passDB, "doc_nodes") {
		return nil // table absent (pre-015 store) — nothing to do
	}
	st := knowledge.NewStore(passDB)
	files := make([]knowledge.FileInput, 0, len(scanned))
	for _, f := range scanned {
		files = append(files, knowledge.FileInput{Path: f.Path, Contents: f.Contents})
	}
	_, err := knowledge.RunPasses(ctx, st, files)
	return err
}

// tableMissing reports whether a query error is "no such table" (the table
// predates the relevant migration) — the pass is then a no-op.
func tableMissing(db *sql.DB, table string) bool {
	err := db.QueryRow(`SELECT 1 FROM ` + table + ` LIMIT 0`).Err()
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "no such table")
}

// processLLMStub is the deterministic LLM labeler used by the 03.8 indexer
// hook when no live LLM is configured. It is a clearly-marked stub (not a live
// LLM): it returns a label derived from the flow's entry name, so the process
// pass RUNS and labels at index time (code_processes is populated) without a
// live LLM call. The label is deterministic (a pure function of the flow), so
// an unchanged re-index is a cache hit.
type processLLMStub struct{}

func (processLLMStub) Label(ctx context.Context, summary string) (string, error) {
	// Derive a deterministic label from the flow's entry name (the first
	// "Entry: <name>" line of the summary).
	for _, line := range strings.Split(summary, "\n") {
		if strings.HasPrefix(line, "Entry: ") {
			return "flow: " + strings.TrimSpace(strings.TrimPrefix(line, "Entry: ")), nil
		}
	}
	return "flow", nil
}

// pruneOrphanSymbols deletes symbols whose uid is not in the declared target
// set. The file_id cascade (for deleted files) has already removed their
// symbols; this catches symbols in surviving files that were rewritten (e.g.
// a removed function) and whose uid is no longer produced by extraction.
// seedTargetUIDsFromSkippedFiles adds the UIDs of all symbols belonging to
// skipped (unchanged) files to targetUIDs. This ensures the orphan prune does
// not delete symbols from files that were not re-indexed this run.
func seedTargetUIDsFromSkippedFiles(tx *sql.Tx, fileIDs []int64, targetUIDs map[string]struct{}) error {
	if len(fileIDs) == 0 {
		return nil
	}
	placeholders := make([]string, len(fileIDs))
	args := make([]interface{}, len(fileIDs))
	for i, id := range fileIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	query := fmt.Sprintf(`SELECT uid FROM symbols WHERE file_id IN (%s)`, strings.Join(placeholders, ","))
	rows, err := tx.Query(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err != nil {
			return err
		}
		targetUIDs[uid] = struct{}{}
	}
	return rows.Err()
}

func pruneOrphanSymbols(tx *sql.Tx, targetUIDs map[string]struct{}) error {
	rows, err := tx.Query(`SELECT id, uid FROM symbols`)
	if err != nil {
		return err
	}
	var orphanIDs []int64
	for rows.Next() {
		var id int64
		var uid string
		if err := rows.Scan(&id, &uid); err != nil {
			rows.Close()
			return err
		}
		if _, ok := targetUIDs[uid]; !ok {
			orphanIDs = append(orphanIDs, id)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, id := range orphanIDs {
		if _, err := tx.Exec(`DELETE FROM symbols WHERE id = ?`, id); err != nil {
			return err
		}
	}
	return nil
}

// embedPass runs the eager dual-granularity embedding pass. It embeds
// symbol-level vectors (name + signature) and upserts only rows that are new
// or whose content changed. The embedding_model guard in embed_meta ensures a
// model swap triggers a full re-embed.
func (idx *Indexer) embedPass(ctx context.Context, tx *sql.Tx) error {
	model := idx.emb.Model()
	dim := idx.emb.Dimension()
	if model == "" || dim <= 0 {
		return nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	// Model-swap guard: if the indexed model differs, clear all vectors.
	var indexedModel string
	_ = tx.QueryRow(`SELECT value FROM embed_meta WHERE key = 'embedding_model'`).Scan(&indexedModel)
	if indexedModel != "" && indexedModel != model {
		if _, err := tx.Exec(`DELETE FROM embeddings`); err != nil {
			return fmt.Errorf("clear stale embeddings: %w", err)
		}
		if _, err := tx.Exec(`INSERT OR REPLACE INTO embed_meta (key, value) VALUES ('embedding_model', ?)`, model); err != nil {
			return err
		}
	} else if indexedModel == "" {
		if _, err := tx.Exec(`INSERT OR REPLACE INTO embed_meta (key, value) VALUES ('embedding_model', ?)`, model); err != nil {
			return err
		}
	}
	// Symbol-level embedding: embed each symbol's name + signature.
	rows, err := tx.Query(`
		SELECT s.id, s.name, s.signature
		FROM symbols s
		ORDER BY s.id
	`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := ctx.Err(); err != nil {
			return err
		}
		var symID int64
		var name, sig string
		if err := rows.Scan(&symID, &name, &sig); err != nil {
			return err
		}
		// Skip if already embedded with the current model (model-swap clears
		// all rows above, so an existing row is by definition current-model).
		var existingModel string
		err := tx.QueryRow(`SELECT model FROM embeddings WHERE symbol_id = ?`, symID).Scan(&existingModel)
		if err == nil && existingModel == model {
			continue
		}
		text := name
		if sig != "" {
			text += "\n" + sig
		}
		vec, err := idx.emb.Embed(ctx, text)
		if err != nil {
			continue
		}
		if len(vec.Data) != dim {
			continue
		}
		blob := memory.EncodeVector(vec)
		if _, err := tx.Exec(`
			INSERT INTO embeddings (symbol_id, model, dim, vector, updated_at)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(symbol_id) DO UPDATE SET
			  model = excluded.model,
			  dim = excluded.dim,
			  vector = excluded.vector,
			  updated_at = excluded.updated_at
		`, symID, model, dim, blob, now); err != nil {
			return err
		}
	}
	return rows.Err()
}

// extractFile runs the Extractor for one scanned file and returns its symbols,
// edges, and rationale nodes. Per-file extraction failures fall back to regex
// and never abort the run.
func (idx *Indexer) extractFile(file ScannedFile) ([]extract.Symbol, []extract.Edge, []extract.Rationale, error) {
	ex := extract.Default()
	g, err := ex.ExtractFile(file.Path, file.Contents)
	if err != nil {
		return nil, nil, nil, err
	}
	return g.Symbols, g.Edges, g.Rationales, nil
}

// extractRoutes runs the route extractor for one scanned file and persists
// route nodes + references/navigates edges in the SAME transaction as the 005
// extraction. It is non-fatal: a malformed routing file yields zero route rows
// (the index continues). The per-file 005 symbols (just upserted above) are
// the source of truth for handler resolution (same-file first).
func (idx *Indexer) extractRoutes(ctx context.Context, tx *sql.Tx, fileID int64, file ScannedFile) ([]string, error) {
	// Source-aware candidate check: run the extractor on the file's contents
	// and skip files that yield neither routes nor navigations. A file-type
	// check alone (ExtractFile(path, nil)) would make EVERY .go file a
	// candidate once the nethttp handler is registered (its dispatch is
	// extension-based), so the extraction itself is the filter.
	fr := route.ExtractFile(file.Path, file.Contents)
	if len(fr.Routes) == 0 && len(fr.Navigates) == 0 {
		return nil, nil
	}
	st := &txRouteStore{tx: tx, fileID: fileID}
	_, err := route.Run(ctx, st, fileID, file.Path, file.Contents, nil)
	if err != nil {
		return st.routeUIDs(), err
	}
	return st.routeUIDs(), nil
}

// txRouteStore implements route.Store over the indexer's open transaction. It
// persists route nodes (kind=route symbols) and references/navigates edges in
// the SAME tx as the 005 extraction, and applies the drop-not-guess policy in
// ResolveHandler (same-file first, then a unique global match). fileSyms is
// loaded lazily on first use (the 005 symbols are already upserted in the tx).
type txRouteStore struct {
	tx       *sql.Tx
	fileID   int64
	fileSyms []route.FileSymbol
	loaded   bool
	uuids    []string
}

// routeUIDs returns the UIDs of route nodes stored by this store (for the
// indexer's orphan-prune target set).
func (s *txRouteStore) routeUIDs() []string {
	return s.uuids
}

// loadFileSyms loads the file's 005 symbols once (name/uid/id), cached.
func (s *txRouteStore) loadFileSyms() error {
	if s.loaded {
		return nil
	}
	s.loaded = true
	rows, err := s.tx.Query(`SELECT name, uid, id FROM symbols WHERE file_id = ? ORDER BY start_line, id`, s.fileID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var sym route.FileSymbol
		if err := rows.Scan(&sym.Name, &sym.UID, &sym.ID); err != nil {
			return err
		}
		s.fileSyms = append(s.fileSyms, sym)
	}
	return rows.Err()
}

func (s *txRouteStore) FirstSymbolID(fileID int64) (int64, error) {
	var id int64
	err := s.tx.QueryRow(`SELECT id FROM symbols WHERE file_id = ? ORDER BY start_line, id LIMIT 1`, fileID).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return id, err
}

func (s *txRouteStore) StoreRouteNode(node route.RouteNode) (int64, error) {
	// Upsert the route node into the 005 symbols table, byte-identical to the
	// 005 symbol upsert (route nodes are kind=route symbols).
	if _, err := s.tx.Exec(`
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		VALUES (?, ?, ?, 'route', ?, ?, ?, ?, ?, ?)
		ON CONFLICT(uid) DO UPDATE SET
		  file_id = excluded.file_id,
		  name = excluded.name,
		  qualified_name = excluded.qualified_name,
		  kind = excluded.kind,
		  language = excluded.language,
		  signature = excluded.signature,
		  start_line = excluded.start_line,
		  end_line = excluded.end_line,
		  content_hash = excluded.content_hash`,
		s.fileID, node.Name, node.Name, node.Language, node.PathPattern,
		node.Line, node.Line, node.ContentHash, node.UID,
	); err != nil {
		return 0, fmt.Errorf("upsert route node %s: %w", node.Name, err)
	}
	var id int64
	if err := s.tx.QueryRow(`SELECT id FROM symbols WHERE uid = ?`, node.UID).Scan(&id); err != nil {
		return 0, err
	}
	if err := s.writeSymbolSegments(node.Name); err != nil {
		return 0, err
	}
	s.uuids = append(s.uuids, node.UID)
	return id, nil
}

// writeSymbolSegments fills the symbol_segments reverse index in target-state
// for one symbol name: drop the name's prior segments, then re-insert.
func (s *txRouteStore) writeSymbolSegments(name string) error {
	if _, err := s.tx.Exec(`DELETE FROM symbol_segments WHERE symbol_name = ?`, name); err != nil {
		return err
	}
	for _, seg := range nameSegments(name) {
		if _, err := s.tx.Exec(`INSERT OR IGNORE INTO symbol_segments (segment, symbol_name) VALUES (?, ?)`, seg, name); err != nil {
			return err
		}
	}
	return nil
}

func (s *txRouteStore) StoreRouteMeta(fileID, symbolID int64, node route.RouteNode) error {
	// Remove this route's prior meta (target-state) then insert.
	if _, err := s.tx.Exec(`DELETE FROM route_meta WHERE symbol_id = ?`, symbolID); err != nil {
		return err
	}
	_, err := s.tx.Exec(`INSERT INTO route_meta (symbol_id, file_id, framework, method, path_pattern, screen)
		VALUES (?, ?, ?, ?, ?, ?)`,
		symbolID, fileID, node.Framework, node.Method, node.PathPattern, "")
	return err
}

func (s *txRouteStore) ResolveHandler(fileID int64, name string) (int64, string, string, int) {
	// 1) Same-file match: a handler defined in the same file is an explicit
	// reference (EXTRACTED) and wins over any global guess. The global match
	// count is still reported (candidates) even when the same-file match wins.
	var global int
	_ = s.tx.QueryRow(`SELECT COUNT(*) FROM symbols WHERE name = ?`, name).Scan(&global)
	if err := s.loadFileSyms(); err != nil {
		return 0, "", "", global
	}
	for _, fs := range s.fileSyms {
		if fs.Name == name {
			return fs.ID, fs.UID, route.ConfidenceExtracted, global
		}
	}
	// 2) Name-only match: no same-file / explicit handler, but a unique global
	// symbol by this name. This is a best-effort name guess → AMBIGUOUS
	// (stored, low-confidence). Multiple or zero global matches are unresolvable
	// → dropped (drop-not-guess); the match count is reported either way so a
	// dropped ref can log it.
	var id int64
	var uid string
	if global != 1 {
		return 0, "", "", global
	}
	if err := s.tx.QueryRow(`SELECT id, uid FROM symbols WHERE name = ? LIMIT 1`, name).Scan(&id, &uid); err != nil {
		return 0, "", "", global
	}
	return id, uid, route.ConfidenceAmbiguous, global
}

func (s *txRouteStore) StoreReferencesEdges(fileID int64, nodes []route.RouteNode, _ int64) (int, error) {
	// Target-state: prune this file's prior route-originated references edges
	// (kind=references whose from is a route node in this file), then upsert.
	if _, err := s.tx.Exec(`
		DELETE FROM edges
		WHERE kind = 'references' AND file_id = ?
		  AND from_id IN (SELECT id FROM symbols WHERE file_id = ? AND kind = 'route')`,
		fileID, fileID); err != nil {
		return 0, err
	}
	stored := 0
	for _, n := range nodes {
		if n.HandlerSymbol == 0 {
			continue // dropped (drop-not-guess) — no references edge fabricated
		}
		var routeID int64
		if err := s.tx.QueryRow(`SELECT id FROM symbols WHERE uid = ?`, n.UID).Scan(&routeID); err != nil {
			return stored, err
		}
		// The confidence comes from the resolution policy: EXTRACTED for a
		// same-file/explicit handler, AMBIGUOUS for a name-only best-effort
		// guess. (INFERRED is not a references-edge confidence — it labels
		// convention-derived navigates, not route->handler references.)
		conf := n.HandlerConfidence
		if conf == "" {
			conf = route.ConfidenceExtracted
		}
		if _, err := s.tx.Exec(`
			INSERT INTO edges (kind, from_id, file_id, to_id, to_name, target_path, confidence, line, valid_from)
			VALUES ('references', ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(kind, from_id, file_id, to_id, to_name, target_path, line) DO UPDATE SET
			  confidence = excluded.confidence,
			  valid_from = excluded.valid_from`,
			routeID, fileID, n.HandlerSymbol, n.HandlerName, n.PathPattern, conf, n.Line, time.Now().Unix(),
		); err != nil {
			return stored, fmt.Errorf("upsert references edge %s: %w", n.HandlerName, err)
		}
		stored++
	}
	return stored, nil
}

func (s *txRouteStore) StoreNavigatesEdges(fileID int64, navs []route.NavigationNode) (int, error) {
	// Target-state: prune this file's prior navigates edges, then upsert.
	if _, err := s.tx.Exec(`
		DELETE FROM edges
		WHERE kind = 'navigates' AND file_id = ?
		  AND from_id IN (SELECT id FROM symbols WHERE file_id = ?)`,
		fileID, fileID); err != nil {
		return 0, err
	}
	// The sending function is the file's first 005 symbol (top-level
	// navigation); a file with no symbols has no resolvable source.
	fromID, fromErr := s.FirstSymbolID(fileID)
	if fromErr != nil {
		fromID = 0
	}
	stored := 0
	for _, n := range navs {
		src := n.FromSymbol
		if src == 0 {
			src = fromID
		}
		if src == 0 {
			continue // no sending function → unresolved, not fabricated
		}
		if _, err := s.tx.Exec(`
			INSERT INTO edges (kind, from_id, file_id, to_id, to_name, target_path, confidence, line, valid_from)
			VALUES ('navigates', ?, ?, NULL, ?, ?, ?, ?, ?)
			ON CONFLICT(kind, from_id, file_id, to_id, to_name, target_path, line) DO UPDATE SET
			  confidence = excluded.confidence,
			  valid_from = excluded.valid_from`,
			src, fileID, n.ToName, n.ToName, n.Confidence, n.Line, time.Now().Unix(),
		); err != nil {
			return stored, fmt.Errorf("upsert navigates edge %s (from_id=%d file_id=%d): %w", n.ToName, src, fileID, err)
		}
		stored++
	}
	return stored, nil
}

func (s *txRouteStore) StoreRouteDrops(fileID int64, dropped int) error {
	_, err := s.tx.Exec(`
		INSERT INTO route_drops (file_id, dropped) VALUES (?, ?)
		ON CONFLICT(file_id) DO UPDATE SET dropped = excluded.dropped`,
		fileID, dropped)
	return err
}

func (s *txRouteStore) StoreUnresolvedRefs(fileID int64, refs []route.DroppedRef, now string) error {
	// Target-state: this file's prior unresolved rows are replaced (a ref that
	// now resolves drops out of the table).
	if _, err := s.tx.Exec(`DELETE FROM unresolved_refs WHERE file_id = ?`, fileID); err != nil {
		return err
	}
	for _, r := range refs {
		if _, err := s.tx.Exec(`
			INSERT INTO unresolved_refs (file_id, reference_name, reference_kind, line, candidates, status, first_seen_at, last_seen_at)
			VALUES (?, ?, ?, ?, ?, 'failed', ?, ?)
			ON CONFLICT(file_id, reference_name, reference_kind, line) DO UPDATE SET
			  candidates = excluded.candidates,
			  last_seen_at = excluded.last_seen_at`,
			fileID, r.Name, r.Kind, r.Line, r.Candidates, now, now,
		); err != nil {
			return err
		}
	}
	return nil
}

// fileFirstSymbol caches the first (lowest id) symbol of a file for the
// duration of a writeFileGraph call, used as a default edge source. It is a
// process-global keyed by file id (not by store), so a prior test in the same
// process can leave stale entries that collide with a later test's ids.
var fileFirstSymbol = map[int64]int64{}

// ResetFileFirstSymbol clears the fileFirstSymbol cache. Tests that open a
// fresh store must call this (via TestMain or a fixture) to avoid stale id
// collisions with a prior test's store.
func ResetFileFirstSymbol() {
	for k := range fileFirstSymbol {
		delete(fileFirstSymbol, k)
	}
}

// writeFileGraph upserts a file's target-state symbols, edges, and rationale.
// It declares the target rows (the file's extracted symbols), upserts them,
// prunes the file's now-orphaned symbols (and their edges/vectors/buckets via
// cascade), then upserts the edges and rationale nodes.
func writeFileGraph(tx *sql.Tx, fileID int64, syms []extract.Symbol, edges []extract.Edge, rationale []extract.Rationale) (int, error) {
	targetUIDs := make(map[string]struct{}, len(syms))
	for _, s := range syms {
		targetUIDs[s.UID] = struct{}{}
	}
	// Upsert symbols (and their FTS rows via trigger).
	var upserted int
	for _, s := range syms {
		if _, err := tx.Exec(`
			INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(uid) DO UPDATE SET
			  file_id = excluded.file_id,
			  name = excluded.name,
			  qualified_name = excluded.qualified_name,
			  kind = excluded.kind,
			  language = excluded.language,
			  signature = excluded.signature,
			  start_line = excluded.start_line,
			  end_line = excluded.end_line,
			  content_hash = excluded.content_hash`,
			fileID, s.Name, s.QualifiedName, s.Kind, s.Language, s.Signature, s.StartLine, s.EndLine, s.ContentHash, s.UID,
		); err != nil {
			return 0, fmt.Errorf("upsert symbol %s: %w", s.Name, err)
		}
		upserted++
	}
	// Resolve UIDs to ids for edges, and record the file's first symbol.
	uidToIDFinal := map[string]int64{}
	rr, err := tx.Query(`SELECT id, uid FROM symbols WHERE file_id = ? ORDER BY start_line, id`, fileID)
	if err != nil {
		return 0, err
	}
	for rr.Next() {
		var id int64
		var uid string
		if err := rr.Scan(&id, &uid); err != nil {
			rr.Close()
			return 0, err
		}
		uidToIDFinal[uid] = id
		if _, ok := fileFirstSymbol[fileID]; !ok {
			fileFirstSymbol[fileID] = id
		}
	}
	rr.Close()
	if err := rr.Err(); err != nil {
		return 0, err
	}
	// Symbol segment vocabulary (target-state): drop this file's current
	// symbol names' segments, then re-insert. Route nodes are handled in
	// StoreRouteNode (they upsert after this pass).
	if _, err := tx.Exec(`DELETE FROM symbol_segments WHERE symbol_name IN (SELECT name FROM symbols WHERE file_id = ?)`, fileID); err != nil {
		return 0, err
	}
	for _, s := range syms {
		for _, seg := range nameSegments(s.Name) {
			if _, err := tx.Exec(`INSERT OR IGNORE INTO symbol_segments (segment, symbol_name) VALUES (?, ?)`, seg, s.Name); err != nil {
				return 0, err
			}
		}
	}
	// Also resolve cross-file target UIDs (a call to a symbol in another file
	// that is already indexed).
	if len(edges) > 0 {
		allUIDs, err := tx.Query(`SELECT uid, id FROM symbols`)
		if err == nil {
			for allUIDs.Next() {
				var uid string
				var id int64
				if err := allUIDs.Scan(&uid, &id); err == nil {
					if _, ok := uidToIDFinal[uid]; !ok {
						uidToIDFinal[uid] = id
					}
				}
			}
			allUIDs.Close()
		}
	}
	// Target-state: drop this file's existing outgoing edges before upserting
	// the freshly extracted set. Pruning by file_id (not by the from-symbol's
	// id) is essential: the edges table has no per-symbol cascade that can
	// reach a file's edges once its symbols are re-keyed on rewrite, so a
	// re-index would otherwise leave stale rows behind.
	if _, err := tx.Exec(`DELETE FROM edges WHERE file_id = ?`, fileID); err != nil {
		return 0, fmt.Errorf("prune edges for file %d: %w", fileID, err)
	}
	// Upsert edges. Edges require a resolvable from_id (the symbol they
	// originate from); import edges are stored with the file's package/first
	// symbol as the source when no specific symbol is bound.
	for _, e := range edges {
		fromID, okFrom := uidToIDFinal[e.FromUID]
		if !okFrom {
			// The extractor binds call/heritage edges to their enclosing
			// definition, but a name-only edge (or one the extractor left
			// unbound) must still resolve to a real source: the file's first
			// symbol. Edges carry a NOT-NULL from_id, so an unresolved source
			// must fall back here rather than silently store a null source.
			defID, ok := fileFirstSymbol[fileID]
			if !ok {
				continue
			}
			fromID = defID
		}
		var toID sql.NullInt64
		if e.ToUID != "" {
			if id, ok := uidToIDFinal[e.ToUID]; ok {
				toID.Valid = true
				toID.Int64 = id
			}
		}
		if _, err := tx.Exec(`
			INSERT INTO edges (kind, from_id, file_id, to_id, to_name, target_path, confidence, line, valid_from)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(kind, from_id, file_id, to_id, to_name, target_path, line) DO UPDATE SET
			  confidence = excluded.confidence,
			  valid_from = excluded.valid_from`,
			e.Kind, fromID, fileID, toID, e.ToName, e.TargetPath, e.Confidence, e.Line, time.Now().Unix(),
		); err != nil {
			return 0, fmt.Errorf("upsert edge %s: %w", e.Kind, err)
		}
	}
	// Upsert rationale nodes. A rationale links to the nearest enclosing
	// symbol (by UID, resolved to an id); a rationale with no resolvable
	// symbol is dropped (no fabricated link).
	if len(rationale) > 0 {
		// Remove this file's existing rationale rows first (target-state: the
		// re-extracted set is the full target; orphans are pruned). Rationale
		// rows reference the file's symbols; symbol id changes on rewrite.
		if _, err := tx.Exec(`
			DELETE FROM rationale WHERE symbol_id IN (SELECT id FROM symbols WHERE file_id = ?)
		`, fileID); err != nil {
			return 0, fmt.Errorf("prune rationale for %d: %w", fileID, err)
		}
		for _, r := range rationale {
			if r.SymbolUID == "" {
				continue
			}
			symID, ok := uidToIDFinal[r.SymbolUID]
			if !ok {
				continue
			}
			if _, err := tx.Exec(`
				INSERT INTO rationale (symbol_id, text, kind, line)
				VALUES (?, ?, ?, ?)
			`, symID, r.Text, r.Kind, r.Line); err != nil {
				return 0, fmt.Errorf("insert rationale: %w", err)
			}
		}
	}
	return upserted, nil
}

// pruneFileFootprint deletes a file's whole graph footprint in one pass.
// Deleting the file cascades to chunks and symbols; deleting each symbol
// cascades to edges, embeddings, LSH buckets, rationale, and FTS rows.
func pruneFileFootprint(tx *sql.Tx, fileID int64) error {
	if _, err := tx.Exec(`DELETE FROM files WHERE id = ?`, fileID); err != nil {
		return err
	}
	return nil
}

func loadExistingFiles(db *sql.DB) (map[string]existingFile, error) {
	rows, err := db.Query(`SELECT id, path, mtime_ns, size, content_hash FROM files`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]existingFile)
	for rows.Next() {
		var f existingFile
		var path string
		if err := rows.Scan(&f.ID, &path, &f.MtimeNs, &f.Size, &f.ContentHash); err != nil {
			return nil, err
		}
		out[path] = f
	}
	return out, rows.Err()
}

func upsertFile(tx *sql.Tx, file ScannedFile, indexedAt string) (int64, error) {
	if _, err := tx.Exec(
		`INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(path) DO UPDATE SET
		   mtime_ns = excluded.mtime_ns,
		   size = excluded.size,
		   content_hash = excluded.content_hash,
		   indexed_at = excluded.indexed_at`,
		file.Path, file.MtimeNs, file.Size, file.Hash, indexedAt,
	); err != nil {
		return 0, err
	}
	// Resolve the rowid by path. LastInsertId is NOT reliable on the
	// upsert UPDATE branch (modernc returns the last inserted rowid in the
	// session, not the upserted row's rowid), so always re-look-up by the
	// unique path. This is a local SQLite PK lookup — cost is negligible.
	var fileID int64
	if err := tx.QueryRow(`SELECT id FROM files WHERE path = ?`, file.Path).Scan(&fileID); err != nil {
		return 0, err
	}
	return fileID, nil
}

// Status reports aggregate index statistics.
type Status struct {
	FileCount   int    `json:"file_count"`
	ChunkCount  int    `json:"chunk_count"`
	LastIndexed string `json:"last_indexed,omitempty"`
}

// WatchStatus reports the auto-sync watcher state for the index status section.
type WatchStatus struct {
	Enabled bool     `json:"enabled"`
	Pending []string `json:"pending,omitempty"`
}

// GetWatchStatus reports whether the watcher is disabled (manual index) and the
// set of pending (unsynced) source files. pending is empty when the watcher is
// off or nothing is pending.
func GetWatchStatus(w *Watcher) WatchStatus {
	ws := WatchStatus{Enabled: !WatchDisabled()}
	if w != nil {
		ws.Pending = w.Pending()
	}
	return ws
}

// GetStatus returns current index stats from the store.
func GetStatus(st *store.Store) (Status, error) {
	var status Status
	if st == nil || st.DB == nil {
		return status, fmt.Errorf("store not initialized")
	}
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM files`).Scan(&status.FileCount); err != nil {
		return status, err
	}
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM chunks`).Scan(&status.ChunkCount); err != nil {
		return status, err
	}
	var last sql.NullString
	if err := st.DB.QueryRow(`SELECT MAX(indexed_at) FROM files`).Scan(&last); err != nil {
		return status, err
	}
	if last.Valid {
		status.LastIndexed = last.String
	}
	return status, nil
}
