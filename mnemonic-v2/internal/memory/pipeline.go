package memory

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/config"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/syncid"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/intelligence/embedder"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/intelligence/llm"
)

// Save pipeline — spec 01-memory/001-observations §2. Service.Save is the
// single entry point for every save surface (MCP/CLI/HTTP funnel here).
//
// Steps implemented here: 1 VALIDATE, 2 RESOLVE (project + session policy),
// 3 REDACT (strip private spans + truncate), 4.5 REPLAY LEDGER (the
// operation-idempotency check — Task 6), 5 DEDUP (the hash floor; the LLM
// path's internals land in Task 8 — the call site is final), 6 UPSERT/INSERT
// (topic-key revision with observation_versions snapshot, fresh insert with
// review_after + importance), 7 ENQUEUE (sync journal, enrollment-gated),
// 8 COMMIT (one local transaction).
//
// Step 4 (conflict pre-filter) is Task 9; step 10 (Publish) is Task 7.

// ErrSessionEnded is the terminal 409: the session row exists and has
// ended. The gate (core/store) returns store.ErrSessionEnded; the pipeline
// wraps it into this sentinel so the transport layer (Task 13) maps
// errors.Is(err, ErrSessionEnded) → 409.
var ErrSessionEnded = errors.New("memory: session ended")

// ErrSessionProjectMismatch is the 409 for a project_owned session whose
// project does not match the resolved project (the gate returns
// store.ErrSessionProjectMismatch; wrapped the same way).
var ErrSessionProjectMismatch = errors.New("memory: session project mismatch")

// ErrOperationConflict is the 409 for a replay whose operation id is already
// committed under a different payload fingerprint (spec 01-memory/001 §4).
// The transport layer (Task 13) maps errors.Is(err, ErrOperationConflict) →
// 409; the committed observation and ledger row are left untouched.
var ErrOperationConflict = errors.New("memory: operation conflict")

// ErrOperationExpired is the 410 for a replay whose ledger row exists but
// its observation is gone — the target was hard-deleted and the ledger FK
// (ON DELETE SET NULL, Task 1's DDL) nulled observation_id, leaving the row
// as a tombstone (spec 01-memory/001 §4). Mapped to 410 by Task 13.
var ErrOperationExpired = errors.New("memory: operation expired")

// wrapSessionErr re-wraps the gate's session sentinels into the memory
// package's sentinels (errors.Is against the store's values).
func wrapSessionErr(err error) error {
	switch {
	case errors.Is(err, store.ErrSessionEnded):
		return ErrSessionEnded
	case errors.Is(err, store.ErrSessionProjectMismatch):
		return ErrSessionProjectMismatch
	}
	return err
}

// observationTypes is the type vocabulary (spec 002-schema §2.2.1).
var observationTypes = map[string]bool{
	"decision": true, "architecture": true, "bugfix": true, "pattern": true,
	"config": true, "discovery": true, "learning": true, "preference": true,
	"convention": true, "session_summary": true,
}

// validScopes is the scope vocabulary (spec 01-memory/001 §2 step 1).
var validScopes = map[string]bool{"project": true, "personal": true, "global": true}

// topicKeyRe is the kebab-case ≤2-level format: levels of [a-z0-9] joined by
// at most one '/' (spec 01-memory/001 §2 step 1).
var topicKeyRe = regexp.MustCompile(`^[a-z0-9]+(/[a-z0-9]+)?$`)

// Service is the memory-function service: the save pipeline over the store
// + LLM/embedder seams (the seams may be nil — the INV-3 degraded floors).
type Service struct {
	st  store.Store
	cfg config.Config
	bus *EventBus
	llm llm.LLM
	emb embedder.Embedder
}

// NewService builds the service. l and e may be nil (absent seams → the
// deterministic floors flag degraded).
func NewService(st store.Store, cfg config.Config, bus *EventBus, l llm.LLM, e embedder.Embedder) *Service {
	return &Service{st: st, cfg: cfg, bus: bus, llm: l, emb: e}
}

// SaveRequest is one save (spec 01-memory/001 §2 input).
type SaveRequest struct {
	SessionID, Type, Title, Content, ToolName, Project, Scope, TopicKey, OperationID string
	// CapturePrompt is tri-state (spec 004-transport-mcp mem_save row,
	// amended): nil → true (the MCP "absent" default), explicit false → off.
	CapturePrompt   *bool
	CurrentPrompt   string // the current turn's prompt, when capture is on
}

