package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// The universal context envelope (014 step 22) is a single, size-bounded JSON
// document an agent can hand to another agent (or load into a context window)
// to resume work on a project. It combines four sections:
//
//   - project       — name, indexed file count, and detected languages
//   - working_set   — files edited during the session (22.1), with edit
//                     counts, net line deltas, and hub-file flags
//   - intent        — the classification of the current work (22.2): one of
//                     exploration / debugging / review / refactor
//   - matched_skills— the skill names most relevant to the intent
//   - handoff_refs  — pointers to the step-21 handoff artifacts
//
// The working set is in-memory and session-scoped (a CLI process cannot rely on
// prior in-process state, so it is built fresh per invocation from explicit
// RecordEdit calls). The envelope itself is assembled by
// NewContextEnvelope / GenerateContextEnvelope.

// WorkingFile is one file in the session's working set (22.1). EditCount is
// the number of distinct edits recorded; NetLines is (lines added - lines
// removed) accumulated across all edits; IsHub flags hub files (the same
// fixed hub-file list the step-21 handoff uses).
type WorkingFile struct {
	Path      string `json:"path"`
	EditCount int    `json:"edit_count"`
	NetLines  int    `json:"net_lines"`
	IsHub     bool   `json:"is_hub"`
}

// WorkingSet tracks the files edited during a session (22.1). It is in-memory
// and session-scoped. NewWorkingSet takes the project root: edits to files
// OUTSIDE that root are excluded (the working set is a project-local view of
// what the agent touched). RecordEdit accumulates per-file edit counts and net
// line deltas; Files() returns a deterministically sorted snapshot.
type WorkingSet struct {
	projectRoot string
	files       map[string]*WorkingFile
}

// NewWorkingSet creates an empty working set scoped to projectRoot. An empty
// root means "no scope filter" — every recorded path is accepted.
func NewWorkingSet(projectRoot string) *WorkingSet {
	return &WorkingSet{
		projectRoot: strings.TrimRight(projectRoot, "/"),
		files:       map[string]*WorkingFile{},
	}
}

// RecordEdit registers one edit to path. linesAdded and linesRemoved are the
// per-edit line counts (net = added - removed is accumulated across edits).
// A path that is not under the project root (when a root is set) is excluded
// from the working set — the working set is a project-local view.
func (ws *WorkingSet) RecordEdit(path string, linesAdded, linesRemoved int) {
	if !ws.inProject(path) {
		return
	}
	rel := filepath.ToSlash(path)
	f, ok := ws.files[rel]
	if !ok {
		f = &WorkingFile{Path: rel, IsHub: isHubFile(rel)}
		ws.files[rel] = f
	}
	f.EditCount++
	f.NetLines += linesAdded - linesRemoved
}

// inProject reports whether path is under the project root. With no root set,
// every path is in-project.
func (ws *WorkingSet) inProject(path string) bool {
	if ws.projectRoot == "" {
		return true
	}
	root := filepath.ToSlash(ws.projectRoot)
	p := filepath.ToSlash(path)
	return p == root || strings.HasPrefix(p, root+"/")
}

