package store

import "context"

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

	// test-only hooks (SQLite store; a no-op-capable method on the interface
	// so both dialects satisfy it)
	TouchForTest(ctx context.Context, id int64, lastSeenAt string, revisionCount int) error
	SetTxnHook(hook func())

	// lifecycle
	Migrate(ctx context.Context) error
	Close() error
	Ping(ctx context.Context) error
}
