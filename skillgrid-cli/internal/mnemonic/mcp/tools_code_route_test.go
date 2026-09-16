package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/codeindex"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

// routeMCPFixture indexes a small web app (Express route + Next.js navigation)
// and pins the project so the temp-dir fixture resolves to one stable bucket.
func routeMCPFixture(t *testing.T) string {
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
	write("server.js", "function listUsers() { return []; }\n\napp.get('/users', listUsers);\napp.get('/mystery', unservedHandler);\n")
	write("page.tsx", "import { useRouter } from 'next/router';\n\nexport function Page() {\n  const router = useRouter();\n  return (\n    <button onClick={() => router.push('/about')}>About</button>\n    <Link href='/settings'>Settings</Link>\n  );\n}\n")
	// The default config include is **/*.go, **/*.ts, **/*.tsx, **/*.md — it
	// does not scan .js. Add an indexing.yaml so the Express fixture (server.js)
	// is indexed, matching the codeindex route fixture's include set.
	if err := os.MkdirAll(filepath.Join(abs, "config.d"), 0o755); err != nil {
		t.Fatalf("mkdir config.d: %v", err)
	}
	write("config.d/indexing.yaml", "mnemonic:\n  include:\n    - '**/*.go'\n    - '**/*.js'\n    - '**/*.ts'\n    - '**/*.tsx'\n    - '**/*.py'\n    - '**/*.rb'\n    - '**/*.java'\n    - '**/*.svelte'\n  exclude:\n    - '**/node_modules/**'\n    - '**/.git/**'\n")
	t.Setenv("MNEMONIC_PROJECT", "routemcp-probe")
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

// TestRouteToolsRegistered covers @step-01 (Scenario: Route tools register and
// code_search stays stable): the route/navigates query tools register with
// distinct code_* names; code_search name + required `query` param schema is
// unchanged; 005/008 tools stay intact; bad route args are rejected with a
// clear validation error.
func TestRouteToolsRegistered(t *testing.T) {
	routeMCPFixture(t)

	// Distinct code_* route tool names registered.
	tools := NewServer().ListTools()
	for _, name := range []string{"code_route", "code_navigates"} {
		if _, ok := tools[name]; !ok {
			t.Errorf("expected route tool %q to be registered", name)
		}
	}

	// code_search name + required `query` param schema unchanged (005 baseline).
	assertCodeToolStable(t, codeSearchTool(), "code_search", []string{"query"})

	// 005/008 tools stay intact on the live server (baseline lock).
	for _, name := range []string{"code_search", "code_read", "code_communities", "code_impact", "code_explore"} {
		if _, ok := tools[name]; !ok {
			t.Errorf("005/008 tool %q no longer registered", name)
		}
	}
}

// TestCodeRouteQuery covers @step-01 (route query surfaces the URL pattern for
// a handler, and a handler's callers surface the route). A handler name
// returns the routes that serve it.
func TestCodeRouteQuery(t *testing.T) {
	routeMCPFixture(t)

	// Query by handler: the /users route (which references listUsers) surfaces.
	res, err := handleCodeRoute(context.Background(), newCallTool("code_route", map[string]any{"handler": "listUsers"}))
	if err != nil {
		t.Fatalf("handleCodeRoute: %v", err)
	}
	if res.IsError {
		t.Fatalf("code_route errored: %s", callResultText(t, res))
	}
	var out struct {
		Routes []struct {
			Path          string `json:"path"`
			Handler       string `json:"handler"`
			Confidence    string `json:"references_confidence"`
			Framework     string `json:"framework"`
		} `json:"routes"`
	}
	if err := json.Unmarshal([]byte(callResultText(t, res)), &out); err != nil {
		t.Fatalf("code_route output not JSON: %v (text %s)", err, callResultText(t, res))
	}
	if len(out.Routes) == 0 {
		t.Fatalf("expected route(s) for handler listUsers, got none")
	}
	foundUsers := false
	for _, r := range out.Routes {
		if r.Path == "/users" {
			foundUsers = true
			if r.Handler != "listUsers" {
				t.Errorf("route /users handler = %q, want listUsers", r.Handler)
			}
			if r.Confidence != "EXTRACTED" {
				t.Errorf("route /users references confidence = %q, want EXTRACTED", r.Confidence)
			}
		}
	}
	if !foundUsers {
		t.Errorf("expected the /users route to surface for handler listUsers, got %+v", out.Routes)
	}
}

// TestCodeNavigatesQuery covers @step-01 (navigates query returns screen ->
// screen edges with confidence labels). A to-screen query returns the senders.
func TestCodeNavigatesQuery(t *testing.T) {
	routeMCPFixture(t)

	res, err := handleCodeNavigates(context.Background(), newCallTool("code_navigates", map[string]any{"to": "/about"}))
	if err != nil {
		t.Fatalf("handleCodeNavigates: %v", err)
	}
	if res.IsError {
		t.Fatalf("code_navigates errored: %s", callResultText(t, res))
	}
	var out struct {
		Navigations []struct {
			Screen     string `json:"screen"`
			Confidence string `json:"confidence"`
			From       string `json:"from"`
		} `json:"navigations"`
	}
	if err := json.Unmarshal([]byte(callResultText(t, res)), &out); err != nil {
		t.Fatalf("code_navigates output not JSON: %v (text %s)", err, callResultText(t, res))
	}
	if len(out.Navigations) == 0 {
		t.Fatalf("expected navigation(s) to /about, got none")
	}
	for _, n := range out.Navigations {
		if n.Confidence != "EXTRACTED" {
			t.Errorf("navigates to /about (literal) should be EXTRACTED, got %q", n.Confidence)
		}
	}
}

// TestRouteToolsBadArgs covers @step-01 (bad route args rejected clearly):
// code_route with neither handler nor path, and code_navigates with neither
// from nor to, are rejected with a clear validation error (not an invented
// result).
func TestRouteToolsBadArgs(t *testing.T) {
	routeMCPFixture(t)

	res, err := handleCodeRoute(context.Background(), newCallTool("code_route", map[string]any{}))
	if err != nil {
		t.Fatalf("handleCodeRoute dispatch: %v", err)
	}
	if !res.IsError {
		t.Errorf("code_route with no handler/path should be a validation error, got: %s", callResultText(t, res))
	}
	if text := callResultText(t, res); !strings.Contains(strings.ToLower(text), "handler") && !strings.Contains(strings.ToLower(text), "path") {
		t.Errorf("bad-arg error should name the handler/path problem, got: %s", text)
	}

	res2, err := handleCodeNavigates(context.Background(), newCallTool("code_navigates", map[string]any{}))
	if err != nil {
		t.Fatalf("handleCodeNavigates dispatch: %v", err)
	}
	if !res2.IsError {
		t.Errorf("code_navigates with no from/to should be a validation error, got: %s", callResultText(t, res2))
	}
	if text := callResultText(t, res2); !strings.Contains(strings.ToLower(text), "from") && !strings.Contains(strings.ToLower(text), "to") {
		t.Errorf("bad-arg error should name the from/to problem, got: %s", text)
	}
}