// SaveResult is the save's return shape (spec 01-memory/001 §2 step 9).
// INV-6: every cross-reference is a sync_id, never a local rowid.
type SaveResult struct {
	ID            int64
	SyncID        string
	RevisionCount int
	Replayed      bool
	Dedup         struct {
		Verdict, Reason          string
		MergedIntoSyncID         *string
	}
	Candidates         []store.Relation // conflict candidates (Task 9 fills; empty until then)
	JudgmentRequired   bool
	Degraded           bool
}

// Save runs the save pipeline. It calls only the store.Store interface and
// the LLM/embedder seams — never a raw DB handle (INV-2).
func (s *Service) Save(ctx context.Context, req SaveRequest) (SaveResult, error) {
	// 1. VALIDATE
	if err := validateRequest(req); err != nil {
		return SaveResult{}, err
	}

	// 2. RESOLVE — the store's write path resolves project inside the
	// transaction (explicit > session's project), the same store insert
	// SessionStart uses (the pipeline does not re-resolve beyond the
	// detector: INV-7).

	// 3. REDACT — strip private spans, then truncate (rune count, per the
	// brief) — the hash is over the redacted+truncated content.
	content := truncateContent(stripPrivate(req.Content, s.cfg.Memory.PrivateTag), s.cfg.Memory.ContentMaxBytes)

	// 4.5. REPLAY LEDGER (spec 01-memory/001 §4) — the operation-idempotency
	// check, one ledger lookup before dedup. Identical fingerprint → return
	// the committed id early (dedup/upsert/journal run exactly once — the
	// whole point: a plugin retrying a flaky save never double-writes).
	// Changed fingerprint → ErrOperationConflict (409). Ledger row whose
	// target was deleted (observation_id NULL via ON DELETE SET NULL) →
	// ErrOperationExpired (410). No row (or no operation id) → proceed to
	// dedup; the ledger row is inserted inside the commit transaction by
	// the gate (step 8).
	if req.OperationID != "" {
		out, done, err := s.replayLedger(ctx, s.st, req, content)
		if err != nil {
			return SaveResult{}, err
		}
		if done {
			return out, nil
		}
	}

	// 5. DEDUP — final call site. Task 8 changes only the interior of
	// runDedup when the seam is non-nil; the pipeline never changes again.
	scope := strings.TrimSpace(req.Scope)
	if scope == "" {
		scope = "project"
	}
	decision, degraded := runDedup(ctx, s.llm, s.st, req.Project, scope, content, req.Type, req.Title)

	// 6. UPSERT/INSERT (+ 7 ENQUEUE + 8 COMMIT) in one local transaction.
	row, err := s.commitSave(ctx, s.st, saveTxnInput{
		req:            req,
		content:        content,
		redactedPrompt: s.redactPrompt(req.CurrentPrompt),
		decision:       decision,
		captureOn:      captureOn(req),
	})
	if err != nil {
		return SaveResult{}, err
	}

	// 7/8 happened inside commitSave: the sync mutation (enrollment-gated)
	// rows in the same transaction as the observation + versions.

	// 9. RESULT
	res := SaveResult{
		ID:            row.id,
		SyncID:        row.syncID,
		RevisionCount: row.revisionCount,
		Replayed:      row.replayed,
		Degraded:      degraded,
	}
	res.Dedup.Verdict = string(decision.Verdict)
	res.Dedup.Reason = decision.Reason
	if decision.Verdict == llm.VerdictUpdate || decision.Verdict == llm.VerdictDelete {
		// Task 8's LLM path produces these: the merge lands on
		// CandidateID, and the result references the target by sync_id
		// (INV-6). The merge write is Task 8's; this branch is the final
		// result-shape slot.
		target, err := s.st.GetObservation(ctx, decision.CandidateID, false)
		if err == nil {
			res.Dedup.MergedIntoSyncID = &target.SyncID
		}
	}

	// Task 7: wire bus.Publish(SaveEvent{...}) here — the event carries
	// sync_id, project, scope, type, topic_key, entity, importance,
	// degraded (spec 01-memory/001 §5). Not published in Task 5.
	return res, nil
}

