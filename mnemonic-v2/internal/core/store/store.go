package store

import (
	"context"
	"fmt"
)

// SyncSeam are the sync-core test hooks (Task 15). The SQLite adapter
// implements them; the memory package type-asserts (the pipeline's Probe
// pattern — Go's export data carries unexported methods only within a
// package). Production code never calls these: SetSyncEnabled is wired by
// the transport layer (Task 13) from cfg.Sync.Enabled at startup, and
// SetEnrolledForTest is a test-only enrollment fixture for the memory
// package's gate assertions. The Postgres adapter (Task 18) adopts the same
// seam shape.
type SyncSeam interface {
	SetSyncEnabled(on bool)
	SetEnrolledForTest(project string) error
}

// Dialect selects the physical backend (spec 001-storage §2).
type Dialect int

const (
	// SQLite is the default local-truth backend (one file per project).
	SQLite Dialect = iota
	// Postgres is the remote dialect render (Task 18).
	Postgres
)

// OpenUpTo opens (or creates) a store and applies the embedded migrations
// only up to lastFile (inclusive, the same sort order the runner uses) —
// the test/diagnostic seam for the migration-parity regression (T-CI-3, M1:
// the parity test needs a store *at* a specific version). Production Open
// migrates to completion; OpenUpTo stops before the first unapplied file
// past lastFile and records schema_version for the applied set. lastFile is
// the migration file NAME (e.g. "003_retrieval_trails.sql"); the empty
// string migrates to completion (== Open).
//
// This is a seam on the runner, not a new Store method (the D2 door stays
// closed): it is a package-level Open analog, not part of the Store
// interface.
func OpenUpTo(ctx context.Context, dataDir, project string, dialect Dialect, dsn, lastFile string) (Store, error) {
	if dialect != SQLite {
		// the PG analog mirrors this over migrations_pg (TICKET-08c's file
		// does not exist yet — the PG runner's migrations_pg set is the
		// memory change's 001-003; the loop simply stops at the highest
		// applied file when lastFile is past that set).
		return nil, fmt.Errorf("store.OpenUpTo: PG analog is TICKET-08c's scope (dialect %v)", dialect)
	}
	return openSQLiteUpTo(ctx, dataDir, project, lastFile)
}

