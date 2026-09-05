package service_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/tiered"
)

func seedTieredDoc(t *testing.T, dataDir, project, name, body string) (fullPath string, st *store.Store) {
	t.Helper()
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	root := filepath.Join(dataDir, "content")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	fullPath = filepath.Join(root, name)
	if err := os.WriteFile(fullPath, []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	ts := &tiered.Store{DB: st.DB, Summarizer: tiered.HeuristicSummarizer{}}
	if err := ts.GenerateTiers(context.Background(), project, fullPath, name); err != nil {
		t.Fatalf("tiers: %v", err)
	}
	return fullPath, st
}

func insertLTM(t *testing.T, st *store.Store, project, fullPath, title string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	abs, over := tiered.SidecarPaths(fullPath)
	if _, err := st.DB.Exec(`
		INSERT INTO long_term_memories (project, title, full_path, abstract_path, overview_path, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		project, title, fullPath, abs, over, now, now); err != nil {
		t.Fatalf("ltm: %v", err)
	}
}

func TestSemanticSearchL1OnlyOverview(t *testing.T) {
	dataDir := t.TempDir()
	project := "retproj"
	l2Body := "# Secret L2 body that must not leak\n\nOverview worthy paragraph about widgets.\n"
	path, st := seedTieredDoc(t, dataDir, project, "widgets.md", l2Body)
	defer st.Close()
	insertLTM(t, st, project, path, "widgets")

	svc := service.New(dataDir)
	out, err := svc.SemanticSearch(context.Background(), project, "widgets", "", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(out.Results) == 0 {
		t.Fatal("expected results")
	}
	raw, _ := json.Marshal(out)
	var probe map[string]any
	if err := json.Unmarshal(raw, &probe); err != nil {
		t.Fatal(err)
	}
	results, _ := probe["results"].([]any)
	if len(results) == 0 {
		t.Fatal("no results")
	}
	first, _ := results[0].(map[string]any)
	if _, hasContent := first["content"]; hasContent {
		t.Fatal("semantic_search must not include content/L2 body field")
	}
	hit := out.Results[0]
	if hit.Overview == "" || hit.Abstract == "" || hit.FullPath == "" {
		t.Fatalf("incomplete hit: %+v", hit)
	}
	if hit.Overview == l2Body {
		t.Fatal("overview equals full L2 body")
	}
	if out.TrailID == 0 {
		t.Fatal("expected trail_id")
	}
}

func TestLoadFullDetailsL2(t *testing.T) {
	dataDir := t.TempDir()
	project := "retproj2"
	body := "# Full markdown content here\n"
	path, st := seedTieredDoc(t, dataDir, project, "full.md", body)
	defer st.Close()
	insertLTM(t, st, project, path, "full")

	svc := service.New(dataDir)
	got, err := svc.LoadFullDetails(context.Background(), project, path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got != body {
		t.Fatalf("got %q want %q", got, body)
	}
}

func TestEmbedOffFallbackTrail(t *testing.T) {
	t.Setenv("MNEMONIC_EMBED", "")
	dataDir := t.TempDir()
	project := "embedoff"
	path, st := seedTieredDoc(t, dataDir, project, "alpha.md", "# Alpha\n\nUniqueKeywordZed\n")
	defer st.Close()
	insertLTM(t, st, project, path, "alpha")

	svc := service.New(dataDir)
	out, err := svc.SemanticSearch(context.Background(), project, "UniqueKeywordZed", service.CorpusLTM, 5)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if out.TrailID == 0 {
		t.Fatal("trail required when embeddings off")
	}
	if len(out.Results) == 0 {
		t.Fatal("expected fallback hit")
	}
}

func TestUnknownPathNotFound(t *testing.T) {
	dataDir := t.TempDir()
	project := "nf"
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatal(err)
	}
	st.Close()
	svc := service.New(dataDir)
	_, err = svc.LoadFullDetails(context.Background(), project, filepath.Join(dataDir, "missing.md"))
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("want not found, got %v", err)
	}
}

func TestCorpusLTMFilter(t *testing.T) {
	dataDir := t.TempDir()
	project := "corpus"
	ltmPath, st := seedTieredDoc(t, dataDir, project, "ltm.md", "# LTM doc\n\nLongTermOnlyToken\n")
	defer st.Close()
	insertLTM(t, st, project, ltmPath, "ltm")

	other := filepath.Join(dataDir, "content", "other.md")
	if err := os.WriteFile(other, []byte("# Other\n\nOtherOnlyToken\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ts := &tiered.Store{DB: st.DB, Summarizer: tiered.HeuristicSummarizer{}}
	if err := ts.GenerateTiers(context.Background(), project, other, "other"); err != nil {
		t.Fatal(err)
	}

	svc := service.New(dataDir)
	out, err := svc.SemanticSearch(context.Background(), project, "OtherOnlyToken", "", 10)
	if err != nil {
		t.Fatal(err)
	}
	for _, hit := range out.Results {
		if hit.FullPath == other {
			t.Fatal("default corpus should not include non-LTM tiered path")
		}
	}
}

func TestCorpusAllTierFilter(t *testing.T) {
	dataDir := t.TempDir()
	project := "corpusall"
	ltmPath, st := seedTieredDoc(t, dataDir, project, "ltm.md", "# LTM\n\nShared\n")
	defer st.Close()
	insertLTM(t, st, project, ltmPath, "ltm")

	other := filepath.Join(dataDir, "content", "other.md")
	if err := os.WriteFile(other, []byte("# Other\n\nAllCorpusUniqueToken\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ts := &tiered.Store{DB: st.DB, Summarizer: tiered.HeuristicSummarizer{}}
	if err := ts.GenerateTiers(context.Background(), project, other, "other"); err != nil {
		t.Fatal(err)
	}

	svc := service.New(dataDir)
	out, err := svc.SemanticSearch(context.Background(), project, "AllCorpusUniqueToken", service.CorpusAll, 10)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, hit := range out.Results {
		if hit.FullPath == other {
			found = true
		}
	}
	if !found {
		t.Fatal("all corpus should include tiered non-LTM path")
	}
}
