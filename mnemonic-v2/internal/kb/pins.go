package kb

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// pins.go — the pin-check (006-validate, 011 §3): anchors + claims resolved
// against LIVE code. The span pin is OpenWiki's relocating version token,
// ported to the kb-lines-v1 form (006 §2.1): the token carries the span's
// content hash + 6 relocation fields, so the pin-check distinguishes moved
// (no work, ok) from changed (stale, live value reported, token NOT rewritten)
// from ambiguous (unresolved — fail closed).

const (
	tokenPrefix      = "kb-lines-v1:sha256:"
	contextLineCount = 3
)

// Span resolution results (006 §2.1).
const (
	SpanOK         = "ok"         // resolved in place or relocated (line drift free)
	SpanStale      = "stale"      // content changed at a unique site
	SpanUnresolved = "unresolved" // ambiguous or no match (fail closed)
)

// spanMeta is the token's relocation metadata (fixed field order — the base64url
// payload is a JSON object with exactly these keys).
type spanMeta struct {
	SelectedLineCount         int    `json:"selectedLineCount"`
	FirstSelectedLineHash     string `json:"firstSelectedLineHash"`
	LastSelectedLineHash      string `json:"lastSelectedLineHash"`
	PrecedingContextLineCount int    `json:"precedingContextLineCount"`
	PrecedingContextHash      string `json:"precedingContextHash"`
	FollowingContextLineCount int    `json:"followingContextLineCount"`
	FollowingContextHash      string `json:"followingContextHash"`
}

func lineHash(l string) string {
	h := sha256.Sum256([]byte(l))
	return hex.EncodeToString(h[:])
}

func spanHash(lines []string) string {
	h := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	return hex.EncodeToString(h[:])
}

// TokenForSpan pins lines[line-1 : line-1+count) of content with a relocating
// version token (line is 1-based). Returns "" when the window is empty.
func TokenForSpan(content string, line, count int) string {
	lines := strings.Split(content, "\n")
	if line < 1 || count < 1 || line+count-1 > len(lines) {
		return ""
	}
	sel := lines[line-1 : line-1+count]
	start, end := line-1, line-1+count // 0-based window [start, end)

	pCount := contextLineCount
	if start < pCount {
		pCount = start
	}
	fCount := contextLineCount
	if remaining := len(lines) - end; remaining < fCount {
		fCount = remaining
	}
	meta := spanMeta{
		SelectedLineCount:         count,
		FirstSelectedLineHash:     lineHash(sel[0]),
		LastSelectedLineHash:      lineHash(sel[count-1]),
		PrecedingContextLineCount: pCount,
		PrecedingContextHash:      spanHash(lines[start-pCount : start]),
		FollowingContextLineCount: fCount,
		FollowingContextHash:      spanHash(lines[end : end+fCount]),
	}
	b, err := json.Marshal(meta)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s%s:%s", tokenPrefix, spanHash(sel), base64.RawURLEncoding.EncodeToString(b))
}

