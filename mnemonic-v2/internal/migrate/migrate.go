// Package migrate is the v1→v2 seed migration (TICKET-19, one-way door D4):
// a one-time EXPORT of the v1 store into the v2 store, not an in-place
// migration. It opens the v1 .sqlite read-only (the v1 files must come back
// byte-identical), maps v1's sessions + observations onto SaveRequests, and
// writes them into v2 through the memory service's Save pipeline (the seed
// is a batch of saves — redaction, dedup, importance, and the sync journal
// all apply; INV-4, no special import path).
//
// The package lives outside internal/memory on purpose (brief Task 19): it
// depends on memory.Service, and keeping it in its own package keeps the
// cutover from coupling to the domain (the memory package's test fixtures
// would otherwise import the migrator's v1 fixtures).
//
// Idempotency: every save carries OperationID "v1migrate:<v1 id>" (Task 6's
// replay ledger), so a re-run of Migrate over the same v2 store is a
// per-row no-op — the ledger replays each source row instead of
// re-inserting it.
package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/config"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/memory"
)

// WriterV1Migrate is the provenance.writer stamped on every seeded row
// (brief: the v1 store has no provenance column; the seed sets
// provenance.writer = "v1-migrate", model = ""). Migrate sets it on every
// SaveRequest; the save pipeline carries it through GateInsert.Writer to
// the row's INV-1 provenance record (the seed's rows are the only rows
// with a non-empty writer — the normal save surfaces leave it "").
const WriterV1Migrate = "v1-migrate"

// OpIDPrefix is the replay-ledger operation-id prefix for seeded rows
// (brief: OperationID = "v1migrate:" + v1id — a re-run is idempotent per
// source row, Task 6's ledger).
const OpIDPrefix = "v1migrate:"

// MigrationState is the doctor's per-project migration state (brief doctor
// completeness: a v1 dir detected but not migrated → a migrate-pending
// advisory line).
const (
	// StatePending: the v1 store has rows but no v1migrate: ledger rows —
	// the seed has not run (or ran partially).
	StatePending = "migrate-pending"
	// StateDone: at least one v1migrate: ledger row exists — the seed ran.
	StateDone = "migrate-done"
)

// MigrateReport is the migration's result (brief: {exported, skipped,
// errors []}). Exported counts the v1 live observations that landed in v2
// on this run; Skipped counts the rows the replay ledger replayed (a re-run
// skips everything); Errors carries one entry per row that failed to save
// (the run continues — a bad row must not wedge the rest of the seed).
// Session seeding (a v2 row per v1 session) is not counted — the report
// is over the v1 observations, the brief's seed unit.
type MigrateReport struct {
	Exported int
	Skipped  int
	Errors   []string
}

// MigrateSessionOpID is the replay-ledger operation id for a session's
// seed save (brief: sessions first, so observation FKs resolve; the same
// per-source-row idempotency as the observations, keyed by the v1 session
// id).
func MigrateSessionOpID(sessionID string) string { return OpIDPrefix + "session:" + sessionID }

// v1TypeToV2 maps the v1 type vocabulary onto the v2 vocabulary (spec
// 002-schema §2.2.1). The v2 vocabulary covers the v1 core types
// one-to-one; v1's extra types (learning, …) fall to the default arm,
// which is "discovery" (the v2 type for findings that predate the v2
// vocabulary). A v1 row whose type is outside both vocabularies still
// falls to discovery: a seed must not drop data over a vocabulary miss.
var v1TypeToV2 = map[string]string{
	"decision":     "decision",
	"architecture": "architecture",
	"bugfix":       "bugfix",
	"pattern":      "pattern",
	"config":       "config",
	"discovery":    "discovery",
	"preference":   "preference",
	"convention":   "convention",
}

// v1TypeDefault is the mapping's default arm (v1-only / unmapped types).
const v1TypeDefault = "discovery"

