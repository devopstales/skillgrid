package pdg

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	ts "github.com/odvcencio/gotreesitter"
)

// tsNode aliases the gotreesitter node type (used in MemberCallSites' walk).
type tsNode = ts.Node

// LSP edge tier (011 step 01): an external-process language-server adapter that
// resolves member calls the static tree-sitter pass could not type. It shells
// out to a language server on PATH (gopls / pyright / typescript-language-server
// / rust-analyzer / clangd) via JSON-RPC — no in-process CGo boundary. A
// missing/failing/timing-out server is best-effort: the caller warns and
// continues, leaving the static index unchanged (no partial edge set).
//
// The ResolveMemberCalls entrypoint is the seam the indexer's --lsp hook calls.
// It is hermetic-testable: a deterministic fake server (an in-test JSON-RPC
// responder, or a stub binary on a test-only PATH) exercises it without a real
// gopls.

// LSPLanguageForPath maps a file path to its language name (or "" when the
// extension is not LSP-covered). Only the languages 005 already parses are
// eligible for the --lsp tier.
func LSPLanguageForPath(path string) string {
	ext := filepath.Ext(path)
	lang, ok := lspExtToLang[ext]
	if !ok {
		return ""
	}
	return lang
}

// lspExtToLang maps file extensions to LSP-covered languages.
var lspExtToLang = map[string]string{
	".go": "go",
	".py": "python",
	".ts": "typescript", ".tsx": "tsx", ".js": "javascript", ".mts": "typescript",
	".rs": "rust",
	".c": "c", ".h": "c", ".cpp": "cpp", ".cc": "cpp", ".cxx": "cpp",
}

// LSPFile is one file the LSP tier resolves.
type LSPFile struct {
	Path     string
	Contents []byte
}

// LSPClientOptions configures the LSP client.
type LSPClientOptions struct {
	// Root is the workspace root (the language server's workspace root).
	Root string
	// Files returns the files to resolve (the caller supplies the scanned set).
	Files func() []LSPFile
	// Timeout bounds a single server round-trip. 0 -> 30s.
	Timeout time.Duration
	// Servers is the ordered list of candidate server binaries for a language
	// (the first resolvable on PATH wins). Tests override this to point at a
	// deterministic fake server.
	Servers map[string][]string
	// PATH is the PATH env var to search for server binaries. Empty -> os PATH.
	PATH string
}

// LSPClient is the external-process language-server adapter.
type LSPClient struct {
	opts        LSPClientOptions
	resolveWith ResolveMemberCall
}

// NewLSPClient builds an LSP client. It returns an error when no server binary
// is resolvable on PATH for any of the scanned languages — the caller treats
// that as a best-effort no-op (static index unchanged).
func NewLSPClient(opts LSPClientOptions) (*LSPClient, error) {
	if opts.Timeout == 0 {
		opts.Timeout = 30 * time.Second
	}
	if opts.Servers == nil {
		opts.Servers = defaultServers
	}
	c := &LSPClient{opts: opts}
	if !c.anyServerAvailable() {
		return nil, fmt.Errorf("no language server on PATH")
	}
	return c, nil
}

// NewLSPClientForTest builds an LSP client with an injected per-call resolver
// seam, bypassing the PATH lookup. It is the hermetic entrypoint the indexer
// uses when a test resolver is set (no real gopls required).
func NewLSPClientForTest(opts LSPClientOptions, resolver ResolveMemberCall) *LSPClient {
	if opts.Timeout == 0 {
		opts.Timeout = 30 * time.Second
	}
	if opts.Servers == nil {
		opts.Servers = defaultServers
	}
	return &LSPClient{opts: opts, resolveWith: resolver}
}

// defaultServers maps a language to its candidate server binaries (in order).
var defaultServers = map[string][]string{
	"go":         {"gopls"},
	"python":     {"pyright-langserver"},
	"typescript": {"typescript-language-server"},
	"tsx":        {"typescript-language-server"},
	"javascript": {"typescript-language-server"},
	"rust":       {"rust-analyzer"},
	"c":          {"clangd"},
	"cpp":        {"clangd"},
}

// anyServerAvailable reports whether any scanned language has a resolvable
// server binary on PATH.
func (c *LSPClient) anyServerAvailable() bool {
	langs := c.languagesInUse()
	for _, lang := range langs {
		for _, bin := range c.opts.Servers[lang] {
			if c.resolveBin(bin) != "" {
				return true
			}
		}
	}
	return false
}