func parseSpanToken(token string) (contentHash string, meta spanMeta, ok bool) {
	rest, found := strings.CutPrefix(token, tokenPrefix)
	if !found {
		return "", meta, false
	}
	hash, b64, found := strings.Cut(rest, ":")
	if !found || b64 == "" {
		return "", meta, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(b64)
	if err != nil {
		return "", meta, false
	}
	if err := json.Unmarshal(raw, &meta); err != nil {
		return "", meta, false
	}
	if meta.SelectedLineCount < 1 {
		return "", meta, false
	}
	return hash, meta, true
}

// contextMatches verifies the decision anchors (006 §2.1: the first/last
// selected-line hashes prefilter only — the context hashes decide).
func contextMatches(lines []string, start int, m spanMeta) bool {
	pEnd := start
	if pEnd > m.PrecedingContextLineCount {
		pEnd = m.PrecedingContextLineCount
	}
	if pEnd != m.PrecedingContextLineCount {
		return false
	}
	if pEnd > 0 && spanHash(lines[start-pEnd:start]) != m.PrecedingContextHash {
		return false
	}
	if pEnd == 0 && m.PrecedingContextHash != spanHash(nil) {
		return false
	}
	fEnd := start + m.SelectedLineCount
	if remaining := len(lines) - fEnd; remaining < m.FollowingContextLineCount {
		return false
	}
	if m.FollowingContextLineCount > 0 {
		if spanHash(lines[fEnd:fEnd+m.FollowingContextLineCount]) != m.FollowingContextHash {
			return false
		}
	} else if m.FollowingContextHash != spanHash(nil) {
		return false
	}
	return true
}

// ResolveSpanToken re-resolves a token against the live content.
// Returns (result, resolvedLine 1-based, liveSnippet).
func ResolveSpanToken(content, token string, hintLine int) (string, int, string) {
	hash, meta, ok := parseSpanToken(token)
	if !ok {
		return SpanUnresolved, 0, ""
	}
	lines := strings.Split(content, "\n")
	size := meta.SelectedLineCount

	// Step 1 — hit at the hinted lines.
	if w := window(lines, hintLine, size); w != nil && spanHash(w) == hash {
		return SpanOK, hintLine, strings.Join(w, "\n")
	}
	// Step 2 — relocate (unchanged content, moved): prefilter by first/last
	// line hashes, then context hashes decide, then content must match.
	// Step 3 — content changed at a unique site: context matches, content
	// differs → stale with the live value (the token is NOT rewritten).
	var okHits, staleHits []int
	for start := 0; start+size <= len(lines); start++ {
		w := lines[start : start+size]
		if lineHash(w[0]) != meta.FirstSelectedLineHash || lineHash(w[size-1]) != meta.LastSelectedLineHash {
			continue
		}
		if !contextMatches(lines, start, meta) {
			continue
		}
		if spanHash(w) == hash {
			okHits = append(okHits, start)
		} else {
			staleHits = append(staleHits, start)
		}
	}
	if len(okHits) == 1 {
		w := lines[okHits[0] : okHits[0]+size]
		return SpanOK, okHits[0] + 1, strings.Join(w, "\n")
	}
	if len(staleHits) == 1 && len(okHits) == 0 {
		w := lines[staleHits[0] : staleHits[0]+size]
		return SpanStale, staleHits[0] + 1, strings.Join(w, "\n")
	}
	return SpanUnresolved, 0, ""
}

func window(lines []string, hintLine, size int) []string {
	if hintLine < 1 || hintLine+size-1 > len(lines) {
		return nil
	}
	return lines[hintLine-1 : hintLine-1+size]
}

// Check is one anchor/claim check result (006 §3).
type Check struct {
	Kind     string `json:"kind"` // token | span | symbol | claim
	Path     string `json:"path,omitempty"`
	Symbol   string `json:"symbol,omitempty"`
	Subject  string `json:"subject,omitempty"`
	Literal  string `json:"literal,omitempty"`
	Result   string `json:"result"` // ok | stale | unresolved
	Expected string `json:"expected,omitempty"`
	Actual   string `json:"actual,omitempty"`
	Note     string `json:"note,omitempty"`
	Degraded bool   `json:"degraded,omitempty"`
}

// PageCheck is one grounded page's pin-check (006 §3). Status is the worst of
// its checks (unresolved > stale > ok); a grounded page with no checks at all
// is unattested → unresolved (KB-INV-2).
type PageCheck struct {
	RelPath     string  `json:"rel_path"`
	Status      string  `json:"status"` // fresh | stale | unresolved
	Checks      []Check `json:"checks"`
	StaleChecks int     `json:"stale_checks"`
	Degraded    bool    `json:"degraded,omitempty"`
}

// ValidatePage runs the full pin-check for one grounded page: every anchor
// (token/span/symbol) + every claim against live code at repoRoot.
func ValidatePage(ctx context.Context, st store.Store, repoRoot, relPath string, fm Frontmatter) (PageCheck, error) {
	pc := PageCheck{RelPath: relPath, Status: "fresh"}
	worst := 0 // 0 ok, 1 stale, 2 unresolved
	set := func(c Check) {
		pc.Checks = append(pc.Checks, c)
		pc.Degraded = pc.Degraded || c.Degraded
		switch c.Result {
		case SpanStale:
			pc.StaleChecks++
			if worst < 1 {
				worst = 1
			}
		case SpanUnresolved:
			if worst < 2 {
				worst = 2
			}
		}
	}

	readFile := func(p string) (string, error) {
		b, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(p)))
		if err != nil {
			return "", err
		}
		return string(b), nil
	}

	anchorPaths := map[string]bool{}
	for _, a := range fm.Anchors {
		anchorPaths[a.Path] = true
		switch a.Kind() {
		case "token":
			c := Check{Kind: "token", Path: a.Path, Result: SpanOK, Expected: strings.Join(a.AllOf, " ")}
			content, err := readFile(a.Path)
			if err != nil {
				c.Result = SpanUnresolved
				c.Note = "anchor file missing"
			} else {
				for _, tok := range a.AllOf {
					if !strings.Contains(content, tok) {
						c.Result = SpanUnresolved
						c.Note = fmt.Sprintf("token %q not present", tok)
						break
					}
				}
			}
			set(c)
		case "span":
			c := Check{Kind: "span", Path: a.Path, Result: SpanOK, Expected: a.Version, Actual: a.Version}
			if a.Version == "" {
				// Bare line: repocodex degradation — ok iff the line still
				// contains every token of a co-located all_of anchor on the
				// same path; otherwise under-specified.
				var toks []string
				for _, o := range fm.Anchors {
					if o.Path == a.Path && len(o.AllOf) > 0 {
						toks = o.AllOf
						break
					}
				}
				if len(toks) == 0 {
					c.Result = SpanUnresolved
					c.Note = "bare line: no co-located all_of tokens"
				} else if content, err := readFile(a.Path); err != nil {
					c.Result = SpanUnresolved
					c.Note = "anchor file missing"
				} else {
					lines := strings.Split(content, "\n")
					line := ""
					if a.Line >= 1 && a.Line <= len(lines) {
						line = lines[a.Line-1]
					}
					for _, tok := range toks {
						if !strings.Contains(line, tok) {
							c.Result = SpanUnresolved
							c.Note = fmt.Sprintf("token %q not on line %d", tok, a.Line)
							break
						}
					}
				}
			} else if content, err := readFile(a.Path); err != nil {
				c.Result = SpanUnresolved
				c.Note = "anchor file missing"
			} else if res, line, snippet := ResolveSpanToken(content, a.Version, a.Line); res != SpanOK {
				c.Result = res
				if res == SpanStale {
					c.Actual = snippet
					c.Note = fmt.Sprintf("text changed; pin re-resolved to a unique new location (line %d)", line)
				} else {
					c.Note = "span token could not be re-resolved (fail closed)"
				}
			}
			set(c)
		case "symbol":
			set(symbolCheck(ctx, st, a))
		default:
			set(Check{Kind: "span", Path: a.Path, Result: SpanUnresolved, Note: "anchor has no selector"})
		}
	}

	for _, cl := range fm.Claims {
		sites := []string{}
		if cl.Path != "" {
			sites = append(sites, cl.Path)
		} else if len(anchorPaths) == 1 {
			for p := range anchorPaths {
				sites = append(sites, p)
			}
		} else {
			sites = anchorList(anchorPaths)
		}
		c := Check{Kind: "claim", Subject: cl.Subject, Literal: cl.Literal, Result: SpanOK, Expected: cl.Literal}
		if len(sites) == 0 {
			c.Result = SpanUnresolved
			c.Note = "claim has no anchor site (page carries no anchors)"
		} else {
			found := false
			checked := 0
			for _, p := range sites {
				content, err := readFile(p)
				if err != nil {
					continue
				}
				checked++
				if strings.Contains(content, cl.Literal) {
					c.Path = p
					found = true
					break
				}
			}
			if !found {
				if checked == 0 {
					c.Result = SpanUnresolved
					c.Note = "no claim site readable"
				} else {
					c.Result = SpanStale
					c.Note = fmt.Sprintf("literal no longer holds at %d site(s)", checked)
				}
			}
		}
		set(c)
	}

	if len(pc.Checks) == 0 {
		pc.Status = "unresolved"
		pc.Checks = append(pc.Checks, Check{Kind: "claim", Result: SpanUnresolved, Note: "grounded page carries no anchors or claims"})
	} else {
		switch worst {
		case 2:
			pc.Status = "unresolved"
		case 1:
			pc.Status = "stale"
		}
	}
	return pc, nil
}

