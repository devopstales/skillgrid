package memory

import (
	"context"
	"errors"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// ownerFixture opens ONE store (a single physical bucket) and returns the
// service plus two distinct owner identities sharing that bucket. Owner
// isolation is modeled by the `owner` column: a "second owner" reading the
// same store sees only observations whose owner matches theirs (or that they
// are granted). This is the single-operator mapping for 013 step 01.
type ownerFixture struct {
	st       *store.Store
	svc      *Service
	ownerA   string
	ownerB   string
	sessionID string
}

func newOwnerFixture(t *testing.T, project string) *ownerFixture {
	t.Helper()
	dataDir := t.TempDir()
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	svc := New(st, project)
	if _, err := st.DB.Exec(`
		INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES ('s1', ?, '/tmp', '2026-01-01T00:00:00Z', 'active')`, project); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	return &ownerFixture{
		st:        st,
		svc:       svc,
		ownerA:    "ownerA",
		ownerB:    "ownerB",
		sessionID: "s1",
	}
}

// TestGovernanceTools covers @step-01 RED "Mnemonic tool surface":
// mem_share / mem_governance are callable (registered at the MCP layer),
// a new observation is private-by-default with an owner, mem_update appends a
// recoverable version (latest = read path, revision_count advances), and bad
// governance args are rejected with a clear error leaving visibility intact.
func TestGovernanceTools(t *testing.T) {
	fx := newOwnerFixture(t, "gov")
	ctx := context.Background()

	// 1. New observation is private-by-default + owned by the creating agent.
	id, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessionID,
		Type:      "decision",
		Title:     "Governing asset",
		Content:   "v1 body",
		Owner:     fx.ownerA,
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	obs, err := fx.svc.Get(ctx, id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if obs.Visibility != "private" {
		t.Errorf("new observation must be private by default, got %q", obs.Visibility)
	}
	if obs.Owner != fx.ownerA {
		t.Errorf("owner must be the creating identity, got %q", obs.Owner)
	}

	// 2. mem_share is the explicit widen (bad target rejected, visibility
	//    unchanged).
	if err := fx.svc.Share(ctx, id, ShareInput{Visibility: "galaxy"}); err == nil {
		t.Errorf("mem_share to unknown target should be rejected")
	}
	still, _ := fx.svc.Get(ctx, id)
	if still.Visibility != "private" {
		t.Errorf("bad mem_share must leave visibility unchanged (still private), got %q", still.Visibility)
	}

	// 3. mem_share to team widens.
	if err := fx.svc.Share(ctx, id, ShareInput{Visibility: "team"}); err != nil {
		t.Fatalf("mem_share team: %v", err)
	}
	shared, _ := fx.svc.Get(ctx, id)
	if shared.Visibility != "team" {
		t.Errorf("mem_share team should set visibility team, got %q", shared.Visibility)
	}

	// 4. mem_update appends a recoverable version (latest = read path).
	if err := fx.svc.Update(ctx, id, UpdateInput{Content: "v2 body"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, err := fx.svc.Get(ctx, id)
	if err != nil {
		t.Fatalf("get after update: %v", err)
	}
	if got.Content != "v2 body" {
		t.Errorf("latest version must be the read path, got %q", got.Content)
	}
	if got.RevisionCount != 1 {
		t.Errorf("revision_count must advance to 1, got %d", got.RevisionCount)
	}

	// 5. mem_governance returns owner/version history/status/usage/visibility.
	gov, err := fx.svc.Governance(ctx, id)
	if err != nil {
		t.Fatalf("governance: %v", err)
	}
	if gov.Owner != fx.ownerA {
		t.Errorf("governance owner = %q, want %q", gov.Owner, fx.ownerA)
	}
	if gov.Visibility != "team" {
		t.Errorf("governance visibility = %q, want team", gov.Visibility)
	}
	if gov.Status != "active" {
		t.Errorf("governance status = %q, want active", gov.Status)
	}
	if len(gov.Versions) != 1 || gov.Versions[0].Content != "v1 body" {
		t.Errorf("governance version history must recover prior content, got %+v", gov.Versions)
	}
	if got.RevisionCount != gov.RevisionCount {
		t.Errorf("governance revision_count %d != observation %d", gov.RevisionCount, got.RevisionCount)
	}

	// 6. Explicit status set (never inferred): superseded.
	if err := fx.svc.SetStatus(ctx, id, "superseded"); err != nil {
		t.Fatalf("set status: %v", err)
	}
	if got, _ := fx.svc.Get(ctx, id); got.Status != "superseded" {
		t.Errorf("status must be explicitly set, got %q", got.Status)
	}

	// 7. Bad mem_governance args: unknown id → clear error.
	if _, err := fx.svc.Governance(ctx, 999999); err == nil {
		t.Errorf("governance for unknown id should error")
	}
}

// TestVisibility covers @step-01 RED "Data leak / visibility": a private
// observation is invisible to a second owner until shared; a restricted
// observation is enforced by ACL (granted read, non-granted absent); a
// restricted observation with no grants is owner-only (surfaced, not an
// error); a private observation is absent from the admin cross-owner list.
func TestVisibility(t *testing.T) {
	fx := newOwnerFixture(t, "vis")
	ctx := context.Background()

	// --- private is invisible to the second owner until share ---
	idA, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessionID, Type: "decision",
		Title: "widget alpha", Content: "widget alpha", Owner: fx.ownerA,
	})
	if err != nil {
		t.Fatalf("save A: %v", err)
	}

	// Second owner's search (modelled: owner-B scoped read) must not see it
	// while private. (Single-token FTS term, matching the style of
	// TestSaveAndSearch.)
	second, err := fx.svc.SearchOwner(ctx, fx.ownerB, "widget", "any", 10)
	if err != nil {
		t.Fatalf("search as B: %v", err)
	}
	if containsID(second, idA) {
		t.Errorf("private observation leaked to a second owner before share")
	}
	// First owner still sees its own private observation.
	own, err := fx.svc.SearchOwner(ctx, fx.ownerA, "widget", "any", 10)
	if err != nil {
		t.Fatalf("search as A: %v", err)
	}
	if !containsID(own, idA) {
		t.Errorf("owner A must still see its own private observation")
	}

	// After an explicit share to team, the second owner's search returns it.
	if err := fx.svc.Share(ctx, idA, ShareInput{Visibility: "team"}); err != nil {
		t.Fatalf("share: %v", err)
	}
	second2, err := fx.svc.SearchOwner(ctx, fx.ownerB, "widget", "any", 10)
	if err != nil {
		t.Fatalf("search as B after share: %v", err)
	}
	if !containsID(second2, idA) {
		t.Errorf("after share the second owner's search must return the observation")
	}

	// --- restricted ACL enforcement ---
	idR, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessionID, Type: "config",
		Title: "zebra restricted", Content: "zebra restricted", Owner: fx.ownerA,
	})
	if err != nil {
		t.Fatalf("save R: %v", err)
	}
	// Grant agent-x only.
	if err := fx.svc.Share(ctx, idR, ShareInput{Visibility: "restricted", Grants: []string{"agent-x"}}); err != nil {
		t.Fatalf("share restricted: %v", err)
	}
	granted, err := fx.svc.ReadAs(ctx, fx.ownerB, idR, "agent-x")
	if err != nil {
		t.Fatalf("granted agent read: %v", err)
	}
	if !containsID([]Observation{granted}, idR) {
		t.Errorf("granted agent must be able to read the restricted observation")
	}
	if _, err := fx.svc.ReadAs(ctx, fx.ownerB, idR, "agent-y"); !errors.Is(err, ErrNotFoundForReader) {
		t.Errorf("non-granted agent must see the restricted observation as absent, got %v", err)
	}

	// --- restricted with no grants is owner-only (surfaced, not an error) ---
	idNR, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessionID, Type: "config",
		Title: "quartz restricted", Content: "quartz restricted", Owner: fx.ownerA,
	})
	if err != nil {
		t.Fatalf("save NR: %v", err)
	}
	if err := fx.svc.Share(ctx, idNR, ShareInput{Visibility: "restricted", Grants: nil}); err != nil {
		t.Fatalf("share no-grant restricted: %v", err)
	}
	govNR, err := fx.svc.Governance(ctx, idNR)
	if err != nil {
		t.Fatalf("governance NR: %v", err)
	}
	if !govNR.RestrictedNoGrants {
		t.Errorf("restricted with no grants must surface RestrictedNoGrants (owner-only), not raise an error")
	}
	// Owner can read; another owner sees it as absent.
	if _, err := fx.svc.ReadAs(ctx, fx.ownerA, idNR, fx.ownerA); err != nil {
		t.Errorf("owner must read its no-grant restricted observation: %v", err)
	}
	if _, err := fx.svc.ReadAs(ctx, fx.ownerB, idNR, fx.ownerB); !errors.Is(err, ErrNotFoundForReader) {
		t.Errorf("another owner must see no-grant restricted as absent, got %v", err)
	}

	// --- private absent from admin cross-owner list ---
	// An admin is an owner who is NOT the creator of the private observation.
	// Use a third owner (the admin) — ownerA's private obs is absent from the
	// admin's cross-owner list, but the admin's OWN obs is still listed.
	adminOwner := "adminC"
	idPriv, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessionID, Type: "decision",
		Title: "privnote asset", Content: "privnote asset", Owner: fx.ownerA,
	})
	if err != nil {
		t.Fatalf("save private for admin test: %v", err)
	}
	idAdminOwn, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessionID, Type: "decision",
		Title: "adminown asset", Content: "adminown asset", Owner: adminOwner,
	})
	if err != nil {
		t.Fatalf("save admin own: %v", err)
	}
	adminList, err := fx.svc.AdminCrossOwnerList(ctx, adminOwner)
	if err != nil {
		t.Fatalf("admin list: %v", err)
	}
	if containsID(adminList, idPriv) {
		t.Errorf("a private observation must be absent from the admin cross-owner list")
	}
	if !containsID(adminList, idAdminOwn) {
		t.Errorf("the admin's OWN observation must be listed in the cross-owner list")
	}
}

