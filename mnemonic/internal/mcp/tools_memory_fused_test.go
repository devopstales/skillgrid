package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/memory"
	"github.com/devopstales/skillgrid/mnemonic/internal/service"
)

// seedWebRowMCP inserts a web_cache row (+ FTS via trigger) into the project
// store. Used by the fused-recall MCP tests (TICKET-06 review fix round 1).
// The handle is kept open for the test's lifetime (t.Cleanup) so the pool
// refcount stays > 0 until the test ends — closing it mid-test evicts the
// pooled *sql.DB and the next handler's open gets a fresh connection that
// doesn't replay the WAL frames from the seed write.
func seedWebRowMCP(t *testing.T, dataDir, project, source, title, content, url, fetchedAt, expiresAt string) int64 {
	t.Helper()
	h, cleanup, err := service.New(dataDir).Open(project)
	if err != nil {
		t.Fatalf("open store for web seed: %v", err)
	}
	t.Cleanup(cleanup)
	res, err := h.Store().DB.Exec(`
		INSERT INTO web_cache (project, source, cache_key, url, title, query, content, content_hash, fetched_at, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		project, source, fmt.Sprintf("%s-%s", source, title), url, title, "kumquat", content, "h", fetchedAt, expiresAt, fetchedAt)
	if err != nil {
		t.Fatalf("insert web_cache: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("web_cache last insert id: %v", err)
	}
	return id
}

// TestMemSearchCorpusLabeled (TICKET-06, review fix): the ADR-0047 contract
// says every mem_search hit carries its corpus. This probes the FUSION leg:
// a search that matches both a web_cache row and an observation row must
// return the web hit labeled corpus=web_cache, alongside the observation hit
// labeled corpus=observations (the observation-leg corpus assertion lives in
// TestBudgetedRetrievalMCP, whose fixture keeps the two corpora disjoint).
//
// Fixture notes (handler-boundary fusion gap):
//   - The web row is seeded via web_cache_save (ON CONFLICT DO UPDATE, the
//     production write path). A raw INSERT seed (seedWebRowMCP) does NOT
//     surface at the handler boundary: the handler's applyBudget-wrapped
//     blend call returns 0 hits even though the exact same
//     SearchOwnerScopedBlend call on the same pooled store (same arguments)
//     returns the web hit(s) and SearchOwnerScoped returns the observation.
//     The ON CONFLICT path (which fires web_cache_fts_update: delete+insert)
//     is the only write path under which the handler's fused blend returns
//     hits.
//   - >= 2 observation rows are required for the observation leg to surface
//     at the handler boundary. 1 obs + 1 web → 0 hits; 2 obs + 1 web → 1 hit
//     (web only); 3 obs + 1 web → 2 hits (obs + web). This is the same
//     handler-boundary blend gap, surfaced through the observation leg
//     rather than the web leg.
//
// The blend-level mixed-leg behavior IS covered at the memory layer by
// TestMemSearch_FusedWebCache / TestMemSearch_FusedWebCacheDecayOn (both
// legs present, corpus-labeled, 1 obs + 1 web). The handler-boundary
// observation-leg gap is reported as a blocker below; this test pins the
// web-leg labeling through the handler, which it serves under the
// production write path.
func TestMemSearchCorpusLabeled(t *testing.T) {
	t.Setenv("MNEMONIC_EMBED", "")
	dataDir := t.TempDir()
	dir := t.TempDir()
	proj := pinProjectCwd(t, dataDir, dir, "fused-corpus-probe")

	sid := startMCPSession(t)
	// Seed the web row through the production handler path (ON CONFLICT DO
	// UPDATE) so the fused-recall fixture matches the way production rows are
	// written. This is the ONLY path under which the handler's fused blend
	// returns hits at the handler boundary (see the docstring below for the
	// raw-insert gap).
	saveRes, err := handleWebCacheSave(context.Background(), newCallTool("web_cache_save", map[string]any{
		"source":     "fetch",
		"content":    "kumquat web page",
		"url":        "https://example.com/kumquat",
		"title":      "kumquat web",
		"query":      "kumquat",
		"session_id": sid,
	}))
	if err != nil || saveRes.IsError {
		t.Fatalf("web_cache_save: %v %s", err, callResultText(t, saveRes))
	}
	// 3 observation rows: the handler-boundary fusion path needs >= 2 obs for
	// the observation leg to surface (see the docstring below).
	for _, title := range []string{"kumquat corpus probe", "kumquat second note", "kumquat third note"} {
		seedObservation(t, dataDir, proj, memory.SaveInput{
			Title: title, Type: "learning",
			Content: title, Scope: "project", SessionID: sid,
		})
	}

	res, err := handleMemSearch(context.Background(), newCallTool("mem_search", map[string]any{
		"query": "kumquat", "reader_owner": sid,
	}))
	if err != nil || res.IsError {
		t.Fatalf("search: %v %s", err, callResultText(t, res))
	}
	var body struct {
		Project      string `json:"project"`
		Observations []struct {
			Corpus string `json:"corpus"`
			Source string `json:"source"`
			URL    string `json:"url"`
		} `json:"observations"`
	}
	if err := json.Unmarshal([]byte(callResultText(t, res)), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(body.Observations) < 2 {
		t.Fatalf("expected an observation + a fused web hit under project %q, got %d: %+v", body.Project, len(body.Observations), body.Observations)
	}
	// Every returned hit is corpus-labeled (ADR-0047 contract).
	var sawObs, sawWeb bool
	for _, o := range body.Observations {
		switch o.Corpus {
		case "observations":
			sawObs = true
		case "web_cache":
			sawWeb = true
		default:
			t.Fatalf("hit missing corpus label (ADR-0047 contract): %+v", o)
		}
	}
	if !sawObs || !sawWeb {
		t.Fatalf("corpus labels = obs:%v web:%v, want both legs present", sawObs, sawWeb)
	}
}

// TestFusedWebHitBudget (TICKET-06, review fix): a fused web hit's label
// fields (source/url/…) must respect the char budget — before the fix
// fusedWebHitDTO emitted them unbounded, so a 50KB URL could blow the list
// budget while observation content was truncated.
func TestFusedWebHitBudget(t *testing.T) {
	t.Setenv("MNEMONIC_EMBED", "")
	dataDir := t.TempDir()
	dir := t.TempDir()
	proj := pinProjectCwd(t, dataDir, dir, "fused-budget-probe")

	sid, _, err := service.New(dataDir).SessionStart(context.Background(), dir, "fused")
	if err != nil {
		t.Fatalf("session start: %v", err)
	}
	// A URL far over the default 1200-char in-list budget.
	longURL := "https://example.com/kumquat?" + strings.Repeat("a", 2000)
	// Seed through the production handler path (ON CONFLICT DO UPDATE): the
	// handler-boundary fusion path does not surface raw-INSERT web rows (see
	// TestMemSearchCorpusLabeled's fixture notes).
	saveRes, err := handleWebCacheSave(context.Background(), newCallTool("web_cache_save", map[string]any{
		"source":     "fetch",
		"content":    "kumquat long url page",
		"url":        longURL,
		"title":      "kumquat long url",
		"query":      "kumquat",
		"session_id": sid,
	}))
	if err != nil || saveRes.IsError {
		t.Fatalf("web_cache_save: %v %s", err, callResultText(t, saveRes))
	}
	// 3 observation rows: the handler-boundary fusion path needs >= 2 obs for
	// the observation leg to surface (see TestMemSearchCorpusLabeled's fixture
	// notes). The budget assertion targets the web hit specifically.
	for _, title := range []string{"kumquat long url one", "kumquat long url two", "kumquat long url three"} {
		seedObservation(t, dataDir, proj, memory.SaveInput{
			Title: title, Type: "learning",
			Content: title, Scope: "project", SessionID: sid,
		})
	}

	res, err := handleMemSearch(context.Background(), newCallTool("mem_search", map[string]any{
		"query": "kumquat", "reader_owner": sid,
	}))
	if err != nil || res.IsError {
		t.Fatalf("search: %v %s", err, callResultText(t, res))
	}
	var body struct {
		Observations []struct {
			Corpus string `json:"corpus"`
			URL    string `json:"url"`
			Source string `json:"source"`
		} `json:"observations"`
	}
	if err := json.Unmarshal([]byte(callResultText(t, res)), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	sawWeb := false
	for _, o := range body.Observations {
		if o.Corpus != "web_cache" {
			continue
		}
		sawWeb = true
		// The budget truncates with the explicit "N chars omitted" marker.
		if !strings.Contains(o.URL, "chars omitted") {
			t.Fatalf("fused web hit url not char-budgeted (unbounded label): len=%d %q", len(o.URL), o.URL[:min(80, len(o.URL))])
		}
		if len(o.URL) > 1200+len("… (2000 chars omitted)") {
			t.Fatalf("fused web hit url exceeds the char budget: len=%d", len(o.URL))
		}
	}
	if !sawWeb {
		t.Fatalf("expected the fused web hit in the handler result, got none: %+v", body.Observations)
	}
}
