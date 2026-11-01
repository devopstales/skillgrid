package memory

import (
	"context"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/config"
)

var ctx = context.Background()

// [save-persists-with-sync-id] a save persists with provenance + project and
// returns {id, obs- sync id}; [save-result-shape] the LLM-absent result is
// degraded with the hash reason.
func TestSavePersistsAndReturnsShape(t *testing.T) {
	svc := newTestService(t) // real SQLite store + nil LLM (degraded)
	r, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision", Title: "T",
		Content: "C", Project: "p", Scope: "project"})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if r.ID == 0 || !strings.HasPrefix(r.SyncID, "obs-") {
		t.Fatalf("bad shape %+v", r)
	}
	if !r.Degraded {
		t.Fatalf("no LLM → degraded must be true")
	}
	if r.Dedup.Reason != "hash" {
		t.Fatalf("floor reason must be hash, got %q", r.Dedup.Reason)
	}
	if r.Dedup.Verdict != "add" {
		t.Fatalf("first save verdict must be add, got %q", r.Dedup.Verdict)
	}
	got, err := svc.st.GetObservation(ctx, r.ID, false)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Project == nil || *got.Project != "p" || got.Provenance.SessionID != "s" {
		t.Fatalf("provenance/project not set: %+v", got)
	}
	// the row must be retrievable by its sync id as well (BDD: "by both its
	// id and its sync id").
	if findIDBySyncID(t, svc.st, r.SyncID) != r.ID {
		t.Fatalf("sync id %s did not resolve to id %d", r.SyncID, r.ID)
	}
}

// [save-redact-truncate] private spans are stripped and content truncated
// before storage AND before hashing — a save differing only inside a private
// span is a hash-equal noop/replay.
func TestSaveRedactsPrivateAndTruncates(t *testing.T) {
	svc := newTestServiceWith(t, config.MemoryCfg{PrivateTag: "<private>", ContentMaxBytes: 16})
	r, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "learning", Title: "T",
		Content: "a <private>secret</private> bbbbbbbbbbbbbbbbb", Project: "p", Scope: "project"})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := svc.st.GetObservation(ctx, r.ID, false)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if strings.Contains(got.Content, "secret") {
		t.Fatalf("private tag not stripped: %q", got.Content)
	}
	if strings.Contains(got.Content, "<private>") {
		t.Fatalf("private tag markers not removed: %q", got.Content)
	}
	if n := utf8.RuneCountInString(got.Content); n > 16 {
		t.Fatalf("not truncated to content_max_bytes: %d runes (%q)", n, got.Content)
	}
	// the hash must be over the REDACTED+TRUNCATED content (spec 004-dedup §7):
	// differing only inside <private> must hash-equal → noop on the same row.
	r2, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "learning", Title: "T",
		Content: "a <private>other</private> bbbbbbbbbbbbbbbbb", Project: "p", Scope: "project"})
	if err != nil {
		t.Fatalf("save 2: %v", err)
	}
	if r2.ID != r.ID || r2.SyncID != r.SyncID || !r2.Replayed {
		t.Fatalf("differing only in <private> must hash-equal → noop/replay: %+v vs %+v", r2, r)
	}
}

// [hash-dedupe-noop] an exact re-save is a first-class noop: same id, no new
// row, duplicate_count + last_seen_at bumped, degraded with the hash reason.
func TestHashFloorNoopBumpsCounters(t *testing.T) {
	svc := newTestService(t)
	r1, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "discovery", Title: "D", Content: "exact same", Project: "p", Scope: "project"})
	if err != nil {
		t.Fatalf("save 1: %v", err)
	}
	r2, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "discovery", Title: "D", Content: "exact same", Project: "p", Scope: "project"})
	if err != nil {
		t.Fatalf("save 2: %v", err)
	}
	if r2.Dedup.Verdict != "noop" || r2.ID != r1.ID || r2.SyncID != r1.SyncID {
		t.Fatalf("exact dupe must noop on same id: %+v", r2)
	}
	if !r2.Degraded || r2.Dedup.Reason != "hash" {
		t.Fatalf("noop must be flagged degraded with hash reason: %+v", r2)
	}
	got, err := svc.st.GetObservation(ctx, r1.ID, false)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.DuplicateCount != 1 || got.LastSeenAt == nil {
		t.Fatalf("counters not bumped: %+v", got)
	}
	if liveObsCount(t, svc.st) != 1 {
		t.Fatalf("noop must not write a new row; live count = %d", liveObsCount(t, svc.st))
	}
}

