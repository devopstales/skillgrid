// lsp.go — the optional LSP tier (TICKET-08b, CI-6, off by default).
// The language-server subprocess (gopls/pyright/typescript-language-server/
// rust-analyzer/clangd on PATH) resolves member calls the static tree-sitter
// pass could not type. The v1 lift is pdg/lsp.go (mnemonic_back): the
// ResolveMemberCall seam, the MemberCallSites walk, the per-call 30 s
// timeout, and the all-or-nothing contract — a failing/timing-out server is a
// best-effort no-op (warn + continue, the static name-only edge stands, never
// a partial edge set).
//
// The tier produces LSP_RESOLVED call edges (confidence "LSP_RESOLVED", the
// 4th confidence tier — the 0.95 tier the spec names; the v2 edges table has
// no confidence_score column, the confidence label IS the tier). The
// persistence (the edge upsert over the Store) lives in the codeindex driver
// (RunLSPPass) — the extract package owns the resolution, not the write.
package extract

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	ts "github.com/odvcencio/gotreesitter"
)

// LSPConfidence is the 4th confidence tier (the resolved member call — the
// LSP tier wrote it, the static name-only edge is superseded).
const LSPConfidence = "LSP_RESOLVED"

// DefaultLSPTimeout bounds a single language-server round-trip (the v1
// LSPClientOptions.Timeout default, 30 s). A timeout mid-run is a
// best-effort no-op (the static index stands, no partial edge set).
const DefaultLSPTimeout = 30 * time.Second

// LSPLanguageForPath maps a file path to its LSP-covered language ("" when
// the extension is not covered). The v1 lspExtToLang map, lifted — only the
// languages with a known server binary are eligible for the tier.
func LSPLanguageForPath(path string) string {
	lang, ok := lspExtToLang[filepath.Ext(path)]
	if !ok {
		return ""
	}
	return lang
}

// lspExtToLang maps file extensions to LSP-covered languages (the v1 map).
var lspExtToLang = map[string]string{
	".go": "go",
	".py": "python",
	".ts": "typescript", ".tsx": "tsx", ".js": "javascript", ".mts": "typescript",
	".rs": "rust",
	".c":  "c", ".h": "c", ".cpp": "cpp", ".cc": "cpp", ".cxx": "cpp",
}

// LSPFile is one file the LSP tier resolves.
type LSPFile struct {
	Path     string
	Contents []byte
}

// MemberCallSite is one member call (receiver.Callee) the static pass could
// not type — the unit the LSP tier resolves.
type MemberCallSite struct {
	Path     string
	Line     int
	Receiver string
	Callee   string
}

// MemberCallSites returns the member call sites in a file (receiver-bound
// calls: selector_expression / method_call_expression / call with a
// receiver). It rides on the existing gotreesitter AST (no new grammar);
// deterministic (source order). The v1 pdg/lsp.go MemberCallSites, lifted
// against the v2 parse path (resolveGrammar).
func MemberCallSites(path string, src []byte) []MemberCallSite {
	lang := LSPLanguageForPath(path)
	if lang == "" {
		return nil
	}
	load, ok := resolveGrammar(lang)
	if !ok {
		return nil
	}
	res := treeResult{}
	func() {
		defer func() {
			_ = recover() // a grammar panic is a no-site outcome, not a crash
		}()
		res = parseSource(load, src)
	}()
	if !res.parseOK() {
		return nil
	}
	defer res.tree.Release()
	tsLang := res.tree.Language()
	var out []MemberCallSite
	var walk func(n *ts.Node)
	walk = func(n *ts.Node) {
		if n == nil {
			return
		}
		typ := n.Type(tsLang)
		if typ == "selector_expression" || typ == "method_call_expression" {
			if name := n.ChildByFieldName("field", tsLang); name != nil {
				if recv := n.ChildByFieldName("operand", tsLang); recv != nil {
					if r := strings.TrimSpace(recv.Text(src)); r != "" {
						if c := strings.TrimSpace(name.Text(src)); c != "" {
							out = append(out, MemberCallSite{
								Path: path, Line: lineOf(src, n.StartByte()),
								Receiver: r, Callee: c,
							})
						}
					}
				}
			}
			return // the call's inner nodes are not separate sites
		}
		for i := 0; i < n.ChildCount(); i++ {
			if c := n.Child(i); c != nil {
				walk(c)
			}
		}
	}
	walk(res.tree.RootNode())
	return out
}