// Store is the dialect-agnostic data layer (spec 001-storage §7).
// Functions see only this interface (INV-2); dialect SQL lives inside the
// SQLiteStore / PostgresStore adapters.
type Store interface {
	// observations
	AddObservation(ctx context.Context, p AddObservationParams) (AddObservationResult, error)
	GetObservation(ctx context.Context, id int64, includeHistory bool) (Observation, error)
	UpdateObservationForProject(ctx context.Context, id int64, expectedProject string, p UpdateParams) error
	DeleteObservationForProject(ctx context.Context, id int64, expectedProject string, hard bool) error
	ConsolidateObservations(ctx context.Context, p ConsolidateParams) (ConsolidateResult, error)

	// relations
	AddRelation(ctx context.Context, p AddRelationParams) (Relation, error)
	JudgeRelation(ctx context.Context, id int64, p JudgeParams) error

	// retrieval
	Search(ctx context.Context, q SearchQuery) (SearchResult, error)
	VectorSearch(ctx context.Context, q VecQuery) (VecResult, error)
	Context(ctx context.Context, q ContextQuery) (ContextResult, error)

	// sync
	EnqueueMutation(ctx context.Context, m Mutation) (int64, error)
	ListPendingMutations(ctx context.Context, cursor int64, limit int) ([]Mutation, error)
	ApplyPulledMutation(ctx context.Context, m Mutation) error
	Tombstone(ctx context.Context, entity, key string) error

	// additive methods (beyond spec 001-storage §7 — S-2 rule: Go types only)
	MarkReviewed(ctx context.Context, id int64) error
	BumpRetrievalUsage(ctx context.Context, ids []int64) error
	SearchPrompts(ctx context.Context, q PromptSearchQuery) (PromptSearchResult, error)
	// JudgeMany runs the verdict batch inside ONE transaction (spec
	// 004-transport-mcp §4: mem_judge_many is one queued write = one
	// transaction). base carries the provenance defaults (actor/kind); a
	// per-request Model overrides base.MarkedByModel when non-empty.
	JudgeMany(ctx context.Context, rs []JudgeRequest, base JudgeParams) ([]Relation, error)

	// test-only hooks (SQLite store; a no-op-capable method on the interface
	// so both dialects satisfy it)
	TouchForTest(ctx context.Context, id int64, lastSeenAt string, revisionCount int) error
	SetTxnHook(hook func())

	// lifecycle
	Migrate(ctx context.Context) error
	Close() error
	Ping(ctx context.Context) error

	// code-index (ADR-0046 — additive to the memory change's Store; the
	// write path is the per-file target-state tx the Index driver composes,
	// K1: the caller wraps the five per-file writes in ONE tx via Begin).
	UpsertFile(ctx context.Context, f FileRow) (int64, error)
	UpsertSymbols(ctx context.Context, fileID int64, syms []Symbol) (int64, error)
	ReplaceEdgesForFile(ctx context.Context, fileID int64, edges []Edge) error
	ReplaceUnresolvedForFile(ctx context.Context, fileID int64, refs []UnresolvedRef) error
	ReplaceChunksForFile(ctx context.Context, fileID int64, chunks []Chunk) error
	DeleteFile(ctx context.Context, path string) error
	PruneOrphanSymbols(ctx context.Context, targetUIDs []string) error
	UpdateIndexFreshness(ctx context.Context, f FreshnessRecord) error
	// graph reads (the dialect-agnostic walker's surface, spec 004)
	EdgesByFrom(ctx context.Context, fromID int64, kinds []string) ([]EdgeRow, error)
	EdgesByTo(ctx context.Context, toID int64, kinds []string) ([]EdgeRow, error)
	EdgesByNameOnlyToName(ctx context.Context, toName string, kinds []string) ([]EdgeRow, error)
	SymbolByName(ctx context.Context, name string) ([]SymbolRow, error)
	SymbolByUID(ctx context.Context, uid string) (SymbolRow, error)
	FileByPath(ctx context.Context, path string) (FileRow, error)
	// vectors (M2: the vectorstore.Engine holds the Store, not a *sql.DB —
	// the vector SQL is dialect-rendered; BLOB tables here, ANN in 08a)
	EmbedSymbol(ctx context.Context, symbolID int64, model string, vec []float32) error
	EmbedChunk(ctx context.Context, chunkID int64, model string, vec []float32) error
	ClearVectors(ctx context.Context) error
	// trail (T15/INV-5 — the shared retrieval_trails table, migration 003)
	RecordRetrieval(ctx context.Context, t RetrievalTrail) error
	// the table-adding seam (T21 — the KB's KBStore adds its kb_* tables
	// through this without forking the dialect code; multi-statement)
	ExecDDL(ctx context.Context, statements []string) error
	// K1 — the single per-file tx surface: Begin opens the transaction the
	// driver composes the five per-file writes inside (the idempotency
	// claim rests on it — one file's write all commits or all rolls back).
	Begin(ctx context.Context) (Txn, error)
}

// Txn is the per-file transaction surface (K1). It carries the code-index
// write methods over the open transaction; Commit/Rollback settle it. The
// caller composes UpsertFile → UpsertSymbols → ReplaceEdgesForFile →
// ReplaceUnresolvedForFile → ReplaceChunksForFile inside one Txn.
type Txn interface {
	UpsertFile(ctx context.Context, f FileRow) (int64, error)
	UpsertSymbols(ctx context.Context, fileID int64, syms []Symbol) (int64, error)
	ReplaceEdgesForFile(ctx context.Context, fileID int64, edges []Edge) error
	ReplaceUnresolvedForFile(ctx context.Context, fileID int64, refs []UnresolvedRef) error
	ReplaceChunksForFile(ctx context.Context, fileID int64, chunks []Chunk) error
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}