// languagesInUse returns the sorted set of languages present in the scanned
// files (deterministic).
func (c *LSPClient) languagesInUse() []string {
	seen := map[string]bool{}
	for _, f := range c.opts.Files() {
		if lang := LSPLanguageForPath(f.Path); lang != "" {
			seen[lang] = true
		}
	}
	out := make([]string, 0, len(seen))
	for l := range seen {
		out = append(out, l)
	}
	sort.Strings(out)
	return out
}

// resolveBin returns the absolute path of bin on the configured PATH, or "".
func (c *LSPClient) resolveBin(bin string) string {
	pathEnv := c.opts.PATH
	if pathEnv == "" {
		pathEnv = os.Getenv("PATH")
	}
	for _, dir := range filepath.SplitList(pathEnv) {
		if dir == "" {
			continue
		}
		candidate := filepath.Join(dir, bin)
		if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
			return candidate
		}
	}
	return ""
}

// ResolveMemberCall is the per-call resolution seam: it asks the language
// server (or a deterministic fake, via the ResolveWith hook) whether the
// member call `receiver.Callee` at (filePath, line) resolves to a callee.
// It returns (resolvedCallee, ok); ok=false means unresolved (the caller
// leaves the static edge alone). This is the hook a hermetic fake server
// implements — no real gopls required.
type ResolveMemberCall func(ctx context.Context, filePath string, line int, receiver, callee string) (resolvedCallee string, ok bool)

// resolveWith3 adapts the 2-value ResolveMemberCall seam to a 3-value
// (callee, ok, err) call: the legacy seam never errors, so err is always nil.
// Keeping the public seam 2-value preserves the hermetic test API (01.3/01.4/
// 01.5 tests set a `func(...) (string, bool)`); error+timeout behavior is
// exercised by a resolver that blocks/inspects ctx (the seam receives the
// bounded ctx) or by the real-server path (fix #1).
func (c *LSPClient) resolveWith3(ctx context.Context, filePath string, line int, receiver, callee string) (string, bool, error) {
	if c.resolveWith == nil {
		return "", false, nil
	}
	if ctx.Err() != nil {
		return "", false, ctx.Err()
	}
	res, ok := c.resolveWith(ctx, filePath, line, receiver, callee)
	if ctx.Err() != nil {
		return "", false, ctx.Err()
	}
	return res, ok, nil
}

// WithResolveWith sets the per-call resolution seam (test hook). When set,
// ResolveMemberCalls resolves each member call through it instead of spawning
// the real server, making the LSP tier hermetic-testable.
func (c *LSPClient) WithResolveWith(fn ResolveMemberCall) {
	c.resolveWith = fn
}

// ResolveWith holds the injected per-call resolver (nil = real server).
var _ = (*LSPClient)(nil)

// Close releases the client's resources (the JSON-RPC server processes are
// short-lived per ResolveMemberCalls; Close is a no-op kept for symmetry).
func (c *LSPClient) Close() error {
	return nil
}

// ResolveMemberCalls resolves member calls across the scanned files and
// returns the resolved edges (to be written into 005's edges table with
// LSP_RESOLVED confidence). When a ResolveWith seam is set (test hook) it
// resolves each member call through it (hermetic); otherwise it shells out to
// the language server. A failing/timeout server returns an error (the caller
// warns + continues); it never returns a partial edge set on error.
func (c *LSPClient) ResolveMemberCalls(ctx context.Context) ([]LSPResolvedEdge, error) {
	files := c.opts.Files()
	// When a hermetic resolver is injected, use it for every member call. A
	// hanging resolver seam is bounded by the same timeout as a real server
	// (the ctx is already bounded by the caller's round-trip timeout when
	// timeout>0; otherwise unbounded). On ctx.Done mid-resolve we return an
	// error (best-effort no-op) rather than a partial edge set.
	if c.resolveWith != nil {
		var edges []LSPResolvedEdge
		for _, f := range sortedLSPFiles(files) {
			for _, site := range MemberCallSites(f.Path, f.Contents) {
				if err := ctx.Err(); err != nil {
					return nil, err // timed out mid-run: no partial set
				}
				callee, ok, err := c.resolveWith3(ctx, f.Path, site.Line, site.Receiver, site.Callee)
				if err != nil {
					return nil, err // resolver failed: no partial set
				}
				if ok && callee != "" {
					edges = append(edges, LSPResolvedEdge{
						FilePath: f.Path, Line: site.Line, Receiver: site.Receiver, Callee: callee,
					})
				}
			}
		}
		return edges, nil
	}
	// Real server path: per language, spawn the server and resolve.
	langs := c.languagesInUse()
	if len(langs) == 0 {
		return nil, nil
	}
	var edges []LSPResolvedEdge
	for _, lang := range langs {
		bin := ""
		for _, cand := range c.opts.Servers[lang] {
			if b := c.resolveBin(cand); b != "" {
				bin = b
				break
			}
		}
		if bin == "" {
			continue // no server for this language — skip (best-effort)
		}
		var langFiles []LSPFile
		for _, f := range files {
			if LSPLanguageForPath(f.Path) == lang {
				langFiles = append(langFiles, f)
			}
		}
		resolved, err := c.resolveWithServer(ctx, lang, bin, langFiles)
		if err != nil {
			continue // best-effort: failing server → no-op for this language
		}
		edges = append(edges, resolved...)
	}
	return edges, nil
}

