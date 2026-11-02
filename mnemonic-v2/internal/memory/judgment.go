package memory

// judgment.go — the explicit judgment surface (spec 01-memory/003 §4–§5):
// Judge / JudgeMany record the agent's verdicts on pending candidates
// (the engine stores, the agent judges — INV-7; the LLM seam's Judge
// method is for the agent-side tooling, the MCP mem_judge records what
// the agent sent), Compare proactively compares two observations through
// the LLM seam and creates a judged relation directly (idempotent per
// unordered pair), and Consolidate is the two-phase fingerprint-gated
// merge of ≥2 live observations (spec §5 — the survivor inherits the
// newest source's session/type/scope, carries a consolidated_from footer,
// the sources are soft-deleted with supersession provenance).

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/syncid"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/intelligence/degrade"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/intelligence/llm"
)

// judgeManyMax is the batch bound (spec 004-transport-mcp mem_judge_many:
// "1–32 judgments in one queued write").
const judgeManyMax = 32

// ErrConsolidateRefused is the structural refusal of a consolidation
// selection (spec 01-memory/003 §5): fewer than two ids, duplicate ids,
// missing or deleted sources, or an unresolvable project. Refusals fire
// before any write.
var ErrConsolidateRefused = errors.New("memory: consolidation refused")

// ErrConsolidateFingerprintMismatch is the fingerprint-gated refusal
// (spec 01-memory/003 §5): the source state or the resolved survivor
// changed between preview and confirm. The error carries a fresh plan in
// the ConsolidateResult the caller receives — inspect and re-confirm the
// new fingerprint; never auto-confirm a replacement.
var ErrConsolidateFingerprintMismatch = errors.New("memory: consolidation fingerprint mismatch")

// ErrCompareDegradedFloor is the clean sentinel for a compare whose LLM
// seam is absent or failing after the bounded retry: the deterministic
// floor ran and created no relation (a compare is a semantic judgment —
// without the seam there is no verdict to record, and an unjudged
// candidate is not evidence of anything).
var ErrCompareDegradedFloor = errors.New("memory: compare degraded to floor (no relation created)")

// JudgeRequest is one agent verdict on a pending candidate (the MCP
// mem_judge shape — spec 004-transport-mcp §3). The store keeps its own
// store.JudgeRequest twin for the batch seam; the service maps between
// them at the JudgeMany boundary.
type JudgeRequest = store.JudgeRequest

// Judge records the agent's verdict on a pending candidate (spec
// 01-memory/003 §4, INV-1 provenance): the row is loaded by id (a
// missing id or an id of a judged row is an idempotent no-op returning
// the existing row) and the verdict + the judging provenance are stored
// (marked_by_actor='agent', marked_by_kind='mcp', marked_by_model = the
// agent-supplied model name or "").
func (s *Service) Judge(ctx context.Context, r JudgeRequest) (store.Relation, error) {
	rel, err := s.getRelation(ctx, r.JudgmentID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return store.Relation{}, fmt.Errorf("memory: relation %d not found", r.JudgmentID)
		}
		return store.Relation{}, err
	}
	if rel.JudgmentStatus == "judged" {
		// Idempotent no-op: the row already carries a verdict; return it
		// untouched (no second write, no field churn).
		return rel, nil
	}
	conf := r.Confidence
	if err := s.st.JudgeRelation(ctx, r.JudgmentID, store.JudgeParams{
		Relation: r.Relation, Reason: r.Reason, Evidence: r.Evidence,
		Confidence: &conf, MarkedByActor: "agent", MarkedByKind: "mcp",
		MarkedByModel: r.Model,
	}); err != nil {
		return store.Relation{}, err
	}
	return s.getRelation(ctx, r.JudgmentID)
}

