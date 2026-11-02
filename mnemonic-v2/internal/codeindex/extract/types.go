// Package extract turns a source file into a queryable graph of symbols,
// edges, and chunks via a 3-tier tree-sitter ladder (rich-6 one-pass →
// map-tier node-type data table → regex fallback), with a self-healing
// grammar pool, a real content-hash extractor stamp (D-STAMP), an AST-first
// chunker, and knowledge-node classification. It is the extraction engine
// TICKET-01's codeindex.Index ExtractFn seam wires into.
package extract

// Tier is the per-language extraction tier (D-TIER) recorded in the
// index_freshness extractor_tiers map.
type Tier string

const (
	// TierRich is the gotreesitter one-pass tier (the 6 rich languages:
	// go, python, java, javascript, typescript, tsx).
	TierRich Tier = "rich"
	// TierMap is the node-type data-table tier (the 24 distinct maps + the
	// c→cpp alias = 25 rows).
	TierMap Tier = "map"
	// TierRegex is the regex fallback tier (unmapped extensions, malformed
	// files, quarantined grammars) — the deterministic floor (INV-3).
	TierRegex Tier = "regex"
)

// Symbol is one extracted code unit (function, method, type, ...). It mirrors
// the store.Symbol shape the driver writes (the extract package owns its own
// type so the ladder has no store import; the driver converts).
type Symbol struct {
	Name          string
	QualifiedName string
	Kind          string
	Language      string
	Signature     string
	StartLine     int
	EndLine       int
	Visibility    string
	IsExported    bool
	ReturnType    string
	ParamTypes    string
	ContentHash   string
	UID           string
}

// Edge connects two symbols (or a symbol and an unresolved target name). ToID
// is a pointer: nil = name-only (unresolved — CI-INV-3, drop-not-guess).
type Edge struct {
	Kind       string
	FromID     int64
	ToID       *int64
	ToName     string
	Confidence string
	Line       int
	Context    string
}

// UnresolvedRef is one "we saw this reference but could not resolve it" ledger
// row (CI-INV-3 — the name-only edge's companion record).
type UnresolvedRef struct {
	Reference  string
	Kind       string
	Line       int
	Candidates string
}

// Chunk is one FTS + semantic-leg text unit. Kind is "ast" | "lines".
type Chunk struct {
	Kind        string
	StartLine   int
	EndLine     int
	Text        string
	ContentHash string
}

// FileGraph is the per-file extraction output: the file's symbols, edges,
// unresolved ledger rows, and chunks. Indexed=false (oversized) is the
// not-indexed outcome the driver counts in files_oversized.
type FileGraph struct {
	Symbols    []Symbol
	Edges      []Edge
	Unresolved []UnresolvedRef
	Chunks     []Chunk
	// Oversized is true when the file was skipped for being over the max
	// size (the driver counts it, does not write it).
	Oversized bool
	// Tier is the extraction tier that produced this graph (D-TIER).
	Tier Tier
	// Error is set when the primary extractor failed and the fallback was
	// used (the quarantine/panic path).
	Error string
}

// ExtractOption configures an ExtractFile call.
type ExtractOption func(*extractConfig)

type extractConfig struct {
	maxFileSize int
	pool        *Pool
}

// defaultMaxFileSize is the oversized threshold (500 KiB — the spec 001 §2
// max-file-size constant). A file over it is counted, not indexed.
const defaultMaxFileSize = 500 * 1024

// WithMaxFileSize sets the oversized threshold. A file over it is not indexed
// (counted in files_oversized) — the T1 oversized outcome.
func WithMaxFileSize(n int) ExtractOption {
	return func(c *extractConfig) {
		if n > 0 {
			c.maxFileSize = n
		}
	}
}

// WithPool supplies the self-healing grammar pool (the circuit-breaker state).
// A nil pool (the default) means a fresh per-call pool (stateless extraction).
func WithPool(p *Pool) ExtractOption {
	return func(c *extractConfig) {
		c.pool = p
	}
}

// applyOptions folds the variadic options over the defaults.
func applyOptions(opts []ExtractOption) extractConfig {
	c := extractConfig{maxFileSize: defaultMaxFileSize, pool: NewPool(PoolConfig{})}
	for _, o := range opts {
		if o != nil {
			o(&c)
		}
	}
	return c
}