// validateRequest enforces step 1: non-empty title/type/content, scope and
// type in vocabulary, topic_key kebab-case ≤2 levels when given.
func validateRequest(req SaveRequest) error {
	if strings.TrimSpace(req.Title) == "" {
		return errors.New("memory: title is required")
	}
	if strings.TrimSpace(req.Content) == "" {
		return errors.New("memory: content is required")
	}
	if !observationTypes[req.Type] {
		return fmt.Errorf("memory: unknown observation type %q", req.Type)
	}
	scope := strings.TrimSpace(req.Scope)
	if scope == "" {
		scope = "project" // the store's default; the empty form is legal
	}
	if !validScopes[scope] {
		return fmt.Errorf("memory: unknown scope %q", req.Scope)
	}
	if req.TopicKey != "" && !topicKeyRe.MatchString(req.TopicKey) {
		return fmt.Errorf("memory: topic_key %q must be kebab-case with at most two levels", req.TopicKey)
	}
	return nil
}

// stripPrivate removes every private-tag span (case-insensitive,
// nested-safe; an unterminated opener removes through end of string), then
// collapses the horizontal-space runs the removal leaves. Ported from v1
// mnemonic_back/internal/memory/privacy.go (StripPrivate) with the tag
// parameterized (config MemoryCfg.PrivateTag); the empty-tag form leaves
// content untouched.
func stripPrivate(s, tag string) string {
	if s == "" || tag == "" {
		return s
	}
	open := strings.ToLower(tag)
	// the closing tag: the opener's inner name wrapped in </…>
	// ("<private>" → "</private>")
	inner := strings.TrimSuffix(strings.TrimPrefix(tag, "<"), ">")
	close_ := strings.ToLower("</" + inner + ">")
	var b strings.Builder
	for {
		start := strings.Index(strings.ToLower(s), open)
		if start < 0 {
			break
		}
		// keep the text before the opener
		b.WriteString(s[:start])
		// scan for the matching closer (nested-safe)
		depth := 1
		i := start + len(open)
		closed := false
		for i < len(s) {
			nextOpen := strings.Index(strings.ToLower(s[i:]), open)
			nextClose := strings.Index(strings.ToLower(s[i:]), close_)
			if nextClose < 0 {
				break // unterminated: drop through end of string
			}
			if nextOpen >= 0 && nextOpen < nextClose {
				depth++
				i += nextOpen + len(open)
				continue
			}
			depth--
			i += nextClose + len(close_)
			if depth == 0 {
				closed = true
				break
			}
		}
		if !closed {
			break
		}
		s = s[i:]
		b.WriteRune(' ')
	}
	b.WriteString(s)
	return collapseHorizontalSpace(b.String())
}

// collapseHorizontalSpace folds space/tab runs to one space (newlines stay)
// and trims the result. Ported from v1 privacy.go.
func collapseHorizontalSpace(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	prevSpace := false
	for _, r := range s {
		if r == ' ' || r == '\t' {
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
			continue
		}
		b.WriteRune(r)
		if r != '\n' && r != '\r' {
			prevSpace = false
		}
	}
	return strings.TrimSpace(b.String())
}

// truncateContent cuts content to maxRunes runes (the brief's rune count of
// content_max_bytes). maxRunes <= 0 means unbounded.
func truncateContent(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return s
	}
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	return string(r[:maxRunes])
}

// captureOn applies the tri-state: nil → true (MCP "absent" default).
func captureOn(req SaveRequest) bool {
	return req.CapturePrompt == nil || *req.CapturePrompt
}

// stripPrivateForPrompt redacts a captured prompt (same tag, no truncation
// here — the prompt is bounded by the caller's turn length).
func (s *Service) redactPrompt(p string) string {
	return truncateContent(stripPrivate(p, s.cfg.Memory.PrivateTag), s.cfg.Memory.ContentMaxBytes)
}

// ── the transaction (steps 6–8) ─────────────────────────────────────────────

type saveTxnInput struct {
	req            SaveRequest
	content        string
	redactedPrompt string // the redacted+truncated current-turn prompt ("" = none)
	decision       llm.DedupDecision
	captureOn      bool
}

type saveTxnOutput struct {
	id            int64
	syncID        string
	revisionCount int
	replayed      bool
}

