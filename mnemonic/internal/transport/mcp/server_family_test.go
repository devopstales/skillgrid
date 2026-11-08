package mcp

// server_family_test.go — the per-family split pins (TICKET-03, D4):
// the initialize shape per family (memory←engram, code←gitnexus,
// projectwiki←openwiki), the exact per-family/profile tools/list sets
// with zero cross-family names, the serial stdio invariant (one tool
// call in flight, the mutation gate for WithWorkerPoolSize(1)), the
// protocol-only stdout invariant, and the two-servers-one-store
// parallelism (no database-is-locked, ≤1 sync-lease holder).

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	mcplib "github.com/mark3labs/mcp-go/mcp"

	"github.com/devopstales/skillgrid/mnemonic/internal/core/config"
	"github.com/devopstales/skillgrid/mnemonic/internal/core/store"
	"github.com/devopstales/skillgrid/mnemonic/internal/memory"
	"github.com/devopstales/skillgrid/mnemonic/internal/transport/handler"
)

// ────────────────────────────────────────────────────────────────────────────
// helpers
// ────────────────────────────────────────────────────────────────────────────

// initResult is the initialize response's result object (the wire shape
// the client sees — serverInfo identity, the instructions text, and the
// capabilities the server declares).
type initResult struct {
	ServerInfo struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"serverInfo"`
	Instructions string                     `json:"instructions"`
	Capabilities map[string]json.RawMessage `json:"capabilities"`
}

// initResultOf parses the initialize response frame (frames[0] from
// HandshakeAndCall) into the full result object.
func initResultOf(t *testing.T, frames [][]byte) initResult {
	t.Helper()
	if len(frames) == 0 {
		t.Fatal("no frames from the handshake")
	}
	var rpc struct {
		Result initResult `json:"result"`
	}
	if err := json.Unmarshal(frames[0], &rpc); err != nil {
		t.Fatalf("initialize frame is not JSON: %v (%s)", err, frames[0])
	}
	return rpc.Result
}

// stdioToolsList sends initialize + tools/list over the mcp-go message
// path (the same path the stdio loop dispatches through) and returns the
// registered tool names.
func stdioToolsList(t *testing.T, s *Server) map[string]bool {
	t.Helper()
	ctx := context.Background()
	initReq := `{"jsonrpc":"2.0","id":0,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}`
	if s.mcp.HandleMessage(ctx, json.RawMessage(initReq)) == nil {
		t.Fatal("no initialize response")
	}
	resp := s.mcp.HandleMessage(ctx, json.RawMessage(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	if resp == nil {
		t.Fatal("no tools/list response")
	}
	out, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal tools/list: %v", err)
	}
	var parsed struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("tools/list is not JSON: %v", err)
	}
	names := map[string]bool{}
	for _, tool := range parsed.Result.Tools {
		names[tool.Name] = true
	}
	return names
}

// assertNameSet checks the wire tools/list set equals want exactly (the
// exact-set pin: no missing tools, no foreign tools).
func assertNameSet(t *testing.T, label string, got, want map[string]bool) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s: %d tools, want %d", label, len(got), len(want))
	}
	for n := range want {
		if !got[n] {
			t.Errorf("%s: missing tool %q", label, n)
		}
	}
	for n := range got {
		if !want[n] {
			t.Errorf("%s: unexpected tool %q (cross-family or unlisted)", label, n)
		}
	}
}

// linePipe is a mutex-guarded byte buffer that acts as the stdio loop's
// stdout (the loop's writer goroutine and the test's reader run
// concurrently).
type linePipe struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (p *linePipe) Write(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.buf.Write(b)
}

// lines returns the complete lines written so far.
func (p *linePipe) lines() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := strings.Split(p.buf.String(), "\n")
	return out
}

// waitFor polls cond until it is true or the deadline passes.
func waitFor(t *testing.T, d time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !cond() {
		t.Fatalf("condition not met within %s", d)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// [memory-initialize-follows-engram]
// [projectwiki-initialize-follows-openwiki]
// [code-initialize-follows-gitnexus]
// ────────────────────────────────────────────────────────────────────────────

func TestInitializeShapePerFamily(t *testing.T) {
	t.Run("memory-agent", func(t *testing.T) {
		s := NewServer(newTestHandler(t), Agent, FamilyMemory)
		defer s.Close()
		frames, err := s.HandshakeAndCall(context.Background(), "mem_current_project")
		if err != nil {
			t.Fatalf("handshake: %v", err)
		}
		res := initResultOf(t, frames)
		if res.ServerInfo.Name != "mnemonic-memory" || res.ServerInfo.Version != "0.2.0" {
			t.Fatalf("memory serverInfo = %s/%s, want mnemonic-memory/0.2.0", res.ServerInfo.Name, res.ServerInfo.Version)
		}
		instr := res.Instructions
		// engram shape: every agent-profile tool is listed with its
		// one-liner; the admin tools are named as deferred.
		for _, tl := range memAgentToolLines {
			if !strings.Contains(instr, tl.name+" — "+tl.line) {
				t.Errorf("agent-mode instructions must list %q with its one-liner", tl.name)
			}
		}
		if !strings.Contains(instr, "DEFERRED TOOLS (admin profile only):") {
			t.Fatal("agent mode must carry the DEFERRED TOOLS section")
		}
		for _, tl := range memAdminToolLines {
			if !strings.Contains(instr, tl.name) {
				t.Errorf("agent mode must name the deferred admin tool %q", tl.name)
			}
		}
		// single-family: no code/wiki routing in the memory block.
		for _, foreign := range []string{"code_", "projectwiki_"} {
			if strings.Contains(instr, foreign) {
				t.Errorf("the memory block must not name %s* tools (family split)", foreign)
			}
		}
	})

	t.Run("memory-all", func(t *testing.T) {
		s := NewServer(newTestHandler(t), All, FamilyMemory)
		defer s.Close()
		frames, err := s.HandshakeAndCall(context.Background(), "mem_current_project")
		if err != nil {
			t.Fatalf("handshake: %v", err)
		}
		instr := initResultOf(t, frames).Instructions
		// all mode: the admin tools are core (listed with one-liners),
		// no DEFERRED section.
		for _, tl := range memAdminToolLines {
			if !strings.Contains(instr, tl.name+" — "+tl.line) {
				t.Errorf("all-mode instructions must list the admin tool %q as core", tl.name)
			}
		}
		if strings.Contains(instr, "DEFERRED TOOLS") {
			t.Fatal("all mode must not carry a DEFERRED section (every tool is core)")
		}
	})

	t.Run("code", func(t *testing.T) {
		s := NewServer(newTestHandler(t), Agent, FamilyCode)
		defer s.Close()
		s.RegisterCodeTools(newTestCodeHandler(t))
		frames, err := s.HandshakeAndCall(context.Background(), "code_status")
		if err != nil {
			t.Fatalf("handshake: %v", err)
		}
		res := initResultOf(t, frames)
		if res.ServerInfo.Name != "mnemonic-code" || res.ServerInfo.Version != "0.2.0" {
			t.Fatalf("code serverInfo = %s/%s, want mnemonic-code/0.2.0", res.ServerInfo.Name, res.ServerInfo.Version)
		}
		// gitnexus shape: no instructions field at all (the tool
		// descriptions carry the routing).
		if res.Instructions != "" {
			t.Fatalf("the code server must serve no instructions (gitnexus shape), got: %s", res.Instructions)
		}
		// the pinned capabilities: exactly tools/resources/prompts,
		// each empty (no list-changed, no resources, no prompts).
		if len(res.Capabilities) != 3 {
			t.Fatalf("code capabilities = %v, want exactly {tools,resources,prompts}", res.Capabilities)
		}
		for _, k := range []string{"tools", "resources", "prompts"} {
			raw, ok := res.Capabilities[k]
			if !ok {
				t.Fatalf("code capabilities missing %q: %v", k, res.Capabilities)
			}
			if string(raw) != "{}" {
				t.Errorf("code capabilities.%s = %s, want {} (gitnexus pin)", k, raw)
			}
		}
	})

	t.Run("projectwiki", func(t *testing.T) {
		s, _, _ := newProjectWikiFixture(t, Agent)
		frames, err := s.HandshakeAndCall(context.Background(), "projectwiki_status")
		if err != nil {
			t.Fatalf("handshake: %v", err)
		}
		res := initResultOf(t, frames)
		if res.ServerInfo.Name != "mnemonic-projectwiki" || res.ServerInfo.Version != "0.2.0" {
			t.Fatalf("wiki serverInfo = %s/%s, want mnemonic-projectwiki/0.2.0", res.ServerInfo.Name, res.ServerInfo.Version)
		}
		instr := res.Instructions
		// openwiki shape: the retrieval discipline + the page-job
		// lifecycle (generate/index/validate/lint).
		if !strings.Contains(instr, "Do not enumerate, preload, or search the vault at task start") {
			t.Fatal("the projectwiki block must carry the retrieval discipline")
		}
		for _, name := range []string{"projectwiki_generate", "projectwiki_index", "projectwiki_validate", "projectwiki_lint"} {
			if !strings.Contains(instr, name) {
				t.Errorf("the projectwiki block must name the lifecycle tool %s", name)
			}
		}
		// single-project vault: no workspace/link wording.
		lower := strings.ToLower(instr)
		for _, word := range []string{"workspace", "link"} {
			if strings.Contains(lower, word) {
				t.Errorf("the projectwiki block must carry no %q wording (single-project vault), got: %s", word, instr)
			}
		}
	})
}

// ────────────────────────────────────────────────────────────────────────────
// [memory-server-serves-memory-only]
// [all-profile-adds-admin-tools-only]
// [code-and-wiki-servers-are-isolated]
// ────────────────────────────────────────────────────────────────────────────

// codeAgentToolNames is the D1 code surface's agent-profile set (the 20
// working tools — the pin is the literal set, not a count).
var codeAgentToolNames = []string{
	"code_status", "code_index", "code_search", "code_explore", "code_read",
	"code_orient", "code_explain", "code_path", "code_impact",
	"code_affected", "code_grep", "code_semantic", "code_get_callers",
	"code_get_callees", "code_get_dependents", "code_get_implementors",
	"code_get_hierarchy", "code_get_tests_for", "code_resolve",
	"code_packages",
}

func TestToolIsolationPerFamily(t *testing.T) {
	memAgentWant := map[string]bool{}
	for _, tl := range memAgentToolLines {
		memAgentWant[tl.name] = true
	}
	memAllWant := map[string]bool{}
	for n := range memAgentWant {
		memAllWant[n] = true
	}
	for _, tl := range memAdminToolLines {
		memAllWant[tl.name] = true
	}
	codeAgentWant := map[string]bool{}
	for _, n := range codeAgentToolNames {
		codeAgentWant[n] = true
	}
	codeAllWant := map[string]bool{}
	for n := range codeAgentWant {
		codeAllWant[n] = true
	}
	for _, n := range codeAdminOnlyTools {
		codeAllWant[n] = true
	}
	wikiAgentWant := map[string]bool{}
	for _, n := range kbAgentTools {
		wikiAgentWant[n] = true
	}
	wikiAllWant := map[string]bool{}
	for _, n := range append(append([]string{}, kbAgentTools...), kbAdminTools...) {
		wikiAllWant[n] = true
	}

	cases := []struct {
		label string
		srv   *Server
		want  map[string]bool
	}{
		{"memory/agent", NewServer(newTestHandler(t), Agent, FamilyMemory), memAgentWant},
		{"memory/all", NewServer(newTestHandler(t), All, FamilyMemory), memAllWant},
	}
	// the code/wiki servers need their registration seams.
	codeAgent := NewServer(newTestHandler(t), Agent, FamilyCode)
	codeAgent.RegisterCodeTools(newTestCodeHandler(t))
	codeAll := NewServer(newTestHandler(t), All, FamilyCode)
	codeAll.RegisterCodeTools(newTestCodeHandler(t))
	wikiAgent, _, _ := newProjectWikiFixture(t, Agent)
	wikiAll, _, _ := newProjectWikiFixture(t, All)
	cases = append(cases,
		struct {
			label string
			srv   *Server
			want  map[string]bool
		}{"code/agent", codeAgent, codeAgentWant},
		struct {
			label string
			srv   *Server
			want  map[string]bool
		}{"code/all", codeAll, codeAllWant},
		struct {
			label string
			srv   *Server
			want  map[string]bool
		}{"projectwiki/agent", wikiAgent, wikiAgentWant},
		struct {
			label string
			srv   *Server
			want  map[string]bool
		}{"projectwiki/all", wikiAll, wikiAllWant},
	)
	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			defer tc.srv.Close()
			got := stdioToolsList(t, tc.srv)
			assertNameSet(t, tc.label, got, tc.want)
		})
	}
}

// ────────────────────────────────────────────────────────────────────────────
// [concurrent-calls-answer-one-at-a-time]
// [deep-concurrency-stays-serial]  (mutation gate: WithWorkerPoolSize(1))
// ────────────────────────────────────────────────────────────────────────────

func TestSerialToolCalls(t *testing.T) {
	s := NewServer(newTestHandler(t), Agent, FamilyMemory)
	defer s.Close()

	// A blocking probe tool: the handler reports when it starts, blocks
	// until released, and tracks the max concurrent in-flight count. With
	// the serial stdio option (one worker) the second call cannot start
	// until the first completes — a mutated pool of N>1 starts both
	// immediately and maxSeen reaches 2.
	var (
		mu       sync.Mutex
		inflight int
		maxSeen  int
		started1 = make(chan struct{})
		release  = make(chan struct{})
	)
	probe := mcplib.NewTool("test_slow_probe", mcplib.WithDescription("serial stdio probe"))
	s.mcp.AddTool(probe, func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		mu.Lock()
		inflight++
		if inflight > maxSeen {
			maxSeen = inflight
		}
		first := inflight == 1
		mu.Unlock()
		if first {
			close(started1)
		}
		<-release
		mu.Lock()
		inflight--
		mu.Unlock()
		return &mcplib.CallToolResult{}, nil
	})

	// Drive the production stdio loop (StartOn = the pinned options) over
	// pipes.
	inR, inW := io.Pipe()
	out := &linePipe{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- s.StartOn(ctx, inR, out) }()

	write := func(line string) {
		t.Helper()
		if _, err := inW.Write([]byte(line + "\n")); err != nil {
			t.Fatalf("write %s: %v", line, err)
		}
	}
	waitForID := func(id int) {
		t.Helper()
		waitFor(t, 5*time.Second, func() bool {
			for _, l := range out.lines() {
				var m struct {
					ID json.RawMessage `json:"id"`
				}
				if json.Unmarshal([]byte(l), &m) == nil && strings.Contains(string(m.ID), fmt.Sprintf("%d", id)) {
					return true
				}
			}
			return false
		})
	}

	write(`{"jsonrpc":"2.0","id":0,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}`)
	waitForID(0)

	// Two concurrent tool calls (the client sends both before any
	// response — the MCP client's "parallel" tool calls).
	write(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"test_slow_probe","arguments":{}}}`)
	write(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"test_slow_probe","arguments":{}}}`)

	<-started1 // the first call is in flight
	// If the pool were >1, the second call would already be in flight.
	time.Sleep(150 * time.Millisecond)
	mu.Lock()
	got := maxSeen
	mu.Unlock()
	if got != 1 {
		t.Fatalf("two tool calls were in flight at once (maxSeen=%d) — the serial stdio option (WithWorkerPoolSize(1)) is not in effect", got)
	}
	// Release both: the second call now runs (one at a time, complete
	// responses for both).
	close(release)
	waitForID(1)
	waitForID(2)

	cancel()
	if err := <-done; err != nil && ctx.Err() == nil {
		t.Fatalf("stdio loop ended with an unexpected error: %v", err)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// [stdout-carries-only-protocol-frames]
// ────────────────────────────────────────────────────────────────────────────

func TestStdoutProtocolOnly(t *testing.T) {
	s := NewServer(newTestHandler(t), Agent, FamilyMemory)
	defer s.Close()

	// The diagnostic seam (R5): post-startup diagnostics land on the
	// stderr writer, never stdout.
	var stderrBuf bytes.Buffer
	s.logW = &stderrBuf

	// A tool that emits a serving diagnostic while in flight (the
	// post-startup window: after the stdio loop is serving).
	const diag = "diagnostic: probe completed"
	probe := mcplib.NewTool("test_diag_probe", mcplib.WithDescription("stderr diagnostic probe"))
	s.mcp.AddTool(probe, func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		s.Logf("%s\n", diag)
		return &mcplib.CallToolResult{}, nil
	})

	inR, inW := io.Pipe()
	out := &linePipe{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- s.StartOn(ctx, inR, out) }()

	write := func(line string) {
		t.Helper()
		if _, err := inW.Write([]byte(line + "\n")); err != nil {
			t.Fatalf("write %s: %v", line, err)
		}
	}
	waitForID := func(id int) {
		t.Helper()
		waitFor(t, 5*time.Second, func() bool {
			for _, l := range out.lines() {
				var m struct {
					ID json.RawMessage `json:"id"`
				}
				if json.Unmarshal([]byte(l), &m) == nil && strings.Contains(string(m.ID), fmt.Sprintf("%d", id)) {
					return true
				}
			}
			return false
		})
	}

	write(`{"jsonrpc":"2.0","id":0,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}`)
	waitForID(0)
	write(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"test_diag_probe","arguments":{}}}`)
	waitForID(1)

	cancel()
	<-done

	// stdout: every complete line is a JSON-RPC frame (the protocol
	// frames only — no diagnostic, no log, no stray byte).
	for _, l := range out.lines() {
		if l == "" {
			continue
		}
		var m struct {
			JSONRPC string `json:"jsonrpc"`
		}
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Fatalf("stdout carries a non-JSON line (protocol-only invariant broken): %q", l)
		}
		if m.JSONRPC != "2.0" {
			t.Fatalf("stdout carries a non-JSON-RPC line: %q", l)
		}
	}
	// stderr seam: the in-flight diagnostic landed here.
	if !strings.Contains(stderrBuf.String(), diag) {
		t.Fatalf("the serving diagnostic must land on the stderr seam, got stderr: %q", stderrBuf.String())
	}
	if strings.Contains(strings.Join(out.lines(), "\n"), diag) {
		t.Fatal("the serving diagnostic reached stdout (protocol-only invariant broken)")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// [two-servers-one-store-parallel]
// ────────────────────────────────────────────────────────────────────────────

// countingTransport is a no-op MutationTransport that counts the calls
// that reach it (the lease-holder proof: a stood-down worker makes ZERO
// transport calls — only the lease winner reaches the network).
type countingTransport struct {
	mu    sync.Mutex
	calls int
}

func (c *countingTransport) bump() {
	c.mu.Lock()
	c.calls++
	c.mu.Unlock()
}

func (c *countingTransport) Calls() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls
}

