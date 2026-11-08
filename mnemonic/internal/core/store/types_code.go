package store

// The code-index store types (spec 02-code-index 001/002; ADR-0046). Go
// types only (the S-2 rule): the dialect SQL lives in the adapters, the
// functions see only the Store interface + these shapes.

// FileRow is one `files` row (spec 001 §2 — the indexing unit). The caller
// fills the walk-observed fields; the store owns ID (local, ephemeral),
// SyncID (minted on upsert, INV-6), and IndexedAt (set on upsert). Project
// is implicit — one DB per project (CI-O2), so the column is not a field.
type FileRow struct {
	ID          int64
	SyncID      string
	Path        string
	MtimeNS     int64
	Size        int64
	ContentHash string
	Language    string
	ASTHash     string
	IndexedAt   string
}

// Symbol is one `symbols` row (spec 002 §2 — the single node table, CI-2;
// every node, code + knowledge, is a kind-tagged row). The caller fills the
// extract fields; the store owns ID + SyncID (minted on first insert,
// INV-6). UID is the local upsert key (stable per file+name+kind, CI-3).
type Symbol struct {
	ID            int64
	SyncID        string
	Name          string
	QualifiedName string
	Kind          string
	Language      string
	Signature     string
	StartLine     int
	EndLine       int
	Visibility    string
	IsExported    bool
	ReturnType    string
	ParamTypes    string
	ContentHash   string
	UID           string
}

// Edge is one `edges` row (spec 002 §4). ToID is a pointer: nil = name-only
// (unresolved — CI-INV-3, drop-not-guess; the edge the extractor saw but
// could not bind, never a fabricated resolved edge). FromID/FileID are
// implicit from the per-file write context (K1). The store owns ID.
type Edge struct {
	ID         int64
	Kind       string
	FromID     int64
	ToID       *int64
	ToName     string
	FileID     int64
	Confidence string
	Line       int
	Context    string
}

// UnresolvedRef is one `unresolved_edges` ledger row (spec 001 §5 — the
// per-edge record of "we saw this reference but could not resolve it",
// CI-INV-3). Candidates is the JSON candidate list (empty = none).
// FileID is implicit from the per-file write context.
type UnresolvedRef struct {
	Reference  string
	Kind       string
	Line       int
	Candidates string
}

// Chunk is one `chunks` row (spec 001 §3 — the FTS + semantic leg text).
// Kind is 'ast' | 'lines'. FileID is implicit from the per-file write
// context. The store owns ID.
type Chunk struct {
	ID          int64
	FileID      int64
	Kind        string
	StartLine   int
	EndLine     int
	Text        string
	ContentHash string
}

// FileGraph is the per-file extract output the Index driver writes in one
// tx (K1): the file's symbols, edges, unresolved ledger rows, and chunks.
// The extractor (Task 3's ladder; Task 1's test double) produces it.
type FileGraph struct {
	Symbols    []Symbol
	Edges      []Edge
	Unresolved []UnresolvedRef
	Chunks     []Chunk
	// Oversized is true when the file was skipped for being over the max
	// size (the driver counts it in files_oversized, not silent — T1). The
	// zero value (false) means "not oversized" so a test double that doesn't
	// set it is treated as a normal indexed file. An additive data field
	// (not a Store method — the D2 door is about interface methods, not this
	// shape).
	Oversized bool
}

// FreshnessRecord is the one `index_freshness` row (spec 001 §4 — the
// query-path staleness contract). Project is the table's PK.
// ExtractorTiers is the D-TIER per-language tier map (the spec 001 §4 DDL
// predates the D-TIER fold; the column is created by 100_code_index).
type FreshnessRecord struct {
	Project           string
	ExtractorStamp    string
	SchemaFingerprint string
	LastIndexedAt     *string
	FilesIndexed      int
	FilesDeleted      int
	FilesOversized    int
	SymbolsCount      int
	EdgesCount        int
	ChunksCount       int
	UnresolvedCount   int
	CoverageFair      float64
	Capabilities      string
	ExtractorTiers    string
	UpdatedAt         string
}

// RetrievalTrail is one `retrieval_trails` row (the shared INV-5 table,
// migration 003 — the memory change owns the table; the code-index owns
// the code-side writes, T15/M7).
type RetrievalTrail struct {
	Route     string
	Query     string
	Project   string
	Filters   string
	IDs       string
	Degraded  bool
	Timestamp string
}

// EdgeRow is the graph-read projection of an `edges` row (the walker's
// surface — spec 004). ToID is a pointer: nil = name-only.
type EdgeRow struct {
	ID         int64
	Kind       string
	FromID     int64
	ToID       *int64
	ToName     string
	FileID     int64
	Confidence string
	Line       int
	Context    string
}

// SymbolRow is the read projection of a `symbols` row.
type SymbolRow struct {
	ID            int64
	SyncID        string
	Name          string
	QualifiedName string
	Kind          string
	Language      string
	Signature     string
	StartLine     int
	EndLine       int
	UID           string
	FileID        int64
}
