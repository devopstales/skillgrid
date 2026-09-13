package memfs

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// newMemFSTestFixture opens a store, creates a memory service, and returns
// the fixture.
type memfsFixture struct {
	fs  *MemFS
	mem *memory.Service
	st  *store.Store
}

func newMemFSTestFixture(t *testing.T, project string) *memfsFixture {
	t.Helper()
	dataDir := t.TempDir()
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	mem := memory.New(st, project)
	// Create a session for saving observations.
	if _, err := st.DB.Exec(`
		INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES ('sess-memfs', ?, '/tmp', '2026-01-01T00:00:00Z', 'active')`, project); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	return &memfsFixture{
		fs:  New(st, project),
		mem: mem,
		st:  st,
	}
}

// saveObs saves an observation with the given topic_key and memory_type.
func (fx *memfsFixture) saveObs(t *testing.T, title, topicKey, memType, content string) {
	t.Helper()
	ctx := context.Background()
	_, err := fx.mem.Save(ctx, memory.SaveInput{
		SessionID:  "sess-memfs",
		Type:       "learning",
		Title:      title,
		Content:    content,
		TopicKey:   topicKey,
		MemoryType: memType,
	})
	if err != nil {
		t.Fatalf("save obs %q: %v", title, err)
	}
}

// TestMemLSScopeListing is 25.1 [RED] — `mem ls` lists observations in a scope.
// Observations in project/A/preferences, project/A/entities, user/B/profile.
// `mem ls project/A/preferences` → only preferences;
// `mem ls project/A/` → all under A;
// `mem ls user/B/` → only B.
func TestMemLSScopeListing(t *testing.T) {
	fx := newMemFSTestFixture(t, "memfs-ls-test")
	ctx := context.Background()

	// Create observations in different scopes.
	fx.saveObs(t, "pref-a", "project/A/preferences/lang", "preferences", "Use gofmt")
	fx.saveObs(t, "pref-b", "project/A/preferences/lint", "preferences", "Use golangci-lint")
	fx.saveObs(t, "ent-a", "project/A/entities/user_model", "entities", "User entity")
	fx.saveObs(t, "ent-b", "project/A/entities/order_model", "entities", "Order entity")
	fx.saveObs(t, "profile-b", "user/B/profile/basic", "profile", "User B profile")

	// ls project/A/preferences → only preferences observations.
	res, err := fx.fs.List(ctx, "project/A/preferences")
	if err != nil {
		t.Fatalf("ls project/A/preferences: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("ls project/A/preferences: expected 2, got %d: %+v", len(res), res)
	}
	for _, o := range res {
		if !strings.Contains(o.TopicKey, "preferences") {
			t.Errorf("ls project/A/preferences: unexpected observation %q (topic_key=%q)", o.Title, o.TopicKey)
		}
	}

	// ls project/A/ → all under project A (preferences + entities).
	res, err = fx.fs.List(ctx, "project/A/")
	if err != nil {
		t.Fatalf("ls project/A/: %v", err)
	}
	if len(res) != 4 {
		t.Fatalf("ls project/A/: expected 4, got %d: %+v", len(res), res)
	}

	// ls user/B/ → only user B's observations.
	res, err = fx.fs.List(ctx, "user/B/")
	if err != nil {
		t.Fatalf("ls user/B/: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("ls user/B/: expected 1, got %d: %+v", len(res), res)
	}
	if res[0].Title != "profile-b" {
		t.Errorf("ls user/B/: expected profile-b, got %q", res[0].Title)
	}
}

// TestMemTreeHierarchical is 25.2 [RED] — `mem tree` shows a hierarchical view.
// Observations in nested scopes project/A/preferences/sub1, sub2, project/A/entities.
// `mem tree project/A/` → shows the directory tree.
func TestMemTreeHierarchical(t *testing.T) {
	fx := newMemFSTestFixture(t, "memfs-tree-test")
	ctx := context.Background()

	// Create nested observations.
	fx.saveObs(t, "sub1-obs", "project/A/preferences/sub1", "preferences", "Sub1 content")
	fx.saveObs(t, "sub2-obs", "project/A/preferences/sub2", "preferences", "Sub2 content")
	fx.saveObs(t, "ent-obs", "project/A/entities", "entities", "Entities content")

	tree, err := fx.fs.Tree(ctx, "project/A/")
	if err != nil {
		t.Fatalf("tree project/A/: %v", err)
	}

	// Verify the tree shows the hierarchical structure.
	if !strings.Contains(tree, "preferences/") {
		t.Errorf("tree missing preferences/ directory:\n%s", tree)
	}
	if !strings.Contains(tree, "entities/") {
		t.Errorf("tree missing entities/ directory:\n%s", tree)
	}
	if !strings.Contains(tree, "sub1") {
		t.Errorf("tree missing sub1:\n%s", tree)
	}
	if !strings.Contains(tree, "sub2") {
		t.Errorf("tree missing sub2:\n%s", tree)
	}

	// Verify indentation: preferences/ should contain sub1 and sub2 as children.
	lines := strings.Split(strings.TrimSpace(tree), "\n")
	// Find the preferences/ line and verify sub1/sub2 are indented under it.
	prefIdx := -1
	for i, l := range lines {
		if strings.Contains(l, "preferences/") {
			prefIdx = i
			break
		}
	}
	if prefIdx == -1 {
		t.Fatalf("tree: preferences/ not found in output:\n%s", tree)
	}
	// sub1 and sub2 should appear after preferences/ with more indentation.
	foundSub1 := false
	foundSub2 := false
	for i := prefIdx + 1; i < len(lines); i++ {
		l := lines[i]
		if strings.Contains(l, "sub1") {
			foundSub1 = true
		}
		if strings.Contains(l, "sub2") {
			foundSub2 = true
		}
		// Stop if we hit a sibling at the same or less indent level.
		// entities/ is a sibling of preferences/, not a child.
		if strings.Contains(l, "entities/") && i > prefIdx {
			break
		}
	}
	if !foundSub1 || !foundSub2 {
		t.Errorf("tree: sub1/sub2 not found under preferences/:\n%s", tree)
	}
}

// TestMemFindPatternMatching is 25.3 [RED] — `mem find` does glob pattern matching.
// Observations with titles "auth.go", "auth_test.go", "payment.go", "user.go".
// `mem find "auth*"` → "auth.go" and "auth_test.go".
// `mem find "*.go"` → all 4.
// `mem find "auth*test*"` → only "auth_test.go".
func TestMemFindPatternMatching(t *testing.T) {
	fx := newMemFSTestFixture(t, "memfs-find-test")
	ctx := context.Background()

	// Create observations with file-like titles.
	fx.saveObs(t, "auth.go", "project/A/preferences/auth.go", "preferences", "auth.go content")
	fx.saveObs(t, "auth_test.go", "project/A/preferences/auth_test.go", "preferences", "auth_test.go content")
	fx.saveObs(t, "payment.go", "project/A/entities/payment.go", "entities", "payment.go content")
	fx.saveObs(t, "user.go", "project/A/entities/user.go", "entities", "user.go content")

	// find "auth*" → auth.go and auth_test.go.
	res, err := fx.fs.Find(ctx, "auth*", "project/A/")
	if err != nil {
		t.Fatalf("find auth*: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("find auth*: expected 2, got %d: %+v", len(res), res)
	}
	names := map[string]bool{}
	for _, o := range res {
		names[o.Title] = true
	}
	if !names["auth.go"] || !names["auth_test.go"] {
		t.Errorf("find auth*: expected auth.go and auth_test.go, got %v", names)
	}

	// find "*.go" → all 4.
	res, err = fx.fs.Find(ctx, "*.go", "project/A/")
	if err != nil {
		t.Fatalf("find *.go: %v", err)
	}
	if len(res) != 4 {
		t.Fatalf("find *.go: expected 4, got %d: %+v", len(res), res)
	}

	// find "auth*test*" → only auth_test.go.
	res, err = fx.fs.Find(ctx, "auth*test*", "project/A/")
	if err != nil {
		t.Fatalf("find auth*test*: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("find auth*test*: expected 1, got %d: %+v", len(res), res)
	}
	if res[0].Title != "auth_test.go" {
		t.Errorf("find auth*test*: expected auth_test.go, got %q", res[0].Title)
	}
}

// TestMemURIResolution is 25.4 [RED] — `mem://` URI resolution maps to scope filters.
func TestMemURIResolution(t *testing.T) {
	// Resolve mem://project/A/preferences.
	f, err := ResolveURI("mem://project/A/preferences")
	if err != nil {
		t.Fatalf("ResolveURI project/A/preferences: %v", err)
	}
	if f.Kind != "project" || f.ID != "A" || f.MemoryType != "preferences" {
		t.Errorf("ResolveURI project/A/preferences: got %+v", f)
	}
	if f.Prefix() != "project/A/preferences" {
		t.Errorf("ResolveURI project/A/preferences prefix: got %q", f.Prefix())
	}

	// Resolve mem://user/B/.
	f, err = ResolveURI("mem://user/B/")
	if err != nil {
		t.Fatalf("ResolveURI user/B/: %v", err)
	}
	if f.Kind != "user" || f.ID != "B" || f.MemoryType != "" {
		t.Errorf("ResolveURI user/B/: got %+v", f)
	}
	if f.Prefix() != "user/B" {
		t.Errorf("ResolveURI user/B/ prefix: got %q", f.Prefix())
	}

	// Resolve mem://project/A/ (no memory_type).
	f, err = ResolveURI("mem://project/A/")
	if err != nil {
		t.Fatalf("ResolveURI project/A/: %v", err)
	}
	if f.Kind != "project" || f.ID != "A" || f.MemoryType != "" {
		t.Errorf("ResolveURI project/A/: got %+v", f)
	}

	// Invalid URI: empty.
	_, err = ResolveURI("")
	if err == nil {
		t.Error("ResolveURI empty: expected error, got nil")
	}

	// Invalid URI: bad kind.
	_, err = ResolveURI("mem://org/X/")
	if err == nil {
		t.Error("ResolveURI org/X/: expected error for unknown kind, got nil")
	}

	// Invalid URI: missing id.
	_, err = ResolveURI("mem://project/")
	if err == nil {
		t.Error("ResolveURI project/: expected error for missing id, got nil")
	}

	// Bare path (no mem:// prefix).
	f, err = ResolveURI("project/A/preferences")
	if err != nil {
		t.Fatalf("ResolveURI bare project/A/preferences: %v", err)
	}
	if f.Kind != "project" || f.ID != "A" || f.MemoryType != "preferences" {
		t.Errorf("ResolveURI bare: got %+v", f)
	}

	// Resolution is fast (sub-millisecond) — just verify it completes quickly.
	start := time.Now()
	for i := 0; i < 10000; i++ {
		_, _ = ResolveURI("mem://project/A/preferences")
	}
	elapsed := time.Since(start)
	if elapsed > 100*time.Millisecond {
		t.Errorf("ResolveURI 10000 iterations took %v (expected < 100ms)", elapsed)
	}
}