func (c *countingTransport) PushMutations(ctx context.Context, project string, batch []store.Mutation) (int64, error) {
	c.bump()
	if len(batch) == 0 {
		return 0, nil
	}
	return batch[len(batch)-1].Seq, nil
}

func (c *countingTransport) PullMutations(ctx context.Context, project string, sinceSeq int64) ([]store.Mutation, error) {
	c.bump()
	return nil, nil
}

func (c *countingTransport) EnsureSchema(ctx context.Context, project string) error {
	c.bump()
	return nil
}

func (c *countingTransport) Close() error { return nil }

// newTestCodeHandlerOn builds a CodeHandler over a caller-provided store
// (newTestCodeHandler's two-process variant — the same wiring, the
// shared store file).
func newTestCodeHandlerOn(t *testing.T, st store.Store) *handler.CodeHandler {
	t.Helper()
	c := config.Default()
	c.DataDir = t.TempDir()
	c.Project.Detect = false
	return handler.NewCodeHandler(st, nil, c, "", &detectorStub{resolve: func(cwd, explicit string) (string, error) {
		if explicit != "" {
			return explicit, nil
		}
		return testProject, nil
	}})
}

// syncTestConfig is the autosync test config: sync enabled + auto, a
// resolvable project (the pull leg's transport call needs it), and a
// short debounce (the test's cycle windows).
func syncTestConfig() config.Config {
	c := config.Default()
	c.Project.Detect = false
	c.Project.Name = testProject
	c.Sync.Enabled = true
	c.Sync.Auto = true
	c.Sync.DebounceMs = 50
	return c
}