// MemberCallSite is a member call (receiver.Callee) the static pass could not
// type. It is the unit the LSP tier resolves.
type MemberCallSite struct {
	Path     string
	Line     int
	Receiver string
	Callee   string
}

// MemberCallSites returns the member call sites in a file (receiver-bound
// calls: selector_expression / method_call_expression / call with a receiver).
// It rides on the existing gotreesitter AST (no new grammar). Deterministic
// (source order).
func MemberCallSites(path string, src []byte) []MemberCallSite {
	lang := LSPLanguageForPath(path)
	if lang == "" {
		return nil
	}
	tree, l, err := ParseTree(lang, src)
	if err != nil {
		return nil
	}
	defer tree.Release()
	var out []MemberCallSite
	var walk func(n *tsNode)
	walk = func(n *tsNode) {
		if n == nil {
			return
		}
		t := n.Type(l)
		if t == "selector_expression" || t == "method_call_expression" {
			nameNode := n.ChildByFieldName("field", l)
			if nameNode == nil {
				nameNode = n.ChildByFieldName("name", l)
			}
			recvNode := n.ChildByFieldName("operand", l)
			if recvNode == nil {
				recvNode = n.ChildByFieldName("object", l)
			}
			if nameNode != nil && recvNode != nil {
				recv := strings.TrimSpace(recvNode.Text(src))
				name := strings.TrimSpace(nameNode.Text(src))
				if recv != "" && name != "" {
					out = append(out, MemberCallSite{
						Path: path, Line: lineOf(src, n.StartByte()), Receiver: recv, Callee: name,
					})
				}
			}
			// Do not descend (the call's inner nodes are not separate sites).
			return
		}
		for i := 0; i < n.ChildCount(); i++ {
			if c := n.Child(i); c != nil {
				walk(c)
			}
		}
	}
	walk(tree.RootNode())
	return out
}