// [save-validation] empty title, out-of-vocabulary type, out-of-vocabulary
// scope, and >2-level topic keys are each refused with an error and nothing
// is stored.
func TestSaveValidation(t *testing.T) {
	svc := newTestService(t)
	for _, bad := range []SaveRequest{
		{SessionID: "s", Type: "decision", Title: "", Content: "c", Project: "p", Scope: "project"},     // empty title
		{SessionID: "s", Type: "not-a-type", Title: "t", Content: "c", Project: "p", Scope: "project"}, // bad type
		{SessionID: "s", Type: "decision", Title: "t", Content: "c", Project: "p", Scope: "mars"},      // bad scope
		{SessionID: "s", Type: "decision", Title: "t", Content: "c", Project: "p", Scope: "project", TopicKey: "a/b/c"}, // >2 levels
	} {
		if _, err := svc.Save(ctx, bad); err == nil {
			t.Fatalf("must reject %+v", bad)
		}
	}
	if n := liveObsCount(t, svc.st); n != 0 {
		t.Fatalf("validation failures must store nothing, stored %d", n)
	}
}

// [save-validation] triangulation — the four valid scopes are all accepted
// (different inputs, same validation behavior: in-vocabulary passes).
func TestSaveValidationAcceptsVocabulary(t *testing.T) {
	svc := newTestService(t)
	for _, scope := range []string{"project", "personal", "global"} {
		req := SaveRequest{SessionID: "s", Type: "pattern", Title: "t-" + scope, Content: "c-" + scope, Project: "p", Scope: scope}
		if _, err := svc.Save(ctx, req); err != nil {
			t.Fatalf("scope %s must be accepted: %v", scope, err)
		}
	}
	if n := liveObsCount(t, svc.st); n != 3 {
		t.Fatalf("expected 3 accepted saves, stored %d", n)
	}
}

// [save-redact-truncate] triangulation — a multi-line, multi-span secret is
// fully stripped (different inputs than the single-span test above).
func TestSaveRedactsMultiLineSecrets(t *testing.T) {
	svc := newTestService(t) // default 32000-rune limit: no truncation interference
	secret := "AKIA-MULTI-LINE"
	other := "another-secret"
	// two spans, one spanning a newline (a multi-line secret): both must be
	// stripped, and the surviving text must not leak the span boundaries.
	content := "alpha <private>" + secret + "</private> beta\n<private>" + other + "</private>"
	r, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "learning", Title: "T",
		Content: content, Project: "p", Scope: "project"})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := svc.st.GetObservation(ctx, r.ID, false)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	for _, s := range []string{secret, other, "<private>", "</private>"} {
		if strings.Contains(got.Content, s) {
			t.Fatalf("secret not fully stripped (%q left in %q)", s, got.Content)
		}
	}
	if !strings.Contains(got.Content, "alpha") || !strings.Contains(got.Content, "beta") {
		t.Fatalf("surrounding text lost: %q", got.Content)
	}
	// a save differing only inside the spans must hash-equal the first
	r2, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "learning", Title: "T",
		Content: "alpha <private>DIFFERENT</private> beta\n<private>OTHER</private>", Project: "p", Scope: "project"})
	if err != nil {
		t.Fatalf("save 2: %v", err)
	}
	if r2.ID != r.ID || !r2.Replayed {
		t.Fatalf("multi-line span difference must hash-equal → noop: %+v vs %+v", r2, r)
	}
}

