// Package process traces precomputed execution flows from 010's entry points
// (route/handler/CLI-main symbols) through the 005 call edges into named
// processes/process_steps, with a cross-community flag (from 01's
// communities) and LLM labels cached by content-hash. The trace is advisory,
// never load-bearing: it reads the graph but adds no symbols/edges of its own,
// so 005 search is unchanged. The flow is deterministic (depth-capped BFS,
// stable ordering); only the LLM label is non-deterministic, so it is cached
// by the process's content-hash.
package process

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// LLM is the pluggable labeler. A nil LLM (or one that errors) leaves the
// flow cached UNLABELED (empty label, label_status="unlabeled") — never a
// fabricated label. Tests inject a stub to exercise caching without a live
// LLM. There is no CGo LLM client in this package.
type LLM interface {
	Label(ctx context.Context, summary string) (string, error)
}

// Entry is one seed for the process pass: a symbol id plus the kind of entry
// point it represents (route | handler | cli-main).
type Entry struct {
	SymbolID int64
	Kind     string
}

// Step is one hop in a traced flow: the symbol reached, the edge kind and
// confidence used to get there (empty on the entry step), and the line.
type Step struct {
	SymbolID   int64
	Name       string
	Kind       string
	Confidence string
	Line       int
}

// StopNote is a "stops at <symbol> (<reason>)" annotation the trace emits
// when it reaches a dispatch boundary. It reuses the 005 graph-stops idea:
// the flow is truncated at the boundary and named, not silently cut.
type StopNote struct {
	Symbol string `json:"symbol"`
	Reason string `json:"reason"`
	Line   int    `json:"line"`
}

// Process is one traced flow: its entry, the ordered steps, the cross-
// community flag, the stop note (when it truncated at a boundary), the
// content-hash key, and the LLM label + label status.
type Process struct {
	Name           string    `json:"name"`
	EntrySymbolID  int64     `json:"entry_symbol_id"`
	EntryKind      string    `json:"entry_kind"`
	CrossCommunity bool      `json:"cross_community"`
	ContentHash    string    `json:"content_hash"`
	Steps          []Step    `json:"steps"`
	Stop           *StopNote `json:"stop,omitempty"`
	Label          string    `json:"label,omitempty"`
	LabelStatus    string    `json:"label_status"`
	updatedAt      time.Time
}

// RunOptions tunes the process pass.
type RunOptions struct {
	// MaxDepth caps the BFS depth from the entry (0 = 5). A cap prevents a
	// deep call graph from exploding into one giant flow.
	MaxDepth int
	// LLMLimit caps the number of LLM label calls in a single pass (0 = 20).
	// Excess flows stay cached unlabeled (never fabricated).
	LLMLimit int
}