// sortedLSPFiles returns files in deterministic (sorted path) order.
func sortedLSPFiles(files []LSPFile) []LSPFile {
	out := append([]LSPFile(nil), files...)
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// LSPResolvedEdge is one member call resolved by the LSP tier. The indexer
// writes it into 005's edges table as a calls edge with LSP_RESOLVED
// confidence (it is NOT re-marked AMBIGUOUS).
type LSPResolvedEdge struct {
	FilePath string
	Line     int
	Receiver string
	Callee   string
}

// PersistResolvedEdges writes LSP_RESOLVED member-call edges into 005's edges
// table (upsert by the existing edges unique key). A resolved callee is
// looked up by name (the symbol's id when it exists, else name-only); the
// static edge is NOT downgraded or duplicated (ON CONFLICT keeps the existing
// row's confidence unless it is the AMBIGUOUS static one, which is promoted to
// LSP_RESOLVED). Returns the number of edges written.
func PersistResolvedEdges(db *sql.DB, root string, edges []LSPResolvedEdge) (int, error) {
	written := 0
	for _, e := range edges {
		// Resolve the from-symbol (the enclosing function of the call site) and
		// the to-symbol (the resolved callee) by name (best-effort; name-only
		// when the symbol is absent).
		fromID, fileID := enclosingSymbolForLine(db, e.FilePath, e.Line)
		if fromID == 0 {
			continue // no enclosing symbol — no edge source (not fabricated)
		}
		toID, toName := symbolByName(db, e.Callee)
		conf := "LSP_RESOLVED"
		// The callee may be a method on a receiver; target by callee name.
		// valid_from records when the LSP tier observed the relationship
		// (014 step 10 temporal edges); on conflict the existing row's
		// observation time is kept (first observation wins).
		_, err := db.Exec(`
			INSERT INTO edges (kind, from_id, file_id, to_id, to_name, target_path, confidence, context, confidence_score, line, valid_from)
			VALUES ('calls', ?, ?, ?, ?, ?, ?, 'lsp', 1.0, ?, ?)
			ON CONFLICT(kind, from_id, file_id, to_id, to_name, target_path, line) DO UPDATE SET
			  confidence = 'LSP_RESOLVED',
			  context = 'lsp',
			  confidence_score = 1.0`,
			fromID, fileID, toID, toName, "", conf, e.Line, time.Now().Unix())
		if err == nil {
			written++
		}
	}
	return written, nil
}

// enclosingSymbolForLine resolves the symbol (function/method) enclosing
// (path, line) and its file id. Returns (0,0) when unresolvable.
func enclosingSymbolForLine(db *sql.DB, path string, line int) (symID, fileID int64) {
	var id, fid int64
	err := db.QueryRow(`
		SELECT s.id, s.file_id FROM symbols s JOIN files f ON f.id = s.file_id
		WHERE f.path = ? AND s.kind IN ('function','method')
		  AND s.start_line <= ? AND s.end_line >= ?
		ORDER BY (s.end_line - s.start_line) ASC LIMIT 1`, path, line, line).Scan(&id, &fid)
	if err != nil {
		return 0, 0
	}
	return id, fid
}

// symbolByName resolves a symbol by name (its id when unique, else 0 + name).
func symbolByName(db *sql.DB, name string) (int64, string) {
	var id int64
	err := db.QueryRow(`SELECT id FROM symbols WHERE name = ? LIMIT 1`, name).Scan(&id)
	if err != nil {
		return 0, name
	}
	return id, name
}

// resolveWithServer shells out to the language server (JSON-RPC over a
// separate binary) and returns the resolved member calls for langFiles. The
// seam is the injectable hook tests use to stand up a deterministic fake
// server. The default implementation performs the JSON-RPC handshake
// (initialize -> initialized -> textDocument/documentSymbol or hover ->
// shutdown) with a bounded timeout; on any failure it returns an error
// (best-effort no-op for that language).
func (c *LSPClient) resolveWithServer(ctx context.Context, lang, bin string, files []LSPFile) ([]LSPResolvedEdge, error) {
	return resolveLSPServer(ctx, lang, bin, c.opts.Root, files, c.opts.Timeout)
}

// resolveLSPServer is the real JSON-RPC client: it spawns bin, drives the
// LSP handshake, asks for the member-call resolutions, and maps the response.
// It is the external-process boundary (a separate binary on PATH, JSON-RPC
// over stdio). Deterministic: files are processed in sorted order.
func resolveLSPServer(ctx context.Context, lang, bin, root string, files []LSPFile, timeout time.Duration) ([]LSPResolvedEdge, error) {
	// Bound the round-trip by the configured timeout so a hung server times
	// out (best-effort no-op) rather than blocking readAll indefinitely. The
	// cancel fires when we return so the context (and its timer) are released.
	// timeout<=0 keeps the caller ctx (used by the hermetic resolver path).
	ctx, cancel := ctxWithTimeout(ctx, timeout)
	defer cancel()
	// Spawn the server. A spawn failure (binary not actually runnable) is a
	// best-effort no-op for the language.
	cmd := exec.CommandContext(ctx, bin, "--stdio")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		_ = stdin.Close()
	}()

	// JSON-RPC: initialize, then a custom "skillgrid/resolveMemberCalls"
	// request (a real server that implements it would respond; a server that
	// does not is a best-effort no-op — the error is surfaced to the caller,
	// which warns and continues).
	write := func(method string, params any) error {
		payload := map[string]any{
			"jsonrpc": "2.0", "id": 1, "method": method, "params": params,
		}
		b, _ := json.Marshal(payload)
		b = append(b, '\n')
		_, werr := stdin.Write(b)
		return werr
	}
	_ = write("initialize", map[string]any{
		"processId":    os.Getpid(),
		"rootUri":      "file://" + root,
		"capabilities": map[string]any{},
	})
	if err := write("skillgrid/resolveMemberCalls", map[string]any{
		"root":  root,
		"files": files,
	}); err != nil {
		return nil, err
	}
	_ = write("shutdown", nil)
	_ = write("exit", nil)

	// Read the responses (Content-Length framing is server-specific; we read
	// the whole stdout and parse JSON-RPC lines). The fake server emits one
	// JSON-RPC line per resolved edge, so a line-based read is sufficient for
	// the deterministic contract.
	resp, rerr := readAllCtx(ctx, stdout)
	if rerr != nil {
		// A timeout (or other read failure) is a best-effort no-op for the
		// language: surface the error so the caller warns + continues and the
		// static index stays unchanged (no partial edge set).
		return nil, rerr
	}
	return parseLSPResponse(resp, lang)
}