// commitSave runs steps 6–8 through the store's save-transaction gate
// (core/store.SaveTXN — INV-2: the pipeline never touches a raw handle).
func (s *Service) commitSave(ctx context.Context, st store.Store, in saveTxnInput) (saveTxnOutput, error) {
	// ponytail: the gate cast is the INV-2 boundary — the pipeline sees
	// only the gate interface; the SQLite adapter (core/store/txngate.go)
	// is the single implementation, and Task 18's Postgres adapter plugs
	// into the same seam shape.
	txn, err := beginSaveTxn(ctx, st)
	if err != nil {
		return saveTxnOutput{}, err
	}
	defer func() {
		if err != nil {
			_ = txn.Rollback()
		}
	}()

	// Step 2 (in-transaction): session policy + resolved project. The gate
	// returns its own sentinels (store.ErrSessionEnded /
	// store.ErrSessionProjectMismatch); the pipeline wraps them into the
	// memory package's sentinels so the transport layer (Task 13) maps
	// errors.Is(err, ErrSessionEnded) → 409.
	project, err := txn.CheckSession(ctx, in.req.SessionID, in.req.Project)
	if err != nil {
		return saveTxnOutput{}, wrapSessionErr(err)
	}

	scope := strings.TrimSpace(in.req.Scope)
	if scope == "" {
		scope = "project"
	}

	// Step 6a: exact-dedupe hit → noop (bump counters, no new row) — the
	// hash floor's verdict. (The LLM's update/delete verdicts are the
	// merge-into-target path, Task 8.)
	if in.decision.Verdict == llm.VerdictNoop && in.decision.CandidateID > 0 {
		if err := txn.MarkDuplicate(ctx, in.decision.CandidateID); err != nil {
			return saveTxnOutput{}, err
		}
		// the noop re-touches the row; the result carries the matched row's
		// identity (read inside the txn before commit).
		syncID, err := txn.LookupSyncID(ctx, in.decision.CandidateID)
		if err != nil {
			return saveTxnOutput{}, err
		}
		if err := txn.Commit(ctx); err != nil {
			return saveTxnOutput{}, err
		}
		return saveTxnOutput{id: in.decision.CandidateID, syncID: syncID, replayed: true}, nil
	}

	// Step 6a (topic-key upsert): a re-save with a topic_key matching an
	// existing row for the same project+scope+topic_key revises in place
	// (a hash-miss with a topic_key hit: new content for a bounded topic).
	// A hash-miss on an identical topic-key save is a revision, not a noop:
	// the content changed, so the bounded topic is updated.
	reviseID := int64(0)
	reviseSyncID := ""
	if in.req.TopicKey != "" {
		id, found, err := txn.LookupTopicKey(ctx, project, scope, in.req.TopicKey)
		if err != nil {
			return saveTxnOutput{}, err
		}
		if found {
			reviseID = id
			if reviseSyncID, err = txn.LookupSyncID(ctx, reviseID); err != nil {
				return saveTxnOutput{}, err
			}
		}
	}

	p := store.GateInsert{
		SessionID:      in.req.SessionID,
		Type:           in.req.Type,
		Title:          in.req.Title,
		Content:        in.content,
		ToolName:       in.req.ToolName,
		Project:        project,
		Scope:          scope,
		TopicKey:       in.req.TopicKey,
		OperationID:    in.req.OperationID,
		SyncID:         syncid.NewSyncID("obs"),
		NormalizedHash: storeNormalizedHash(in.content),
		ReviewAfter:    s.reviewAfterFor(in.req.Type),
		Importance:     Importance(0, 0, in.req.Type), // fresh insert: usage 0, age 0
		PromptContent:  promptFor(in),
	}

	var out saveTxnOutput
	if reviseID > 0 {
		// In-place revision: snapshot the pre-state first (the gate reads
		// the current row inside the txn). The revision's importance is
		// recomputed over the surviving row's retrieval_usage at read time
		// (Task 11's usage bumps feed it); the gate writes p.Importance as
		// the fresh-state score for now.
		rev, err := txn.ReviseObservation(ctx, reviseID, p)
		if err != nil {
			return saveTxnOutput{}, err
		}
		out = saveTxnOutput{id: reviseID, syncID: reviseSyncID, revisionCount: rev}
	} else {
		id, syncID, err := txn.InsertObservation(ctx, p)
		if err != nil {
			return saveTxnOutput{}, err
		}
		out = saveTxnOutput{id: id, syncID: syncID}
	}
	if err := txn.Commit(ctx); err != nil {
		return saveTxnOutput{}, err
	}
	return out, nil
}