// JudgeMany records 1–32 verdicts in ONE committed transaction (spec
// 004-transport-mcp mem_judge_many: "batch 1–32 judgments in one queued
// write, ordered, independent verdicts" — the write queue serializes it
// at the transport layer, Task 13). The store's JudgeMany seam runs the
// whole batch inside one transaction; already-judged ids are idempotent
// no-ops, a missing id is an error (the whole batch rolls back).
//
// ponytail: the batch reaches the store through the judgeTxnSeam type
// assertion rather than the Store interface (the SQLite adapter implements
// the seam; the interface stays stable for Task 18's Postgres adapter).
// Ceiling: a JudgeMany method on Store itself when the dialect surface is
// proven stable.
func (s *Service) JudgeMany(ctx context.Context, rs []JudgeRequest) ([]store.Relation, error) {
	if len(rs) < 1 || len(rs) > judgeManyMax {
		return nil, fmt.Errorf("memory: judge_many requires 1–%d judgments, got %d", judgeManyMax, len(rs))
	}
	// The whole batch runs in ONE committed transaction (spec
	// 004-transport-mcp §4: one queued write = one transaction). The
	// store's JudgeMany seam owns it. A dialect without the seam REFUSES
	// the batch (fix round 3: the per-judgment fallback committed one
	// transaction per judgment, violating the one-transaction guarantee —
	// fail-safe over silent N-transaction degradation; both shipped
	// dialects implement the seam, so only a future adapter is affected).
	// The single-judgment Judge path stays available.
	seam, ok := s.st.(judgeTxnSeam)
	if !ok {
		return nil, fmt.Errorf("memory: store %T does not expose the JudgeMany transaction seam", s.st)
	}
	// The seam stamps the per-request Model on its own (INV-1
	// marked_by_model); the base carries the transport's provenance.
	base := store.JudgeParams{MarkedByActor: "agent", MarkedByKind: "mcp"}
	return seam.JudgeMany(ctx, rs, base)
}

// judgeTxnSeam is the store's one-transaction JudgeMany seam (the SQLite
// adapter implements it; a dialect without it refuses the batch — the
// single-judgment Judge path stays available).
type judgeTxnSeam interface {
	JudgeMany(ctx context.Context, rs []JudgeRequest, base store.JudgeParams) ([]store.Relation, error)
}

// getRelation reads a memory_relations row through the store's read seam
// (the SQLite adapter's GetRelation; the seam keeps the store interface
// stable — the read is a single-row primary-key lookup, no FTS surface).
func (s *Service) getRelation(ctx context.Context, id int64) (store.Relation, error) {
	st, ok := s.st.(interface {
		GetRelation(ctx context.Context, id int64) (store.Relation, error)
	})
	if !ok {
		return store.Relation{}, fmt.Errorf("memory: store %T does not expose the GetRelation seam", s.st)
	}
	return st.GetRelation(ctx, id)
}

// Compare proactively compares two observations through the LLM seam and
// creates a judged relation directly (spec 01-memory/003 §4, system
// provenance). Idempotent per unordered pair: an existing judged row for
// the pair (either direction) is returned instead of a second row. A
// missing/deleted/identical pair is refused before the seam call; a
// degraded seam (absent, or failing after the bounded retry) creates no
// relation and returns ErrCompareDegradedFloor.
func (s *Service) Compare(ctx context.Context, aID, bID int64) (store.Relation, error) {
	if aID == bID {
		return store.Relation{}, fmt.Errorf("memory: compare requires two distinct observations")
	}
	a, err := s.st.GetObservation(ctx, aID, false)
	if err != nil {
		return store.Relation{}, fmt.Errorf("memory: compare observation %d: %w", aID, err)
	}
	b, err := s.st.GetObservation(ctx, bID, false)
	if err != nil {
		return store.Relation{}, fmt.Errorf("memory: compare observation %d: %w", bID, err)
	}
	// Idempotence: an existing judged row for the unordered pair (either
	// direction) is the result — no seam call, no second row.
	if existing, ok, err := s.existingJudgedPair(ctx, a.SyncID, b.SyncID); err != nil {
		return store.Relation{}, err
	} else if ok {
		return existing, nil
	}
	if s.llm == nil {
		return store.Relation{}, ErrCompareDegradedFloor
	}
	lo, ro := a, b
	if a.ID > b.ID {
		lo, ro = b, a // the pair is unordered; the seam sees a stable order
	}
	res, degraded := degrade.Run(ctx, &compareSeam{llm: s.llm, a: toLLMObservation(lo), b: toLLMObservation(ro)},
		func() any { return llm.Judgment{} }, "compare")
	if degraded {
		return store.Relation{}, ErrCompareDegradedFloor
	}
	j, ok := res.(llm.Judgment)
	if !ok || j.Relation == "" {
		// A seam that returns no verdict is treated as the floor: nothing
		// to record (the floor is the no-relation result).
		return store.Relation{}, ErrCompareDegradedFloor
	}
	// re-check after the seam call (a concurrent compare may have landed
	// the row in the window — the unordered-pair lookup is the guard).
	if existing, ok, err := s.existingJudgedPair(ctx, a.SyncID, b.SyncID); err != nil {
		return store.Relation{}, err
	} else if ok {
		return existing, nil
	}
	conf := j.Confidence
	rel, err := s.addJudgedRelation(ctx, a.SyncID, b.SyncID, j.Relation, j.Evidence, j.Model, &conf)
	if err != nil {
		return store.Relation{}, err
	}
	return rel, nil
}

