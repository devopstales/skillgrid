package store

import "context"

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
}