// Step 6a — a topic_key set on a re-save with the same project+scope+topic_key
// revises the row in place (revision_count++) and snapshots the pre-state into
// observation_versions.
func TestSaveTopicKeyUpsertSnapshotsVersion(t *testing.T) {
	svc := newTestService(t)
	r1, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision", Title: "Dec",
		Content: "first version", Project: "p", Scope: "project", TopicKey: "auth/model"})
	if err != nil {
		t.Fatalf("save 1: %v", err)
	}
	r2, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision", Title: "Dec",
		Content: "second version", Project: "p", Scope: "project", TopicKey: "auth/model"})
	if err != nil {
		t.Fatalf("save 2: %v", err)
	}
	// same identity: in-place revision, not a new row
	if r2.ID != r1.ID || r2.SyncID != r1.SyncID {
		t.Fatalf("topic-key re-save must revise in place: %+v vs %+v", r2, r1)
	}
	if r2.RevisionCount != 1 {
		t.Fatalf("revision_count must be 1, got %d", r2.RevisionCount)
	}
	got, err := svc.st.GetObservation(ctx, r1.ID, false)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Content != "second version" {
		t.Fatalf("in-place revision lost: %q", got.Content)
	}
	if got.RevisionCount != 1 || got.TopicKey == nil || *got.TopicKey != "auth/model" {
		t.Fatalf("revision/topic state wrong: %+v", got)
	}
	// the pre-state snapshot is version 1, holding the FIRST content
	n, err := probeStore(t, svc.st).ProbeInt(
		`SELECT COUNT(*) FROM observation_versions WHERE observation_id = ? AND version = 1 AND content = 'first version'`,
		r1.ID)
	if err != nil {
		t.Fatalf("version probe: %v", err)
	}
	if n != 1 {
		t.Fatalf("pre-state snapshot missing or wrong: %d matching version rows", n)
	}
	if liveObsCount(t, svc.st) != 1 {
		t.Fatalf("in-place revision must not add a row; live count = %d", liveObsCount(t, svc.st))
	}
}

// Step 6b — review_after comes from the type→months config map.
func TestSaveSetsReviewAfterFromConfig(t *testing.T) {
	svc := newTestServiceWith(t, config.MemoryCfg{
		Review:          map[string]int{"decision": 6, "learning": 3},
		ContentMaxBytes: 32000, PrivateTag: "<private>",
	})
	rd, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision", Title: "D", Content: "c", Project: "p", Scope: "project"})
	if err != nil {
		t.Fatalf("save decision: %v", err)
	}
	got, err := svc.st.GetObservation(ctx, rd.ID, false)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ReviewAfter == nil {
		t.Fatalf("scheduled type must set review_after; row: %+v", got)
	}
	rl, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "learning", Title: "L", Content: "c", Project: "p", Scope: "project"})
	if err != nil {
		t.Fatalf("save learning: %v", err)
	}
	gotL, err := svc.st.GetObservation(ctx, rl.ID, false)
	if err != nil {
		t.Fatalf("get learning: %v", err)
	}
	// 3-month review_after lands strictly before 6-months from now
	raL, err := timeParseUTC(*gotL.ReviewAfter)
	if err != nil {
		t.Fatalf("parse review_after: %v", err)
	}
	raD, err := timeParseUTC(*got.ReviewAfter)
	if err != nil {
		t.Fatalf("parse review_after: %v", err)
	}
	if !raL.Before(raD) {
		t.Fatalf("3-month review_after must precede 6-month: learning=%s decision=%s", *gotL.ReviewAfter, *got.ReviewAfter)
	}
}