// compareSeam adapts llm.LLM.Compare onto degrade.Seam (the floor returns
// the same concrete type, llm.Judgment — the zero value = no relation).
type compareSeam struct {
	llm llm.LLM
	a   llm.Observation
	b   llm.Observation
}

func (c *compareSeam) Call(ctx context.Context, a, b any) (any, error) {
	return c.llm.Compare(ctx, c.a, c.b)
}

// toLLMObservation narrows a store row to the seam's minimal observation.
func toLLMObservation(o store.Observation) llm.Observation {
	return llm.Observation{ID: o.ID, SyncID: o.SyncID, Type: o.Type, Title: o.Title, Content: o.Content}
}

// existingJudgedPair looks up a judged memory_relations row for an
// unordered sync-id pair (either direction). A read failure is not
// evidence of absence (spec 004-dedup §6 posture) — it errors; a missing
// row returns ok=false.
func (s *Service) existingJudgedPair(ctx context.Context, a, b string) (store.Relation, bool, error) {
	st, ok := s.st.(interface {
		RawQuery(query string, args ...any) ([][]any, error)
	})
	if !ok {
		return store.Relation{}, false, fmt.Errorf("memory: store %T does not expose the RawQuery seam the compare idempotence reads through", s.st)
	}
	rows, err := st.RawQuery(`
		SELECT id FROM memory_relations
		 WHERE judgment_status = 'judged'
		   AND ((source_id = ? AND target_id = ?) OR (source_id = ? AND target_id = ?))
		 ORDER BY id LIMIT 1`, a, b, b, a)
	if err != nil {
		return store.Relation{}, false, err
	}
	if len(rows) == 0 {
		return store.Relation{}, false, nil
	}
	id, ok2 := rows[0][0].(int64)
	if !ok2 {
		return store.Relation{}, false, fmt.Errorf("memory: compare idempotence row id: %T", rows[0][0])
	}
	rel, err := s.getRelation(ctx, id)
	return rel, true, err
}

// addJudgedRelation inserts a judged relation with system provenance
// (spec 01-memory/003 §4: mem_compare "creates a judged relation directly
// (system provenance)"). It writes through the store's own write path
// (AddRelation — the sync_id mint + journal posture) and then records the
// verdict through JudgeRelation: the store's relation vocabulary defaults
// to pending on insert, so the judged insert is a two-step through the
// store's stable methods.
func (s *Service) addJudgedRelation(ctx context.Context, source, target, relation, evidence, model string, confidence *float64) (store.Relation, error) {
	rel, err := s.st.AddRelation(ctx, store.AddRelationParams{
		SourceID: source, TargetID: target, Relation: "pending",
		Reason: "mem_compare: proactive LLM comparison", Evidence: evidence,
	})
	if err != nil {
		return store.Relation{}, err
	}
	if err := s.st.JudgeRelation(ctx, rel.ID, store.JudgeParams{
		Relation: relation, Reason: "mem_compare: proactive LLM comparison", Evidence: evidence,
		Confidence: confidence, MarkedByActor: "system", MarkedByKind: "system", MarkedByModel: model,
	}); err != nil {
		return store.Relation{}, err
	}
	return s.getRelation(ctx, rel.ID)
}

// ConsolidateRequest is the two-phase consolidation input (spec
// 01-memory/003 §5, the MCP mem_consolidate shape).
type ConsolidateRequest struct {
	IDsToDelete     []int64
	NewTitle        string
	NewContent      string
	ExpectedProject string
	Type            *string
	TopicKey        *string
	Fingerprint     string
	Confirm         bool
}

// ConsolidateResult is the consolidation outcome: the plan is always
// present (preview builds it; confirm re-resolves it; a fingerprint
// mismatch returns the fresh plan alongside its error), and Survivor is
// set only on a confirmed merge.
type ConsolidateResult struct {
	Plan     *ConsolidatePlan
	Survivor *store.Observation
}

// ConsolidatePlan is the preview/confirm plan (spec 01-memory/003 §5):
// the sources (in id order), the survivor's resolved identity, the
// project, and the opaque Fingerprint.
type ConsolidatePlan struct {
	Sources     []store.Observation
	SurvivorID  int64
	Project     string
	Fingerprint string
}

