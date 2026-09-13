package main

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// TestMemRelationsCLI covers 14.2: `mem relations <observation_id>` lists the
// related observations with their relation type and confidence. An observation
// A related to B (mentions), C (depends_on), and D (contradicts) shows all
// three in the output; an observation with no relations produces an empty list;
// a missing observation id fails with a clear error.
func TestMemRelationsCLI(t *testing.T) {
	dataDir := t.TempDir()
	project := "memcli-relations"

	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := st.DB.Exec(`
		INSERT INTO sessions (id, project, directory, started_at, status, title, summary)
		VALUES ('sess-rel', ?, '/tmp', '2026-01-01T00:00:00Z', 'ended', 'rel session', 'rel session summary')`,
		project); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	mem := memory.New(st, project)
	ctx := context.Background()
	save := func(title, content string) int64 {
		id, err := mem.Save(ctx, memory.SaveInput{
			SessionID: "sess-rel",
			Type:      "decision",
			Title:     title,
			Content:   content,
		})
		if err != nil {
			t.Fatalf("save %q: %v", title, err)
		}
		return id
	}
	a := save("rel A", "content A")
	b := save("rel B", "content B")
	c := save("rel C", "content C")
	d := save("rel D", "content D")
	iso := save("rel isolated", "content iso")

	if _, err := mem.AddRelation(ctx, a, b, "mentions", 0.9); err != nil {
		t.Fatalf("AddRelation A->B: %v", err)
	}
	if _, err := mem.AddRelation(ctx, a, c, "depends_on", 0.8); err != nil {
		t.Fatalf("AddRelation A->C: %v", err)
	}
	if _, err := mem.AddRelation(ctx, a, d, "contradicts", 0.7); err != nil {
		t.Fatalf("AddRelation A->D: %v", err)
	}
	st.Close()

	// `mem relations A` lists B, C, D with their relation types and confidences.
	out := runMemCLI(t, dataDir, "relations", strconv.FormatInt(a, 10), "--project", project, "--dir", dataDir)
	for _, title := range []string{"rel B", "rel C", "rel D"} {
		if !strings.Contains(out, title) {
			t.Fatalf("mem relations A missing related obs %q: %s", title, out)
		}
	}
	for _, rt := range []string{"mentions", "depends_on", "contradicts"} {
		if !strings.Contains(out, rt) {
			t.Fatalf("mem relations A missing relation type %q: %s", rt, out)
		}
	}
	if !strings.Contains(out, "0.9") || !strings.Contains(out, "0.8") || !strings.Contains(out, "0.7") {
		t.Fatalf("mem relations A missing confidence scores: %s", out)
	}

	// `--min-confidence 0.85` filters down to only the 0.9 edge (14.3 CLI).
	out = runMemCLI(t, dataDir, "relations", strconv.FormatInt(a, 10), "--min-confidence", "0.85", "--project", project, "--dir", dataDir)
	if !strings.Contains(out, "rel B") {
		t.Fatalf("mem relations A --min-confidence 0.85 must keep the 0.9 edge: %s", out)
	}
	if strings.Contains(out, "rel C") || strings.Contains(out, "rel D") {
		t.Fatalf("mem relations A --min-confidence 0.85 must drop the 0.8/0.7 edges: %s", out)
	}

	// An observation with no relations produces an empty list (graceful).
	out = runMemCLI(t, dataDir, "relations", strconv.FormatInt(iso, 10), "--project", project, "--dir", dataDir)
	if !strings.Contains(out, `"relations": []`) {
		t.Fatalf("mem relations for an isolated obs must print an empty list: %s", out)
	}

	// A missing observation id fails with a clear error (non-zero exit).
	out, err = runMemCLIExpectError(t, dataDir, "relations", "999999", "--project", project, "--dir", dataDir)
	if err == nil {
		t.Fatalf("mem relations with a missing id should fail, got: %s", out)
	}
	if !strings.Contains(out, "not found") {
		t.Fatalf("missing observation id should be rejected clearly, got: %s", out)
	}
}
