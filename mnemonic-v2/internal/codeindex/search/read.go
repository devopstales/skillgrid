package search

import (
	"context"
	"os"
	"regexp"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// ReadMode is the code_read mode (spec 006 §7 — the progressive-disclosure
// drill handle, T13). full (default) = the live source span; content = the
// drill handle (signature + neighbor ids, no bodies; a file → the TOC).
type ReadMode string

const (
	ReadModeFull    ReadMode = "full"
	ReadModeContent ReadMode = "content"
)

// NeighborID is one neighbor pointer in a content-mode read (the callee/caller
// ids the agent copies to code_read deeper — the bodies are empty, T13).
type NeighborID struct {
	ToID       int64
	ToName     string
	Kind       string
	Confidence string
	Body       string // empty in content mode
}

// ReadResult is the code_read answer. Text is the live source span (full
// mode) or empty (content mode on a file/symbol — the TOC / neighbor ids
// carry the answer). TOC is the file's symbol list (content mode on a file).
// NeighborIDs are the callee/caller pointers (content mode on a symbol).
// Signature is the symbol's signature (content mode on a symbol).
type ReadResult struct {
	Text        string
	TOC         []store.SymbolRow
	NeighborIDs []NeighborID
	Signature   string
	Degraded    string
}

// secretRe matches a secret-bearing assignment: token= / password= / secret=
// (and the common variants), with the VALUE masked but the KEY kept (the v1
// property — spec 006 §7, T13). Group 1 = key+operator, group 3 = quote (or
// empty), group 4 = value. RE2 has no backreferences, so the closing quote is
// not matched (the value charset excludes the quote, so the mask is clean).
var secretRe = regexp.MustCompile(`(?i)(\b(token|password|passwd|secret|api[_-]?key|access[_-]?key|private[_-]?key|bearer)\b\s*[:=]\s*)(["']?)([A-Za-z0-9+/._~-]{1,})`)

// redactSecrets masks secret values in a source span, keeping the keys (the v1
// property — values masked, keys kept).
func redactSecrets(text string) string {
	return secretRe.ReplaceAllStringFunc(text, func(m string) string {
		mm := secretRe.FindStringSubmatch(m)
		if mm == nil {
			return m
		}
		// mm[1] = key+operator, mm[3] = opening quote (or empty), mm[4] = value.
		// The value charset excludes the quote, so re-appending the opening
		// quote + mask + the same quote is clean for a quoted value, and a bare
		// mask for an unquoted one.
		return mm[1] + mm[3] + "********" + mm[3]
	})
}

// Read returns the live source span for path/start..end (full mode) or the
// file TOC (content mode — the indexed symbol list, line order; T13 drill
// handle) (spec 006 §7). Full mode reads the WORKING TREE (live), not a
// snapshot; secret values in the fetched slice are masked (T13). st is the
// Store (the content-mode TOC comes from the index: FileByPath + SymbolsByFile).
func Read(ctx context.Context, st store.Store, path string, start, end int, mode ReadMode) (ReadResult, error) {
	if mode == ReadModeContent {
		// Content mode on a file → the TOC, not the body (T13 drill handle).
		file, err := st.FileByPath(ctx, path)
		if err != nil {
			// Not indexed → no TOC (empty, not an error: the file may simply
			// not be in the index).
			return ReadResult{}, nil
		}
		toc, err := st.SymbolsByFile(ctx, file.ID)
		if err != nil {
			return ReadResult{}, err
		}
		return ReadResult{TOC: toc}, nil
	}
	// Full mode: the live source span.
	data, err := os.ReadFile(path)
	if err != nil {
		return ReadResult{}, err
	}
	lines := strings.Split(string(data), "\n")
	lo := start
	if lo < 1 {
		lo = 1
	}
	hi := end
	if hi < lo || hi > len(lines) {
		hi = len(lines)
	}
	span := strings.Join(lines[lo-1:hi], "\n")
	return ReadResult{Text: redactSecrets(span)}, nil
}

// ReadSymbol returns the symbol's source span (full mode — start_line..end_line)
// or the drill handle (content mode — the signature + callee/caller ids, no
// bodies; T13). st is the Store (for the symbol row + the graph edges).
func ReadSymbol(ctx context.Context, st store.Store, symID int64, mode ReadMode) (ReadResult, error) {
	sym, err := st.SymbolByID(ctx, symID)
	if err != nil {
		return ReadResult{}, err
	}
	if mode == ReadModeContent {
		// Content mode: the signature + callee/caller ids (the drill handle,
		// T13). The neighbors are the symbol's calls/references edges (forward
		// = callees, reverse = callers), confidence-labeled; their bodies are
		// empty (the agent copies an id to code_read deeper).
		var neighbors []NeighborID
		callerKinds := []string{"calls", "references"}
		// Forward (callees).
		fwd, err := st.EdgesByFrom(ctx, symID, callerKinds)
		if err == nil {
			for _, e := range fwd {
				n := NeighborID{Kind: e.Kind, Confidence: e.Confidence}
				if e.ToID != nil {
					n.ToID = *e.ToID
				}
				n.ToName = e.ToName
				neighbors = append(neighbors, n)
			}
		}
		// Reverse (callers).
		rev, err := st.EdgesByTo(ctx, symID, callerKinds)
		if err == nil {
			for _, e := range rev {
				n := NeighborID{ToID: e.FromID, Kind: e.Kind, Confidence: e.Confidence}
				// Resolve the caller's name (best-effort).
				if c, err := st.SymbolByID(ctx, e.FromID); err == nil {
					n.ToName = c.Name
				}
				neighbors = append(neighbors, n)
			}
		}
		return ReadResult{
			Signature:   sym.Signature,
			NeighborIDs: neighbors,
		}, nil
	}
	// Full mode: the symbol's source span (start_line..end_line) from the live
	// working tree.
	if sym.FileID == 0 {
		return ReadResult{Signature: sym.Signature}, nil
	}
	file, err := st.FileByID(ctx, sym.FileID)
	if err != nil {
		return ReadResult{Signature: sym.Signature}, nil
	}
	data, err := os.ReadFile(file.Path)
	if err != nil {
		return ReadResult{Signature: sym.Signature, Degraded: "file_not_found"}, nil
	}
	lines := strings.Split(string(data), "\n")
	lo := sym.StartLine
	if lo < 1 {
		lo = 1
	}
	hi := sym.EndLine
	if hi < lo || hi > len(lines) {
		hi = len(lines)
	}
	span := strings.Join(lines[lo-1:hi], "\n")
	return ReadResult{
		Text:      redactSecrets(span),
		Signature: sym.Signature,
	}, nil
}
