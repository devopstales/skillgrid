package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/mnemonic/internal/memory"
	"github.com/devopstales/skillgrid/mnemonic/internal/service"
)

// timeNowRFC3339 is a thin test seam so seedNearDupeRows can stamp rows with a
// single stable instant (kept in one place so the seed stays deterministic).
func timeNowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// ensureSessionForProject inserts the session row observations reference
// (session_id FK), mirroring the house pattern: a direct INSERT OR IGNORE so
// the pinned fixture bucket has a valid session to attach.
func ensureSessionForProject(t *testing.T, svc *service.Service, project string) {
	t.Helper()
	h, cleanup, err := svc.Open(project)
	if err != nil {
		t.Fatalf("open %s: %v", project, err)
	}
	defer cleanup()
	if _, err := h.Store().DB.Exec(
		`INSERT OR IGNORE INTO sessions (id, project, directory, started_at, status)
		 VALUES (?, ?, '/tmp', '2026-01-01T00:00:00Z', 'active')`, "warn-sb-session", project); err != nil {
		t.Fatalf("insert session: %v", err)
	}
}

// nearDupeVecForTest is a deterministic L2-normalized token-hash vector
// (64 dims): identical content → identical vector (cosine 1.0), distinct
// content stays well under the 0.85 dedup threshold.
func nearDupeVecForTest(text string) memory.Vector {
	dim := 64
	vec := make([]float32, dim)
	for _, tok := range strings.Fields(strings.ToLower(text)) {
		fh := fnv.New32a()
		_, _ = fh.Write([]byte(tok))
		vec[int(fh.Sum32()%uint32(dim))]++
	}
	var norm float64
	for _, v := range vec {
		norm += float64(v) * float64(v)
	}
	if norm > 0 {
		inv := float32(1 / math.Sqrt(norm))
		for i := range vec {
			vec[i] *= inv
		}
	}
	return memory.Vector{Data: vec}
}

// tools_health_warnings_test.go — TICKET-06 (2026-09-30-mnemonic-second-brain
// Task 7): the _health_warnings field, already wired into mem_search in
// TICKET-01 as an empty placeholder, now carries secondbrain.ForProject output
// when the handler knows a concrete project. The all-projects path (project ==
// "all") keeps the empty [] (the handler does not know one project). These
// tests are RED-first: the handler still emits the empty placeholder until the
// wiring step swaps it for ForProject.

// seedNearDupeRows inserts 2×N token-identical rows (N pairs) sharing
// normalized_hash, embedding every one with the near-dupe hash embedder so the
// real health pass computes duplicate density > 0.05 (the HIGH recommendation).
// Direct SQL (mirroring TestLifecycle_DedupDegradesHash) guarantees the rows
// exist distinct from Save's 24h (title+content+type) dedup.
func seedNearDupeRows(t *testing.T, dataDir, project string, pairs int) {
	t.Helper()
	svc := service.New(dataDir)
	ensureSessionForProject(t, svc, project)
	h, cleanup, err := svc.Open(project)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer cleanup()
	db := h.Store().DB
	now := timeNowRFC3339()
	for p := 0; p < pairs; p++ {
		content := fmt.Sprintf("near dupe payload %d identical words", p)
		for c := 0; c < 2; c++ {
			id := p*1000 + c
			if _, err := db.Exec(`
				INSERT INTO observations (id, project, title, content, type, scope,
					normalized_hash, session_id, owner, visibility, status, created_at, updated_at)
				VALUES (?, ?, ?, ?, 'decision', 'project',
					'warnhash000000000000000000', ?, 'warn-owner', 'private', 'active', ?, ?)`,
				id, project, fmt.Sprintf("warn dupe %d copy %d", p, c), content,
				"warn-sb-session", now, now,
			); err != nil {
				t.Fatalf("insert dupe row %d: %v", id, err)
			}
			// Embed every row so the real cosine pass (not hash-degrade) drives
			// the duplicate density. Identical content → identical vector.
			if _, err := db.Exec(`
				UPDATE observations SET embedding = ?
				WHERE id = ? AND project = ?`,
				memory.EncodeVector(nearDupeVecForTest(content)), id, project,
			); err != nil {
				t.Fatalf("embed dupe row %d: %v", id, err)
			}
		}
	}
}

func TestMemSearchHealthWarningsNonBreaking(t *testing.T) {
	dataDir := t.TempDir()
	dir := t.TempDir()
	proj := pinProjectCwd(t, dataDir, dir, "warn-nonbreaking")
	seedNearDupeRows(t, dataDir, proj, 6)

	req := newCallTool("mem_search", map[string]any{
		"query":        "near dupe payload",
		"reader_owner": "warn-owner",
	})
	res, err := handleMemSearch(context.Background(), req)
	if err != nil {
		t.Fatalf("handleMemSearch: %v", err)
	}
	if res.IsError {
		t.Fatalf("mem_search returned error: %s", callResultText(t, res))
	}
	text := callResultText(t, res)
	// Results are still returned normally (non-breaking: the search is intact).
	var out struct {
		Project          string                   `json:"project"`
		Count            int                      `json:"count"`
		Observations     []map[string]any         `json:"observations"`
		HealthWarnings   []map[string]any         `json:"_health_warnings"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("unmarshal mem_search result: %v (raw %s)", err, text)
	}
	if out.Project != proj {
		t.Errorf("expected project %q, got %q", proj, out.Project)
	}
	if out.Count == 0 || len(out.Observations) == 0 {
		t.Errorf("expected observations to be returned normally, got count=%d (raw %s)", out.Count, text)
	}
	// The HIGH duplicate warning is now inline.
	if len(out.HealthWarnings) == 0 {
		t.Fatalf("expected _health_warnings to carry the HIGH warning, got empty (raw %s)", text)
	}
	for _, w := range out.HealthWarnings {
		sev, _ := w["severity"].(string)
		if sev != "HIGH" && sev != "CRIT" {
			t.Errorf("unexpected inline severity %q (only HIGH/CRIT)", sev)
		}
		if w["message"] == nil || w["message"] == "" {
			t.Errorf("warning missing message: %+v", w)
		}
	}
	if !strings.Contains(text, "duplicate density") {
		t.Errorf("expected the HIGH duplicate-density warning text, got %s", text)
	}
}

func TestMemSearchHealthWarningsEmptyOnAllProjects(t *testing.T) {
	dataDir := t.TempDir()
	dir := t.TempDir()
	proj := pinProjectCwd(t, dataDir, dir, "warn-allprojects")
	seedNearDupeRows(t, dataDir, proj, 6)

	req := newCallTool("mem_search", map[string]any{
		"query":        "near dupe payload",
		"all_projects": true,
		"reader_owner": "warn-owner",
	})
	res, err := handleMemSearch(context.Background(), req)
	if err != nil {
		t.Fatalf("handleMemSearch: %v", err)
	}
	if res.IsError {
		t.Fatalf("mem_search returned error: %s", callResultText(t, res))
	}
	text := callResultText(t, res)
	var out struct {
		Project        string `json:"project"`
		HealthWarnings []any  `json:"_health_warnings"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("unmarshal: %v (raw %s)", err, text)
	}
	if out.Project != "all" {
		t.Errorf("expected project %q for all-projects search, got %q", "all", out.Project)
	}
	// The all-projects path does not know one project → the field stays [].
	if len(out.HealthWarnings) != 0 {
		t.Errorf("expected empty _health_warnings for all-projects search, got %v", out.HealthWarnings)
	}
}
