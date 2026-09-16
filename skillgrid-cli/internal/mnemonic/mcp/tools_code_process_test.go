package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/codeindex"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/process"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

// processMCPFixture indexes a small web app (an Express route serving
// handleListUsers, which calls loadUsers -> queryDB) and pins the project so
// the temp-dir fixture resolves to one stable bucket. It returns the data dir.
func processMCPFixture(t *testing.T) string {
	t.Helper()
	dataDir := t.TempDir()
	raw := t.TempDir()
	abs, err := filepath.Abs(raw)
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(abs, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	// An Express route serving handleListUsers, plus the call chain.
	write("server.js", "function handleListUsers() { return loadUsers(); }\nfunction loadUsers() { return queryDB(); }\nfunction queryDB() { return [1,2,3]; }\n\napp.get('/users', handleListUsers);\n")
	if err := os.MkdirAll(filepath.Join(abs, "config.d"), 0o755); err != nil {
		t.Fatalf("mkdir config.d: %v", err)
	}
	write("config.d/indexing.yaml", "mnemonic:\n  include:\n    - '**/*.go'\n    - '**/*.js'\n    - '**/*.ts'\n    - '**/*.tsx'\n  exclude:\n    - '**/node_modules/**'\n    - '**/.git/**'\n")
	t.Setenv("MNEMONIC_PROJECT", "processmcp-probe")
	svc := service.New(dataDir)
	SetService(svc)
	t.Cleanup(func() { SetService(nil) })
	codeindex.ResetFileFirstSymbol()
	if _, err := svc.RunCodeIndex(context.Background(), abs); err != nil {
		t.Fatalf("index: %v", err)
	}
	oldDir, _ := os.Getwd()
	if err := os.Chdir(abs); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(oldDir) })
	return dataDir
}

// seedEntries resolves the fixture's entry points (the route's handler, which
// has a traceable call chain) and returns them as process seeds.
func seedEntries(t *testing.T, svc *service.Service, projectID string) []process.Entry {
	t.Helper()
	h, cleanup, err := svc.Open(projectID)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer cleanup()
	db := h.Store().DB
	var out []process.Entry
	// 010's entry points: kind='route' symbols. For each resolved route, the
	// seed is the route's handler (the traceable call-chain start).
	rows, err := db.Query(`SELECT e.to_id FROM edges e JOIN symbols r ON r.id = e.from_id WHERE r.kind = 'route' AND e.kind = 'references' AND e.to_id IS NOT NULL`)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var to int64
		if err := rows.Scan(&to); err != nil {
			return out
		}
		out = append(out, process.Entry{SymbolID: to, Kind: "handler"})
	}
	return out
}

// runProcessPass seeds entry points and runs the process pass once.
func runProcessPass(t *testing.T, svc *service.Service, projectID string) {
	t.Helper()
	entries := seedEntries(t, svc, projectID)
	if len(entries) == 0 {
		t.Fatalf("no entry points found in fixture")
	}
	if _, err := svc.CodeProcessTrace(context.Background(), projectID, entries, nil, process.RunOptions{}); err != nil {
		t.Fatalf("CodeProcessTrace: %v", err)
	}
}

