package route

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

// RouteNode is one route node to be upserted into the 005 symbols table.
type RouteNode struct {
	Name            string
	UID             string
	PathPattern     string
	Method          string
	Framework       string
	Language        string
	Line            int
	ContentHash     string
	HandlerName     string
	HandlerSymbol   int64 // 0 when the handler could not be resolved
	HandlerUID      string
	HandlerExplicit bool
	// HandlerConfidence is the Confidence Label for the route->handler
	// references edge: EXTRACTED for a same-file/explicit handler, AMBIGUOUS
	// for a name-only best-effort guess. Empty when the handler was dropped.
	HandlerConfidence string
}

// NavigationNode is one navigation edge to be upserted into the 005 edges
// table (kind = navigates).
type NavigationNode struct {
	FromSymbol int64
	ToName     string // the screen (path-like)
	Confidence string
	Line       int
}

// Build is the per-framework node map. Given the raw extraction for a file,
// the symbols extracted by 005 (for the same file), and the symbol index, it
// produces the route nodes + references/navigates edges. This is the single
// place where the drop-rather-than-guess policy is applied to references.
func Build(path string, src []byte, fileSyms []FileSymbol, index SymbolIndex) *Built {
	fr := ExtractFile(path, src)
	if fr == nil {
		fr = &FileRoutes{}
	}
	sortRoutes(fr)

	built := &Built{
		Framework: fr.Framework,
		Nodes:     make([]RouteNode, 0, len(fr.Routes)),
		Navs:      make([]NavigationNode, 0, len(fr.Navigates)),
	}
	built.buildRoutes(path, src, fr, fileSyms, index)
	built.buildNavigates(path, src, fr, fileSyms, index)
	return built
}

// Built is the output of Build: the route nodes and navigates edges to store,
// plus the drop-not-guess warnings.
type Built struct {
	Framework  string
	Nodes      []RouteNode
	Navs       []NavigationNode
	Dropped    int
	DropSample string
	Drops      []DroppedRef
}

// DroppedRef is one unresolved reference dropped at extraction, logged for
// the unresolved_refs table (never stored as an edge).
type DroppedRef struct {
	Name       string
	Kind       string // "route_handler" | "navigation"
	Line       int
	Candidates int // global symbols matched by name
}

// Store is the narrow persistence surface the indexer uses to persist route
// nodes + references/navigates edges in the SAME transaction as the 005
// symbol/edge extraction. The indexer implements it over *sql.Tx.
type Store interface {
	// StoreRouteNode upserts a route node into the 005 symbols table
	// (kind=route) and returns its resolved symbol id. The 005 upsert is
	// byte-identical to the one the indexer already performs for 005 symbols.
	StoreRouteNode(node RouteNode) (int64, error)
	// StoreReferencesEdges upserts the references (route -> handler) edges
	// for a file's route nodes. It prunes the file's prior route-originated
	// references/navigates edges first (target-state), then upserts.
	StoreReferencesEdges(fileID int64, nodes []RouteNode, fileSymbolID int64) (int, error)
	// StoreNavigatesEdges upserts the navigates (function -> screen) edges.
	StoreNavigatesEdges(fileID int64, navs []NavigationNode) (int, error)
	// StoreRouteDrops records the per-file drop-not-guess count.
	StoreRouteDrops(fileID int64, dropped int) error
	// StoreRouteMeta records the framework-specific route metadata.
	StoreRouteMeta(fileID int64, symbolID int64, node RouteNode) error
	// FirstSymbolID returns the file's first 005 symbol id (the default
	// source for top-level navigates edges), or 0 when the file has none.
	FirstSymbolID(fileID int64) (int64, error)
	// ResolveHandler resolves a handler name to its symbol id + uid +
	// confidence, applying the drop-not-guess policy: a same-file (explicit)
	// match resolves EXTRACTED; a name-only match that hits a unique global
	// symbol resolves AMBIGUOUS (a best-effort guess, stored low-confidence);
	// multiple or zero global matches are unresolvable → (0, "", "") so the
	// caller drops the reference. candidates is the number of global symbols
	// matched by name (logged when the ref is dropped).
	ResolveHandler(fileID int64, name string) (id int64, uid string, confidence string, candidates int)
	// StoreUnresolvedRefs persists the file's dropped references
	// (target-state: the file's prior rows are replaced).
	StoreUnresolvedRefs(fileID int64, refs []DroppedRef, now string) error
}

