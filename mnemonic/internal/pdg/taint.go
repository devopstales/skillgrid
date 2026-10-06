package pdg

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

// Taint (011 step 02): intraprocedural source->sink reachability over the
// opt-in per-function PDG data-dependence edges. It answers "does untrusted
// input reach this sink" as a query over the persisted PDG. It is pure Go
// (no CGo) and deterministic: findings are enumerated in sorted (source,
// sink) order and carry no wall-clock or map-order nondeterminism.
//
// Confidence semantics (per global constraint): every hop in a finding's path
// carries a Confidence Label in EXTRACTED | INFERRED | AMBIGUOUS |
// LSP_RESOLVED. The PATH label is the worst hop: a path is EXTRACTED only
// when EVERY hop is a resolved data-dependence (EXTRACTED). A hop through a
// call boundary resolved only by the --lsp tier is LSP_RESOLVED (a resolved
// call boundary — the path continues through it), which makes the path NOT
// EXTRACTED. A hop through an unresolved call boundary is AMBIGUOUS and the
// path TRUNCATES there ("stops at <boundary>") — never fabricated beyond it.

// SourceKind is one deterministic, configurable taint source class.
type SourceKind struct {
	// Name is the stable kind identifier (e.g. "request_param").
	Name string
	// Description is a human summary of the class (for tool output).
	Description string
	// MatchSource reports whether a source candidate (name, note) belongs to
	// this class. Deterministic (name/substring based, no env/registry lookup).
	MatchSource func(name, note string) bool
}

// SinkKind is one deterministic, configurable taint sink class.
type SinkKind struct {
	Name        string
	Description string
	MatchSink   func(name, note string) bool
}

// TaintConfig holds the source/sink sets the solver matches against. The
// defaults (DefaultTaintConfig) are deterministic built-ins; tests and callers
// may override either set (02.10 configurable). An empty set means "no
// sources/sinks of that class" — the solver finds nothing for it (never a
// fabricated match).
type TaintConfig struct {
	Sources []SourceKind
	Sinks   []SinkKind
}

// DefaultTaintConfig is the deterministic built-in source/sink set:
// sources = request params, env vars, file reads; sinks = SQL exec, shell
// exec, template render, file write. Matching is name-substring based so the
// set is stable across runs and languages (the PDG is language-agnostic).
func DefaultTaintConfig() TaintConfig {
	return TaintConfig{
		Sources: []SourceKind{
			{Name: "request_param", Description: "request parameter/header/body",
				MatchSource: nameMatchesAny("param", "query", "form", "header", "request", "route", "uri")},
			{Name: "env_var", Description: "environment variable",
				MatchSource: nameMatchesAny("env", "getenv", "environ")},
			{Name: "file_read", Description: "file read",
				MatchSource: nameMatchesAny("readfile", "readall", "fread", "open", "fgets")},
		},
		Sinks: []SinkKind{
			{Name: "sql_exec", Description: "SQL execution",
				MatchSink: nameMatchesAny("sqlexec", "query", "prepare", "sql")},
			{Name: "shell_exec", Description: "shell/command execution",
				MatchSink: nameMatchesAny("shexec", "system", "popen", "spawn")},
			{Name: "template_render", Description: "template render",
				MatchSink: nameMatchesAny("render", "template")},
			{Name: "file_write", Description: "file write",
				MatchSink: nameMatchesAny("writefile", "fwrite", "writeall", "savefile")},
		},
	}
}

// nameMatchesAny reports whether name (lowercased) contains any of subs.
func nameMatchesAny(subs ...string) func(name, note string) bool {
	return func(name, note string) bool {
		l := strings.ToLower(name)
		for _, s := range subs {
			if strings.Contains(l, s) {
				return true
			}
		}
		return false
	}
}

// TaintHop is one hop of a taint path: a PDG data-dependence edge the tainted
// value crossed, or a boundary truncation.
type TaintHop struct {
	Line       int    `json:"line"`     // the line the hop lands on
	Name       string `json:"name"`     // the statement/call at that line
	Confidence string `json:"confidence"` // EXTRACTED|INFERRED|AMBIGUOUS|LSP_RESOLVED
	Note       string `json:"note,omitempty"`
}