// Consolidate runs the two-phase fingerprint-gated merge (spec
// 01-memory/003 §5). Preview (Confirm=false): the selection is resolved
// and the plan + Fingerprint are built; nothing is written. Confirm
// (Confirm=true): the selection is re-resolved, the fingerprint is
// compared (mismatch → ErrConsolidateFingerprintMismatch + a fresh plan,
// no writes), and the merge executes in one local transaction — the
// survivor inherits the newest source's session/type/scope (explicit
// type/topic_key override), its content carries the consolidated_from
// footer with the source sync_ids, and the sources are soft-deleted with
// superseded_by_sync_id → the survivor.
func (s *Service) Consolidate(ctx context.Context, r ConsolidateRequest) (ConsolidateResult, error) {
	plan, err := s.buildConsolidatePlan(ctx, r)
	if err != nil {
		// A structural refusal (pre-write): no plan to return.
		return ConsolidateResult{}, err
	}
	if !r.Confirm {
		return ConsolidateResult{Plan: plan}, nil
	}
	if r.Fingerprint != "" && r.Fingerprint != plan.Fingerprint {
		// Fingerprint gate: the source state or the resolved survivor
		// changed since the preview — refuse with the fresh plan (the
		// caller inspects and re-confirms; never auto-confirm).
		return ConsolidateResult{Plan: plan}, ErrConsolidateFingerprintMismatch
	}
	survivor, err := s.confirmConsolidate(ctx, r, plan)
	if err != nil {
		return ConsolidateResult{}, err
	}
	return ConsolidateResult{Plan: plan, Survivor: &survivor}, nil
}

// buildConsolidatePlan resolves the selection (refusals first, before any
// write) and computes the plan + Fingerprint (spec 01-memory/003 §5):
// sha256 hex over the canonical {"version":"consolidate:v2","sources":[…
// sorted by ID],…} projection of the resolved selection.
func (s *Service) buildConsolidatePlan(ctx context.Context, r ConsolidateRequest) (*ConsolidatePlan, error) {
	project := strings.TrimSpace(r.ExpectedProject)
	if project == "" {
		return nil, fmt.Errorf("%w: unresolvable project (expected_project is required)", ErrConsolidateRefused)
	}
	if len(r.IDsToDelete) < 2 {
		return nil, fmt.Errorf("%w: at least 2 ids are required, got %d", ErrConsolidateRefused, len(r.IDsToDelete))
	}
	seen := map[int64]bool{}
	for _, id := range r.IDsToDelete {
		if seen[id] {
			return nil, fmt.Errorf("%w: duplicate id %d in the selection", ErrConsolidateRefused, id)
		}
		seen[id] = true
	}
	sources := make([]store.Observation, 0, len(r.IDsToDelete))
	for _, id := range r.IDsToDelete {
		o, err := s.st.GetObservation(ctx, id, false)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, fmt.Errorf("%w: observation %d is missing or deleted", ErrConsolidateRefused, id)
			}
			return nil, err
		}
		// the ownership assertion (spec 01-memory/001 §6): the source must
		// belong to the expected project (the store's own guard, mirrored
		// here so the refusal fires before the transaction, not inside it).
		if proj := derefStr(o.Project); proj != "" && proj != project {
			return nil, fmt.Errorf("%w: observation %d belongs to project %q, not %q", ErrConsolidateRefused, id, proj, project)
		}
		sources = append(sources, o)
	}
	// id order: the fingerprint's canonical source order AND the
	// "newest source" resolution (the largest id is the newest row — the
	// store's AUTOINCREMENT id order).
	sort.Slice(sources, func(i, j int) bool { return sources[i].ID < sources[j].ID })
	newest := sources[len(sources)-1]
	survivorType := newest.Type
	if r.Type != nil && *r.Type != "" {
		survivorType = *r.Type
	}
	if r.NewTitle == "" || strings.TrimSpace(r.NewContent) == "" {
		return nil, fmt.Errorf("%w: new_title and new_content are required", ErrConsolidateRefused)
	}
	content := r.NewContent + consolidatedFooter(sources)
	survivor := map[string]any{
		"title":      r.NewTitle,
		"content":    content,
		"session_id": newest.SessionID,
		"type":       survivorType,
		"scope":      newest.Scope,
		"project":    project,
		"topic_key":  derefStr(r.TopicKey),
		"sync_id":    "", // minted at confirm; the fingerprint covers the selection, not the mint
	}
	payload, err := json.Marshal(map[string]any{
		"version":  "consolidate:v2",
		"sources":  sources,
		"survivor": survivor,
	})
	if err != nil {
		return nil, fmt.Errorf("memory: consolidate plan: %w", err)
	}
	h := sha256.Sum256(payload)
	return &ConsolidatePlan{
		Sources:     sources,
		SurvivorID:  newest.ID,
		Project:     project,
		Fingerprint: hex.EncodeToString(h[:]),
	}, nil
}