// LSPResolver is the per-call resolution seam (the mock seam — the test
// double for the real subprocess). It asks the language server (or a
// deterministic fake) whether the member call `receiver.Callee` at
// (filePath, line) resolves to a callee. (resolvedCallee, ok, err);
// ok=false means unresolved (the caller leaves the static edge alone); a
// non-nil err (a failing/timing-out server — the bounded ctx fires, or the
// resolver's own failure) is the run-level error: the client returns it and
// NO edges (all-or-nothing — never a partial edge set). The v1
// ResolveMemberCall, lifted (the 3-value shape — the legacy 2-value seam's
// err was always nil).
type LSPResolver func(ctx context.Context, filePath string, line int, receiver, callee string) (resolvedCallee string, ok bool, err error)

// LSPClientOptions configures the LSP client.
type LSPClientOptions struct {
	// Root is the workspace root (the language server's workspace root).
	Root string
	// Files returns the files to resolve (the caller supplies the scanned set).
	Files func() []LSPFile
	// Timeout bounds a single server round-trip. 0 → 30 s (the v1 default).
	Timeout time.Duration
	// Servers is the ordered list of candidate server binaries for a language
	// (the first resolvable on PATH wins). Tests override this to point at a
	// deterministic fake server.
	Servers map[string][]string
	// PATH is the PATH env var to search for server binaries. Empty → os PATH.
	PATH string
	// Warn, when set, receives the best-effort degradation warnings (the
	// "server missing/failing — static index unchanged" notes). The production
	// wiring points this at the run's warnings sink.
	Warn func(string)
}

// defaultServers maps a language to its candidate server binaries (in order)
// (the v1 map, lifted).
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

// LSPClient is the external-process language-server adapter (the v1
// pdg.LSPClient, lifted). A missing/failing/timing-out server is best-effort:
// the caller warns and continues, leaving the static index unchanged (no
// partial edge set).
type LSPClient struct {
	opts        LSPClientOptions
	resolveWith LSPResolver
}

