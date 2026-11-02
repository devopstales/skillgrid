package store

import "time"

// Observation is the memory unit (spec 002-schema §2.2).
type Observation struct {
	ID                 int64
	SyncID             string
	SessionID          string
	Type               string
	Title              string
	Content            string
	ToolName           *string
	Project            *string
	Scope              string
	TopicKey           *string
	NormalizedHash     string
	RevisionCount      int
	DuplicateCount     int
	LastSeenAt         *string
	Pinned             bool
	ReviewAfter        *string
	SupersededBySyncID *string
	ExpiresAt          *string
	Importance         float64
	RetrievalUsage     int
	Provenance         Provenance
	Embedding          []byte
	EmbeddingModel     string
	CreatedAt          string
	UpdatedAt          string
	DeletedAt          *string
}

// State derives the virtual lifecycle state from ReviewAfter (Engram
// observationState semantics).
func (o Observation) State() string {
	if o.ReviewAfter == nil || *o.ReviewAfter == "" {
		return "active"
	}
	ra, err := time.Parse("2006-01-02 15:04:05", *o.ReviewAfter)
	if err != nil {
		return "active"
	}
	if !ra.After(time.Now().UTC()) {
		return "needs_review"
	}
	return "active"
}

// Provenance is the INV-1 immutable provenance record.
type Provenance struct {
	SessionID string
	Writer    string
	Model     string
	CreatedAt string
}

// AddObservationParams carries the fields for a new observation save.
type AddObservationParams struct {
	SessionID   string
	Type        string
	Title       string
	Content     string
	ToolName    string
	Project     string
	Scope       string
	TopicKey    string
	OperationID string
}

// AddObservationResult is returned by AddObservation.
type AddObservationResult struct {
	ID       int64
	SyncID   string
	Replayed bool
}

// UpdateParams carries optional field updates for UpdateObservationForProject.
type UpdateParams struct {
	Type     *string
	Title    *string
	Content  *string
	TopicKey *string
	Pinned   *bool
}

// ConsolidateParams is the input for ConsolidateObservations (a confirmed
// consolidation — the memory service owns the preview/fingerprint/merge
// semantics; the store owns the transactional effect).
type ConsolidateParams struct {
	SourceIDs     []int64
	TargetProject string
	// SurvivorID is the local id of the merged observation row (the caller
	// inserted it first; it is excluded from the sources' soft-delete).
	SurvivorID int64
	// SurvivorSyncID is the survivor's sync_id — the value stamped on every
	// source's superseded_by_sync_id (structured supersession provenance).
	SurvivorSyncID string
}

// ConsolidateResult is returned by ConsolidateObservations.
type ConsolidateResult struct {
	SurvivorID int64
	DeletedIDs []int64
}

// SearchQuery is the input for Search.
type SearchQuery struct {
	Query   string
	Project string
	Scope   string
	Type    string
	Limit   int
}

// SearchResult is returned by Search.
type SearchResult struct {
	Observations []Observation
	Degraded     bool
}

// VecQuery is the input for VectorSearch.
type VecQuery struct {
	QueryVector []float32
	Project     string
	TopK        int
}

// VecResult is returned by VectorSearch.
type VecResult struct {
	Observations []Observation
	Degraded     bool
}

// ContextQuery is the input for Context.
type ContextQuery struct {
	SessionID string
	Project   string
	MaxBytes  int
}

// ContextResult is returned by Context.
type ContextResult struct {
	Observations []Observation
	TotalBytes   int
}

// Mutation is a sync journal entry.
type Mutation struct {
	Seq        int64
	TargetKey  string
	Entity     string
	EntityKey  string
	Op         string
	Payload    string
	Source     string
	Project    string
	OccurredAt string
}

// Relation is a memory_relations row.
type Relation struct {
	ID             int64
	SyncID         string
	SourceID       string
	TargetID       string
	Relation       string
	Reason         string
	Evidence       string
	SessionID      string
	JudgmentStatus string
	// Verdict + provenance (spec 002-schema §2.5, INV-1 on judged rows).
	// Confidence is 0 for rows judged without a numeric value (NULL).
	Confidence    float64
	MarkedByActor string
	MarkedByKind  string
	MarkedByModel string
	CreatedAt     string
}

// AddRelationParams is the input for AddRelation.
type AddRelationParams struct {
	SourceID  string
	TargetID  string
	Relation  string
	Reason    string
	Evidence  string
	SessionID string
}

// JudgeParams is the input for JudgeRelation.
type JudgeParams struct {
	Relation      string
	Reason        string
	Evidence      string
	Confidence    *float64
	MarkedByActor string
	MarkedByKind  string
	MarkedByModel string
}

// JudgeRequest is one verdict in a JudgeMany batch (the store-level twin
// of the memory service's JudgeRequest — the store owns the row, the
// service owns the tooling shape; the service maps between them). Model is
// the agent's model name (INV-1 marked_by_model provenance; the transport
// is where it enters — the service hardcodes "" otherwise).
type JudgeRequest struct {
	JudgmentID int64
	Relation   string
	Reason     string
	Evidence   string
	Confidence float64
	Model      string
}

// PromptSearchQuery is the input for SearchPrompts.
type PromptSearchQuery struct {
	Query   string
	Project string
	Limit   int
}

// PromptRow is a single captured-prompt hit with its identity (spec 002-schema
// §2.4). Later tasks build prompt identity on these fields, so they are part of
// the contract (not just the content).
type PromptRow struct {
	ID        int64
	SyncID    string
	SessionID string
	Project   string
	Content   string
	CreatedAt string
}

// PromptSearchResult is returned by SearchPrompts.
type PromptSearchResult struct {
	Prompts  []PromptRow
	Degraded bool
}

// ErrNotImplemented is returned by methods that are on the Store interface
// but not yet implemented on the current adapter.
type ErrNotImplemented struct{ Method string }

func (e ErrNotImplemented) Error() string {
	return "store: " + e.Method + " not implemented yet"
}