// Migrate seeds the v2 store for project from the v1 store at
// <v1Dir>/<project>.sqlite (brief). It opens the v1 file read-only
// (file:<path>?mode=ro — the v1 dir must be byte-identical afterward),
// seeds the v1 sessions (so the observation FKs resolve), then seeds the
// v1 live observations in id order through Service.Save. Returns the
// report; an open failure (no v1 store) is an error, a per-row save
// failure is a report entry.
func Migrate(ctx context.Context, v1Dir, v2DataDir, project string) (MigrateReport, error) {
	v1Path := filepath.Join(v1Dir, project+".sqlite")
	if fi, err := os.Stat(v1Path); err != nil {
		return MigrateReport{}, fmt.Errorf("migrate: v1 store %s: %w", v1Path, err)
	} else if fi.IsDir() {
		return MigrateReport{}, fmt.Errorf("migrate: v1 store %s is a directory", v1Path)
	}
	v1, err := openV1ReadOnly(v1Path)
	if err != nil {
		return MigrateReport{}, fmt.Errorf("migrate: open v1 store: %w", err)
	}
	defer v1.Close()

	st, err := openV2Store(ctx, v2DataDir, project)
	if err != nil {
		return MigrateReport{}, err
	}
	defer st.Close()
	svc := memory.NewService(st, v2Config(v2DataDir), memory.NewEventBus(), nil, nil)

	rep := MigrateReport{}
	// Sessions first (so observation FKs resolve): the v1 session rows are
	// auto-registered by the save pipeline's CheckSession, so the seed
	// walks the v1 sessions to guarantee their v2 rows exist before any
	// observation save. A session-seed save is one per v1 session (id
	// order); its replay-safe operation id makes a re-run a no-op. The
	// content is a fixed marker row (type session_summary — the v2
	// vocabulary's session type) whose title carries the v1 session id.
	var sessions []string
	rows, err := v1.Query(`SELECT id FROM sessions ORDER BY id`)
	if err != nil {
		return MigrateReport{}, fmt.Errorf("migrate: read v1 sessions: %w", err)
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return MigrateReport{}, fmt.Errorf("migrate: scan v1 session: %w", err)
		}
		sessions = append(sessions, id)
	}
	if err := rows.Err(); err != nil {
		return MigrateReport{}, fmt.Errorf("migrate: read v1 sessions: %w", err)
	}
	rows.Close()
	for _, id := range sessions {
		if _, err := svc.Save(ctx, memory.SaveRequest{
			SessionID:   id,
			Type:        "session_summary",
			Title:       "v1 session " + id,
			Content:     "v1 session row carried over by the v1→v2 seed migration",
			Project:     project,
			Scope:       "project",
			OperationID: MigrateSessionOpID(id),
			Writer:      WriterV1Migrate,
		}); err != nil {
			return MigrateReport{}, fmt.Errorf("migrate: session %s: %w", id, err)
		}
	}

	// Observations in v1 id order (the brief: sessions first, then
	// observations in id order). Deleted v1 rows (deleted_at set) are not
	// seeded — the v2 store starts from the v1 live set.
	obs, err := v1.Query(`
		SELECT id, session_id, type, title, content,
		       ifnull(project, ''), ifnull(scope, ''), ifnull(topic_key, '')
		FROM observations
		WHERE deleted_at IS NULL OR deleted_at = ''
		ORDER BY id`)
	if err != nil {
		return MigrateReport{}, fmt.Errorf("migrate: read v1 observations: %w", err)
	}
	defer obs.Close()
	for obs.Next() {
		var (
			id        int64
			sessionID string
			v1type    string
			title     string
			content   string
			proj      string
			scope     string
			topicKey  string
		)
		if err := obs.Scan(&id, &sessionID, &v1type, &title, &content, &proj, &scope, &topicKey); err != nil {
			return MigrateReport{}, fmt.Errorf("migrate: scan v1 observation %d: %w", id, err)
		}
		res, err := svc.Save(ctx, memory.SaveRequest{
			SessionID:   sessionID,
			Type:        v1TypeToV2Type(v1type),
			Title:       title,
			Content:     content,
			Project:     proj,
			Scope:       scope,
			TopicKey:    topicKey,
			OperationID: OpIDPrefix + fmt.Sprint(id),
			Writer:      WriterV1Migrate,
		})
		if err != nil {
			rep.Errors = append(rep.Errors, fmt.Sprintf("v1 observation %d: %v", id, err))
			continue
		}
		if res.Replayed {
			rep.Skipped++
			continue
		}
		rep.Exported++
	}
	if err := obs.Err(); err != nil {
		return MigrateReport{}, fmt.Errorf("migrate: read v1 observations: %w", err)
	}
	return rep, nil
}

// v1TypeToV2Type applies the type mapping (known v1 types one-to-one,
// everything else to the default arm).
func v1TypeToV2Type(v1type string) string {
	if t, ok := v1TypeToV2[v1type]; ok {
		return t
	}
	return v1TypeDefault
}