// Run traces every entry point into a process and persists processes +
// process_steps (plus the content-hash cache key). It consults the
// process_meta_cache first: a matching content-hash returns the stored
// partition (FromCache=true) without re-tracing or re-calling the LLM.
func Run(ctx context.Context, db *sql.DB, entries []Entry, llm LLM, opts RunOptions) (*Result, error) {
	if opts.MaxDepth <= 0 {
		opts.MaxDepth = 5
	}
	if opts.LLMLimit <= 0 {
		opts.LLMLimit = 20
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	key, err := contentKey(db)
	if err != nil {
		return nil, fmt.Errorf("content key: %w", err)
	}
	cacheKey := "proc:" + key

	if cached, err := cachedProcesses(db, cacheKey); err == nil && cached != nil {
		cached.FromCache = true
		return cached, nil
	}

	res, err := runCore(ctx, db, entries, llm, opts, cacheKey)
	if err != nil {
		return nil, err
	}
	if err := writeMeta(db, cacheKey); err != nil {
		return res, fmt.Errorf("process pass ok but cache write failed: %w", err)
	}
	return res, nil
}

// Result is the Run output: the traced processes and whether the pass was
// served from cache.
type Result struct {
	Processes []Process `json:"processes"`
	FromCache bool      `json:"from_cache"`
	CacheKey  string    `json:"cache_key"`
	Updated   time.Time `json:"updated_at"`
}

// runCore traces each entry, computes the cross-community flag, applies the
// depth/step cap + dispatch-boundary truncation, and persists the flows.
func runCore(ctx context.Context, db *sql.DB, entries []Entry, llm LLM, opts RunOptions, cacheKey string) (*Result, error) {
	res := &Result{CacheKey: cacheKey, Updated: time.Now().UTC()}
	comm, err := communityOf(db)
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		p, err := traceOne(db, e, opts.MaxDepth)
		if err != nil {
			return nil, err
		}
		// Cross-community flag from 01's communities: the flow touches more
		// than one community id.
		p.updatedAt = res.Updated
		commIDs := map[int]bool{}
		for _, s := range p.Steps {
			if c, ok := comm[s.SymbolID]; ok {
				commIDs[c] = true
			}
		}
		p.CrossCommunity = len(commIDs) > 1
		// Persist the flow so code_processes serves it without traversal.
		id, err := persistProcess(ctx, db, p)
		if err != nil {
			return nil, err
		}
		_ = id
		res.Processes = append(res.Processes, *p)
	}

	// LLM-label the flows (best-effort; a nil/error LLM leaves them
	// unlabeled, never fabricated).
	if llm != nil {
		labelFlows(ctx, db, res.Processes, llm, opts.LLMLimit)
	}
	return res, nil
}

// traceOne performs the depth-capped BFS over the 005 call edges from the
// entry symbol, stopping (with a note) at the first dispatch boundary.
// Deterministic: neighbors are visited in (symbol id) order, a symbol is
// visited once, and the depth is capped.
func traceOne(db *sql.DB, entry Entry, maxDepth int) (*Process, error) {
	name, err := symbolName(db, entry.SymbolID)
	if err != nil {
		return nil, err
	}
	p := &Process{
		EntrySymbolID: entry.SymbolID,
		EntryKind:     entry.Kind,
		Name:          processName(entry, name),
		Steps:         []Step{{SymbolID: entry.SymbolID, Name: name}},
		LabelStatus:   "unlabeled",
	}

	queue := []int64{entry.SymbolID}
	depth := map[int64]int{entry.SymbolID: 0}
	visited := map[int64]bool{entry.SymbolID: true}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if depth[cur] >= maxDepth {
			continue
		}
		neighbors, err := callNeighbors(db, cur)
		if err != nil {
			return nil, err
		}
		// Stable ordering by symbol id.
		sort.Slice(neighbors, func(a, b int) bool { return neighbors[a].symbolID < neighbors[b].symbolID })

		stop := false
		for _, n := range neighbors {
			if visited[n.symbolID] {
				continue
			}
			// Dispatch-boundary detection: the target symbol is a dispatch
			// kind (interface / message-bus / callback) — the static trace
			// cannot know which implementation runs, so it stops here with a
			// note (reusing the 005 graph-stops idea), not silently cut. The
			// boundary symbol IS recorded as the last step (reached) so the
			// trace is not silently cut.
			if isDispatchKind(n.kind) {
				visited[n.symbolID] = true
				p.Steps = append(p.Steps, Step{
					SymbolID:   n.symbolID,
					Name:       n.name,
					Kind:       n.edgeKind,
					Confidence: n.confidence,
					Line:       n.line,
				})
				p.Stop = &StopNote{
					Symbol: n.name,
					Reason: dispatchReason(n.kind),
					Line:   n.line,
				}
				stop = true
				break
			}
			if n.symbolID == 0 {
				// Unresolved name-only callee: the graph stops here too.
				p.Stop = &StopNote{Symbol: n.name, Reason: "unresolved callee", Line: n.line}
				stop = true
				break
			}
			visited[n.symbolID] = true
			depth[n.symbolID] = depth[cur] + 1
			queue = append(queue, n.symbolID)
			p.Steps = append(p.Steps, Step{
				SymbolID:   n.symbolID,
				Name:       n.name,
				Kind:       n.edgeKind,
				Confidence: n.confidence,
				Line:       n.line,
			})
		}
		if stop {
			break
		}
	}
	ch, err := contentHash(p)
	if err != nil {
		return nil, err
	}
	p.ContentHash = ch
	return p, nil
}

