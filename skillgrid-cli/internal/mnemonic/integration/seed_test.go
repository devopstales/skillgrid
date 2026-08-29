package integration

import (
	"context"
	stdhttp "net/http"
	"os"
	"path/filepath"
	"testing"

	mnemonichttp "github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/http"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/webcache"
)

const seedProject = "seed-test"

// seedWorkspace creates a temp workspace with a pinned project id and a few
// source files for the code indexer. It returns the workspace path.
func seedWorkspace(t *testing.T) string {
	t.Helper()
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, ".skillgrid"), 0o755); err != nil {
		t.Fatalf("mkdir .skillgrid: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, ".skillgrid", "config.json"),
		[]byte(`{"project":"`+seedProject+`"}`), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}
	pkg := filepath.Join(workspace, "pkg")
	if err := os.MkdirAll(pkg, 0o755); err != nil {
		t.Fatalf("mkdir pkg: %v", err)
	}
	files := map[string]string{
		filepath.Join(pkg, "app.go"):          "package app\n\nfunc Greet(name string) string { return \"hi \" + name }\n",
		filepath.Join(pkg, "util.ts"):         "export function add(a: number, b: number) { return a + b }\n",
		filepath.Join(workspace, "README.md"): "# seed workspace\nhello code index Greet\n",
	}
	for p, c := range files {
		if err := os.WriteFile(p, []byte(c), 0o644); err != nil {
			t.Fatalf("write %s: %v", p, err)
		}
	}
	return workspace
}

func asFloat(t *testing.T, v any) float64 {
	t.Helper()
	f, ok := v.(float64)
	if !ok {
		t.Fatalf("expected number, got %T (%v)", v, v)
	}
	return f
}