// reviewAfterFor maps a type to its review clock from cfg.Memory.Review
// (spec 002-schema §2.2.2: config/preference 3 mo, discovery/pattern/
// decision 6 mo, architecture/bugfix/policy 12 mo, learning/
// session_summary/unmapped none). A 0 or missing entry means no schedule.
func (s *Service) reviewAfterFor(typ string) *string {
	months, ok := s.cfg.Memory.Review[typ]
	if !ok || months <= 0 {
		return nil
	}
	return monthsStamp(months)
}

// promptFor returns the captured prompt's content when capture is on and a
// prompt was provided ("" = store nothing). Redaction + truncation happen
// at the call site (the prompt is redacted with the same tag/limit as the
// observation content) so promptFor stays a pure selector.
func promptFor(in saveTxnInput) string {
	if !in.captureOn || in.req.CurrentPrompt == "" {
		return ""
	}
	return in.redactedPrompt
}

func monthsStamp(months int) *string {
	s := time.Now().UTC().AddDate(0, months, 0).Format("2006-01-02 15:04:05")
	return &s
}

// ── step 4.5: the replay ledger ─────────────────────────────────────────────

// ledgerFingerprint reads the stored fingerprint for an operation id via
// the store's Probe test seam (the bool-returning seam the pipeline can
// cross-package-assert — Go's export data carries unexported methods only
// within the package, so the pipeline can't see the store's unexported
// methods; the Probe seam is the narrow cross-package read surface).
// notFound:true means the ledger has no row. A failed read (other than
// ErrNoRows) is NOT evidence of absence (spec 004-dedup §6, the hash
// floor's posture): it errors — the ledger is the idempotency guard.
func (s *Service) ledgerFingerprint(st store.Store, operationID, want string) (matched, found bool, err error) {
	p, ok := st.(interface {
		Probe(query string, args ...any) (bool, error)
	})
	if !ok {
		return false, false, fmt.Errorf("memory: store %T does not expose the Probe seam the replay ledger reads through", st)
	}
	// ponytail: the fingerprint is compared by equality probe (the seam
	// returns bool only, so the stored value is recovered by asking
	// "does my computed value match?"). ErrNoRows = no ledger row (first
	// save under this id). Upgrade path: a SaveTXN gate method when Task
	// 18's Postgres adapter needs the read transactional.
	matched, err = p.Probe(
		`SELECT fingerprint = ? FROM observation_save_operations WHERE operation_id = ?`,
		want, operationID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	return matched, true, nil
}

// ledgerObservationID reads the ledger row's observation_id for an
// operation id (0 = no row OR the ON DELETE SET NULL tombstone — the
// caller disambiguates via ledgerFingerprint first).
func (s *Service) ledgerObservationID(st store.Store, operationID string) (int64, error) {
	p, ok := st.(interface {
		ProbeInt(query string, args ...any) (int, error)
	})
	if !ok {
		return 0, fmt.Errorf("memory: store %T does not expose the ProbeInt seam the replay ledger reads through", st)
	}
	// ponytail: ifnull(observation_id, 0) maps the tombstone's NULL to 0,
	// which the caller treats as "not live." This assumes rowids start at
	// 1 (SQLite/Postgres default); a Task 18 adapter that starts rowids at
	// 0 must re-check this assumption (a rowid-0 observation would be
	// misread as a tombstone).
	n, err := p.ProbeInt(
		`SELECT ifnull(observation_id, 0) FROM observation_save_operations WHERE operation_id = ?`,
		operationID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return int64(n), err
}

// replayLedger runs the step 4.5 check (spec 01-memory/001 §4) before
// dedup. Two read-only Probe calls (fingerprint equality, then
// observation_id) over the ledger table; branches on the row's state:
//
//	no row                             → done:false, proceed to dedup
//	(the ledger row is inserted inside the commit transaction by the
//	gate, step 8).
//	row + different fingerprint        → ErrOperationConflict (409)
//	row + same fingerprint + NULL
//	observation_id (the ON DELETE SET
//	NULL tombstone)                    → ErrOperationExpired (410)
//	row + same fingerprint + live
//	observation                        → the committed result
//	(Replayed:true), returned early — dedup/upsert/journal run exactly
//	once (the whole point: a plugin retrying a flaky save never
//	double-writes).
//
// The check runs BEFORE session-ownership resolution: the fingerprint is
// computed over the request's redacted+truncated content (spec §4), not
// the resolved project, so a retry of the same request fingerprints
// identically whether or not its session row is present.
// ponytail: the read goes through the store's Probe/ProbeInt test seams
// (the narrow cross-package read surface — Go's export data carries
// unexported methods only within the package, so the pipeline can't
// assert against a store-internal ledgerLookup) rather than a new SaveTXN
// gate method; the upgrade path is a gate method when Task 18's Postgres
// adapter needs the ledger read transactional inside the save txn.
func (s *Service) replayLedger(ctx context.Context, st store.Store, req SaveRequest, content string) (SaveResult, bool, error) {
	fingerprint := s.opFingerprintFor(req, content)
	matched, found, err := s.ledgerFingerprint(st, req.OperationID, fingerprint)
	if err != nil {
		return SaveResult{}, false, fmt.Errorf("memory: replay ledger lookup: %w", err)
	}
	if !found {
		return SaveResult{}, false, nil // no row: first save under this id
	}
	if !matched {
		// row exists but different fingerprint → 409.
		return SaveResult{}, true, ErrOperationConflict
	}
	// fingerprint match: read the observation_id (NULL → target was
	// deleted → 410; non-NULL → the committed row).
	obsID, err := s.ledgerObservationID(st, req.OperationID)
	if err != nil {
		return SaveResult{}, false, fmt.Errorf("memory: replay ledger lookup: %w", err)
	}
	if obsID == 0 {
		return SaveResult{}, true, ErrOperationExpired
	}
	// committed result: read the row's sync_id (INV-6 identity). A row
	// that vanished between the ledger lookup and this read (not found)
	// is the 410 path; any other read error is propagated (it is not
	// evidence the target was deleted).
	o, err := st.GetObservation(ctx, obsID, false)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return SaveResult{}, true, ErrOperationExpired
		}
		return SaveResult{}, false, fmt.Errorf("memory: replay ledger observation read: %w", err)
	}
	return SaveResult{ID: o.ID, SyncID: o.SyncID, Replayed: true}, true, nil
}

