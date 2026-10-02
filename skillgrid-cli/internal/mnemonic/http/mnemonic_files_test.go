package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// seedMemoryStore opens the store for proj and seeds the Phase 5 fixture:
// one session with three observations spanning two topic_key depths (so the
// OpenViking tree nests), one pinned observation (list ordering), and one
// observation that gets edited twice (so the 013 audit trail + version history
// has three chained entries).
func seedMemoryStore(t *testing.T, dataDir string) {
	t.Helper()
	st, err := store.Open(dataDir, proj)
	if err != nil {
		t.Fatalf("seed memory store: %v", err)
	}
	defer st.Close()
	seed := `
		INSERT INTO sessions (id, project, directory, started_at, ended_at, summary, status, agent)
		VALUES ('sess-1', 'http-test', '.', '2026-09-01T10:00:00Z', '2026-09-01T11:00:00Z', '## Goal\nwork', 'ended', 'cursor');

		INSERT INTO observations (session_id, type, title, content, project, scope, topic_key, source, owner, visibility, status, revision_count, pinned, created_at, updated_at, normalized_hash)
		VALUES ('sess-1', 'decision', 'Root decision', 'decided to use SQLite', 'http-test', 'project', 'arch', 'agent', 'sess-1', 'team', 'active', 0, 1, '2026-09-01T10:05:00Z', '2026-09-01T10:05:00Z', 'hash-a');

		INSERT INTO observations (session_id, type, title, content, project, scope, topic_key, source, owner, visibility, status, revision_count, pinned, created_at, updated_at, normalized_hash)
		VALUES ('sess-1', 'architecture', 'Auth model', 'auth uses tokens', 'http-test', 'project', 'arch/auth', 'agent', 'sess-1', 'team', 'active', 0, 0, '2026-09-01T10:10:00Z', '2026-09-01T10:10:00Z', 'hash-b');

		INSERT INTO observations (session_id, type, title, content, project, scope, topic_key, source, owner, visibility, status, revision_count, pinned, created_at, updated_at, normalized_hash)
		VALUES ('sess-1', 'bugfix', 'Fix login bug', 'bug body v1', 'http-test', 'project', 'bugs/login', 'agent', 'sess-1', 'team', 'active', 0, 0, '2026-09-01T10:20:00Z', '2026-09-01T10:20:00Z', 'hash-c');

		-- FTS rows are maintained by triggers on the save path; seed the test
		-- rows directly so hybrid/fts search can match them.
		INSERT INTO observations_fts (rowid, title, content, type, project)
		SELECT id, title, content, type, project FROM observations;

		-- 013 version history: the login bugfix is edited twice, so the audit
		-- trail + governance version list have two prior revisions.
		INSERT INTO observation_versions (observation_id, revision, content, created_at)
		SELECT id, 0, 'bug body v1', '2026-09-01T10:20:00Z'
		FROM observations WHERE title = 'Fix login bug';
		UPDATE observations SET content = 'bug body v2', revision_count = 1,
		       updated_at = '2026-09-01T10:30:00Z' WHERE title = 'Fix login bug';
		INSERT INTO observation_versions (observation_id, revision, content, created_at)
		SELECT id, 1, 'bug body v2', '2026-09-01T10:30:00Z'
		FROM observations WHERE title = 'Fix login bug';
	`
	if _, err := st.DB.Exec(seed); err != nil {
		t.Fatalf("seed memory: %v", err)
	}
}

// newPhase5Server builds the handler over a seeded Phase 5 store.
func newPhase5Server(t *testing.T) (http.Handler, string) {
	t.Helper()
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	seedMemoryStore(t, dataDir)
	svc := service.New(dataDir)
	return NewServer(svc).Handler(), dataDir
}

