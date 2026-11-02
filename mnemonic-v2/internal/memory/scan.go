package memory

// scan.go — the project conflict scan (spec 01-memory/003 §7): the
// page-batched re-run of candidate detection over a project, surfacing
// NEW pending relations (a pair with any existing relation row — pending
// or judged — is skipped) and returning the count of newly materialized
// rows. This is the post-pull "are there conflicts I haven't judged?"
// pass: observations that a sync pull introduced (and that never went
// through the save pipeline's step 7.5 materialization) are the ones only
// the scan catches.
//
// Detection is the same lexical heuristic the write path runs (Task 9's
// pre-filter + gate — spec §3 note: synonyms and one-term overlaps can
// miss; the LLM is the semantic layer). The scan adds no new heuristic.

import (
	"context"
	"fmt"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// scanPageSize is the page size for the scan's live-row walk (spec
// 01-memory/003 §7 "page-batched"). 500 rows per page bounds the in-
// memory set between RawQuery calls.
const scanPageSize = 500

// Scan re-runs candidate detection over a project (spec
// 01-memory/003 §7) and materializes new pending relations for rows that
// have no relation row yet (a row with any existing relation row — pending
// or judged, either direction — is skipped). It returns the count of
// newly materialized pending rows. A store without the RawQuery seam
// (Task 18's Postgres before it lands) cannot walk the live rows — the
// scan errors (the write path's save-time materialization is unaffected).
func (s *Service) Scan(ctx context.Context, project string) (int, error) {
	project = strings.TrimSpace(project)
	if project == "" {
		return 0, fmt.Errorf("memory: scan requires a project")
	}
	rows, err := s.scanLiveRows(ctx, project)
	if err != nil {
		return 0, err
	}
	newCount := 0
	for _, r := range rows {
		if !s.scanSeen(ctx, r) {
			for _, cand := range s.scanCandidates(ctx, r, project) {
				if s.addPendingRelation(ctx, r, cand) {
					newCount++
				}
			}
		}
	}
	return newCount, nil
}

// scanRow is one live observation row the scan walks (id order — the
// scan's deterministic output order).
type scanRow struct {
	id       int64
	syncID   string
	title    string
	scope    string
	topicKey string
}

// scanLiveRows lists the project's live observations in id order (page
// batches of scanPageSize).
func (s *Service) scanLiveRows(ctx context.Context, project string) ([]scanRow, error) {
	_ = ctx
	rq, ok := s.st.(interface {
		RawQuery(query string, args ...any) ([][]any, error)
	})
	if !ok {
		return nil, fmt.Errorf("memory: store %T does not expose the RawQuery seam the scan reads through", s.st)
	}
	var out []scanRow
	lastID := int64(0)
	for {
		rows, err := rq.RawQuery(`
			SELECT id, ifnull(sync_id,''), ifnull(title,''), ifnull(scope,''), ifnull(topic_key,'')
			 FROM observations
			 WHERE LOWER(project) = ? AND deleted_at IS NULL AND id > ?
			 ORDER BY id LIMIT ?`, project, lastID, scanPageSize)
		if err != nil {
			return nil, err
		}
		if len(rows) == 0 {
			break
		}
		for _, row := range rows {
			id, ok := row[0].(int64)
			if !ok {
				return nil, fmt.Errorf("memory: scan row id: %T", row[0])
			}
			lastID = id
			out = append(out, scanRow{
				id:       id,
				syncID:   strVal(row, 1),
				title:    strVal(row, 2),
				scope:    strVal(row, 3),
				topicKey: strVal(row, 4),
			})
		}
		if len(rows) < scanPageSize {
			break
		}
	}
	return out, nil
}

// scanSeen reports whether the row has any relation row at all (pending or
// judged, either direction) — the "already-seen pair" skip (spec §7).
func (s *Service) scanSeen(ctx context.Context, r scanRow) bool {
	_ = ctx
	ok, err := s.st.(interface {
		Probe(query string, args ...any) (bool, error)
	}).Probe(`
		SELECT COUNT(*) > 0 FROM memory_relations
		 WHERE source_id = ? OR target_id = ?`, r.syncID, r.syncID)
	if err != nil {
		// A failed seen-check is not evidence the row is unseen: the scan
		// skips the row rather than re-surfacing pairs it may have already
		// seen (a duplicate pending row is a miss, not a wrong verdict —
		// the same posture as the write path's materialization).
		return true
	}
	return ok
}

// scanCandidates runs the write path's detection (pre-filter + gate) for
// one scanned row: the row's title is the FTS query (the pre-filter's
// project+scope scope, the row's scope as the literal), the gate compares
// the scanned row against each pre-filter hit (a candidate must be a
// different row — the pre-filter's self-hit is skipped by id).
func (s *Service) scanCandidates(ctx context.Context, r scanRow, project string) []store.Observation {
	var out []store.Observation
	for _, c := range s.conflictPreFilter(ctx, project, r.scope, r.title) {
		if c.ID == r.id {
			continue
		}
		hitTopic := ""
		if c.TopicKey != nil {
			hitTopic = *c.TopicKey
		}
		if candidateGate(titleTopic{Title: r.title, TopicKey: r.topicKey},
			titleTopic{Title: c.Title, TopicKey: hitTopic}) {
			out = append(out, c)
		}
	}
	return out
}

// addPendingRelation materializes one pending candidate (source = the
// scanned row, target = the candidate) and reports whether the write
// landed (a failure drops only that candidate — the scan's posture: a
// miss, not a wrong verdict; the wiki reads pending rows, a missing one
// is a miss). The seen-check already skipped rows with any relation row,
// so a write that lands is by construction a new pending row.
func (s *Service) addPendingRelation(ctx context.Context, r scanRow, cand store.Observation) bool {
	_, err := s.st.AddRelation(ctx, store.AddRelationParams{
		SourceID: r.syncID,
		TargetID: cand.SyncID,
		Relation: "pending",
		Reason:   "lexical conflict candidate (project scan, spec 01-memory/003 §7)",
	})
	return err == nil
}

// MigrationState is the v1→v2 seed migration state reader (the doctor's
// migration-state check, TICKET-19 doctor completeness). It is a one-way
// hook the internal/migrate package registers in its init (the dependency
// direction memory → migrate forbids the Service from importing migrate
// directly — the same one-way wire as the store's
// SetApplyPulledMutationHook; the doctor calls it with the configured v1
// dir, a missing v1 dir is not an error, the check stays advisory-ok).
var MigrationState func(ctx context.Context, v1Dir, project string) (any, error)

// MigrationStateHook is the v1→v2 seed migration state reader (the
// doctor's migration-state check, TICKET-19 doctor completeness). It is a
// one-way hook the internal/migrate package registers in its init (the
// dependency direction memory → migrate keeps the Service from importing
// migrate directly — the store's SetApplyPulledMutationHook pattern). The
// doctor calls the Service's MigrationState with the configured v1 dir; a
// missing v1 dir is not an error (the check stays advisory-ok).
var MigrationStateHook func(ctx context.Context, v1Dir, project string) (any, error)

// MigrationState runs the seed-migration state check for the service's
// project (the doctor's wiring point). It delegates to the hook the
// internal/migrate package registers; a missing v1 dir is not an error
// (the check stays advisory-ok).
func (s *Service) MigrationState(ctx context.Context, v1Dir, project string) (any, error) {
	if MigrationStateHook == nil {
		return nil, nil // hook not bound (no v1 dir to check)
	}
	return MigrationStateHook(ctx, v1Dir, project)
}