// symbolCheck resolves a symbol anchor through the code graph (006 §2). Graph
// absent (no indexed files) → ripgrep-degraded occurrence check, degraded:true.
func symbolCheck(ctx context.Context, st store.Store, a Anchor) Check {
	c := Check{Kind: "symbol", Symbol: a.Symbol, Result: SpanOK}
	n, _ := st.Query(ctx, `SELECT COUNT(*) FROM files`)
	var total int64
	if len(n) > 0 {
		total, _ = n[0][0].(int64)
	}
	if total == 0 {
		// Degraded: occurrence check over the pinned files (call-site not checked).
		c.Degraded = true
		sites := a.CalledFrom
		if len(sites) == 0 {
			sites = []string{a.Path}
		}
		for _, p := range sites {
			if b, err := os.ReadFile(filepath.Join(repoRootOf(p), filepath.FromSlash(p))); err == nil &&
				strings.Contains(string(b), a.Symbol) {
				return c
			}
		}
		c.Result = SpanUnresolved
		c.Note = "symbol not found (degraded occurrence check, no graph)"
		return c
	}
	rows, err := st.SymbolByName(ctx, a.Symbol)
	if err != nil || len(rows) == 0 {
		c.Result = SpanUnresolved
		c.Note = "symbol not in graph"
		return c
	}
	if len(rows) > 1 {
		c.Result = SpanUnresolved
		c.Note = "symbol ambiguous in graph"
		return c
	}
	if len(a.CalledFrom) > 0 {
		edges, err := st.EdgesByNameOnlyToName(ctx, a.Symbol, []string{"calls"})
		if err != nil {
			c.Result = SpanUnresolved
			c.Note = "call-site lookup failed"
			return c
		}
		want := map[string]bool{}
		for _, cf := range a.CalledFrom {
			want[cf] = true
		}
		for _, e := range edges {
			from, err := st.SymbolByID(ctx, e.FromID)
			if err != nil {
				continue
			}
			f, err := st.FileByID(ctx, from.FileID)
			if err != nil {
				continue
			}
			if want[f.Path] {
				return c
			}
		}
		c.Result = SpanStale
		c.Note = "pinned call-site gone"
	}
	return c
}

// repoRootOf is a package var so tests can point the degraded occurrence
// check at a fixture tree (production: the CLI passes the repo root).
var repoRootOf = func(p string) string { return "." }

func anchorList(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
