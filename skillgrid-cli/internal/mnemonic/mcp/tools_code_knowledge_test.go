package mcp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/codeindex"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/knowledge"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

// runKnowledgePasses runs the knowledge extractors over the files under root
// (the indexer hook 03.8 calls this in-tx; the fixture calls it post-index to
// seed the knowledge tables). It opens the project store, scans root for
// doc/config/SQL files, and persists the knowledge nodes + edges.
func runKnowledgePasses(t *testing.T, svc *service.Service, projectID, root string) {
	t.Helper()
	h, cleanup, err := svc.Open(projectID)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer cleanup()
	st := knowledge.NewStore(h.Store().DB)
	var files []knowledge.FileInput
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, "config.d/") {
			return nil
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		files = append(files, knowledge.FileInput{Path: rel, Contents: contents})
		return nil
	})
	if _, err := knowledge.RunPasses(context.Background(), st, files); err != nil {
		t.Fatalf("knowledge passes: %v", err)
	}
}

// knowledgeMCPFixture indexes a small project with a doc (markdown link), a
// config (a configures ref), and a SQL schema (DDL + DML) and pins the
// project so the temp-dir fixture resolves to one stable bucket.
func knowledgeMCPFixture(t *testing.T) string {
	t.Helper()
	dataDir := t.TempDir()
	raw := t.TempDir()
	abs, err := filepath.Abs(raw)
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	write := func(name, content string) {
		full := filepath.Join(abs, name)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	write("docs/a.md", "# A\n\nSee [B](./b.md).\n")
	write("docs/b.md", "# B\nplain doc\n")
	write("config/app.yaml", "server:\n  handler: loadUsers\n")
	write("db/schema.sql", "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT);\n")
	write("app/users.go", "package app\n\nfunc loadUsers() { return }\n\nvar sql = `SELECT id FROM users;`\n")
	if err := os.MkdirAll(filepath.Join(abs, "config.d"), 0o755); err != nil {
		t.Fatalf("mkdir config.d: %v", err)
	}
	write("config.d/indexing.yaml", "mnemonic:\n  include:\n    - '**/*.go'\n    - '**/*.md'\n    - '**/*.yaml'\n    - '**/*.sql'\n  exclude:\n    - '**/node_modules/**'\n    - '**/.git/**'\n")
	t.Setenv("MNEMONIC_PROJECT", "knowledgemcp-probe")
	svc := service.New(dataDir)
	SetService(svc)
	t.Cleanup(func() { SetService(nil) })
	codeindex.ResetFileFirstSymbol()
	if _, err := svc.RunCodeIndex(context.Background(), abs); err != nil {
		t.Fatalf("index: %v", err)
	}
	// Run the knowledge passes over the indexed files (the indexer hook 03.8
	// wires this into Indexer.Run; the fixture calls it directly to seed the
	// knowledge tables).
	runKnowledgePasses(t, svc, "knowledgemcp-probe", abs)
	oldDir, _ := os.Getwd()
	if err := os.Chdir(abs); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(oldDir) })
	return dataDir
}

// TestKnowledgeTools covers @step-03 (Scenario: Knowledge tools register and
// reject bad args): the knowledge query tools register with distinct code_*
// names; the 005 tools stay name/param-stable; bad/missing args are rejected
// clearly.
func TestKnowledgeTools(t *testing.T) {
	knowledgeMCPFixture(t)

	// Distinct code_* knowledge tool names registered.
	tools := NewServer().ListTools()
	for _, name := range []string{"code_docs", "code_configs", "code_sql_schema", "code_sql_access"} {
		if _, ok := tools[name]; !ok {
			t.Errorf("expected knowledge tool %q to be registered", name)
		}
	}

	// 005 tools stay intact on the live server (baseline lock).
	for _, name := range []string{"code_search", "code_read", "code_path", "code_explore", "code_communities", "code_processes"} {
		if _, ok := tools[name]; !ok {
			t.Errorf("005/008 tool %q no longer registered", name)
		}
	}
	// code_search name + required `query` param schema unchanged (005 baseline).
	assertCodeToolStable(t, codeSearchTool(), "code_search", []string{"query"})

	// The tool surface grows additively: 67 baseline + 4 knowledge + 2
	// affected/rename + 1 code_pdg_query + 1 code_taint + 2 governance +
	// 1 mem_layers + 2 session = 80 (all keep their names + required params).
	if len(tools) != 83 {
		t.Errorf("expected 83 tools (67 baseline + 4 knowledge + 2 affected/rename + 1 pdg_query + 1 taint + 2 governance + 1 mem_layers + 2 session + 2 status/compact), got %d", len(tools))
	}

	// code_docs returns the indexed docs (a query, not an error).
	res, err := handleCodeDocs(context.Background(), newCallTool("code_docs", map[string]any{}))
	if err != nil {
		t.Fatalf("handleCodeDocs dispatch: %v", err)
	}
	if res.IsError {
		t.Errorf("code_docs errored: %s", callResultText(t, res))
	}
	if text := callResultText(t, res); !strings.Contains(text, "docs/a.md") {
		t.Errorf("code_docs should surface the indexed doc, got: %s", text)
	}

	// code_sql_schema returns the indexed SQL tables.
	res, err = handleCodeSQLSchema(context.Background(), newCallTool("code_sql_schema", map[string]any{"table": "users"}))
	if err != nil {
		t.Fatalf("handleCodeSQLSchema dispatch: %v", err)
	}
	if res.IsError {
		t.Errorf("code_sql_schema errored: %s", callResultText(t, res))
	}
	if text := callResultText(t, res); !strings.Contains(text, "users") {
		t.Errorf("code_sql_schema should surface the users table, got: %s", text)
	}
}