// TestExplainSymbolProcess covers 02.1 (Scenario: Symbol explanation surfaces
// process participation and existing search tools stay stable): after the
// process pass, code_explain_symbol (code_orient) surfaces which processes the
// symbol participates in (step N/M); the 005 tool names + required params are
// unchanged; the process tools register with distinct code_* names.
func TestExplainSymbolProcess(t *testing.T) {
	processMCPFixture(t)
	svc := service.New(t.TempDir())
	// Re-resolve: the fixture set the global svc; grab it via rootService.
	_ = svc
	got, _ := rootService()
	projectID := "processmcp-probe"
	runProcessPass(t, got, projectID)

	// code_explain_symbol (code_orient) on loadUsers (a mid-chain symbol that
	// participates in the traced flow) surfaces its process participation.
	res, err := handleCodeOrient(context.Background(), newCallTool("code_orient", map[string]any{"symbol": "loadUsers"}))
	if err != nil {
		t.Fatalf("handleCodeOrient: %v", err)
	}
	if res.IsError {
		t.Fatalf("code_orient errored: %s", callResultText(t, res))
	}
	var out struct {
		Found     bool `json:"found"`
		Processes []struct {
			Process string `json:"process"`
			Step    int    `json:"step"`
			Total   int    `json:"total"`
		} `json:"processes"`
	}
	if err := json.Unmarshal([]byte(callResultText(t, res)), &out); err != nil {
		t.Fatalf("code_orient output not JSON: %v (text %s)", err, callResultText(t, res))
	}
	if !out.Found {
		t.Fatalf("expected loadUsers to be found")
	}
	if len(out.Processes) == 0 {
		t.Fatalf("expected loadUsers to surface at least one process participation, got none (text %s)", callResultText(t, res))
	}
	for _, p := range out.Processes {
		if p.Process == "" {
			t.Errorf("process participation has no process name")
		}
		if p.Step < 1 || p.Total < p.Step {
			t.Errorf("process participation step position invalid: step=%d total=%d", p.Step, p.Total)
		}
	}

	// 005 tool names + required params unchanged (baseline lock).
	assertCodeToolStable(t, codeSearchTool(), "code_search", []string{"query"})
	assertCodeToolStable(t, codeOrientTool(), "code_orient", []string{"symbol"})

	// Process tools register with distinct code_* names.
	tools := NewServer().ListTools()
	for _, name := range []string{"code_processes", "code_process"} {
		if _, ok := tools[name]; !ok {
			t.Errorf("expected process tool %q to be registered", name)
		}
	}
}

// TestProcessDetail covers 02.5 (Scenario: Process detail returns the full
// step-by-step trace): code_process <name> returns the full trace with each
// hop's confidence label.
func TestProcessDetail(t *testing.T) {
	processMCPFixture(t)
	got, _ := rootService()
	projectID := "processmcp-probe"
	runProcessPass(t, got, projectID)

	// First list processes to get a real name.
	listRes, err := handleCodeProcesses(context.Background(), newCallTool("code_processes", map[string]any{}))
	if err != nil {
		t.Fatalf("handleCodeProcesses: %v", err)
	}
	if listRes.IsError {
		t.Fatalf("code_processes errored: %s", callResultText(t, listRes))
	}
	var listOut struct {
		Processes []struct {
			Name      string `json:"name"`
			Cross     bool   `json:"cross_community"`
			Label     string `json:"label"`
			LabelStat string `json:"label_status"`
			Steps     []struct {
				Step       int    `json:"step"`
				Symbol     string `json:"symbol"`
				Confidence string `json:"confidence"`
			} `json:"steps"`
		} `json:"processes"`
	}
	if err := json.Unmarshal([]byte(callResultText(t, listRes)), &listOut); err != nil {
		t.Fatalf("code_processes output not JSON: %v (text %s)", err, callResultText(t, listRes))
	}
	if len(listOut.Processes) == 0 {
		t.Fatalf("expected at least one precomputed process, got none (text %s)", callResultText(t, listRes))
	}
	name := listOut.Processes[0].Name

	// Now request it by name: the full step-by-step trace, each hop confidence-labeled.
	res, err := handleCodeProcess(context.Background(), newCallTool("code_process", map[string]any{"name": name}))
	if err != nil {
		t.Fatalf("handleCodeProcess: %v", err)
	}
	if res.IsError {
		t.Fatalf("code_process errored: %s", callResultText(t, res))
	}
	var detail struct {
		Name  string `json:"name"`
		Steps []struct {
			Step       int    `json:"step"`
			Symbol     string `json:"symbol"`
			Confidence string `json:"confidence"`
		} `json:"steps"`
	}
	if err := json.Unmarshal([]byte(callResultText(t, res)), &detail); err != nil {
		t.Fatalf("code_process output not JSON: %v (text %s)", err, callResultText(t, res))
	}
	if detail.Name != name {
		t.Errorf("code_process returned name %q, want %q", detail.Name, name)
	}
	if len(detail.Steps) < 2 {
		t.Fatalf("expected a full multi-step trace, got %d steps: %+v", len(detail.Steps), detail.Steps)
	}
	for i, s := range detail.Steps {
		if i > 0 && s.Confidence == "" {
			t.Errorf("hop %d (-> %s) has no confidence label", i, s.Symbol)
		}
	}
}