// TestLegacyEmptyOwnerConsistency covers the review finding: a legacy
// (pre-017) row whose owner column is empty must read identically via search
// (visibilityFilter) and via ReadAs (canRead). Both treat an empty owner as
// the default (private, owner "legacy") so no live reader can see it — and
// ReadAs does NOT surface ErrVisibilityNotSet (the two read paths agree).
func TestLegacyEmptyOwnerConsistency(t *testing.T) {
	fx := newOwnerFixture(t, "legacyempty")
	ctx := context.Background()

	// Simulate a legacy (pre-017) observation: the row exists with an empty
	// owner and the default visibility. The FTS trigger indexes it on insert.
	var legacyID int64
	if err := fx.st.DB.QueryRow(`
		INSERT INTO observations (session_id, type, title, content, project, scope, normalized_hash, created_at, updated_at, source)
		VALUES ('s1', 'decision', 'legacyempty note', 'legacyempty note body', 'legacyempty', 'project', 'h-legacy',
		        '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z', 'agent')
		RETURNING id`).Scan(&legacyID); err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}

	// Both read paths must agree: the row is private (COALESCE default) and its
	// owner falls back to the sentinel, so NO live reader (owner A or B) can see
	// it via search.
	for _, reader := range []string{fx.ownerA, fx.ownerB} {
		hits, err := fx.svc.SearchOwner(ctx, reader, "legacyempty", "any", 10)
		if err != nil {
			t.Fatalf("search as %s: %v", reader, err)
		}
		if containsID(hits, legacyID) {
			t.Errorf("legacy empty-owner row leaked to reader %s via search", reader)
		}
	}

	// ReadAs must agree with search: gated as absent (not-found), NOT
	// ErrVisibilityNotSet, for every live reader.
	for _, reader := range []string{fx.ownerA, fx.ownerB} {
		if _, err := fx.svc.ReadAs(ctx, reader, legacyID, reader); err != nil {
			if errors.Is(err, ErrVisibilityNotSet) {
				t.Errorf("ReadAs must not surface ErrVisibilityNotSet for a legacy empty-owner row (reader %s)", reader)
			}
			if !errors.Is(err, ErrNotFoundForReader) {
				t.Errorf("ReadAs for legacy empty-owner row must be absent for reader %s, got %v", reader, err)
			}
		}
	}
}

