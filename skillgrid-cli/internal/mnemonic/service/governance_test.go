package service

import (
	"context"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// serviceGovernanceFixture opens the "gov" project store and returns the
// service plus its memory sub-service (for governance assertions). The store
// is created by Open(projectID, ".") — the same path openProject uses — so
// the schema (incl. the 017 governance migration) is applied.
func serviceGovernanceFixture(t *testing.T) (*Service, *memory.Service) {
	t.Helper()
	dataDir := t.TempDir()
	svc := New(dataDir)
	st, err := store.Open(dataDir, "gov")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	mem := memory.New(st, "gov")
	if _, err := st.DB.Exec(`
		INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES ('s1', 'gov', '/tmp', '2026-01-01T00:00:00Z', 'active')`); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	return svc, mem
}

// TestPrivateDefault covers @step-01 (Scenario: new-observation-private-by-default-with-owner):
// a mem_save produces a `private` observation owned by the creating identity;
// a second owner's search excludes it until shared.
func TestPrivateDefault(t *testing.T) {
	_, mem := serviceGovernanceFixture(t)
	ctx := context.Background()
	id, err := mem.Save(ctx, memory.SaveInput{
		SessionID: "s1", Type: "decision",
		Title: "private default", Content: "private default body", Owner: "ownerA",
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	obs, err := mem.Get(ctx, id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if obs.Visibility != "private" {
		t.Errorf("new observation must be private by default, got %q", obs.Visibility)
	}
	if obs.Owner != "ownerA" {
		t.Errorf("owner must be the creating identity, got %q", obs.Owner)
	}
	// Second owner's search excludes it (private-by-default).
	second, err := mem.SearchOwner(ctx, "ownerB", "private", "any", 10)
	if err != nil {
		t.Fatalf("search as ownerB: %v", err)
	}
	if idInList(second, id) {
		t.Errorf("private observation must be excluded from a second owner's search")
	}
	// After an explicit share to team, the second owner's search returns it.
	if err := mem.Share(ctx, id, memory.ShareInput{Visibility: "team"}); err != nil {
		t.Fatalf("share: %v", err)
	}
	second2, err := mem.SearchOwner(ctx, "ownerB", "private", "any", 10)
	if err != nil {
		t.Fatalf("search as ownerB after share: %v", err)
	}
	if !idInList(second2, id) {
		t.Errorf("after share the second owner's search must return the observation")
	}
}

// TestMemShare covers @step-01 (Scenario: mem-share-unknown-target-rejected):
// mem_share to an unknown target is rejected with a clear error; visibility
// is unchanged (still private).
func TestMemShare(t *testing.T) {
	_, mem := serviceGovernanceFixture(t)
	ctx := context.Background()
	id, err := mem.Save(ctx, memory.SaveInput{
		SessionID: "s1", Type: "decision",
		Title: "share target", Content: "share body", Owner: "ownerA",
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	for _, bad := range []string{"", "galaxy", "private"} {
		if err := mem.Share(ctx, id, memory.ShareInput{Visibility: bad}); err == nil {
			t.Errorf("Share(%q) should be rejected", bad)
		}
	}
	got, _ := mem.Get(ctx, id)
	if got.Visibility != "private" {
		t.Errorf("unknown-target share must leave visibility private, got %q", got.Visibility)
	}
	// A valid share widens to team.
	if err := mem.Share(ctx, id, memory.ShareInput{Visibility: "team"}); err != nil {
		t.Fatalf("valid share: %v", err)
	}
	got, _ = mem.Get(ctx, id)
	if got.Visibility != "team" {
		t.Errorf("valid share should set team, got %q", got.Visibility)
	}
}

// TestUpdateVersion covers @step-01 (Scenario: mem-update-appends-recoverable-version):
// mem_update appends a version row; the prior content is recoverable via the
// governance query; the latest version is the read path; revision_count advances.
func TestUpdateVersion(t *testing.T) {
	_, mem := serviceGovernanceFixture(t)
	ctx := context.Background()
	id, err := mem.Save(ctx, memory.SaveInput{
		SessionID: "s1", Type: "decision",
		Title: "versioned", Content: "original content", Owner: "ownerA",
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := mem.Update(ctx, id, memory.UpdateInput{Content: "second content"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, err := mem.Get(ctx, id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Content != "second content" {
		t.Errorf("latest version must be the read path, got %q", got.Content)
	}
	if got.RevisionCount != 1 {
		t.Errorf("revision_count must be 1 after one update, got %d", got.RevisionCount)
	}
	gov, err := mem.Governance(ctx, id)
	if err != nil {
		t.Fatalf("governance: %v", err)
	}
	if len(gov.Versions) != 1 || gov.Versions[0].Content != "original content" {
		t.Errorf("prior content must be recoverable via governance, got %+v", gov.Versions)
	}
}

// TestStatusExplicit covers @step-01 (Scenario: superseded-status-set-explicitly):
// superseded/archived status is set explicitly, never inferred from content.
func TestStatusExplicit(t *testing.T) {
	_, mem := serviceGovernanceFixture(t)
	ctx := context.Background()
	id, err := mem.Save(ctx, memory.SaveInput{
		SessionID: "s1", Type: "decision",
		Title: "status asset", Content: "status body", Owner: "ownerA",
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	got, _ := mem.Get(ctx, id)
	if got.Status != "active" {
		t.Errorf("default status must be active, got %q", got.Status)
	}
	if err := mem.SetStatus(ctx, id, "superseded"); err != nil {
		t.Fatalf("set superseded: %v", err)
	}
	got, _ = mem.Get(ctx, id)
	if got.Status != "superseded" {
		t.Errorf("status must be superseded after explicit set, got %q", got.Status)
	}
	// Never inferred: an update that mentions "superseded" in content does not
	// change the status.
	if err := mem.Update(ctx, id, memory.UpdateInput{Content: "still active but mentions superseded"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, _ = mem.Get(ctx, id)
	if got.Status != "superseded" {
		t.Errorf("status must not be inferred from content, got %q", got.Status)
	}
}

// TestUsageCount covers @step-01 (Scenario: retrieval-usage-count-increments-on-search):
// a search hit increments retrieval_usage (distinct from duplicate_count).
func TestUsageCount(t *testing.T) {
	_, mem := serviceGovernanceFixture(t)
	ctx := context.Background()
	id, err := mem.Save(ctx, memory.SaveInput{
		SessionID: "s1", Type: "decision",
		Title: "usage target", Content: "usage target body", Owner: "ownerA",
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	got, _ := mem.Get(ctx, id)
	if got.RetrievalUsage != 0 || got.DuplicateCount != 0 {
		t.Errorf("initial usage/duplicate must be 0/0, got %d/%d", got.RetrievalUsage, got.DuplicateCount)
	}
	for i := 0; i < 2; i++ {
		if _, err := mem.Search(ctx, "usage", "any", 10); err != nil {
			t.Fatalf("search %d: %v", i, err)
		}
	}
	got, _ = mem.Get(ctx, id)
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
	_, mem := serviceGovernanceFixture(t)
	ctx := context.Background()
	id, err := mem.Save(ctx, memory.SaveInput{
		SessionID: "s1", Type: "decision",
		Title: "govq asset", Content: "govq v1", Owner: "ownerA",
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := mem.Update(ctx, id, memory.UpdateInput{Content: "govq v2"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if err := mem.Share(ctx, id, memory.ShareInput{Visibility: "team"}); err != nil {
		t.Fatalf("share: %v", err)
	}
	if err := mem.SetStatus(ctx, id, "superseded"); err != nil {
		t.Fatalf("status: %v", err)
	}
	if _, err := mem.Search(ctx, "govq", "any", 10); err != nil {
		t.Fatalf("search: %v", err)
	}
	gov, err := mem.Governance(ctx, id)
	if err != nil {
		t.Fatalf("governance: %v", err)
	}
	if gov.Owner != "ownerA" {
		t.Errorf("governance owner = %q, want ownerA", gov.Owner)
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

func idInList(obs []memory.Observation, id int64) bool {
	for _, o := range obs {
		if o.ID == id {
			return true
		}
	}
	return false
}