// confirmConsolidate executes a confirmed merge: the survivor is inserted
// through the save gate (step-6/7/8 posture — its provenance, review
// clock, and journal ride on the gate insert), and the sources are
// soft-deleted with supersession provenance through the store's
// ConsolidateObservations (its own transaction, enrollment-gated journal:
// the survivor upsert + the source deletes).
//
// ponytail: the merge runs as two committed transactions (the survivor
// insert, then the sources' soft-delete) rather than one — the store's
// ConsolidateObservations is the single method that stamps
// superseded_by_sync_id, and the SaveTXN gate has no relation/observation
// soft-delete method (a gate method would be the Task 18 shape). Ceiling:
// a Consolidate gate method when Task 18's Postgres adapter lands, so the
// merge commits atomically. The window between the two commits is bounded
// by the write queue (Task 13 serializes the agent's writes) — Engram's
// disclaimer stands (spec 01-memory/003 §5: "not distributed atomicity").
func (s *Service) confirmConsolidate(ctx context.Context, r ConsolidateRequest, plan *ConsolidatePlan) (store.Observation, error) {
	newest := plan.Sources[len(plan.Sources)-1]
	survivorType := newest.Type
	if r.Type != nil && *r.Type != "" {
		survivorType = *r.Type
	}
	scope := newest.Scope
	topicKey := derefStr(r.TopicKey)
	content := r.NewContent + consolidatedFooter(plan.Sources)
	syncID := syncid.NewSyncID("obs")

	// the survivor's insert rides on the save gate (one transaction for
	// the insert; the sources' soft-delete is the store's
	// ConsolidateObservations in a second transaction — the merge's
	// atomicity is "the survivor is never orphaned without its sources'
	// supersession stamps being the next committed write"; Engram's
	// disclaimer, spec §5: not distributed atomicity).
	txn, err := beginSaveTxn(ctx, s.st)
	if err != nil {
		return store.Observation{}, err
	}
	if _, err := txn.CheckSession(ctx, newest.SessionID, plan.Project); err != nil {
		_ = txn.Rollback()
		return store.Observation{}, fmt.Errorf("memory: consolidate session: %w", err)
	}
	var id int64
	id, _, err = txn.InsertObservation(ctx, store.GateInsert{
		SessionID:      newest.SessionID,
		Type:           survivorType,
		Title:          r.NewTitle,
		Content:        content,
		Project:        plan.Project,
		Scope:          scope,
		TopicKey:       topicKey,
		SyncID:         syncID,
		NormalizedHash: storeNormalizedHash(content),
		ReviewAfter:    s.reviewAfterFor(survivorType),
		Importance:     Importance(0, 0, survivorType),
	})
	if err != nil {
		_ = txn.Rollback()
		return store.Observation{}, err
	}
	if err := txn.Commit(ctx); err != nil {
		return store.Observation{}, err
	}
	if _, err := s.st.ConsolidateObservations(ctx, store.ConsolidateParams{
		SourceIDs:      idsOf(plan.Sources),
		TargetProject:  plan.Project,
		SurvivorID:     id,
		SurvivorSyncID: syncID,
	}); err != nil {
		return store.Observation{}, err
	}
	o, err := s.st.GetObservation(ctx, id, false)
	if err != nil {
		return store.Observation{}, err
	}
	return o, nil
}

// consolidatedFooter is the survivor's provenance footer (spec
// 01-memory/003 §5: "the survivor content includes the source sync_ids in
// a consolidated_from footer"): the source sync_ids, in id order.
func consolidatedFooter(sources []store.Observation) string {
	ids := make([]string, 0, len(sources))
	for _, o := range sources {
		ids = append(ids, o.SyncID)
	}
	var b strings.Builder
	b.WriteString("\n\n## consolidated_from\n")
	for _, id := range ids {
		b.WriteString("- " + id + "\n")
	}
	return b.String()
}

// idsOf maps observations to their local ids (id order, as given).
func idsOf(obs []store.Observation) []int64 {
	out := make([]int64, len(obs))
	for i, o := range obs {
		out[i] = o.ID
	}
	return out
}

// derefStr returns the pointed-to string, or "" for nil.
func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