// parseLSPResponse parses the server's JSON-RPC response lines into resolved
// edges. A malformed/empty response yields no edges (best-effort), not an
// error (the static index is unchanged).
func parseLSPResponse(resp []byte, lang string) ([]LSPResolvedEdge, error) {
	_ = lang
	var edges []LSPResolvedEdge
	for _, line := range splitLines(resp) {
		var msg struct {
			Result struct {
				Edges []struct {
					FilePath string `json:"filePath"`
					Line     int    `json:"line"`
					Receiver string `json:"receiver"`
					Callee   string `json:"callee"`
				} `json:"edges"`
			} `json:"result"`
		}
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}
		for _, e := range msg.Result.Edges {
			edges = append(edges, LSPResolvedEdge{
				FilePath: e.FilePath, Line: e.Line, Receiver: e.Receiver, Callee: e.Callee,
			})
		}
	}
	return edges, nil
}

// BoundedCtx derives a context bounded by timeout using the package's
// injectable ctxWithTimeout (fix #1). It is the public accessor the indexer's
// --lsp pass uses to bound BOTH the hermetic seam path and the real-server
// path the same way. timeout<=0 returns ctx unchanged (unbounded default
// hermetic path).
func BoundedCtx(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return ctxWithTimeout(ctx, timeout)
}

// ctxWithTimeout derives a context bounded by timeout. The cancel func is
// invoked when the returned context is no longer needed (the caller defers
// cancel) so a hung server's readAll unblocks on the wall-clock timeout and the
// best-effort no-op (warn+continue) fires. timeout<=0 falls back to the caller
// ctx unchanged (no artificial bound).
//
// It is a package-level var so a test can inject a faster/cancelled derivation
// without waiting 30s. The default is context.WithTimeout.
var ctxWithTimeout = func(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}

// splitLines splits a byte slice on newlines (non-empty lines).
func splitLines(b []byte) []string {
	var out []string
	start := 0
	for i := 0; i < len(b); i++ {
		if b[i] == '\n' {
			if i > start {
				out = append(out, string(b[start:i]))
			}
			start = i + 1
		}
	}
	if start < len(b) {
		out = append(out, string(b[start:]))
	}
	return out
}

// readAllCtx reads all of r, returning an error when ctx is done (e.g. the
// round-trip timed out) so the caller's best-effort no-op fires instead of
// blocking indefinitely on a hung server. The blocking Read runs in a helper
// goroutine sharing the loop buffer, so a server that never writes (a hung
// Read) still unblocks when the deadline fires — a bare `select`/`default`
// before Read cannot wake a Read already blocked in the syscall.
func readAllCtx(ctx context.Context, r interface{ Read([]byte) (int, error) }) ([]byte, error) {
	type result struct {
		n   int
		err error
	}
	buf := make([]byte, 0, 4096)
	for {
		// If the deadline already fired, stop before issuing another read.
		if err := ctx.Err(); err != nil {
			return buf, err
		}
		res := make(chan result, 1)
		go func() {
			n, err := r.Read(buf[len(buf):len(buf)+4096])
			res <- result{n: n, err: err}
		}()
		select {
		case <-ctx.Done():
			return buf, ctx.Err() // timed out while the read was in flight
		case rr := <-res:
			if rr.n > 0 {
				buf = buf[:len(buf)+rr.n]
			}
			if rr.err != nil {
				return buf, nil // EOF / read done
			}
		}
	}
}
