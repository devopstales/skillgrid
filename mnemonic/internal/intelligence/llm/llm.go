// Package llm is the LLM seam for the mnemonic-v2 memory function (spec
// 003-intelligence §2.1, INV-3). The LLM interface is the seam: a nil seam
// means "absent" (the deterministic floor runs, flagged degraded), and a
// present-but-erroring seam is "failing" (one bounded retry, then the floor).
//
// The OpenAI-compatible provider (openai_compat.go) is the single concrete
// adapter: one HTTP shape (POST {base}/chat/completions) covers an
// OpenAI-compatible cloud host and Ollama (pointed at its /v1 root) — no
// second client.
package llm

import "context"

// LLM is the semantic-judgment seam (spec 003-intelligence §2.1). A nil LLM
// is the "absent" state: the caller's deterministic floor runs and the result
// is flagged degraded (INV-3). The interface is process-lifetime — one client
// per process, constructed at startup.
type LLM interface {
	// Classify 4-way classifies a new observation against pre-filtered
	// candidate contents and returns the verdict + target id + reason
	// (spec 01-memory/004-dedup).
	Classify(ctx context.Context, newContent string, candidates []string) (DedupDecision, error)
	// Judge renders a semantic judgment on a pending relation between two
	// observations (conflict? supersedes? compatible?) and returns the
	// relation + confidence + evidence.
	Judge(ctx context.Context, source, target Observation, rel Relation) (Judgment, error)
	// Compare proactively compares two observations and returns a judged
	// relation.
	Compare(ctx context.Context, a, b Observation) (Judgment, error)
	// Extract pulls discrete candidate observations from free text (passive
	// capture, session summary structuring).
	Extract(ctx context.Context, text string) ([]CandidateObservation, error)
}

// DedupVerdict is the LLM's 4-way classification of a new observation against
// existing candidates. The four valid values are VerdictAdd, VerdictUpdate,
// VerdictDelete, and VerdictNoop (v1 constants, mnemonic_back/internal/memory/
// types.go:57). The zero value is not a valid verdict.
type DedupVerdict string

const (
	// VerdictAdd: the new observation is genuinely new — store it.
	VerdictAdd DedupVerdict = "add"
	// VerdictUpdate: the new observation updates an existing one
	// (DedupDecision.CandidateID names the target).
	VerdictUpdate DedupVerdict = "update"
	// VerdictDelete: the new observation supersedes / deletes an existing one
	// (DedupDecision.CandidateID names the target).
	VerdictDelete DedupVerdict = "delete"
	// VerdictNoop: the new observation matches an existing one and changes
	// nothing — no action required.
	VerdictNoop DedupVerdict = "noop"
)

// DedupDecision is the outcome of the 4-way pre-write dedup classification.
// CandidateID is a 1-based observation id (the LLM's 0-based candidate index
// plus one), and is only meaningful for VerdictUpdate / VerdictDelete. Reason
// is the provenance label (INV-1, no silent fusion): "llm" when the seam made
// the call, "hash" when the deterministic floor did, "" when neither has run.
type DedupDecision struct {
	Verdict     DedupVerdict
	CandidateID int64
	Reason      string
}

// Judgment is a semantic judgment on a relation (spec 003-intelligence §2.1).
// Model is the name of the model that rendered it — recorded on the relation's
// marked_by_model column (INV-1 provenance).
type Judgment struct {
	Relation   string
	Confidence float64
	Evidence   string
	Model      string
}

// Observation is the minimal observation the LLM seam reasons over. It carries
// the identifying + content fields the prompts need, without the store's full
// row (the store's own Observation type is in internal/core/store; the seam
// stays a leaf and does not import it).
type Observation struct {
	ID      int64
	SyncID  string
	Type    string
	Title   string
	Content string
}

// Relation is the pending relation being judged (the memory_relations edge the
// caller is asking the seam to confirm or refute).
type Relation struct {
	SourceID string
	TargetID string
	Relation string
	Reason   string
	Evidence string
}

// CandidateObservation is one observation the Extract call surfaced from free
// text (type + title + content).
type CandidateObservation struct {
	Type    string
	Title   string
	Content string
}