// TaintFinding is one source->sink taint result: the source, the sink, and the
// hop-by-hop path between them (every hop confidence-labeled). SourceKind /
// SinkKind are the configurable-set class the source/sink matched (a finding
// is source->sink, not a guess). PathLabel is the worst hop (EXTRACTED only
// when every hop is EXTRACTED). StopsAt is set when the path TRUNCATED at an
// unresolved boundary (the path never continues past it); it is "" when the
// path reaches the sink fully.
type TaintFinding struct {
	SourceLine int        `json:"source_line"`
	SourceName string     `json:"source_name"`
	SourceKind string     `json:"source_kind"`
	SinkLine   int        `json:"sink_line"`
	SinkName   string     `json:"sink_name"`
	SinkKind   string     `json:"sink_kind"`
	Path       []TaintHop `json:"path"`
	PathLabel  string     `json:"path_label"`
	StopsAt    string     `json:"stops_at,omitempty"`
}

// DataDep is one persisted PDG data-dependence edge (the solver's input).
type DataDep struct {
	FromLine   int
	ToLine     int
	FromName   string
	ToName     string
	Confidence string
	Note       string
}

// CallInfo is the solver's view of a call site: the callee name and its
// resolution status (static vs LSP tier).
type CallInfo struct {
	Name        string
	Resolved    bool
	LSPResolved bool
}

// TaintInput is everything the solver needs for one function (loaded from the
// --pdg tables). Deterministic: the loader sorts dataDeps by (from_line,
// to_line, from_name) and calls by line.
type TaintInput struct {
	SymbolID int64
	DataDeps []DataDep
	ByLine   map[int]*CallInfo // call at a line (absent when the line is not a call)
}