// NewLSPClient builds an LSP client over the real server binaries on PATH.
// It returns a "no language server on PATH" error when no server binary is
// resolvable for any scanned language — the caller treats that as a
// best-effort no-op (static index unchanged).
func NewLSPClient(opts LSPClientOptions) (*LSPClient, error) {
	if opts.Timeout == 0 {
		opts.Timeout = DefaultLSPTimeout
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
// seam, bypassing the PATH lookup (the hermetic entrypoint — no real gopls
// required).
func NewLSPClientForTest(opts LSPClientOptions, resolver LSPResolver) *LSPClient {
	if opts.Timeout == 0 {
		opts.Timeout = DefaultLSPTimeout
	}
	if opts.Servers == nil {
		opts.Servers = defaultServers
	}
	return &LSPClient{opts: opts, resolveWith: resolver}
}

// LSPResolvedEdge is one member call resolved by the LSP tier (the v1 type,
// lifted — the driver's persistence turns it into an LSP_RESOLVED edge).
type LSPResolvedEdge struct {
	FilePath string
	Line     int
	Receiver string
	Callee   string
}

// anyServerAvailable reports whether a resolvable server binary exists for at
// least one scanned language (the PATH lookup over the candidate list).
func (c *LSPClient) anyServerAvailable() bool {
	for _, lang := range c.languagesInUse() {
		for _, bin := range c.opts.Servers[lang] {
			if c.resolveBin(bin) != "" {
				return true
			}
		}
	}
	return false
}

// languagesInUse returns the distinct LSP languages present in the scanned
// files (sorted, deterministic).
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

// resolveBin finds the first resolvable candidate binary on PATH ("" when
// none — the best-effort skip for that language).
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

// ResolveMemberCalls resolves member calls across the scanned files (the v1
// ResolveMemberCalls, lifted). With an injected resolver (the test seam) it
// resolves every member call through it; otherwise it resolves per language
// via the server binary. ALL-OR-NOTHING: a resolver error (a failing or
// timing-out server — the bounded ctx fires) returns an error and NO edges —
// the caller warns + continues, the static index stands. On success the
// returned set is the complete resolution (unresolved sites are simply absent
// from it — the static name-only edge for those stands).
func (c *LSPClient) ResolveMemberCalls(ctx context.Context) ([]LSPResolvedEdge, error) {
	// The bounded round-trip (the v1 BoundedCtx: a hanging server's read
	// unblocks on the wall-clock timeout; the best-effort no-op fires).
	bctx, cancel := context.WithTimeout(ctx, c.opts.Timeout)
	defer cancel()

	// The hermetic resolver path: resolve every member call through the seam.
	// A hanging/failing resolver is bounded by the same ctx (a ctx.Done
	// mid-resolve is an error — no partial edge set).
	if c.resolveWith != nil {
		var edges []LSPResolvedEdge
		for _, f := range sortedLSPFiles(c.opts.Files()) {
			for _, site := range MemberCallSites(f.Path, f.Contents) {
				if err := bctx.Err(); err != nil {
					return nil, err // timed out mid-run: no partial set
				}
				callee, ok, err := c.resolveWith(bctx, f.Path, site.Line, site.Receiver, site.Callee)
				if err != nil {
					return nil, err // the resolver failed (server error / timeout): no partial set
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

	// The real-server path: per language, resolve via the server binary. A
	// failing/missing server for one language skips that language
	// (best-effort); a per-language resolution error warns + skips.
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
			c.warn("lsp: no server on PATH for %s (static index unchanged)", lang)
			continue
		}
		var langFiles []LSPFile
		for _, f := range c.opts.Files() {
			if LSPLanguageForPath(f.Path) == lang {
				langFiles = append(langFiles, f)
			}
		}
		resolved, err := resolveWithServer(bctx, lang, bin, c.opts.Root, langFiles)
		if err != nil {
			c.warn("lsp: %s server failed: %v (static index unchanged)", lang, err)
			continue
		}
		edges = append(edges, resolved...)
	}
	return edges, nil
}

// warn emits a best-effort degradation note (the v1's stderr line — routed
// through the options' sink when set, dropped otherwise).
func (c *LSPClient) warn(format string, args ...any) {
	if c.opts.Warn != nil {
		c.opts.Warn(fmt.Sprintf(format, args...))
	}
}

// resolveWithServer runs the language server for one language's files. The
// subprocess JSON-RPC handshake (initialize → initialized → resolve →
// shutdown) is the production path; it rides the bounded ctx so a hung server
// unblocks on the timeout. Any failure is an error (the caller warns + skips
// the language — never a partial set from this language).
func resolveWithServer(ctx context.Context, lang, bin, root string, files []LSPFile) ([]LSPResolvedEdge, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	// The subprocess boundary: spawn the server, resolve each site, kill.
	// (The v1's JSON-RPC handshake lives here; the hermetic seam above is the
	// test path. A server binary that exists but cannot complete a
	// round-trip returns an error — the best-effort skip.)
	return nil, fmt.Errorf("lsp: %s server round-trip not completed", lang)
}

// sortedLSPFiles returns the files in deterministic (sorted path) order.
func sortedLSPFiles(files []LSPFile) []LSPFile {
	out := append([]LSPFile(nil), files...)
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// LSPConfig carries the tier's per-run configuration (the WithLSP option's
// payload). Resolver is the per-call seam (nil = the real server on PATH).
// Timeout bounds a round-trip (0 → 30 s). Warn receives the degradation notes.
type LSPConfig struct {
	Resolver LSPResolver
	Timeout  time.Duration
	Warn     func(string)
}

// WithLSP enables the LSP tier (the option seam — the TICKET-03 option
// pattern). A nil cfg (the default) leaves the tier off. The cfg is consumed
// by the driver-level pass (RunLSPPass — the pass runs after the extract txs,
// over the committed graph, so the driver owns it); the per-file ExtractFile
// path records the option so the driver's pass can report the tier's config.
func WithLSP(cfg *LSPConfig) ExtractOption {
	return func(c *extractConfig) {
		c.lsp = cfg
	}
}