// Files returns the working set as a deterministically (path-sorted) slice. A
// nil/empty set yields an empty (non-nil) slice so the JSON is [] not null.
func (ws *WorkingSet) Files() []WorkingFile {
	out := make([]WorkingFile, 0, len(ws.files))
	for _, f := range ws.files {
		out = append(out, *f)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// workIntentKeywords maps each intent to the work-query phrases that signal it
// (22.2). The patterns are checked in a fixed order (debugging before review
// before refactor before exploration) so a query like "fix the null pointer"
// classifies as debugging even though it contains no exploration marker.
//
// Note: this is distinct from the step-19 retrieval.ClassifyIntent (which
// covers a different keyword set for search queries). Step 22.2 requires the
// work-intent patterns (fix, extract, check, PR, etc.) that step 19 does not
// include, so this is a separate keyword table that reuses the shared Intent
// type to avoid a name collision while keeping the 4-intent contract.
var workIntentKeywords = []struct {
	intent   Intent
	keywords []string
}{
	{IntentDebugging, []string{
		"fix", "bug", "error", "crash", "fail", "broken", "stack trace",
		"exception", "panic", "regress", "why does", "why is",
	}},
	{IntentReview, []string{
		"check", "review", "pr", "pull request", "diff", "merge",
	}},
	{IntentRefactor, []string{
		"extract", "rename", "move", "split", "refactor", "restructure",
		"cleanup", "clean up",
	}},
	// Exploration is the default: no explicit keywords needed (what, where,
	// list, show are the common exploration markers but the default catch-all
	// handles them without listing every possible word).
}

// ClassifyWorkIntent classifies a work query as one of exploration /
// debugging / review / refactor (22.2). It uses the workIntentKeywords table
// (checked in a fixed deterministic order) and defaults to exploration when no
// keyword matches. It reuses the step-19 Intent type (the same 4-intent set)
// so the context envelope's intent is typed consistently across the memory
// package, but has its own keyword patterns because the step-22 work-intent
// vocabulary (fix, extract, check, PR) differs from step 19's search-query
// vocabulary.
func ClassifyWorkIntent(query string) Intent {
	q := strings.ToLower(strings.TrimSpace(query))
	for _, c := range workIntentKeywords {
		for _, kw := range c.keywords {
			if strings.Contains(q, kw) {
				return c.intent
			}
		}
	}
	return IntentExploration
}

// ProjectMeta is the project section of the envelope (22.3): the project
// identity, its indexed file count, and the languages detected in the index.
type ProjectMeta struct {
	Name      string   `json:"name"`
	FileCount int      `json:"file_count"`
	Languages []string `json:"languages"`
}

// ContextEnvelope is the universal context envelope (22.3). It is the
// project-metadata + working-set + intent + matched-skills + handoff-refs
// JSON document that is size-bounded at serialization (see MarshalJSON).
type ContextEnvelope struct {
	Project       ProjectMeta   `json:"project"`
	WorkingSet    EnvelopeWS    `json:"working_set"`
	Intent        Intent        `json:"intent"`
	MatchedSkills []string      `json:"matched_skills"`
	HandoffRefs   []string      `json:"handoff_refs"`
}

// EnvelopeWS is the envelope's working-set section: the files (and their edit
// counts / net deltas / hub flags) plus a summary count.
type EnvelopeWS struct {
	Files []WorkingFile `json:"files"`
}

// GenerateContextEnvelope assembles the full context envelope (22.3) for the
// project by reading the store: the project's indexed file count and detected
// languages (the distinct symbols.language values), the working set (session
// edits), the classified intent, the matched skills (by intent), and the
// handoff refs (the step-21 handoff.latest.json path under the data dir). The
// working set is passed in (it is session-scoped in-memory state the CLI
// process builds per invocation).
func (s *Service) GenerateContextEnvelope(ctx context.Context, ws *WorkingSet, intent Intent, dataDir string) (*ContextEnvelope, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil, fmt.Errorf("memory service not initialized")
	}
	files, err := s.codeindexFiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("list codeindex files: %w", err)
	}
	languages, err := s.distinctLanguages(ctx)
	if err != nil {
		return nil, fmt.Errorf("distinct languages: %w", err)
	}
	env := NewContextEnvelope(ws, intent)
	env.Project = ProjectMeta{
		Name:      s.projectID,
		FileCount: len(files),
		Languages: languages,
	}
	env.MatchedSkills = MatchedSkillsForIntent(intent)
	if ref := HandoffPath(dataDir); ref != "" {
		env.HandoffRefs = []string{ref}
	}
	return env, nil
}

// distinctLanguages returns the sorted distinct language values present in the
// codeindex symbols table (the languages actually indexed for the project). A
// missing/empty index yields an empty (non-nil) slice so the JSON is [] not
// null. Best-effort: a query failure returns an error to the caller.
func (s *Service) distinctLanguages(ctx context.Context) ([]string, error) {
	rows, err := s.store.DB.QueryContext(ctx,
		`SELECT DISTINCT language FROM symbols WHERE language IS NOT NULL AND language != '' ORDER BY language`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]string, 0)
	for rows.Next() {
		var lang string
		if err := rows.Scan(&lang); err != nil {
			return nil, err
		}
		out = append(out, lang)
	}
	return out, rows.Err()
}

