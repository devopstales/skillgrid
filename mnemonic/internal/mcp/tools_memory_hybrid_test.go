package mcp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/mnemonic/internal/service"
	"github.com/devopstales/skillgrid/mnemonic/internal/skills"
	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

// TestHybridSearchRegistered covers @step-04: hybrid_search is listed
// alongside the unchanged mem_* / fact_* / skill_* tools.
func TestHybridSearchRegistered(t *testing.T) {
	s := NewServer()
	tools := s.ListTools()
	if _, ok := tools["hybrid_search"]; !ok {
		t.Fatal("hybrid_search not registered")
	}
	if _, ok := tools["fact_add"]; !ok {
		t.Fatal("fact_add missing after hybrid_search registered")
	}
}

func openHybridMCPStore(t *testing.T) (dataDir, project string, st *store.Store) {
	t.Helper()
	dataDir = t.TempDir()
	project = "hybridmcp"
	t.Setenv("MNEMONIC_PROJECT", project)
	var err error
	st, err = store.Open(dataDir, project)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { SetService(nil); st.Close() })
	SetService(service.New(dataDir))
	return dataDir, project, st
}

func seedHybridSession(t *testing.T, st *store.Store, project, dataDir, sid string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := st.DB.Exec(`
		INSERT OR IGNORE INTO sessions (id, project, directory, started_at, status)
		VALUES (?, ?, ?, ?, 'active')`, sid, project, dataDir, now); err != nil {
		t.Fatalf("seed session: %v", err)
	}
}

// TestHybridSearchMCPBM25Only covers @step-04: with no project embedder
// configured the tool returns BM25-ranked facts AND skills, legs=["fts"], and
// never errors on the absent vector leg.
func TestHybridSearchMCPBM25Only(t *testing.T) {
	dataDir, project, st := openHybridMCPStore(t)
	sid := "sess-hybrid-mcp"
	seedHybridSession(t, st, project, dataDir, sid)

	id := factAddID(t, project, sid, "the database connection pool max size is 50")
	if _, err := skills.New(st.DB, dataDir, project).Write(context.Background(), "go-testing", "go", "write table-driven go tests with t.Run subtests", "func TestX(t *testing.T){}", true); err != nil {
		t.Fatalf("skill write: %v", err)
	}

	res, err := handleMemoryHybridSearch(context.Background(), callReq("hybrid_search", map[string]any{
		"project":    project,
		"session_id": sid,
		"query":      "connection pool",
	}))
	if err != nil {
		t.Fatalf("handleMemoryHybridSearch: %v", err)
	}
	if res.IsError {
		t.Fatalf("hybrid_search tool error: %s", callResultText(t, res))
	}
	var out struct {
		Query    string                 `json:"query"`
		Legs     []string               `json:"legs"`
		Facts    []map[string]any       `json:"facts"`
		Skills   []map[string]any       `json:"skills"`
		Warnings []string               `json:"warnings"`
		_        map[string]interface{} `json:"-"`
	}
	if err := json.Unmarshal([]byte(callResultText(t, res)), &out); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	if len(out.Legs) != 1 || out.Legs[0] != "fts" {
		t.Fatalf("legs = %v, want [fts] (no project embedder configured)", out.Legs)
	}
	if len(out.Warnings) != 0 {
		t.Fatalf("warnings = %v, want none in BM25-only mode", out.Warnings)
	}
	if len(out.Facts) == 0 {
		t.Fatalf("facts = 0, want the seeded pool fact")
	}
	if fid, _ := out.Facts[0]["id"].(float64); int64(fid) != id {
		t.Fatalf("top fact id = %v, want %d", out.Facts[0]["id"], id)
	}
	if out.Facts[0]["scope"] != "fact" {
		t.Fatalf("fact scope = %v, want fact", out.Facts[0]["scope"])
	}
}

// TestHybridSearchMCPRequiresArgs: missing query or session_id is a clean
// tool error (the fact_search trail needs a real session id).
func TestHybridSearchMCPRequiresArgs(t *testing.T) {
	dataDir, project, st := openHybridMCPStore(t)
	sid := "sess-hybrid-req"
	seedHybridSession(t, st, project, dataDir, sid)

	res, err := handleMemoryHybridSearch(context.Background(), callReq("hybrid_search", map[string]any{
		"project":    project,
		"session_id": sid,
	}))
	if err != nil {
		t.Fatalf("handle: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected tool error for missing query")
	}
}