// Step 7 — sync is config-gated: unenrolled project journals nothing, even
// with sync enabled. (Enrollment journaling is asserted in Task 17.)
func TestSaveUnenrolledJournalsNothing(t *testing.T) {
	svc := newTestServiceWith(t, config.Default().Memory)
	svc.cfg = syncCfg(true)
	_, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision", Title: "T", Content: "C", Project: "p", Scope: "project"})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	n, err := probeStore(t, svc.st).ProbeInt(`SELECT COUNT(*) FROM sync_mutations WHERE entity = 'observation'`)
	if err != nil {
		t.Fatalf("mutation probe: %v", err)
	}
	if n != 0 {
		t.Fatalf("unenrolled project must journal nothing, journaled %d", n)
	}
}

// [save-into-ended-session] saving into an ended session is refused with the
// ended-session sentinel (Task 7 owns the lifecycle surface; the save-side
// check is fixed in this task per the brief).
func TestSaveIntoEndedSessionRefused(t *testing.T) {
	svc := newTestService(t)
	seedObservation(t, svc.st, "p", "decision", "seed", "c", "") // auto-registers session s
	markSessionEnded(t, svc.st, "s")
	_, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision", Title: "T", Content: "C", Project: "p", Scope: "project"})
	if !errors.Is(err, ErrSessionEnded) {
		t.Fatalf("ended session must return ErrSessionEnded (409), got %v", err)
	}
}

// [save-into-ended-session] triangulation — a project_owned session that
// mismatches the resolved project is refused with the mismatch sentinel.
func TestSaveIntoMismatchedOwnedSessionRefused(t *testing.T) {
	svc := newTestService(t)
	seedObservation(t, svc.st, "p", "decision", "seed", "c", "") // session s → project p
	setSessionOwnedProject(t, svc.st, "s", "other-proj")
	_, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision", Title: "T", Content: "C", Project: "q", Scope: "project"})
	if !errors.Is(err, ErrSessionProjectMismatch) {
		t.Fatalf("mismatched owned session must return ErrSessionProjectMismatch (409), got %v", err)
	}
}

// The gate wraps the store's sentinels: the memory package's 409 sentinels
// are the same values the gate returns (errors.Is must work end-to-end).
var (
	_ = ErrSessionEnded
	_ = ErrSessionProjectMismatch
)

// [save-with-unknown-session] a missing session row is auto-registered (shared
// ownership) inside the save — the first save opens its session implicitly.
func TestSaveAutoRegistersMissingSession(t *testing.T) {
	svc := newTestService(t)
	_, err := svc.Save(ctx, SaveRequest{SessionID: "brand-new", Type: "decision", Title: "T", Content: "C", Project: "p", Scope: "project"})
	if err != nil {
		t.Fatalf("save with missing session must succeed (auto-register): %v", err)
	}
	ok, err := probeStore(t, svc.st).Probe(
		`SELECT COUNT(*) > 0 FROM sessions WHERE id = 'brand-new' AND ownership_mode = 'shared'`,
	)
	if err != nil {
		t.Fatalf("session probe: %v", err)
	}
	if !ok {
		t.Fatal("missing session row was not auto-registered with shared ownership")
	}
}

// ponytail: time.Parse with the store's UTC layout — a test-local seam so the
// tests never import the store's nowStamp.
func timeParseUTC(s string) (t timeT, err error) { return parseStoreTime(s) }

// Step 8 — one committed local transaction: the observation and its
// observation_versions snapshot land atomically (txn counter moves once for a
// revision save).
func TestSaveRevisionSingleTransaction(t *testing.T) {
	svc := newTestService(t)
	counts := txnCount(t, svc.st)
	_, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision", Title: "Dec", Content: "v1", Project: "p", Scope: "project", TopicKey: "auth/model"})
	if err != nil {
		t.Fatalf("save 1: %v", err)
	}
	first := *counts
	_, err = svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision", Title: "Dec", Content: "v2", Project: "p", Scope: "project", TopicKey: "auth/model"})
	if err != nil {
		t.Fatalf("save 2: %v", err)
	}
	if *counts-first != 1 {
		t.Fatalf("revision save must commit exactly one transaction, committed %d", *counts-first)
	}
}