// FileSymbol is a minimal view of one 005-extracted symbol in the file, used
// to resolve route handlers. It is intentionally narrow so the route package
// does not import the extract package (no cycle).
type FileSymbol struct {
	Name string
	UID  string
	ID   int64
}

// Run extracts routes/navigations for one file and persists them through st in
// the SAME transaction the indexer already holds. It never errors on a
// malformed file: an empty extraction is persisted as zero rows and the
// indexer continues. It returns the number of route nodes + navigates edges
// stored (for stats). fileSyms may be nil (the Store resolves them lazily).
func Run(ctx context.Context, st Store, fileID int64, path string, src []byte, fileSyms []FileSymbol) (int, error) {
	_ = ctx
	if fileSyms == nil {
		fileSyms = []FileSymbol{}
	}
	b := Build(path, src, fileSyms, resolver{st, fileID})
	if b == nil {
		return 0, nil
	}
	_ = b
	stored := 0
	for _, n := range b.Nodes {
		symID, err := st.StoreRouteNode(n)
		if err != nil {
			return stored, fmt.Errorf("store route node %s: %w", n.Name, err)
		}
		if err := st.StoreRouteMeta(fileID, symID, n); err != nil {
			return stored, fmt.Errorf("store route meta %s: %w", n.Name, err)
		}
		stored++
	}
	if n, err := st.StoreReferencesEdges(fileID, b.Nodes, 0); err != nil {
		return stored, err
	} else {
		stored += n
	}
	if n, err := st.StoreNavigatesEdges(fileID, b.Navs); err != nil {
		return stored, err
	} else {
		stored += n
	}
	if err := st.StoreRouteDrops(fileID, b.Dropped); err != nil {
		return stored, err
	}
	if err := st.StoreUnresolvedRefs(fileID, b.Drops, time.Now().UTC().Format(time.RFC3339)); err != nil {
		return stored, err
	}
	return stored, nil
}

// resolver adapts a Store into the SymbolIndex that Build uses, binding the
// per-file handler resolution to the file's own symbols first.
type resolver struct {
	st     Store
	fileID int64
}

func (r resolver) ResolveHandler(name string) Resolution {
	id, uid, conf, c := r.st.ResolveHandler(r.fileID, name)
	if id == 0 {
		return Resolution{Candidates: c}
	}
	return Resolution{ID: id, UID: uid, Confidence: conf, Candidates: c}
}

// SymbolIndex resolves handler names to their symbol IDs. It is backed by the
// 005 symbols table. It returns a Resolution whose confidence labels how the
// name was resolved: EXTRACTED for a same-file (explicit) match, AMBIGUOUS for
// a name-only best-effort guess (no same-file / explicit match, but a unique
// global match was taken), or Unresolved for the drop-not-guess case.
type SymbolIndex interface {
	ResolveHandler(name string) Resolution
}

// Resolution is the outcome of resolving a handler reference.
type Resolution struct {
	ID         int64
	UID        string
	Confidence string // ConfidenceExtracted | ConfidenceAmbiguous | ""
	// Candidates is the number of global symbols matched by name — always
	// reported (even when a same-file match wins) so a dropped ref can log it.
	Candidates int
}

// Resolved reports whether a handler symbol was found (ok).
func (r Resolution) Resolved() bool { return r.ID != 0 }