// TestSeedMemory loads observations across several valid types into the memory
// store and verifies the status + search surface them.
func TestSeedMemory(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	workspace := seedWorkspace(t)
	svc := service.New(dataDir)
	ctx := context.Background()

	sessID, projectID, err := svc.SessionStart(ctx, workspace, "Seed memory store")
	if err != nil {
		t.Fatalf("session start: %v", err)
	}
	if projectID != seedProject {
		t.Fatalf("expected project %s, got %q", seedProject, projectID)
	}

	seeded := []service.SaveObservationInput{
		{Type: "decision", Title: "Chose SQLite for the local store", Content: "single binary, per-project file", TopicKey: "architecture/store"},
		{Type: "bugfix", Title: "Fixed N+1 query in UserList", Content: "batched the user lookups"},
		{Type: "discovery", Title: "embed resolves relative to package dir", Content: "asset dir moved, silently dropped"},
		{Type: "preference", Title: "Prefer compact table output", Content: "CLI tables, no prose", Scope: "user"},
	}
	for i, in := range seeded {
		if _, err := svc.SaveObservation(ctx, projectID, service.SaveObservationInput{SessionID: sessID, Type: in.Type, Title: in.Title, Content: in.Content, Scope: in.Scope, TopicKey: in.TopicKey}); err != nil {
			t.Fatalf("save %d: %v", i, err)
		}
	}

	st, err := svc.MemoryStatus(ctx, projectID)
	if err != nil {
		t.Fatalf("memory status: %v", err)
	}
	if st.ObservationCount != len(seeded) {
		t.Errorf("expected %d observations, got %d", len(seeded), st.ObservationCount)
	}
	for _, want := range []string{"decision", "bugfix", "discovery", "preference"} {
		if st.ByType[want] != 1 {
			t.Errorf("expected by_type[%s]=1, got %v (all: %v)", want, st.ByType[want], st.ByType)
		}
	}
	if st.ActiveSessions != 1 {
		t.Errorf("expected 1 active session, got %d", st.ActiveSessions)
	}
	if st.NewestCreated == "" {
		t.Errorf("expected newest_created to be set")
	}

	hits, err := svc.SearchObservations(ctx, projectID, "sqlite batched embed", "any", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(hits) < 3 {
		t.Errorf("expected >=3 search hits, got %d", len(hits))
	}
}

// TestSeedCodeIndex writes a small repo, runs the incremental indexer, and
// verifies files, chunks, status, and FTS search.
func TestSeedCodeIndex(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	workspace := seedWorkspace(t)
	svc := service.New(dataDir)
	ctx := context.Background()

	projectID, err := svc.ResolveProject(workspace)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if projectID != seedProject {
		t.Fatalf("expected project %s, got %q", seedProject, projectID)
	}

	stats, err := svc.RunCodeIndex(ctx, workspace)
	if err != nil {
		t.Fatalf("run code index: %v", err)
	}
	if stats.FilesIndexed < 3 {
		t.Errorf("expected >=3 files indexed, got %d", stats.FilesIndexed)
	}

	status, stale, err := svc.CodeStatus(ctx, projectID)
	if err != nil {
		t.Fatalf("code status: %v", err)
	}
	if stale {
		t.Errorf("expected index fresh, got stale")
	}
	if status.FileCount < 3 {
		t.Errorf("expected >=3 files, got %d", status.FileCount)
	}

	files, err := svc.CodeFiles(ctx, projectID)
	if err != nil {
		t.Fatalf("code files: %v", err)
	}
	if len(files) < 3 {
		t.Errorf("expected >=3 indexed paths, got %d", len(files))
	}

	hits, err := svc.CodeSearch(ctx, projectID, "Greet", 5)
	if err != nil {
		t.Fatalf("code search: %v", err)
	}
	if len(hits) < 1 {
		t.Errorf("expected a Greet chunk hit, got %d", len(hits))
	}
}

// TestSeedWebCache persists cached snapshots across every supported source and
// verifies status, lookup, and freshness filtering.
func TestSeedWebCache(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	svc := service.New(dataDir)
	ctx := context.Background()

	entries := []webcache.SaveWebInput{
		{Source: "context7", LibraryID: "/vercel/next.js", Query: "route handlers", Content: "Next.js route handlers doc snippet"},
		{Source: "exa", Query: "express rate limiting", Content: "express-rate-limit howto body"},
		{Source: "deepwiki", RepoName: "facebook/react", Question: "useEffect cleanup", Content: "React useEffect cleanup doc"},
		{Source: "fetch", URL: "https://example.com/spec", Query: "openapi", Content: "fetched spec page"},
		{Source: "manual", Title: "bcrypt notes", Content: "cost 12 balances perf"},
	}
	for i, in := range entries {
		if _, err := svc.WebSave(ctx, seedProject, in); err != nil {
			t.Fatalf("web save %d (%s): %v", i, in.Source, err)
		}
	}

	st, err := svc.WebCacheStatus(ctx, seedProject)
	if err != nil {
		t.Fatalf("web status: %v", err)
	}
	if st.TotalEntries != len(entries) {
		t.Errorf("expected %d entries, got %d", len(entries), st.TotalEntries)
	}
	if st.ExpiredEntries != 0 {
		t.Errorf("expected 0 expired (default TTLs are far-future), got %d", st.ExpiredEntries)
	}
	for _, want := range []string{"context7", "exa", "deepwiki", "fetch", "manual"} {
		if st.BySource[want] != 1 {
			t.Errorf("expected by_source[%s]=1, got %v (all: %v)", want, st.BySource[want], st.BySource)
		}
	}

	lr, err := svc.WebLookup(ctx, seedProject, webcache.LookupInput{
		Source: "context7", LibraryID: "/vercel/next.js", Query: "route handlers",
	})
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if lr.Status != "hit" || !lr.Fresh {
		t.Errorf("expected fresh hit, got status=%q fresh=%v", lr.Status, lr.Fresh)
	}

	hits, err := svc.WebSearch(ctx, seedProject, "route handlers rate limiting", "", true, 10)
	if err != nil {
		t.Fatalf("web search: %v", err)
	}
	if len(hits) < 2 {
		t.Errorf("expected >=2 search hits, got %d", len(hits))
	}
}

// TestSeedAllStoresVisibleOverHTTP seeds all three stores in one workspace and
// asserts the dashboard's three status endpoints expose the data. The memory
// entry + code index + web snapshot share the project id resolved from the
// workspace, so a single store backs all three read paths.
func TestSeedAllStoresVisibleOverHTTP(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	workspace := seedWorkspace(t)
	svc := service.New(dataDir)
	ctx := context.Background()

	// Memory store.
	sessID, projectID, err := svc.SessionStart(ctx, workspace, "All stores seed")
	if err != nil {
		t.Fatalf("session start: %v", err)
	}
	if projectID != seedProject {
		t.Fatalf("expected project %s, got %q", seedProject, projectID)
	}
	if _, err := svc.SaveObservation(ctx, projectID, service.SaveObservationInput{
		SessionID: sessID, Type: "decision", Title: "visible over http", Content: "seeded across all three stores",
	}); err != nil {
		t.Fatalf("save observation: %v", err)
	}

	// Code index store.
	if _, err := svc.RunCodeIndex(ctx, workspace); err != nil {
		t.Fatalf("run code index: %v", err)
	}

	// Web cache store.
	if _, err := svc.WebSave(ctx, projectID, webcache.SaveWebInput{
		Source: "context7", LibraryID: "/lib/x", Query: "visible", Content: "seeded web entry",
	}); err != nil {
		t.Fatalf("web save: %v", err)
	}

	h := mnemonichttp.NewServer(svc).Handler()
	q := "?project=" + projectID

	rr, mem := doHTTP(t, h, stdhttp.MethodGet, "/memory/status"+q, nil)
	if rr.Code != stdhttp.StatusOK {
		t.Fatalf("memory/status: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if c := asFloat(t, mem["observation_count"]); c != 1 {
		t.Errorf("memory observation_count = %v, want 1", c)
	}

	rr, code := doHTTP(t, h, stdhttp.MethodGet, "/code/status"+q, nil)
	if rr.Code != stdhttp.StatusOK {
		t.Fatalf("code/status: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if c := asFloat(t, code["file_count"]); c != 3 {
		t.Errorf("code file_count = %v, want 3", c)
	}

	rr, web := doHTTP(t, h, stdhttp.MethodGet, "/web/status"+q, nil)
	if rr.Code != stdhttp.StatusOK {
		t.Fatalf("web/status: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if c := asFloat(t, web["total_entries"]); c != 1 {
		t.Errorf("web total_entries = %v, want 1", c)
	}
}