// callNeighbor is one resolved callee of a symbol (or its unresolved name).
type callNeighbor struct {
	symbolID   int64
	name       string
	kind       string // the callee symbol's kind (for dispatch detection)
	edgeKind   string // the edge kind (calls/imports/references)
	confidence string
	line       int
}

// callNeighbors returns the resolved callees of symbolID over the call-edge
// kinds, each with the callee's kind + name (for dispatch-boundary detection).
// Deterministic by edge id.
func callNeighbors(db *sql.DB, symbolID int64) ([]callNeighbor, error) {
	rows, err := db.Query(`
		SELECT e.to_id, e.kind, e.confidence, e.line, COALESCE(s.name,''), COALESCE(s.kind,'')
		FROM edges e
		LEFT JOIN symbols s ON s.id = e.to_id
		WHERE e.from_id = ? AND e.kind IN ('calls','imports','references')
		ORDER BY e.id`, symbolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []callNeighbor
	for rows.Next() {
		var toID sql.NullInt64
		var kind, conf, name, symKind string
		var line int
		if err := rows.Scan(&toID, &kind, &conf, &line, &name, &symKind); err != nil {
			return nil, err
		}
		if toID.Valid {
			out = append(out, callNeighbor{
				symbolID: toID.Int64, name: name, kind: symKind,
				edgeKind: kind, confidence: conf, line: line,
			})
		} else {
			// Name-only (unresolved) callee — reported by the trace as a stop.
			var toName string
			_ = db.QueryRow(`SELECT to_name FROM edges WHERE from_id = ? AND to_id IS NULL AND kind IN ('calls','imports','references') AND line = ? LIMIT 1`, symbolID, line).Scan(&toName)
			out = append(out, callNeighbor{symbolID: 0, name: toName, edgeKind: kind, confidence: conf, line: line})
		}
	}
	return out, rows.Err()
}

// dispatchKinds are symbol kinds that mark a dispatch boundary the static
// trace cannot cross deterministically: an interface (which implementation
// runs is decided at runtime), a message bus, or a callback.
var dispatchKinds = map[string]bool{
	"interface": true,
	"bus":       true,
	"callback":  true,
}

func isDispatchKind(kind string) bool { return dispatchKinds[kind] }

func dispatchReason(kind string) string {
	switch kind {
	case "interface":
		return "interface dispatch (implementation selected at runtime)"
	case "bus":
		return "message bus dispatch (handler resolved at runtime)"
	case "callback":
		return "callback (invoked dynamically)"
	default:
		return "dispatch boundary"
	}
}

// processName derives a stable, human-readable process name from the entry
// (the entry symbol's name, or its route pattern for route entries).
func processName(entry Entry, symbolName string) string {
	if entry.Kind == "route" {
		if pattern := routePattern(symbolName); pattern != "" {
			return pattern
		}
	}
	return symbolName
}

// routePattern extracts a readable route pattern from a route symbol name
// (e.g. "GET /users" -> "/users"). Returns "" when there is no pattern.
func routePattern(name string) string {
	fields := strings.Fields(name)
	for _, f := range fields {
		if strings.HasPrefix(f, "/") {
			return f
		}
	}
	return ""
}

// contentHash is the per-process content key: a deterministic digest over the
// flow structure (entry kind + the ordered step symbol ids + each step's
// edge kind/confidence + the stop note). An unchanged re-index produces the
// same hash; a changed flow produces a new one (so it re-labels).
func contentHash(p *Process) (string, error) {
	var b strings.Builder
	b.WriteString(p.EntryKind)
	b.WriteByte(':')
	b.WriteString(strconv.FormatInt(p.EntrySymbolID, 10))
	b.WriteByte(':')
	for _, s := range p.Steps {
		b.WriteString(strconv.FormatInt(s.SymbolID, 10))
		b.WriteByte('|')
		b.WriteString(s.Kind)
		b.WriteByte('|')
		b.WriteString(s.Confidence)
		b.WriteByte(';')
	}
	if p.Stop != nil {
		b.WriteString("@")
		b.WriteString(p.Stop.Symbol)
		b.WriteByte(':')
		b.WriteString(p.Stop.Reason)
	}
	return fnvHex(b.String()), nil
}

// contentKey computes the process-pass content hash over the deterministic
// graph row set (the call/references/imports edge set), mirroring 01's
// community contentKey so an unchanged re-index is a cache hit. The hash is
// computed in Go (modernc.org/sqlite has no sha1() SQL function).
func contentKey(db *sql.DB) (string, error) {
	h := fnvNew()
	rows, err := db.Query(`SELECT from_id, to_id, kind, confidence FROM edges WHERE kind IN ('calls','imports','references') AND to_id IS NOT NULL ORDER BY from_id, to_id, kind`)
	if err != nil {
		return "", err
	}
	for rows.Next() {
		var from, to int64
		var kind, conf string
		if err := rows.Scan(&from, &to, &kind, &conf); err != nil {
			rows.Close()
			return "", err
		}
		_, _ = h.Write([]byte(strconv.FormatInt(from, 10) + ":" + strconv.FormatInt(to, 10) + ":" + kind + ":" + conf + ";"))
	}
	serr := rows.Err()
	rows.Close()
	if serr != nil {
		return "", serr
	}
	// NUL separator so a trailing-edge ambiguity can't collide with the symbol set.
	_, _ = h.Write([]byte{0})
	symRows, err := db.Query(`SELECT id FROM symbols ORDER BY id`)
	if err != nil {
		return "", err
	}
	for symRows.Next() {
		var id int64
		if err := symRows.Scan(&id); err != nil {
			symRows.Close()
			return "", err
		}
		_, _ = h.Write([]byte(strconv.FormatInt(id, 10) + ","))
	}
	symErr := symRows.Err()
	symRows.Close()
	if symErr != nil {
		return "", symErr
	}
	return fnvHexSum(h), nil
}

// communityOf loads the 01 communities table as a symbol id -> community id
// map. An empty map (no communities yet) means no cross-community flag — the
// pass is advisory and still runs.
func communityOf(db *sql.DB) (map[int64]int, error) {
	out := map[int64]int{}
	rows, err := db.Query(`SELECT id, symbol_id FROM communities`)
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			return out, nil
		}
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var sid int64
		if err := rows.Scan(&cid, &sid); err != nil {
			return nil, err
		}
		out[sid] = cid
	}
	return out, rows.Err()
}