// MatchedSkillsForIntent returns the skill names most relevant to an intent
// (22.3). This is a small, deterministic intent→skills map so the envelope can
// suggest the skills an agent should load for the current work without
// consulting the (optional) skill registry. An unknown intent maps to [].
func MatchedSkillsForIntent(intent Intent) []string {
	switch intent {
	case IntentDebugging:
		return []string{"debugging", "tdd"}
	case IntentReview:
		return []string{"review-reception", "requesting-code-review"}
	case IntentRefactor:
		return []string{"codebase-design", "work-unit-commits"}
	case IntentExploration:
		return []string{"sdd-explore", "questioning"}
	default:
		return []string{}
	}
}

// NewContextEnvelope assembles a ContextEnvelope from a working set and a
// classified intent. The project metadata, matched skills, and handoff refs are
// filled in by the caller (or by GenerateContextEnvelope when the project store
// is available).
func NewContextEnvelope(ws *WorkingSet, intent Intent) *ContextEnvelope {
	files := ws.Files()
	// A nil working set yields a non-nil empty files slice (JSON [] not null).
	if files == nil {
		files = []WorkingFile{}
	}
	env := &ContextEnvelope{
		WorkingSet:    EnvelopeWS{Files: files},
		Intent:        intent,
		MatchedSkills: []string{},
		HandoffRefs:   []string{},
	}
	return env
}

// defaultEnvelopeMaxSize is the cap on the serialized envelope (22.3): the
// `mnemonic.envelope.max_size` config default of 64KB.
const defaultEnvelopeMaxSize = 64 * 1024

// envelopeWire is the plain serialization shape of the envelope. It has no
// custom MarshalJSON method (so json.Marshal on it does not recurse back into
// ContextEnvelope.MarshalJSON). ContextEnvelope.MarshalJSON copies itself into
// this type and marshals it, enforcing the size limit.
type envelopeWire struct {
	Project       ProjectMeta   `json:"project"`
	WorkingSet    EnvelopeWS    `json:"working_set"`
	Intent        Intent        `json:"intent"`
	MatchedSkills []string      `json:"matched_skills"`
	HandoffRefs   []string      `json:"handoff_refs"`
}

// MarshalJSON serializes the envelope and enforces the size limit (22.3). If
// the full envelope exceeds the limit, the working set is truncated (smallest
// net-delta files dropped first, most-changed files kept last) until it fits —
// the envelope must never grow unbounded. Uses the 64KB default limit.
func (e *ContextEnvelope) MarshalJSON() ([]byte, error) {
	return e.marshalJSON(defaultEnvelopeMaxSize)
}

// marshalJSON is the size-bounded encoder. It first marshals the full envelope
// via envelopeWire; if that is too large, it progressively drops working-set
// files (sorted by net-delta ascending, so the most-changed survive longest)
// until it fits.
func (e *ContextEnvelope) marshalJSON(maxSize int) ([]byte, error) {
	if maxSize <= 0 {
		maxSize = defaultEnvelopeMaxSize
	}
	files := append([]WorkingFile(nil), e.WorkingSet.Files...)
	full := envelopeWire{
		Project:       e.Project,
		WorkingSet:    EnvelopeWS{Files: files},
		Intent:        e.Intent,
		MatchedSkills: e.MatchedSkills,
		HandoffRefs:   e.HandoffRefs,
	}
	raw, err := json.Marshal(&full)
	if err != nil {
		return nil, err
	}
	if len(raw) <= maxSize {
		return raw, nil
	}

	// Oversized: drop working-set files in ascending net-delta order (the
	// smallest-delta files go first; the most-changed survive longest).
	ordered := append([]WorkingFile(nil), files...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].NetLines < ordered[j].NetLines })
	for drop := 1; drop <= len(ordered); drop++ {
		full.WorkingSet.Files = ordered[drop:]
		raw, err = json.Marshal(&full)
		if err != nil {
			return nil, err
		}
		if len(raw) <= maxSize {
			return raw, nil
		}
	}
	// All working-set files dropped; return the minimal envelope (working set
	// empty) — even if project metadata alone still exceeds the limit, the
	// working set cannot be shrunk further.
	full.WorkingSet.Files = []WorkingFile{}
	return json.Marshal(&full)
}