// opFingerprintFor computes the replay fingerprint for a request (spec
// 01-memory/001 Â§4): "v1:" + hex(sha256(lengthPrefixed(normalizedFields)))
// over the dedup-identity fields (Type, Title, Project, Scope, TopicKey) and
// the redacted+truncated content. The length prefix (big-endian uint64
// before each field) prevents the "ab|c" vs "a|bc" boundary collision a
// plain join would have. Computed before session-ownership resolution, so a
// retry of the same request fingerprints identically whether or not its
// session row is present. This matches the gate's ledger insert
// (txngate.go: storeOpFingerprint), so the pipeline's fingerprint and the
// stored fingerprint agree.
func (s *Service) opFingerprintFor(req SaveRequest, content string) string {
	scope := strings.TrimSpace(req.Scope)
	if scope == "" {
		scope = "project"
	}
	return "v1:" + opFingerprint(req.Type, strings.TrimSpace(req.Title),
		strings.TrimSpace(req.Project), scope, strings.TrimSpace(req.TopicKey), content)
}

// opFingerprint hashes a field list: each field is length-prefixed (a
// big-endian uint64) before its bytes — the length prefix prevents the
// "ab|c" vs "a|bc" boundary collision a plain join would have — then the
// whole stream is SHA-256 hex. (Shared with the store's storeOpFingerprint
// in txngate.go, which must stay in lockstep; the two packages cannot share
// the function without inverting the memory → core/store dependency
// direction.)
func opFingerprint(fields ...string) string {
	var b bytes.Buffer
	for _, f := range fields {
		var l uint64 = uint64(len(f))
		_ = binary.Write(&b, binary.BigEndian, l)
		b.WriteString(f)
	}
	h := sha256.Sum256(b.Bytes())
	return hex.EncodeToString(h[:])
}