// Taint solves intraprocedural source->sink reachability for one function's
// PDG input and returns the findings (deterministic sorted order). It is the
// core of the step-02 solver; PersistTaint writes its output.
//
// Sources are the data-dependence sources (dataDeps' from_line) whose
// statement matches a configured source class. Sinks are any statement (a call
// from ByLine, else a dataDeps to_name at that line) matching a configured
// sink class. From each source the solver walks the data-dependence graph
// (deterministic lowest-line-first expansion), tracking the hop path:
//
//   - a data edge to a CALL SITE is a call boundary: if the callee is
//     LSPResolved the path CONTINUES through it (that hop LSP_RESOLVED — a
//     resolved call boundary, per the "feeder, not a dependency" contract);
//     if statically Resolved it continues as a resolved data-dependence
//     (EXTRACTED hop); otherwise the path TRUNCATES (AMBIGUOUS hop + "stops
//     at <boundary>" note) and the walk stops — not fabricated beyond it.
//   - a data edge to a non-call statement: the value flows into it; continue
//     (hop labeled with the edge's confidence).
//
// A COMPLETE finding is emitted when the walk reaches a sink (StopsAt ""); a
// TRUNCATED finding is emitted when the walk stops at an unresolved boundary
// (StopsAt set). A source with no path to any sink yields NO finding (never
// fabricated); every emitted finding has a non-empty path.
func Taint(in TaintInput, cfg TaintConfig) []TaintFinding {
	var findings []TaintFinding
	// Source candidates (dataDeps from_line matching a source class), sorted.
	type srcInfo struct {
		line int
		name string
		kind string
	}
	seenSrc := map[int]bool{}
	var srcs []srcInfo
	for _, d := range in.DataDeps {
		if seenSrc[d.FromLine] {
			continue
		}
		seenSrc[d.FromLine] = true
		if kind, ok := sourceKind(cfg, d.FromName, d.Note); ok {
			srcs = append(srcs, srcInfo{line: d.FromLine, name: d.FromName, kind: kind})
		}
	}
	sort.Slice(srcs, func(i, j int) bool {
		if srcs[i].line != srcs[j].line {
			return srcs[i].line < srcs[j].line
		}
		return srcs[i].name < srcs[j].name
	})
	// sinkAt: a line is a sink when its statement matches a sink class; returns
	// the sink name + the matched kind.
	sinkAt := func(line int) (string, string, bool) {
		if ci, ok := in.ByLine[line]; ok && ci != nil {
			if kind, ok := sinkKind(cfg, ci.Name, ""); ok {
				return ci.Name, kind, true
			}
		}
		for _, d := range in.DataDeps {
			if d.ToLine == line {
				if kind, ok := sinkKind(cfg, d.ToName, d.Note); ok {
					return d.ToName, kind, true
				}
			}
		}
		return "", "", false
	}
	// hopConfidence labels the hop into line: call boundaries per resolution,
	// else the data edge's confidence.
	hopConfidence := func(d DataDep, line int) string {
		if ci, ok := in.ByLine[line]; ok && ci != nil {
			switch {
			case ci.LSPResolved:
				return ConfidenceLSPResolved
			case ci.Resolved:
				return ConfidenceExtracted
			default:
				return ConfidenceAmbiguous
			}
		}
		return d.Confidence
	}
	// outgoing data edges from line, sorted deterministically.
	outgoing := func(line int) []DataDep {
		var outs []DataDep
		for _, d := range in.DataDeps {
			if d.FromLine == line {
				outs = append(outs, d)
			}
		}
		sort.Slice(outs, func(i, j int) bool {
			if outs[i].ToLine != outs[j].ToLine {
				return outs[i].ToLine < outs[j].ToLine
			}
			return outs[i].FromName < outs[j].FromName
		})
		return outs
	}
	for _, s := range srcs {
		// DFS with backtracking (deterministic sorted order): explores every
		// path from the source; the first sink reached on a branch yields a
		// complete finding; an unresolved boundary truncates that branch.
		visited := map[int]bool{s.line: true}
		var dfs func(line int, path []TaintHop)
		dfs = func(line int, path []TaintHop) {
			if sinkName, sinkKind, isSink := sinkAt(line); isSink {
				findings = append(findings, TaintFinding{
					SourceLine: s.line, SourceName: s.name, SourceKind: s.kind,
					SinkLine: line, SinkName: sinkName, SinkKind: sinkKind,
					Path: path, PathLabel: worstHop(path),
				})
				return
			}
			for _, d := range outgoing(line) {
				if visited[d.ToLine] {
					continue
				}
				ci, isCall := in.ByLine[d.ToLine]
				if isCall && ci != nil && !ci.Resolved {
					// Unresolved boundary: TRUNCATE this branch. The path is
					// reported truncated at the boundary (AMBIGUOUS + stops-at
					// note) — never fabricated beyond it. If the boundary
					// happens to be a sink the finding is a (truncated)
					// source->sink; otherwise it is a source->boundary
					// truncation (sink_name = the boundary callee).
					trunc := append(append([]TaintHop{}, path...), TaintHop{
						Line: d.ToLine, Name: d.ToName, Confidence: ConfidenceAmbiguous,
						Note: "stops at " + d.ToName,
					})
					sinkName, sinkKind, isSink := sinkAt(d.ToLine)
					if !isSink {
						sinkName = d.ToName // the boundary callee (a truncated sink)
					}
					findings = append(findings, TaintFinding{
						SourceLine: s.line, SourceName: s.name, SourceKind: s.kind,
						SinkLine: d.ToLine, SinkName: sinkName, SinkKind: sinkKind,
						Path: trunc, PathLabel: worstHop(trunc),
						StopsAt: "stops at " + d.ToName,
					})
					continue // the walk stops at the boundary
				}
				visited[d.ToLine] = true
				next := append(append([]TaintHop{}, path...), TaintHop{
					Line: d.ToLine, Name: d.ToName, Confidence: hopConfidence(d, d.ToLine), Note: d.Note,
				})
				dfs(d.ToLine, next)
				visited[d.ToLine] = false
			}
		}
		dfs(s.line, []TaintHop{{Line: s.line, Name: s.name, Confidence: ConfidenceExtracted, Note: "source"}})
	}
	// Dedupe by content key (a shared truncated boundary can be reached from
	// multiple branches) and order deterministically.
	seenKey := map[string]bool{}
	var out []TaintFinding
	for _, f := range findings {
		key := fmt.Sprintf("%d|%s|%d|%s|%s|%s", f.SourceLine, f.SourceName, f.SinkLine, f.SinkName, f.PathLabel, pathKey(f.Path))
		if seenKey[key] {
			continue
		}
		seenKey[key] = true
		out = append(out, f)
	}
	sortFindings(out)
	return out
}