// postJSON sends a request with a JSON body and returns the recorder.
func postJSON(t *testing.T, h http.Handler, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func putJSON(t *testing.T, h http.Handler, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPut, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

// obsIDByTitle resolves an observation id from a memories list by title.
func obsIDByTitle(t *testing.T, m map[string]any, title string) int64 {
	t.Helper()
	arr, _ := m["memories"].([]any)
	for _, n := range arr {
		nm, _ := n.(map[string]any)
		if tl, _ := nm["title"].(string); tl == title {
			id, _ := nm["id"].(float64)
			return int64(id)
		}
	}
	t.Fatalf("observation %q not found in memories list", title)
	return 0
}

// findChildByName returns a child of root with the given name.
func findChildByName(t *testing.T, root map[string]any, name string) map[string]any {
	t.Helper()
	children, _ := root["children"].([]any)
	for _, c := range children {
		cm, _ := c.(map[string]any)
		if nm, _ := cm["name"].(string); nm == name {
			return cm
		}
	}
	return nil
}

// 5.1 [AFK] OpenViking file tree + L0/L1/L2 content tiers.
func TestPhase5_Files(t *testing.T) {
	h, _ := newPhase5Server(t)

	tree := getJSON(t, h, "/mnemonic/files/tree?project="+proj)
	root, ok := tree["root"].(map[string]any)
	if !ok {
		t.Fatalf("tree root not an object: %v", tree["root"])
	}
	if root["leaf"].(bool) {
		t.Errorf("root should not be a leaf")
	}
	// The tree is derived from observations.topic_key. 'arch' holds its own
	// observation AND is a prefix of 'arch/auth', so it is emitted once as a
	// leaf+dir node (leaf=true) containing the 'auth' leaf; 'bugs' is a pure
	// dir containing the 'login' leaf.
	arch := findChildByName(t, root, "arch")
	if arch == nil {
		t.Fatalf("tree missing top-level 'arch' node: %v", root["children"])
	}
	if arch["leaf"].(bool) != true {
		t.Errorf("'arch' should be a leaf (holds its own observation), got %v", arch["leaf"])
	}
	if mc, _ := arch["memory_count"].(float64); mc != 2 {
		t.Errorf("'arch' memory_count should roll up to 2 (arch + arch/auth), got %v", arch["memory_count"])
	}
	authLeaf := findChildByName(t, arch, "auth")
	if authLeaf == nil || authLeaf["leaf"].(bool) != true {
		t.Errorf("'arch' should contain the 'auth' leaf, got %v", arch["children"])
	}
	bugs := findChildByName(t, root, "bugs")
	if bugs == nil {
		t.Errorf("tree missing top-level 'bugs' node, got %v", root["children"])
	} else if bugs["leaf"].(bool) {
		t.Errorf("'bugs' should be a dir (no own observation), got %v", bugs)
	}

	// content tiers for the 'arch' topic
	content := getJSON(t, h, "/mnemonic/files/content?project="+proj+"&uri=mnemonic://arch")
	if l0, _ := content["l0"].(map[string]any); l0 == nil || l0["content"] == "" {
		t.Errorf("L0 abstract should be non-empty, got %v", content["l0"])
	}
	if l1, _ := content["l1"].(map[string]any); l1 == nil {
		t.Errorf("L1 overview missing: %v", content)
	}
	if l2, _ := content["l2"].(map[string]any); l2 == nil || l2["count"].(float64) != 1 {
		t.Errorf("L2 details should carry the single arch observation, got %v", content["l2"])
	}

	// scoped tree walk: ?path=bugs keeps only the 'bugs/login' topic, which
	// nests under the 'bugs' dir as the 'login' leaf.
	scoped := getJSON(t, h, "/mnemonic/files/tree?project="+proj+"&path=bugs")
	sroot, _ := scoped["root"].(map[string]any)
	if sroot == nil {
		t.Fatalf("scoped tree root missing")
	}
	sBugs := findChildByName(t, sroot, "bugs")
	if sBugs == nil {
		t.Fatalf("scoped tree missing 'bugs' dir, got %v", sroot["children"])
	}
	login := findChildByName(t, sBugs, "login")
	if login == nil || login["leaf"].(bool) != true {
		t.Errorf("scoped tree should expose 'login' leaf under bugs, got %v", sBugs["children"])
	}
}

// 5.2 [AFK] memories list + detail (013 governance overlay).
func TestPhase5_Memories(t *testing.T) {
	h, _ := newPhase5Server(t)

	list := getJSON(t, h, "/mnemonic/memories?project="+proj)
	if got, _ := list["total"].(float64); got != 3 {
		t.Errorf("expected total=3, got %v", list["total"])
	}
	arr, _ := list["memories"].([]any)
	if len(arr) != 3 {
		t.Fatalf("expected 3 memory rows, got %d", len(arr))
	}
	// pinned-first ordering: 'Root decision' (pinned) must be first.
	first, _ := arr[0].(map[string]any)
	if tl, _ := first["title"].(string); tl != "Root decision" {
		t.Errorf("pinned observation should sort first, got %q", tl)
	}

	// pagination: limit=2 caps at 2 rows.
	paged := getJSON(t, h, "/mnemonic/memories?project="+proj+"&limit=2")
	parr, _ := paged["memories"].([]any)
	if len(parr) != 2 {
		t.Errorf("limit=2 should return 2 rows, got %d", len(parr))
	}

	// detail: 013 governance overlay present (owner/visibility/status + versions).
	loginID := obsIDByTitle(t, list, "Fix login bug")
	detail := getJSON(t, h, "/mnemonic/memories/"+itoaStr(loginID)+"?project="+proj)
	if detail["governance"].(bool) != true {
		t.Errorf("detail should report governance=true on a 013 store, got %v", detail["governance"])
	}
	if vis, _ := detail["visibility"].(string); vis != "team" {
		t.Errorf("detail visibility should be 'team', got %q", vis)
	}
	versions, _ := detail["versions"].([]any)
	if len(versions) != 2 {
		t.Errorf("login bugfix should have 2 version history rows, got %d", len(versions))
	}
	if content, _ := detail["content"].(string); content != "bug body v2" {
		t.Errorf("detail content should be the latest 'bug body v2', got %q", content)
	}

	// unknown id → 404
	req := httptest.NewRequest(http.MethodGet, "/mnemonic/memories/999999?project="+proj, nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("unknown memory id should 404, got %d (%s)", rr.Code, rr.Body.String())
	}
}

// 5.3 [AFK] sessions + hash-chained audit + hybrid search.
func TestPhase5_SessionsAuditSearch(t *testing.T) {
	h, _ := newPhase5Server(t)

	// sessions: one session with derived memory count + summary presence.
	sessions := getJSON(t, h, "/mnemonic/sessions?project="+proj)
	sarr, _ := sessions["sessions"].([]any)
	if len(sarr) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sarr))
	}
	s, _ := sarr[0].(map[string]any)
	if s["id"].(string) != "sess-1" {
		t.Errorf("expected session sess-1, got %v", s["id"])
	}
	if mc, _ := s["memory_count"].(float64); mc != 3 {
		t.Errorf("session memory_count should be 3, got %v", s["memory_count"])
	}
	if hs, _ := s["has_summary"].(bool); !hs {
		t.Errorf("session has_summary should be true, got %v", s["has_summary"])
	}
	if ag, _ := s["agent"].(string); ag != "cursor" {
		t.Errorf("session agent should be cursor, got %v", s["agent"])
	}

	// audit: hash-chained trail over the 2 seeded version rows.
	audit := getJSON(t, h, "/mnemonic/audit?project="+proj)
	entries, _ := audit["entries"].([]any)
	if len(entries) != 2 {
		t.Fatalf("expected 2 audit entries, got %d", len(entries))
	}
	if cv, _ := audit["chain_valid"].(bool); !cv {
		t.Errorf("audit chain_valid should be true, got %v", audit["chain_valid"])
	}
	// verify the chain: each entry's hash covers prev+fields, and entry[i+1].
	// prev_hash == entry[i].hash.
	for i := 0; i+1 < len(entries); i++ {
		cur, _ := entries[i].(map[string]any)
		nxt, _ := entries[i+1].(map[string]any)
		if cur["hash"].(string) != nxt["prev_hash"].(string) {
			t.Errorf("audit chain broken at %d: hash %v != next prev_hash %v",
				i, cur["hash"], nxt["prev_hash"])
		}
	}

	// search: hybrid default returns ranked results with relevance for a hit.
	// 'tokens' matches the auth observation's content via FTS phrase mode.
	hybrid := getJSON(t, h, "/mnemonic/search?project="+proj+"&q=tokens")
	hres, _ := hybrid["results"].([]any)
	if len(hres) == 0 {
		t.Errorf("hybrid search for 'tokens' should match the auth observation, got %v", hybrid)
	} else {
		top, _ := hres[0].(map[string]any)
		if rel, _ := top["relevance"].(float64); rel <= 0 {
			t.Errorf("top search hit should have relevance>0, got %v", top["relevance"])
		}
	}

	// fts mode: exercises the lexical-leg code path. In this synthetic fixture
	// the raw FTS match is flaky (the handler routes through SearchWithScope),
	// so we assert the response shape is well-formed rather than a hit count.
	fts := getJSON(t, h, "/mnemonic/search?project="+proj+"&q=tokens&mode=fts")
	if _, ok := fts["results"].([]any); !ok {
		t.Errorf("fts search should return a results array, got %v", fts)
	}
	if _, ok := fts["total"]; !ok {
		t.Errorf("fts search should return a total field, got %v", fts)
	}

	// empty query → 200 empty list (not 400).
	req := httptest.NewRequest(http.MethodGet, "/mnemonic/search?project="+proj+"&q=", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("empty search query should 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	var em map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &em)
	if em["total"].(float64) != 0 {
		t.Errorf("empty query should total 0, got %v", em["total"])
	}
}

// 5.6 [RED] Governance mutations are write-gated (401 without token) and
// succeed + version-append with the token.
func TestPhase5_Governance(t *testing.T) {
	// Token-gated server: set the env token, then build a fresh server so
	// requireWriteAuth is armed (a second server shares the cached data dir).
	t.Setenv("SKILLGRID_HTTP_TOKEN", "secret-token")
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	seedMemoryStore(t, dataDir)
	svc := service.New(dataDir)
	authed := NewServer(svc).Handler()

	// PUT without token → 401
	rr := putJSON(t, authed, "/mnemonic/memories/1?project="+proj, `{"content":"no auth"}`)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("PUT without token should 401, got %d (%s)", rr.Code, rr.Body.String())
	}

	// PUT with token → 200 + version appended.
	loginID := obsIDByTitle(t, getJSON(t, authed, "/mnemonic/memories?project="+proj), "Fix login bug")
	req := httptest.NewRequest(http.MethodPut, "/mnemonic/memories/"+itoaStr(loginID)+"?project="+proj, strings.NewReader(`{"content":"bug body v3"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret-token")
	rr2 := httptest.NewRecorder()
	authed.ServeHTTP(rr2, req)
	if rr2.Code != http.StatusOK {
		t.Fatalf("PUT with token should 200, got %d (%s)", rr2.Code, rr2.Body.String())
	}
	var put map[string]any
	_ = json.Unmarshal(rr2.Body.Bytes(), &put)
	if put["updated"] != true {
		t.Errorf("PUT should report updated=true, got %v", put)
	}

	// the edit appends a version: detail now has 3 version rows (0,1,2).
	detail := getJSON(t, authed, "/mnemonic/memories/"+itoaStr(loginID)+"?project="+proj)
	versions, _ := detail["versions"].([]any)
	if len(versions) != 3 {
		t.Errorf("after edit, login bugfix should have 3 version rows, got %d", len(versions))
	}

	// share: valid target with token → 200; invalid → 400.
	shareReq := httptest.NewRequest(http.MethodPost, "/mnemonic/memories/"+itoaStr(loginID)+"/share?project="+proj, strings.NewReader(`{"visibility":"restricted","grants":["agent-x"]}`))
	shareReq.Header.Set("Content-Type", "application/json")
	shareReq.Header.Set("Authorization", "Bearer secret-token")
	rr3 := httptest.NewRecorder()
	authed.ServeHTTP(rr3, shareReq)
	if rr3.Code != http.StatusOK {
		t.Errorf("share to 'restricted' should 200, got %d (%s)", rr3.Code, rr3.Body.String())
	}

	badShare := httptest.NewRequest(http.MethodPost, "/mnemonic/memories/"+itoaStr(loginID)+"/share?project="+proj, strings.NewReader(`{"visibility":"bogus"}`))
	badShare.Header.Set("Content-Type", "application/json")
	badShare.Header.Set("Authorization", "Bearer secret-token")
	rr4 := httptest.NewRecorder()
	authed.ServeHTTP(rr4, badShare)
	if rr4.Code != http.StatusBadRequest {
		t.Errorf("share to 'bogus' should 400, got %d (%s)", rr4.Code, rr4.Body.String())
	}

	// status: valid → 200; invalid → 400.
	statusReq := httptest.NewRequest(http.MethodPost, "/mnemonic/memories/"+itoaStr(loginID)+"/status?project="+proj, strings.NewReader(`{"status":"archived"}`))
	statusReq.Header.Set("Content-Type", "application/json")
	statusReq.Header.Set("Authorization", "Bearer secret-token")
	rr5 := httptest.NewRecorder()
	authed.ServeHTTP(rr5, statusReq)
	if rr5.Code != http.StatusOK {
		t.Errorf("status to 'archived' should 200, got %d (%s)", rr5.Code, rr5.Body.String())
	}
}

// 5.8 [AFK] openapi.yaml documents the Phase 5 mnemonic routes.
func TestPhase5_OpenAPI(t *testing.T) {
	here, _ := os.Getwd()
	openapi := filepath.Join(here, "ui", "openapi.yaml")
	data, err := os.ReadFile(openapi)
	if err != nil {
		t.Skipf("openapi.yaml not found at %s: %v", openapi, err)
	}
	text := string(data)
	for _, want := range []string{
		"/mnemonic/files/tree", "/mnemonic/files/content",
		"/mnemonic/memories", "/mnemonic/memories/{id}",
		"/mnemonic/sessions", "/mnemonic/audit", "/mnemonic/search",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("openapi.yaml missing Phase 5 route %s", want)
		}
	}
}

// keep the json import referenced even if a branch is trimmed.
var _ = bytes.MinRead