// SetMigrationStateHook registers the one-way wire (the store's
// SetApplyPulledMutationHook pattern — memory → migrate forbids the Service
// from importing this package, so the hook is registered from the wiring
// layer that imports both: the doctor's CLI command and the memory
// package's migrate_test (its _test import of migrate is a test-only
// cycle, legal in Go — the production wiring is the CLI, which imports
// both packages without a cycle). The hook is the default (a no-op error)
// until the wiring binds it to a store; the doctor's runtime overrides it
// with its own store binding before the check runs.
func SetMigrationStateHook(fn func(ctx context.Context, v1Dir, project string) (any, error)) {
	memory.MigrationStateHook = fn
}

// State is the doctor's per-project migration state (brief doctor
// completeness: the doctor reports the migration state — a v1 dir detected
// but not migrated → a migrate-pending advisory line).
type State struct {
	// V1Present is true when <v1Dir>/<project>.sqlite exists (the v1 store
	// the cutover would seed from).
	V1Present bool
	// V1Rows is the v1 store's live-observation count (0 when absent).
	V1Rows int
	// SeededRows is the v2 store's v1migrate: ledger-row count (0 when the
	// seed has not run).
	SeededRows int
	// MigrationState is StatePending / StateDone / "" (no v1 store to
	// migrate — the doctor's check stays advisory-ok).
	MigrationState string
}

// Report reports the migration state for project: whether the v1 store is
// present, how many live rows it carries, and whether the seed has run
// (the v2 replay ledger's v1migrate: rows). It opens the v1 file
// read-only (the doctor must not touch the v1 files either) and reads the
// v2 ledger through the store's Probe test seam (the pipeline's Probe
// pattern — the narrow cross-package read surface).
func Report(ctx context.Context, v1Dir, v2DataDir, project string, st store.Store) (State, error) {
	out := State{}
	v1Path := filepath.Join(v1Dir, project+".sqlite")
	fi, err := os.Stat(v1Path)
	if err != nil || fi.IsDir() {
		return out, nil // no v1 store: nothing to migrate (advisory-ok)
	}
	out.V1Present = true
	v1, err := openV1ReadOnly(v1Path)
	if err != nil {
		return out, fmt.Errorf("migrate: state open v1 store: %w", err)
	}
	defer v1.Close()
	if err := v1.QueryRow(`
		SELECT COUNT(*) FROM observations
		WHERE deleted_at IS NULL OR deleted_at = ''`).Scan(&out.V1Rows); err != nil {
		return out, fmt.Errorf("migrate: state read v1 observations: %w", err)
	}
	p, ok := st.(interface {
		ProbeInt(query string, args ...any) (int, error)
	})
	if !ok {
		return out, fmt.Errorf("migrate: store %T does not expose the ProbeInt seam the migration state reads through", st)
	}
	n, err := p.ProbeInt(`SELECT COUNT(*) FROM observation_save_operations WHERE operation_id LIKE ?`, OpIDPrefix+"%")
	if err != nil {
		return out, fmt.Errorf("migrate: state read v2 ledger: %w", err)
	}
	out.SeededRows = n
	if n > 0 {
		out.MigrationState = StateDone
	} else {
		out.MigrationState = StatePending
	}
	return out, nil
}

// openV1ReadOnly opens the v1 file as read-only (file:<path>?mode=ro — the
// same pure-Go driver, no v1 dependency: only the table layout is read).
// A read-only open may not rewrite pages or append to a WAL file, so the
// v1 dir is byte-identical after the migrate (the v1-files-untouched
// contract). The busy_timeout pragma absorbs a concurrent v1 writer.
func openV1ReadOnly(path string) (*sql.DB, error) {
	q := url.Values{}
	q.Set("mode", "ro")
	q.Add("_pragma", "busy_timeout(5000)")
	dsn := (&url.URL{Scheme: "file", Path: path}).String() + "?" + q.Encode()
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// v2Config builds the v2 store config the seed's service runs under:
// default config with the caller's data dir, project detection off (the
// seed names the project explicitly — no git shell-out at cutover). The
// intelligence seams are absent (nil in NewService) — the seed's dedup
// runs the deterministic hash floor, the right posture for a one-time
// import (no LLM call per row).
func v2Config(v2DataDir string) config.Config {
	c := config.Default()
	c.DataDir = v2DataDir
	c.Project.Detect = false
	return c
}

// openV2Store opens (or creates) the v2 store for project under
// v2DataDir through the store's normal Open (the seed writes through the
// normal pipeline — INV-4).
func openV2Store(ctx context.Context, v2DataDir, project string) (store.Store, error) {
	return store.Open(ctx, v2DataDir, project, store.SQLite, "")
}