// persistProcess writes one process + its steps (idempotent: replaces any
// prior row with the same content hash). Returns the process id.
func persistProcess(ctx context.Context, db *sql.DB, p *Process) (int64, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var id int64
	err = tx.QueryRowContext(ctx, `
		SELECT id FROM processes WHERE content_hash = ?`, p.ContentHash).Scan(&id)
	if err == sql.ErrNoRows {
		var res sql.Result
		res, err = tx.ExecContext(ctx, `
			INSERT INTO processes (name, entry_symbol_id, entry_kind, cross_community, content_hash, label, label_status, stop_note, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			p.Name, p.EntrySymbolID, p.EntryKind, boolTo(p.CrossCommunity), p.ContentHash,
			p.Label, p.LabelStatus, stopNoteText(p.Stop), p.UpdatedAtOrNow())
		if err != nil {
			return 0, err
		}
		id, err = res.LastInsertId()
		if err != nil {
			return 0, err
		}
	} else if err != nil {
		return 0, err
	}
	if _, err := tx.Exec(`DELETE FROM process_steps WHERE process_id = ?`, id); err != nil {
		return 0, err
	}
	for i, s := range p.Steps {
		if _, err := tx.Exec(`INSERT INTO process_steps (process_id, step, symbol_id, confidence, kind) VALUES (?, ?, ?, ?, ?)`,
			id, i+1, s.SymbolID, s.Confidence, s.Kind); err != nil {
			return 0, err
		}
	}
	return id, tx.Commit()
}

// cachedProcesses rebuilds the stored processes from the tables when the
// content-hash cache key matches. Returns nil (no error) on a miss.
func cachedProcesses(db *sql.DB, cacheKey string) (*Result, error) {
	var stored string
	err := db.QueryRow(`SELECT value FROM process_meta_cache WHERE key = 'cache_key'`).Scan(&stored)
	if err != nil || stored != cacheKey {
		return nil, nil
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM processes`).Scan(&count); err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, nil
	}
	res := &Result{CacheKey: cacheKey, Updated: time.Now().UTC()}
	rows, err := db.Query(`SELECT id, name, entry_symbol_id, entry_kind, cross_community, content_hash, label, label_status, stop_note FROM processes ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type row struct {
		id        int64
		name      string
		entryID   int64
		entryKind string
		cross     int
		hash      string
		label     string
		status    string
		stop      string
	}
	var rs []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.name, &r.entryID, &r.entryKind, &r.cross, &r.hash, &r.label, &r.status, &r.stop); err != nil {
			return nil, err
		}
		rs = append(rs, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, r := range rs {
		steps, err := stepsFor(db, r.id)
		if err != nil {
			return nil, err
		}
		p := Process{
			Name:           r.name,
			EntrySymbolID:  r.entryID,
			EntryKind:      r.entryKind,
			CrossCommunity: r.cross == 1,
			ContentHash:    r.hash,
			Steps:          steps,
			Label:          r.label,
			LabelStatus:    r.status,
		}
		if sn := parseStopNote(r.stop); sn != nil {
			p.Stop = sn
		}
		res.Processes = append(res.Processes, p)
	}
	return res, nil
}

// stepsFor rebuilds a process's ordered steps (with names) from the table.
func stepsFor(db *sql.DB, processID int64) ([]Step, error) {
	rows, err := db.Query(`
		SELECT ps.step, ps.symbol_id, ps.confidence, ps.kind, COALESCE(s.name,'')
		FROM process_steps ps LEFT JOIN symbols s ON s.id = ps.symbol_id
		WHERE ps.process_id = ? ORDER BY ps.step`, processID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Step
	for rows.Next() {
		var step, symbolID int
		var conf, kind, name string
		if err := rows.Scan(&step, &symbolID, &conf, &kind, &name); err != nil {
			return nil, err
		}
		out = append(out, Step{SymbolID: int64(symbolID), Name: name, Kind: kind, Confidence: conf})
	}
	return out, rows.Err()
}

// writeMeta caches the content-hash key so the next unchanged pass is a hit.
func writeMeta(db *sql.DB, cacheKey string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`INSERT INTO process_meta_cache (key, value) VALUES ('cache_key', ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, cacheKey); err != nil {
		return err
	}
	return tx.Commit()
}

// symbolName returns a symbol's name (or "" when unknown — the trace still
// records the step, never a crash).
func symbolName(db *sql.DB, id int64) (string, error) {
	var name string
	err := db.QueryRow(`SELECT name FROM symbols WHERE id = ?`, id).Scan(&name)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return name, err
}

func boolTo(b bool) int {
	if b {
		return 1
	}
	return 0
}

// stopNoteText serializes a stop note for storage; "" when absent.
func stopNoteText(s *StopNote) string {
	if s == nil {
		return ""
	}
	return s.Symbol + "\x00" + s.Reason + "\x00" + strconv.Itoa(s.Line)
}

// parseStopNote is the inverse of stopNoteText; nil when empty.
func parseStopNote(s string) *StopNote {
	if s == "" {
		return nil
	}
	parts := strings.SplitN(s, "\x00", 3)
	if len(parts) < 2 {
		return nil
	}
	line := 0
	if len(parts) == 3 {
		line, _ = strconv.Atoi(parts[2])
	}
	return &StopNote{Symbol: parts[0], Reason: parts[1], Line: line}
}

// UpdatedAtOrNow is a small helper so the persist row uses a stable time
// source (the pass's Updated). It is set by the caller via struct field on
// Process when persisting — see Process.updatedAt.
func (p *Process) UpdatedAtOrNow() string {
	if p.updatedAt.IsZero() {
		return time.Now().UTC().Format(time.RFC3339)
	}
	return p.updatedAt.Format(time.RFC3339)
}

// List returns the stored processes in stable order (by id) — the backing for
// code_processes (precomputed flows, one call, no per-query traversal).
func List(db *sql.DB) ([]Process, error) {
	rows, err := db.Query(`SELECT id, name, entry_symbol_id, entry_kind, cross_community, content_hash, label, label_status, stop_note FROM processes ORDER BY id`)
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			return []Process{}, nil
		}
		return nil, err
	}
	defer rows.Close()
	var out []Process
	for rows.Next() {
		var id int64
		var name string
		var entryID int64
		var entryKind string
		var cross int
		var hash, label, status, stop string
		if err := rows.Scan(&id, &name, &entryID, &entryKind, &cross, &hash, &label, &status, &stop); err != nil {
			return nil, err
		}
		p := Process{
			Name: name, EntrySymbolID: entryID, EntryKind: entryKind,
			CrossCommunity: cross == 1, ContentHash: hash,
			Label: label, LabelStatus: status,
		}
		if sn := parseStopNote(stop); sn != nil {
			p.Stop = sn
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Get returns one stored process by name with its full step-by-step trace
// (each hop's confidence + the stop note) — the backing for
// code_process <name>. A missing name returns sql.ErrNoRows.
func Get(db *sql.DB, name string) (*Process, error) {
	var id int64
	err := db.QueryRow(`SELECT id FROM processes WHERE name = ? ORDER BY id LIMIT 1`, name).Scan(&id)
	if err != nil {
		return nil, err
	}
	var nameCol, entryKind, hash, label, status, stop string
	var entryID int64
	var cross int
	if err := db.QueryRow(`SELECT name, entry_symbol_id, entry_kind, cross_community, content_hash, label, label_status, stop_note FROM processes WHERE id = ?`, id).
		Scan(&nameCol, &entryID, &entryKind, &cross, &hash, &label, &status, &stop); err != nil {
		return nil, err
	}
	p := &Process{
		Name: nameCol, EntrySymbolID: entryID, EntryKind: entryKind,
		CrossCommunity: cross == 1, ContentHash: hash,
		Label: label, LabelStatus: status,
	}
	if sn := parseStopNote(stop); sn != nil {
		p.Stop = sn
	}
	p.Steps, err = stepsFor(db, id)
	return p, err
}

// Participation is a symbol's position in one process (step N/M).
type Participation struct {
	ProcessName string `json:"process"`
	Step        int    `json:"step"`
	Total       int    `json:"total"`
}

// Participations returns every process the symbol participates in, with its
// step position (step N/M) — the additive field surfaced by 005's
// code_explain_symbol. An empty slice (not an error) when the symbol is in no
// process.
func Participations(db *sql.DB, symbolID int64) ([]Participation, error) {
	if _, err := db.Exec(`SELECT 1 FROM process_steps LIMIT 0`); err != nil {
		if strings.Contains(err.Error(), "no such table") {
			return []Participation{}, nil
		}
		return nil, err
	}
	rows, err := db.Query(`
		SELECT p.name, ps.step, MAX(ps2.step)
		FROM process_steps ps
		JOIN processes p ON p.id = ps.process_id
		LEFT JOIN process_steps ps2 ON ps2.process_id = ps.process_id
		WHERE ps.symbol_id = ?
		GROUP BY p.id, ps.step
		ORDER BY p.id, ps.step`, symbolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Participation
	for rows.Next() {
		var name string
		var step, total int
		if err := rows.Scan(&name, &step, &total); err != nil {
			return nil, err
		}
		out = append(out, Participation{ProcessName: name, Step: step, Total: total})
	}
	return out, rows.Err()
}