func TestTwoServersOneStoreParallel(t *testing.T) {
	// Two MCP server instances (a memory server + a code server) over ONE
	// SQLite store file — the two `mnemonic <family> mcp` processes
	// sharing one data dir (two store connections to the same file).
	// (1) Parallel write load reports no database-is-locked;
	// (2) with autosync enabled on both, at most one process holds the
	// sync lease (the loser stands down: zero transport calls).
	ctx := context.Background()
	dir := t.TempDir()
	st1 := openTestStore(t, dir) // process 1 (the memory server)
	st2 := openTestStore(t, dir) // process 2 (the code server), same file

	h1 := newTestHandlerWithStore(t, st1)
	h2 := newTestHandlerWithStore(t, st2)
	memSrv := NewServer(h1, Agent, FamilyMemory)
	defer memSrv.Close()
	codeSrv := NewServer(h2, Agent, FamilyCode)
	defer codeSrv.Close()

	ch := newTestCodeHandlerOn(t, st2)
	root := t.TempDir()
	mustWriteFile(t, root, "main.go", "package main\n\nfunc main() { add(1, 2) }\n")
	mustWriteFile(t, root, "math.go", "package main\n\nfunc add(a, b int) int { return a + b }\n")
	ch.SetRoot(root)
	codeSrv.RegisterCodeTools(ch)

	// (1) Parallel write load: mem_save (memory server) ∥ code_index
	// (code server). SQLite WAL + the busy timeout absorb the
	// cross-connection contention; every write must succeed and none may
	// report a locked database.
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []string
	)
	fail := func(format string, args ...any) {
		mu.Lock()
		errs = append(errs, fmt.Sprintf(format, args...))
		mu.Unlock()
	}
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := range 20 {
			if out := memSrv.CallTool(ctx, "mem_save", saveArgs(i)); strings.Contains(string(out), `"error"`) {
				fail("mem_save %d: %s", i, out)
			}
		}
	}()
	go func() {
		defer wg.Done()
		for i := range 3 {
			if out := codeSrv.CallTool(ctx, "code_index", map[string]any{}); strings.Contains(string(out), `"error"`) {
				fail("code_index %d: %s", i, out)
			}
		}
	}()
	wg.Wait()
	for _, e := range errs {
		if strings.Contains(e, "database is locked") {
			t.Fatalf("database-is-locked under parallel load (one store, two server connections): %s", e)
		}
	}
	if len(errs) > 0 {
		t.Fatalf("parallel writes must all succeed, got %d failures: %v", len(errs), errs[:min(3, len(errs))])
	}

	// (2) The sync lease: autosync on both processes, both marked dirty
	// at once. The lease is sticky (the winner renews it every cycle), so
	// exactly one process ever reaches the transport; the loser stands
	// down with zero calls.
	cfg := syncTestConfig()
	tr1 := &countingTransport{}
	tr2 := &countingTransport{}
	a1 := memory.NewAutosync(memory.NewSyncService(st1, cfg), tr1, cfg)
	a2 := memory.NewAutosync(memory.NewSyncService(st2, cfg), tr2, cfg)
	sctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a1.Start(sctx)
	a2.Start(sctx)
	defer a1.Stop()
	defer a2.Stop()
	// Sequence the race (the fresh-store row has a create-window TOCTOU —
	// the live-lease stand-down is the arbiter once the row exists, so
	// the winner must cycle first, the loser second):
	a1.NotifyDirty()
	// Wait for a1 to cycle: it holds the lease (sticky — renewed every
	// cycle) and its resolvable project makes at least one transport
	// call.
	waitFor(t, 15*time.Second, func() bool { return tr1.Calls() >= 1 })
	// NOW the loser: a2's cycle must see a1's live lease and stand down
	// with zero transport calls.
	a2.NotifyDirty()
	// a2's cycle window (debounce 50 ms + cycle time), generously: a
	// wrong winner would reach the transport within it.
	time.Sleep(750 * time.Millisecond)
	if tr2.Calls() > 0 {
		t.Fatalf("the stood-down process reached the sync transport (tr2=%d) — at most one may hold the lease", tr2.Calls())
	}
	if tr1.Calls() < 1 {
		t.Fatal("the lease winner must have reached the transport")
	}
}