func containsID(obs []Observation, id int64) bool {
	for _, o := range obs {
		if o.ID == id {
			return true
		}
	}
	return false
}

// TestPrivateDefault covers @step-01 (Scenario: new-observation-private-by-default-with-owner):
// a mem_save without an explicit visibility produces a `private` observation
// owned by the creating identity; a second owner's search excludes it.
func TestPrivateDefault(t *testing.T) {
	fx := newOwnerFixture(t, "priv")
	ctx := context.Background()
	id, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessionID, Type: "decision",
		Title: "private default", Content: "private default body", Owner: fx.ownerA,
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	obs, err := fx.svc.Get(ctx, id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if obs.Visibility != "private" {
		t.Errorf("new observation must be private by default, got %q", obs.Visibility)
	}
	if obs.Owner != fx.ownerA {
		t.Errorf("owner must be the creating identity, got %q", obs.Owner)
	}
	// Second owner's search excludes it.
	second, err := fx.svc.SearchOwner(ctx, fx.ownerB, "private", "any", 10)
	if err != nil {
		t.Fatalf("search as B: %v", err)
	}
	if containsID(second, id) {
		t.Errorf("private observation must be excluded from a second owner's search")
	}
}

// TestMemShare covers @step-01 (Scenario: mem-share-unknown-target-rejected):
// mem_share to an unknown target is rejected with a clear error and visibility
// is unchanged (still private); a valid share widens it.
func TestMemShare(t *testing.T) {
	fx := newOwnerFixture(t, "share")
	ctx := context.Background()
	id, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessionID, Type: "decision",
		Title: "share target", Content: "share body", Owner: fx.ownerA,
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	// Unknown target rejected; visibility unchanged.
	for _, bad := range []string{"", "galaxy", "private", "Team"} {
		if err := fx.svc.Share(ctx, id, ShareInput{Visibility: bad}); err == nil {
			t.Errorf("Share(%q) should be rejected", bad)
		}
	}
	got, _ := fx.svc.Get(ctx, id)
	if got.Visibility != "private" {
		t.Errorf("unknown-target share must leave visibility private, got %q", got.Visibility)
	}
	// A valid share widens to team.
	if err := fx.svc.Share(ctx, id, ShareInput{Visibility: "team"}); err != nil {
		t.Fatalf("valid share: %v", err)
	}
	got, _ = fx.svc.Get(ctx, id)
	if got.Visibility != "team" {
		t.Errorf("valid share should set team, got %q", got.Visibility)
	}
}

// TestUpdateVersion covers @step-01 (Scenario: mem-update-appends-recoverable-version):
// mem_update appends a version row; the prior content is recoverable via the
// governance query; the latest version is the read path; revision_count advances.
func TestUpdateVersion(t *testing.T) {
	fx := newOwnerFixture(t, "ver")
	ctx := context.Background()
	id, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessionID, Type: "decision",
		Title: "versioned", Content: "original content", Owner: fx.ownerA,
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	// First update.
	if err := fx.svc.Update(ctx, id, UpdateInput{Content: "second content"}); err != nil {
		t.Fatalf("update 1: %v", err)
	}
	got, err := fx.svc.Get(ctx, id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Content != "second content" {
		t.Errorf("latest version must be the read path, got %q", got.Content)
	}
	if got.RevisionCount != 1 {
		t.Errorf("revision_count must be 1 after one update, got %d", got.RevisionCount)
	}
	// Second update.
	if err := fx.svc.Update(ctx, id, UpdateInput{Content: "third content"}); err != nil {
		t.Fatalf("update 2: %v", err)
	}
	got, _ = fx.svc.Get(ctx, id)
	if got.Content != "third content" {
		t.Errorf("latest version after 2nd update, got %q", got.Content)
	}
	if got.RevisionCount != 2 {
		t.Errorf("revision_count must be 2, got %d", got.RevisionCount)
	}
	// Governance recovers the full version history (prior content).
	gov, err := fx.svc.Governance(ctx, id)
	if err != nil {
		t.Fatalf("governance: %v", err)
	}
	if len(gov.Versions) != 2 {
		t.Fatalf("expected 2 version rows, got %d", len(gov.Versions))
	}
	// Versions are newest-first; the oldest must be the original content.
	foundOriginal := false
	for _, v := range gov.Versions {
		if v.Content == "original content" {
			foundOriginal = true
		}
	}
	if !foundOriginal {
		t.Errorf("prior content (original) must be recoverable via governance, got %+v", gov.Versions)
	}
}

// TestStatusExplicit covers @step-01 (Scenario: superseded-status-set-explicitly):
// superseded/archived status is set explicitly via SetStatus, never inferred.
func TestStatusExplicit(t *testing.T) {
	fx := newOwnerFixture(t, "status")
	ctx := context.Background()
	id, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessionID, Type: "decision",
		Title: "status asset", Content: "status body", Owner: fx.ownerA,
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	// Default is active.
	got, _ := fx.svc.Get(ctx, id)
	if got.Status != "active" {
		t.Errorf("default status must be active, got %q", got.Status)
	}
	// Explicitly mark superseded.
	if err := fx.svc.SetStatus(ctx, id, "superseded"); err != nil {
		t.Fatalf("set superseded: %v", err)
	}
	got, _ = fx.svc.Get(ctx, id)
	if got.Status != "superseded" {
		t.Errorf("status must be superseded after explicit set, got %q", got.Status)
	}
	// Explicitly mark archived.
	if err := fx.svc.SetStatus(ctx, id, "archived"); err != nil {
		t.Fatalf("set archived: %v", err)
	}
	got, _ = fx.svc.Get(ctx, id)
	if got.Status != "archived" {
		t.Errorf("status must be archived after explicit set, got %q", got.Status)
	}
	// Invalid status rejected.
	if err := fx.svc.SetStatus(ctx, id, "weird"); err == nil {
		t.Errorf("SetStatus(weird) should be rejected")
	}
}

// TestUsageCount covers @step-01 (Scenario: retrieval-usage-count-increments-on-search):
// a search hit increments retrieval_usage (distinct from duplicate_count).
func TestUsageCount(t *testing.T) {
	fx := newOwnerFixture(t, "usage")
	ctx := context.Background()
	id, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessionID, Type: "decision",
		Title: "usage target", Content: "usage target body", Owner: fx.ownerA,
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	// Initial state.
	got, _ := fx.svc.Get(ctx, id)
	if got.RetrievalUsage != 0 {
		t.Errorf("initial retrieval_usage must be 0, got %d", got.RetrievalUsage)
	}
	if got.DuplicateCount != 0 {
		t.Errorf("initial duplicate_count must be 0, got %d", got.DuplicateCount)
	}
	// Two searches → usage increments to 2; duplicate_count stays 0.
	for i := 0; i < 2; i++ {
		if _, err := fx.svc.Search(ctx, "usage", "any", 10); err != nil {
			t.Fatalf("search %d: %v", i, err)
		}
	}
	got, _ = fx.svc.Get(ctx, id)
	if got.RetrievalUsage != 2 {
		t.Errorf("retrieval_usage must be 2 after 2 searches, got %d", got.RetrievalUsage)
	}
	if got.DuplicateCount != 0 {
		t.Errorf("duplicate_count must stay 0 (distinct from retrieval_usage), got %d", got.DuplicateCount)
	}
}

// TestGovernanceQuery covers @step-01 (Scenario: mem-governance-surfaces-asset-fields):
// mem_governance returns owner, version history, status, usage, visibility.
func TestGovernanceQuery(t *testing.T) {
	fx := newOwnerFixture(t, "govq")
	ctx := context.Background()
	id, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessionID, Type: "decision",
		Title: "govq asset", Content: "govq v1", Owner: fx.ownerA,
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := fx.svc.Update(ctx, id, UpdateInput{Content: "govq v2"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if err := fx.svc.Share(ctx, id, ShareInput{Visibility: "team"}); err != nil {
		t.Fatalf("share: %v", err)
	}
	if err := fx.svc.SetStatus(ctx, id, "superseded"); err != nil {
		t.Fatalf("status: %v", err)
	}
	// A search to bump usage.
	if _, err := fx.svc.Search(ctx, "govq", "any", 10); err != nil {
		t.Fatalf("search: %v", err)
	}
	gov, err := fx.svc.Governance(ctx, id)
	if err != nil {
		t.Fatalf("governance: %v", err)
	}
	if gov.Owner != fx.ownerA {
		t.Errorf("governance owner = %q, want %q", gov.Owner, fx.ownerA)
	}
	if gov.Visibility != "team" {
		t.Errorf("governance visibility = %q, want team", gov.Visibility)
	}
	if gov.Status != "superseded" {
		t.Errorf("governance status = %q, want superseded", gov.Status)
	}
	if gov.RetrievalUsage < 1 {
		t.Errorf("governance retrieval_usage must be >= 1, got %d", gov.RetrievalUsage)
	}
	if gov.RevisionCount != 1 {
		t.Errorf("governance revision_count = %d, want 1", gov.RevisionCount)
	}
	if len(gov.Versions) != 1 || gov.Versions[0].Content != "govq v1" {
		t.Errorf("governance version history must recover prior content, got %+v", gov.Versions)
	}
}

// TestBadGovernanceArgs covers @step-01 (Scenario: bad-governance-args-rejected):
// bad/missing args are rejected with a clear validation error, no governance
// data is invented, and visibility is unchanged.
func TestBadGovernanceArgs(t *testing.T) {
	fx := newOwnerFixture(t, "badargs")
	ctx := context.Background()
	id, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessionID, Type: "decision",
		Title: "badargs asset", Content: "body", Owner: fx.ownerA,
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	// mem_share: missing/invalid visibility → clear error, visibility intact.
	for _, bad := range []string{"", "galaxy", "PRIVATE"} {
		if err := fx.svc.Share(ctx, id, ShareInput{Visibility: bad}); err == nil {
			t.Errorf("Share(%q) should be rejected", bad)
		}
	}
	// restricted share with an empty grantee → rejected.
	if err := fx.svc.Share(ctx, id, ShareInput{Visibility: "restricted", Grants: []string{"  "}}); err == nil {
		t.Errorf("Share restricted with empty grantee should be rejected")
	}
	// SetStatus: invalid status → rejected.
	if err := fx.svc.SetStatus(ctx, id, "weird"); err == nil {
		t.Errorf("SetStatus(weird) should be rejected")
	}
	// Visibility unchanged through all the rejected mutations.
	got, _ := fx.svc.Get(ctx, id)
	if got.Visibility != "private" || got.Status != "active" {
		t.Errorf("bad args must leave governance unchanged, got visibility=%q status=%q", got.Visibility, got.Status)
	}
}