// sourceKind returns the first matching source class name (or "" when none
// matches). A finding is source->sink (a matched class), not a guess.
func sourceKind(cfg TaintConfig, name, note string) (string, bool) {
	for _, s := range cfg.Sources {
		if s.MatchSource != nil && s.MatchSource(name, note) {
			return s.Name, true
		}
	}
	return "", false
}

// sinkKind returns the first matching sink class name (or "" when none
// matches). A finding is source->sink (a matched class), not a guess.
func sinkKind(cfg TaintConfig, name, note string) (string, bool) {
	for _, s := range cfg.Sinks {
		if s.MatchSink != nil && s.MatchSink(name, note) {
			return s.Name, true
		}
	}
	return "", false
}

// worstHop returns the path-level label: EXTRACTED only when every hop is
// EXTRACTED (a resolved data-dependence); otherwise the worst hop present
// (any AMBIGUOUS wins, then any INFERRED, then any LSP_RESOLVED).
func worstHop(path []TaintHop) string {
	sawAmb, sawInf, sawLSP := false, false, false
	for _, h := range path {
		switch h.Confidence {
		case ConfidenceAmbiguous:
			sawAmb = true
		case ConfidenceInferred:
			sawInf = true
		case ConfidenceLSPResolved:
			sawLSP = true
		}
	}
	switch {
	case sawAmb:
		return ConfidenceAmbiguous
	case sawInf:
		return ConfidenceInferred
	case sawLSP:
		return ConfidenceLSPResolved
	default:
		return ConfidenceExtracted
	}
}

// sortFindings orders findings deterministically (source line, sink line,
// source name, sink name, path key) so repeated runs are byte-for-byte stable.
func sortFindings(findings []TaintFinding) {
	sort.SliceStable(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		if a.SourceLine != b.SourceLine {
			return a.SourceLine < b.SourceLine
		}
		if a.SinkLine != b.SinkLine {
			return a.SinkLine < b.SinkLine
		}
		if a.SourceName != b.SourceName {
			return a.SourceName < b.SourceName
		}
		if a.SinkName != b.SinkName {
			return a.SinkName < b.SinkName
		}
		return pathKey(a.Path) < pathKey(b.Path)
	})
}

// pathKey renders a path to a stable string (for dedupe/sort).
func pathKey(path []TaintHop) string {
	var sb strings.Builder
	for i, h := range path {
		if i > 0 {
			sb.WriteByte('¦')
		}
		fmt.Fprintf(&sb, "%d:%s:%s", h.Line, h.Name, h.Confidence)
	}
	return sb.String()
}

// PersistTaint writes a function's taint findings into taint_findings
// (target-state: clears the function's prior rows first so a re-index is
// idempotent). The persisted row carries the path-level confidence label
// (worst-hop) and the stops-at note (when truncated); the per-hop path is
// derived in-memory by the solver and is not stored as a separate column.
func PersistTaint(db *sql.DB, symbolID int64, findings []TaintFinding) error {
	if _, err := db.Exec(`DELETE FROM taint_findings WHERE symbol_id = ?`, symbolID); err != nil {
		return err
	}
	for _, f := range findings {
		if _, err := db.Exec(`
			INSERT INTO taint_findings (symbol_id, source_line, sink_line, source_name, sink_name, confidence, note)
			VALUES (?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(symbol_id, source_line, sink_line, source_name, sink_name, confidence) DO NOTHING`,
			symbolID, f.SourceLine, f.SinkLine, f.SourceName, f.SinkName, f.PathLabel,
			truncNote(f)); err != nil {
			return fmt.Errorf("persist taint finding: %w", err)
		}
	}
	return nil
}

// truncNote renders the persisted note for a finding (the stops-at note when
// the path truncated at an unresolved boundary, else the path label).
func truncNote(f TaintFinding) string {
	if f.StopsAt != "" {
		return f.StopsAt
	}
	return f.PathLabel
}