// TestProcessArgs covers 02.9 (Scenario: Process tools reject bad args
// clearly): code_process with a missing/empty name and an unknown name are
// rejected with a clear validation error; no process is invented.
func TestProcessArgs(t *testing.T) {
	processMCPFixture(t)

	// code_process with a missing name → clear validation error.
	res, err := handleCodeProcess(context.Background(), newCallTool("code_process", map[string]any{}))
	if err != nil {
		t.Fatalf("handleCodeProcess dispatch: %v", err)
	}
	if !res.IsError {
		t.Errorf("code_process with a missing name should be a validation error, got: %s", callResultText(t, res))
	}

	// code_process with an unknown name → clear "no process named" error
	// (not an invented process).
	res2, err := handleCodeProcess(context.Background(), newCallTool("code_process", map[string]any{"name": "no-such-process"}))
	if err != nil {
		t.Fatalf("handleCodeProcess dispatch: %v", err)
	}
	if !res2.IsError {
		t.Errorf("code_process with an unknown name should be an error (no invented process), got: %s", callResultText(t, res2))
	}
	if text := callResultText(t, res2); !strings.Contains(text, "no-such-process") {
		t.Errorf("unknown-name error should name the process, got: %s", text)
	}

	// The two tools register with distinct code_* names.
	tools := NewServer().ListTools()
	if _, ok := tools["code_processes"]; !ok {
		t.Errorf("code_processes not registered")
	}
	if _, ok := tools["code_process"]; !ok {
		t.Errorf("code_process not registered")
	}
}

// TestProcessParticipationsEmptyForUntracedSymbol is the negative control: a
// symbol that is in no traced process surfaces an empty (not fabricated)
// participation list.
func TestProcessParticipationsEmptyForUntracedSymbol(t *testing.T) {
	db := openProcessStore(t)
	// A symbol with no process_steps row.
	f, _ := db.Exec(`INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at) VALUES ('x.go',1,1,'h','now')`)
	fid, _ := f.LastInsertId()
	if _, err := db.Exec(`INSERT INTO symbols (file_id, name, kind, start_line, end_line, content_hash, uid) VALUES (?, 'orphan', 'function', 1, 1, 'h', 'orphan')`, fid); err != nil {
		t.Fatalf("seed: %v", err)
	}
	var id int64
	if err := db.QueryRow(`SELECT id FROM symbols WHERE name='orphan'`).Scan(&id); err != nil {
		t.Fatalf("lookup: %v", err)
	}
	parts, err := process.Participations(db, id)
	if err != nil {
		t.Fatalf("Participations: %v", err)
	}
	if len(parts) != 0 {
		t.Errorf("expected no participation for an untraced symbol, got %+v", parts)
	}
	_ = sql.ErrNoRows
}

// openProcessStore opens a scratch store with the process tables for the
// mcp-package negative-control test.
func openProcessStore(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	for _, s := range []string{
		`CREATE TABLE files (id INTEGER PRIMARY KEY AUTOINCREMENT, path TEXT NOT NULL, mtime_ns INTEGER, size INTEGER, content_hash TEXT, indexed_at TEXT)`,
		`CREATE TABLE symbols (id INTEGER PRIMARY KEY AUTOINCREMENT, file_id INTEGER NOT NULL, name TEXT NOT NULL, qualified_name TEXT, kind TEXT NOT NULL, language TEXT, signature TEXT, start_line INTEGER NOT NULL, end_line INTEGER NOT NULL, content_hash TEXT NOT NULL, uid TEXT NOT NULL UNIQUE)`,
		`CREATE TABLE edges (id INTEGER PRIMARY KEY AUTOINCREMENT, kind TEXT NOT NULL, from_id INTEGER NOT NULL, file_id INTEGER, to_id INTEGER, to_name TEXT, target_path TEXT, confidence TEXT NOT NULL DEFAULT 'EXTRACTED', line INTEGER)`,
		`CREATE TABLE processes (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, entry_symbol_id INTEGER, entry_kind TEXT, cross_community INTEGER NOT NULL DEFAULT 0, content_hash TEXT NOT NULL, label TEXT NOT NULL DEFAULT '', label_status TEXT NOT NULL DEFAULT 'unlabeled', stop_note TEXT NOT NULL DEFAULT '', updated_at TEXT NOT NULL)`,
		`CREATE TABLE process_steps (process_id INTEGER NOT NULL, step INTEGER NOT NULL, symbol_id INTEGER, confidence TEXT, kind TEXT, PRIMARY KEY (process_id, step))`,
		`CREATE TABLE process_meta_cache (key TEXT PRIMARY KEY, value TEXT)`,
	} {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("create: %v", err)
		}
	}
	return db
}