// buildRoutes turns each raw Route into a RouteNode + (optionally) a
// references edge, applying the drop-not-guess policy to the handler
// reference.
func (b *Built) buildRoutes(path string, src []byte, fr *FileRoutes, fileSyms []FileSymbol, index SymbolIndex) {
	for _, r := range fr.Routes {
		name := routeNodeName(r.Framework, r.PathPattern, r.Method)
		uid := symbolUID(r.Framework, r.PathPattern, r.Method)
		node := RouteNode{
			Name:            name,
			UID:             uid,
			PathPattern:     r.PathPattern,
			Method:          r.Method,
			Framework:       r.Framework,
			Language:        strings.TrimPrefix(filepathExt(path), "."),
			Line:            r.Line,
			ContentHash:     uid,
			HandlerName:     r.Handler,
			HandlerExplicit: r.Explicit,
		}
		if node.Language == "" {
			node.Language = "unknown"
		}
		// Resolve the handler reference. The route node is always created (it
		// is the URL endpoint); the references edge is dropped when the
		// handler is unresolvable (drop-not-guess). A same-file match resolves
		// EXTRACTED; a name-only best-effort guess (unique global hit) resolves
		// AMBIGUOUS (stored, low-confidence).
		if r.Handler != "" {
			res := index.ResolveHandler(r.Handler)
			if !res.Resolved() {
				// Drop-not-guess: no same-file match, no unique global match.
				b.recordDrop(r.Handler, "route_handler", r.Line, res.Candidates)
			} else {
				node.HandlerSymbol = res.ID
				node.HandlerUID = res.UID
				// Default to the explicit confidence; the index may downgrade
				// a name-only guess to AMBIGUOUS.
				node.HandlerConfidence = ConfidenceExtracted
				if res.Confidence != "" {
					node.HandlerConfidence = res.Confidence
				}
			}
		}
		b.Nodes = append(b.Nodes, node)
	}
}

// buildNavigates turns each raw Navigation into a navigates edge. Literal,
// programmatic destinations are EXTRACTED; markup-written links are INFERRED.
// A destination that is computed (not a literal) is left unresolved — it is
// never stored with a fabricated screen, so it is dropped from Navs.
func (b *Built) buildNavigates(path string, src []byte, fr *FileRoutes, fileSyms []FileSymbol, index SymbolIndex) {
	// The sending function is the nearest enclosing function symbol; when the
	// navigation is at top level (module scope) the source is the file's
	// first symbol.
	var fromID int64
	if len(fileSyms) > 0 {
		fromID = fileSyms[0].ID
	}
	for _, n := range fr.Navigates {
		if n.Dest == "" {
			// Computed / no-literal destination: unresolved, not fabricated.
			b.recordDrop(n.Screen, "navigation", n.Line, 0)
			continue
		}
		conf := ConfidenceExtracted
		if n.Markup {
			conf = ConfidenceInferred
		}
		b.Navs = append(b.Navs, NavigationNode{
			FromSymbol: fromID,
			ToName:     n.Screen,
			Confidence: conf,
			Line:       n.Line,
		})
	}
}

// recordDrop tallies a dropped reference for the warning and logs it for the
// unresolved_refs table.
func (b *Built) recordDrop(name, kind string, line, candidates int) {
	b.Dropped++
	if b.DropSample == "" {
		b.DropSample = name
	}
	b.Drops = append(b.Drops, DroppedRef{Name: name, Kind: kind, Line: line, Candidates: candidates})
}

// routeNodeName derives a readable route node name (method + pattern).
func routeNodeName(framework, pattern, method string) string {
	if method == "" {
		return "route:" + pattern
	}
	return method + " " + pattern
}

// filepathExt returns the lowercased extension (without dot) of path.
func filepathExt(path string) string {
	idx := strings.LastIndex(path, ".")
	if idx < 0 {
		return ""
	}
	return strings.ToLower(path[idx+1:])
}

// SortedHandlers returns the sorted unique handler names referenced by a Built
// set (for deterministic drop samples).
func (b *Built) SortedHandlers() []string {
	seen := map[string]bool{}
	var out []string
	for _, n := range b.Nodes {
		if n.HandlerName != "" && !seen[n.HandlerName] {
			seen[n.HandlerName] = true
			out = append(out, n.HandlerName)
		}
	}
	sort.Strings(out)
	return out
}
