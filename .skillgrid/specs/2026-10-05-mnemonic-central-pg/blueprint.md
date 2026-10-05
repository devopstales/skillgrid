# Mnemonic Central PG Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use skillgrid:subagent-execution (recommended) or skillgrid:simple-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Status:** PROPOSED

**Tier:** T3 (high-risk — new trust boundary: network + PG auth + shared data)

**Build shape:** Tracer thread (one observation syncs end-to-end before the rest of the entity set thickens)

**Goal:** Teams share one VM PostgreSQL as mnemonic's shared memory state while every machine keeps its local SQLite store offline-first.

**Architecture:** Every shared-entity local write appends to a SQLite outbox (`sync_mutations`); a lease-guarded `syncpg.Manager` goroutine pushes batches to and pulls batches from a per-project PG schema (`mem_<project>`) via pgx, applying last-write-wins + tombstones. Tokens are PG roles; grants are per-project schema usage; denials are audited. Disabled by default — zero behavior change until `mnemonic.store.central.enabled: true`.

**Tech Stack:** Go 1.22+, `github.com/jackc/pgx/v5` (stdlib `database/sql` path, **new dep — named in ADR-0024**), existing `modernc.org/sqlite` local store untouched, `github.com/google/uuid` (already in go.mod; use `uuid.NewV7()`), hand-rolled embedded-SQL PG migration runner (no migration library, per the "no new dependencies without an ADR" lock), BDD via cucumber-js (`acceptance-tests/`).

**Spec:** `.skillgrid/specs/2026-10-05-mnemonic-central-pg/briefing.md`

**Findings:** `/Users/paladm/.local/share/opencode/plans/mnemonic-central-pg.md` (2026-10-05 opencode plan-mode research: engram cloud architecture + mnemonic store architecture, both subagent-verified)

## Terms

- **Outbox** — local SQLite table `sync_mutations` that queues shared-entity mutations for push. (New in this change.)
- **Journal** — PG table `mem_<project>.mem_mutations`, the ordered mutation stream; server-assigned `seq`. (New.)
- **Materialized table** — PG `mem_<project>.mem_<entity>`, the LWW current-state read model. (New.)
- **Sync ID** — client-generated UUIDv7 stable identity of a synced entity row. (New.)
- **Enroll / unenroll** — client-side per-project opt-in/out to sync. (New.)
- **Token = PG role** — managed token `egm_<env>_<prefix>_<secret>` maps 1:1 to PG role `mem_<prefix>`. (New, per ADR-0024.)
- Existing terms (observation, prompt, fact, skill, web cache, session, edges, team) — see `.skillgrid/artifacts/01-business-terms.md` / `02-technical-terms.md`.

## Must-Haves (goal-backward verification)

**Truths** (observable behaviors that must hold):
- A `mem_save` on machine A is found by `mem_search` on machine B within one sync cycle (≤ poll interval). `backstop` — needs two live stores against a live PG; held-out test: Task 10's two-machine test.
- PG down ⇒ saves succeed locally, outbox grows, no data loss, catch-up on reconnect with no duplicate local rows. `backstop` — held-out test: Task 5 offline→online test.
- Concurrent same-`sync_id` updates converge to the later `occurred_at` on both machines; no errors. `backstop` — held-out test: Task 7 LWW race test.
- Delete propagates as a tombstone; re-save of the same `sync_id` revives it; prune never removes a re-saved `sync_id`.
- A token role granted only project A gets a PG-level denial writing to `mem_B`, and a `mem_auth_audit_log` row appears.
- Two managers on one machine never hold the lease simultaneously; a killed manager's lease is taken over within 60s.
- `enabled: false` ⇒ no PG dial, no manager, `go test ./...` green with no PG present.

**Artifacts** (files that must exist with real implementation, not stubs):
- `skillgrid-cli/internal/mnemonic/syncpg/manager.go` — lease, push, pull, backoff, reason codes.
- `skillgrid-cli/internal/mnemonic/syncpg/pgmigrate/` — embedded PG DDL + runner.
- `skillgrid-cli/internal/mnemonic/syncpg/journal.go` — outbox read/ack against the local store.
- `skillgrid-cli/internal/mnemonic/syncpg/apply.go` — per-entity LWW apply into local SQLite.
- `skillgrid-cli/internal/mnemonic/syncpg/tokens.go` — issue/revoke/verify against PG.
- `skillgrid-cli/internal/mnemonic/syncpg/config.go` — central config load + validation.
- `skillgrid-cli/internal/mnemonic/store/migrations/045_sync_outbox.sql` — local outbox + state tables.
- `skillgrid-cli/cmd/skillgrid/central.go` — `mnemonic central` CLI verbs.
- `skillgrid-cli/internal/mnemonic/outbox/outbox.go` — `Record` helper called by the write sites.
- `acceptance-tests/features/central-pg.feature` (mirror of `acceptance.feature`).

**Key links** (critical connections that must work together):
- `memory.Service.insertObservation` (memory/service.go:872) ⇒ `outbox.Record("observation", syncID, "upsert", payload)` — the tracer bullet end-to-end (Task 4 → Task 10).
- `syncpg.Manager.push` ⇒ PG `mem_mutations` insert ⇒ `journal.go` acks the local outbox only on full batch acceptance.
- `syncpg.Manager.pull` ⇒ `apply.go` upserts local SQLite ⇒ local FTS5 triggers keep search current (no FTS rewrite — this is why local stays SQLite).
- `tokens.go` issue ⇒ PG role + grant ⇒ `manager`'s DSN user is that role ⇒ `doctor` verifies the grant chain.

**One-way-door decisions** (hard to reverse — explicit user approval required before implementing):
1. ⚠ **New dependency `github.com/jackc/pgx/v5`** — permanent go.mod change; approved with ADR-0024 (user accepted the PG direction 2026-10-05).
2. ⚠ **Local migration 045 adds `sync_mutations`, `sync_state`, `sync_enrollments`** to every existing store on next open — additive (new tables only), no column changes to existing tables.
3. ⚠ **PG schema shape `mem_<project>` + journal table `mem_mutations`** is the team's durable contract — renaming or restructuring later requires a PG migration for every team.

## Global Constraints

- Go 1.22+ minimum to build.
- No new dependencies without an ADR — only `pgx/v5`, named in `.skillgrid/artifacts/04-adr-0024-central-pg-shared-memory-state.md`.
- Local store unchanged: per-project SQLite, WAL, pure-Go driver, embedded migrations (per ADR-0012). Migration 045 is additive tables only.
- Bi-temporal save path (per ADR-0011) remains the local classifier; sync transports rows, never re-classifies.
- Fail-open: sync never blocks or fails a local write (per ADR-0016 floors).
- All scopes (`project`, `user`, `global`) sync; scope is a tag, never a privacy boundary (per ADR-0024).
- Token file 0600; TLS mode `require` default; PG 14+ scram-sha-256.
- Trivy advisory-only.
- BDD always on: each task's SATISFIES names its scenario in `acceptance.feature`.

## File Structure

```
skillgrid-cli/
  internal/mnemonic/
    outbox/
      outbox.go            # Record(entity, key, op, payload) — called by the write sites
      outbox_test.go
    syncpg/
      config.go            # CentralConfig load/validate (file + env), DSN resolution
      config_test.go
      pgmigrate/
        pgmigrate.go       # embedded runner: versioned, idempotent, resumable
        migrations/
          001_admin.sql    # mem_admin schema: token_meta, auth_audit_log, project_meta
          002_project.sql  # per-project schema template (rendered with project name)
      journal.go           # outbox read (pending batches) + ack (by pushed seq range)
      journal_test.go
      apply.go             # ApplyPulledMutation(entity, payload) per-entity LWW upsert
      apply_test.go
      manager.go           # lease, loop (debounce+poll), push, pull, backoff, reason codes
      manager_test.go
      tokens.go            # issue/revoke: PG role + grant + token_meta rows
      tokens_test.go
      bootstrap.go         # enroll: import existing local history into the outbox
      bootstrap_test.go
      e2e_test.go          # two-machine end-to-end against a live test PG
      syncpg.go            # New(cfg) -> *SyncPG (wires pgx pool, migrator, journal, apply)
      payload.go           # per-entity payload (de)serialization: local row <-> JSONB
      payload_test.go
    store/
      migrations/
        045_sync_outbox.sql # sync_mutations, sync_state, sync_enrollments (additive)
      sync.go              # low-level: outbox Record/ReadPending/Ack, lease Acquire/Release,
                           #       enroll flags, last_pulled_seq — thin SQL over s.DB
      sync_test.go
    memory/service.go      # MODIFY: insertObservation + update arm call outbox.Record
    prompts/  facts/  skills/  webcache/  teams/
                           # MODIFY: each shared write site calls outbox.Record (one line each)
    service/service.go     # MODIFY: boot wires syncpg when central enabled
  cmd/skillgrid/
    central.go             # mnemonic central status|enroll|unenroll|issue-token|revoke-token|doctor
    central_test.go
  go.mod                   # MODIFY: + github.com/jackc/pgx/v5
acceptance-tests/features/central-pg.feature
```

## Tasks

### Task 1: Dependency, local outbox schema, and the `outbox` package

**Files:**
- Modify: `skillgrid-cli/go.mod` (add `github.com/jackc/pgx/v5`)
- Create: `skillgrid-cli/internal/mnemonic/store/migrations/045_sync_outbox.sql`
- Create: `skillgrid-cli/internal/mnemonic/store/sync.go`
- Create: `skillgrid-cli/internal/mnemonic/store/sync_test.go`
- Create: `skillgrid-cli/internal/mnemonic/outbox/outbox.go`
- Create: `skillgrid-cli/internal/mnemonic/outbox/outbox_test.go`

**Interfaces:**
- Consumes: `*store.Store` (existing; `store.Open` at store/store.go:104), `context.Context`.
- Produces:
  - `func (s *Store) RecordSync(ctx context.Context, entity, key, op, payload string) error` — appends one `sync_mutations` row.
  - `func (s *Store) ReadPendingSync(ctx context.Context, project string, limit int) ([]SyncRow, error)` — oldest-first pending rows for one project.
  - `func (s *Store) AckSync(ctx context.Context, seqs []int64) error` — marks pushed rows `disposition=pushed`.
  - `type SyncRow struct{ Seq int64; Project, Entity, Key, Op, Payload, CreatedAt string }`
  - `func NewRecorder(s *store.Store, enabled func() bool) *Recorder` and `(*Recorder).Record(entity, key, op, payload string)` — write sites use this (nil-safe: nil recorder or disabled = no-op).
- SATISFIES: `happy path observation save records outbox row`, `happy path each shared write site records its outbox row`, `error path outbox failure never fails the local save`

- [ ] **Step 1: Write the failing test** — `store/sync_test.go`:

```go
func TestRecordSyncAppendsPendingRow(t *testing.T) {
	s := testStore(t) // existing helper: opens temp store + migrations
	if err := s.RecordSync(ctx, "observation", "obs-1", "upsert", `{"a":1}`); err != nil {
		t.Fatalf("RecordSync: %v", err)
	}
	rows, err := s.ReadPendingSync(ctx, "testproj", 10)
	if err != nil {
		t.Fatalf("ReadPendingSync: %v", err)
	}
	if len(rows) != 1 || rows[0].Entity != "observation" || rows[0].Key != "obs-1" || rows[0].Op != "upsert" {
		t.Fatalf("unexpected rows: %+v", rows)
	}
	if err := s.AckSync(ctx, []int64{rows[0].Seq}); err != nil {
		t.Fatalf("AckSync: %v", err)
	}
	rows, _ = s.ReadPendingSync(ctx, "testproj", 10)
	if len(rows) != 0 {
		t.Fatalf("expected empty after ack, got %+v", rows)
	}
}

func TestReadPendingSyncFiltersByProject(t *testing.T) {
	s := testStore(t)
	_ = s.RecordSync(ctx, "observation", "a", "upsert", "{}")
	if _, err := s.DB.Exec(`INSERT INTO sync_mutations (project, entity, key, op, payload) VALUES ('otherproj','fact','b','upsert','{}')`); err != nil {
		t.Fatal(err)
	}
	rows, _ := s.ReadPendingSync(ctx, "testproj", 10)
	if len(rows) != 1 || rows[0].Key != "a" {
		t.Fatalf("project filter failed: %+v", rows)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd skillgrid-cli && go test ./internal/mnemonic/store/ -run 'TestRecordSync|TestReadPending' -v`
Expected: FAIL — `s.RecordSync undefined`.

- [ ] **Step 3: Write the migration + implementation**

`store/migrations/045_sync_outbox.sql`:

```sql
-- 045_sync_outbox: local outbox + sync state (additive; central PG feature, ADR-0024)
CREATE TABLE IF NOT EXISTS sync_mutations (
	seq INTEGER PRIMARY KEY AUTOINCREMENT,
	project TEXT NOT NULL,
	entity TEXT NOT NULL,
	key TEXT NOT NULL,
	op TEXT NOT NULL CHECK (op IN ('upsert', 'delete')),
	payload TEXT NOT NULL,
	disposition TEXT NOT NULL DEFAULT 'pending' CHECK (disposition IN ('pending', 'pushed')),
	created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
CREATE INDEX IF NOT EXISTS idx_sync_mutations_pending
	ON sync_mutations (project, seq) WHERE disposition = 'pending';

CREATE TABLE IF NOT EXISTS sync_state (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sync_enrollments (
	project TEXT PRIMARY KEY,
	enrolled INTEGER NOT NULL DEFAULT 1,
	enrolled_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
```

`store/sync.go` (thin SQL; lease helpers added in Task 5):

```go
package store

type SyncRow struct {
	Seq       int64
	Project   string
	Entity    string
	Key       string
	Op        string
	Payload   string
	CreatedAt string
}

func (s *Store) RecordSync(ctx context.Context, entity, key, op, payload string) error {
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO sync_mutations (project, entity, key, op, payload) VALUES (?,?,?,?,?)`,
		s.projectID(), entity, key, op, payload)
	return err
}

func (s *Store) ReadPendingSync(ctx context.Context, project string, limit int) ([]SyncRow, error) {
	rows, err := s.DB.QueryContext(ctx,
		`SELECT seq, project, entity, key, op, payload, created_at
		   FROM sync_mutations WHERE project = ? AND disposition = 'pending'
		  ORDER BY seq LIMIT ?`, project, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SyncRow
	for rows.Next() {
		var r SyncRow
		if err := rows.Scan(&r.Seq, &r.Project, &r.Entity, &r.Key, &r.Op, &r.Payload, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) AckSync(ctx context.Context, seqs []int64) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	for _, seq := range seqs {
		if _, err := tx.ExecContext(ctx,
			`UPDATE sync_mutations SET disposition='pushed' WHERE seq=? AND disposition='pending'`, seq); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}
```

(Note: `s.projectID()` is the store's existing project-id accessor; if the field is unexported, add the accessor in this task — the field lives on `*Store` at store/store.go.)

`outbox/outbox.go`:

```go
package outbox

type Recorder struct {
	store   *store.Store
	enabled func() bool
}

func NewRecorder(s *store.Store, enabled func() bool) *Recorder {
	return &Recorder{store: s, enabled: enabled}
}

// Record appends one outbox row. Fail-open: a nil receiver or disabled flag is a
// no-op; errors are logged, never returned to the save path (ADR-0016 floors).
func (r *Recorder) Record(entity, key, op, payload string) {
	if r == nil || r.enabled == nil || !r.enabled() {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := r.store.RecordSync(ctx, entity, key, op, payload); err != nil {
		log.Printf("mnemonic: outbox record %s/%s: %v", entity, key, err)
	}
}
```

- [ ] **Step 4: Add `pgx/v5` to go.mod**

Run: `cd skillgrid-cli && go get github.com/jackc/pgx/v5`
Expected: go.mod + go.sum updated.

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd skillgrid-cli && go test ./internal/mnemonic/store/ ./internal/mnemonic/outbox/ -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add skillgrid-cli/go.mod skillgrid-cli/go.sum \
  skillgrid-cli/internal/mnemonic/store/migrations/045_sync_outbox.sql \
  skillgrid-cli/internal/mnemonic/store/sync.go \
  skillgrid-cli/internal/mnemonic/store/sync_test.go \
  skillgrid-cli/internal/mnemonic/outbox/
git commit -m "feat(mnemonic): local sync outbox tables and record helper"
```

---

### Task 2: PG migration runner + schema DDL

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/syncpg/pgmigrate/pgmigrate.go`
- Create: `skillgrid-cli/internal/mnemonic/syncpg/pgmigrate/migrations/001_admin.sql`
- Create: `skillgrid-cli/internal/mnemonic/syncpg/pgmigrate/migrations/002_project.sql`
- Create: `skillgrid-cli/internal/mnemonic/syncpg/pgmigrate/pgmigrate_test.go`

**Interfaces:**
- Consumes: `*pgxpool.Pool` (pgx/v5).
- Produces:
  - `func MigrateAdmin(ctx context.Context, pool *pgxpool.Pool) error` — creates `mem_admin` schema + `schema_migrations`, `token_meta`, `auth_audit_log`, `project_meta`.
  - `func MigrateProject(ctx context.Context, pool *pgxpool.Pool, project string) error` — creates `mem_<project>` schema + all shared tables + journal.
  - Validation: project names must match `^[a-z0-9_]{1,63}$` (PG identifier-safe; validated at enroll too — fail loudly here on bad input).
- SATISFIES: `happy path pg migrations apply idempotently`

- [ ] **Step 1: Write the failing test** — `pgmigrate_test.go` (test PG via `PGX_TEST_DSN` env, `t.Skip` if unset — the acceptance runner provides it):

```go
func TestMigrateAdminIdempotent(t *testing.T) {
	pool := testPool(t) // connects to a scratch DB via PGX_TEST_DSN
	if err := pgmigrate.MigrateAdmin(ctx, pool); err != nil {
		t.Fatalf("first migrate: %v", err)
	}
	if err := pgmigrate.MigrateAdmin(ctx, pool); err != nil {
		t.Fatalf("second migrate must be no-op: %v", err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM mem_admin.schema_migrations`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected 1 migration row, got %d", n)
	}
}

func TestMigrateProjectCreatesAllTables(t *testing.T) {
	pool := testPool(t)
	if err := pgmigrate.MigrateProject(ctx, pool, "demo"); err != nil {
		t.Fatal(err)
	}
	tables := querySet(ctx, t, pool,
		`SELECT table_name FROM information_schema.tables WHERE table_schema='mem_demo' ORDER BY 1`)
	for _, want := range []string{"mem_mutations", "mem_observations", "mem_observation_versions",
		"mem_prompts", "mem_facts", "mem_skills", "mem_web_cache", "mem_sessions",
		"mem_session_events", "mem_edges", "mem_teams", "mem_team_members"} {
		if !contains(tables, want) {
			t.Fatalf("missing table %s; have %v", want, tables)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd skillgrid-cli && PGX_TEST_DSN=postgres://mem_admin@localhost:5432/mnemonic_test go test ./internal/mnemonic/syncpg/pgmigrate/ -v`
Expected: FAIL — symbols do not exist.

- [ ] **Step 3: Write the DDL + runner**

`001_admin.sql` (executed as the admin role; `mem_admin` is the shared cross-project space):

```sql
CREATE SCHEMA IF NOT EXISTS mem_admin;

CREATE TABLE IF NOT EXISTS mem_admin.schema_migrations (
	id TEXT PRIMARY KEY,
	applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS mem_admin.token_meta (
	prefix TEXT PRIMARY KEY,
	env TEXT NOT NULL,
	secret_hash TEXT NOT NULL,          -- HMAC-SHA256(pepper || secret)
	role_name TEXT NOT NULL,            -- mem_<prefix>
	revoked_at TIMESTAMPTZ,
	last_used_at TIMESTAMPTZ,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS mem_admin.auth_audit_log (
	id BIGSERIAL PRIMARY KEY,
	actor TEXT NOT NULL,
	project TEXT NOT NULL,
	action TEXT NOT NULL,               -- push | pull | issue | revoke | grant
	outcome TEXT NOT NULL,              -- ok | denied | error
	reason_code TEXT,
	at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS mem_admin.project_meta (
	project TEXT PRIMARY KEY,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	created_by TEXT
);
```

`002_project.sql` (per-project template; `__PROJECT__` is replaced with the validated project identifier before exec):

```sql
CREATE SCHEMA IF NOT EXISTS mem___PROJECT__;

CREATE TABLE IF NOT EXISTS mem___PROJECT__.mem_mutations (
	seq BIGSERIAL,
	entity TEXT NOT NULL,
	entity_key UUID NOT NULL,
	op TEXT NOT NULL CHECK (op IN ('upsert','delete')),
	payload JSONB NOT NULL,
	payload_hash TEXT NOT NULL,
	occurred_at TIMESTAMPTZ NOT NULL,
	created_by TEXT NOT NULL,
	PRIMARY KEY (seq),
	UNIQUE (entity, entity_key, payload_hash)
);
CREATE INDEX IF NOT EXISTS idx___PROJECT__mutations_cursor
	ON mem___PROJECT__.mem_mutations (seq);

CREATE TABLE IF NOT EXISTS mem___PROJECT__.mem_observations (
	sync_id UUID PRIMARY KEY,
	created_by TEXT NOT NULL,
	occurred_at TIMESTAMPTZ NOT NULL,
	deleted_at TIMESTAMPTZ,
	payload JSONB NOT NULL
);
CREATE TABLE IF NOT EXISTS mem___PROJECT__.mem_observation_versions (
	sync_id UUID,
	version INT,
	payload JSONB,
	occurred_at TIMESTAMPTZ,
	created_by TEXT,
	PRIMARY KEY (sync_id, version)
);
CREATE TABLE IF NOT EXISTS mem___PROJECT__.mem_prompts (sync_id UUID PRIMARY KEY, created_by TEXT NOT NULL, occurred_at TIMESTAMPTZ NOT NULL, deleted_at TIMESTAMPTZ, payload JSONB NOT NULL);
CREATE TABLE IF NOT EXISTS mem___PROJECT__.mem_facts (sync_id UUID PRIMARY KEY, created_by TEXT NOT NULL, occurred_at TIMESTAMPTZ NOT NULL, deleted_at TIMESTAMPTZ, payload JSONB NOT NULL);
CREATE TABLE IF NOT EXISTS mem___PROJECT__.mem_skills (sync_id UUID PRIMARY KEY, created_by TEXT NOT NULL, occurred_at TIMESTAMPTZ NOT NULL, deleted_at TIMESTAMPTZ, payload JSONB NOT NULL);
CREATE TABLE IF NOT EXISTS mem___PROJECT__.mem_web_cache (sync_id UUID PRIMARY KEY, created_by TEXT NOT NULL, occurred_at TIMESTAMPTZ NOT NULL, deleted_at TIMESTAMPTZ, payload JSONB NOT NULL);
CREATE TABLE IF NOT EXISTS mem___PROJECT__.mem_sessions (sync_id UUID PRIMARY KEY, created_by TEXT NOT NULL, occurred_at TIMESTAMPTZ NOT NULL, deleted_at TIMESTAMPTZ, payload JSONB NOT NULL);
CREATE TABLE IF NOT EXISTS mem___PROJECT__.mem_session_events (sync_id UUID PRIMARY KEY, created_by TEXT NOT NULL, occurred_at TIMESTAMPTZ NOT NULL, deleted_at TIMESTAMPTZ, payload JSONB NOT NULL);
CREATE TABLE IF NOT EXISTS mem___PROJECT__.mem_edges (sync_id UUID PRIMARY KEY, created_by TEXT NOT NULL, occurred_at TIMESTAMPTZ NOT NULL, deleted_at TIMESTAMPTZ, payload JSONB NOT NULL);
CREATE TABLE IF NOT EXISTS mem___PROJECT__.mem_teams (sync_id UUID PRIMARY KEY, created_by TEXT NOT NULL, occurred_at TIMESTAMPTZ NOT NULL, deleted_at TIMESTAMPTZ, payload JSONB NOT NULL);
CREATE TABLE IF NOT EXISTS mem___PROJECT__.mem_team_members (sync_id UUID PRIMARY KEY, created_by TEXT NOT NULL, occurred_at TIMESTAMPTZ NOT NULL, deleted_at TIMESTAMPTZ, payload JSONB NOT NULL);
```

`pgmigrate.go` — embedded, versioned, resumable (one statement per `Exec`; no multi-statement batches — pgx `database/sql` path):

```go
package pgmigrate

import (
	"context"
	"embed"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var files embed.FS

var projectIDRE = regexp.MustCompile(`^[a-z0-9_]{1,63}$`)

func MigrateAdmin(ctx context.Context, pool *pgxpool.Pool) error {
	return migrateFile(ctx, pool, "001_admin.sql", nil)
}

func MigrateProject(ctx context.Context, pool *pgxpool.Pool, project string) error {
	if !projectIDRE.MatchString(project) {
		return fmt.Errorf("project %q is not a valid PG identifier", project)
	}
	return migrateFile(ctx, pool, "002_project.sql", &project)
}

func migrateFile(ctx context.Context, pool *pgxpool.Pool, file string, project *string) error {
	b, err := files.ReadFile("migrations/" + file)
	if err != nil {
		return err
	}
	var s string
	if project != nil {
		s = strings.ReplaceAll(string(b), "__PROJECT__", *project)
	} else {
		s = string(b)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for _, stmt := range splitStatements(s) {
		if _, err := tx.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("%s: %w", file, err)
		}
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO mem_admin.schema_migrations (id) VALUES ($1) ON CONFLICT DO NOTHING`,
		file); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// splitStatements splits on ';' at end of statement outside single quotes.
// (No dollar-quoting is used in our DDL; the scanner keeps quotes honest.)
func splitStatements(s string) []string { /* scan runes, track inQuote, cut on ';' */ }
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd skillgrid-cli && PGX_TEST_DSN=... go test ./internal/mnemonic/syncpg/pgmigrate/ -v`
Expected: PASS (idempotent + table-set).

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/syncpg/pgmigrate/
git commit -m "feat(mnemonic): embedded PG migration runner and team schema DDL"
```

---

### Task 3: Central config + `central.json` token file

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/syncpg/config.go`
- Create: `skillgrid-cli/internal/mnemonic/syncpg/config_test.go`
- Modify: `skillgrid-cli/internal/mnemonic/config/load.go` (the `mnemonic` section loader) — add the `store.central` block.

**Interfaces:**
- Consumes: existing `mnemonic` config section loader.
- Produces:
  - `type CentralConfig struct{ Enabled bool; DSN, AdminDSN string; Projects []string; PollInterval time.Duration; PushBatch int; TombstoneRetention time.Duration; TLS string }`
  - `func (c *CentralConfig) Load(baseDir string) error` — merges `mnemonic.store.central` YAML + env overrides (`SKILLGRID_MNEMONIC_CENTRAL_DSN`, `_ADMIN_DSN`, `_ENABLED=1`).
  - `func (c *CentralConfig) Validate() error` — enabled ⇒ DSN required; TLS ∈ {require, verify, disable}; poll ≥ 5s; batch 1..500.
  - `type TokenFile struct{ Server, Token string; Projects []string }` — `<dataDir>/central.json`, read/write 0600.
  - `func LoadTokenFile(dataDir string) (*TokenFile, error)` / `func (t *TokenFile) Save(dataDir string) error`.
- SATISFIES: `happy path central config defaults off`

- [ ] **Step 1: Write the failing test** — `config_test.go`:

```go
func TestCentralConfigDefaultsOff(t *testing.T) {
	var c CentralConfig
	if err := c.Load(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if c.Enabled {
		t.Fatal("default must be disabled")
	}
	if c.PollInterval != 30*time.Second || c.PushBatch != 100 || c.TombstoneRetention != 30*24*time.Hour {
		t.Fatalf("bad defaults: %+v", c)
	}
}

func TestCentralConfigEnvWinsOverFile(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "mnemonic:\n  store:\n    central:\n      enabled: true\n      dsn: postgres://file-host/db\n")
	t.Setenv("SKILLGRID_MNEMONIC_CENTRAL_DSN", "postgres://env-host/db")
	var c CentralConfig
	if err := c.Load(dir); err != nil {
		t.Fatal(err)
	}
	if c.DSN != "postgres://env-host/db" {
		t.Fatalf("env must win, got %q", c.DSN)
	}
}

func TestTokenFileRoundTripAndMode(t *testing.T) {
	dir := t.TempDir()
	tf := &TokenFile{Server: "host", Token: "egm_dev_abcd1234_secret", Projects: []string{"demo"}}
	if err := tf.Save(dir); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(filepath.Join(dir, "central.json"))
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v, want 0600", fi.Mode().Perm())
	}
	got, err := LoadTokenFile(dir)
	if err != nil || got.Token != tf.Token {
		t.Fatalf("roundtrip: %+v err=%v", got, err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd skillgrid-cli && go test ./internal/mnemonic/syncpg/ -run 'TestCentralConfig|TestTokenFile' -v`
Expected: FAIL — `CentralConfig` undefined.

- [ ] **Step 3: Implement config + token file**

Add the `store: central: {...}` struct fields to the existing `mnemonic` section in config/load.go following that file's existing per-key-merge style; wire `c.Load` into the same precedence chain (defaults < home < repo file < env).

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd skillgrid-cli && go test ./internal/mnemonic/syncpg/ ./internal/mnemonic/config/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/syncpg/config.go skillgrid-cli/internal/mnemonic/syncpg/config_test.go skillgrid-cli/internal/mnemonic/config/load.go
git commit -m "feat(mnemonic): central config block and 0600 token file"
```

---

### Task 4: Outbox wiring at the observation write sites (the tracer bullet)

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/memory/service.go` (`insertObservation` ~line 872; the update arm of the topic-key upsert path)
- Create: `skillgrid-cli/internal/mnemonic/syncpg/payload.go` (observation payload (de)serialization — the first entity; Tasks 6-8 reuse the pattern)
- Modify: `skillgrid-cli/internal/mnemonic/memory/service.go` — add `SetOutbox(*outbox.Recorder)` + `outboxRecorder()` accessor (nil-safe)
- Modify: `skillgrid-cli/internal/mnemonic/service/service.go` — boot wires the `*outbox.Recorder` into `memory.Service` when central is enabled

**Interfaces:**
- Consumes: `*outbox.Recorder` (Task 1), `memory.Service` internals.
- Produces:
  - `func ObservationPayloadRow(ctx context.Context, s *memory.Service, localID int64) string` — reads the local row (incl. `valid_at`/`invalid_at`/`superseded_by` per ADR-0011), emits JSONB `{"sync_id":..., "occurred_at":..., "created_by":..., <row fields>}`.
  - **Sync ID model (engram's):** each machine mints its own UUIDv7 at first outbox record for a local row, cached in `sync_state` keyed `obs:<localID>`. Two machines saving *new* observations get different sync_ids (both sync — no conflict). A row is created on one machine and pulled by the other, so identity is consistent by construction — the same logical row is never independently created on two machines.
  - `func outboxKeyForObservation(s *memory.Service, localID int64) string` — mint-or-reuse the cached UUIDv7.
- SATISFIES: `happy path observation save records outbox row`, `happy path disabled central is a no-op`

- [ ] **Step 1: Write the failing test** — `memory/service_outbox_test.go`:

```go
func TestInsertObservationRecordsOutbox(t *testing.T) {
	svc := testMemoryServiceWithOutbox(t) // memory.Service + capturing outbox.Recorder
	_, err := svc.Save(ctx, memory.SaveInput{Title: "t", Content: "c", Type: "fact"})
	if err != nil {
		t.Fatal(err)
	}
	got := svc.LastCaptured()
	if got.Entity != "observation" || got.Op != "upsert" || got.Key == "" {
		t.Fatalf("bad outbox capture: %+v", got)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(got.Payload), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["sync_id"] == nil || payload["occurred_at"] == nil || payload["created_by"] == nil {
		t.Fatalf("payload missing identity fields: %v", payload)
	}
}

func TestUpdateObservationRecordsSameKey(t *testing.T) {
	svc := testMemoryServiceWithOutbox(t)
	_, _ = svc.Save(ctx, memory.SaveInput{Title: "t", Content: "c1", TopicKey: "arch/x"})
	first := svc.LastCaptured()
	if _, err := svc.Save(ctx, memory.SaveInput{Title: "t", Content: "c2", TopicKey: "arch/x"}); err != nil {
		t.Fatal(err) // update arm (topic-key upsert)
	}
	got := svc.LastCaptured()
	if got.Entity != "observation" || got.Key != first.Key {
		t.Fatalf("update must re-record with the same key: %+v vs %+v", got, first)
	}
}

func TestSaveSucceedsWhenOutboxFails(t *testing.T) {
	svc := testMemoryServiceWithOutbox(t)
	svc.SetOutboxBroken() // recorder records into a closed store
	if _, err := svc.Save(ctx, memory.SaveInput{Title: "t", Content: "c"}); err != nil {
		t.Fatalf("save must not fail on outbox error: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd skillgrid-cli && go test ./internal/mnemonic/memory/ -run 'TestInsertObservation|TestUpdateObservation|TestSaveSucceedsWhenOutboxFails' -v`
Expected: FAIL — no outbox capture happens.

- [ ] **Step 3: Implement**

In `insertObservation`, after the successful `LastInsertId()` (service.go ~line 895), before `stampImportance`:

```go
	if r := s.outboxRecorder(); r != nil {
		key := outboxKeyForObservation(s, id)
		payload := syncpg.ObservationPayloadRow(ctx, s, id)
		r.Record("observation", key, "upsert", payload)
	}
```

The update arm (topic-key upsert) records the same key after its UPDATE commits. `outboxKeyForObservation` stores the minted UUIDv7 in `sync_state` keyed `obs:<localID>` so updates reuse the key.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd skillgrid-cli && go test ./internal/mnemonic/memory/ -run 'Outbox' -v && go build ./...`
Expected: PASS + build clean.

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/memory/ skillgrid-cli/internal/mnemonic/syncpg/payload.go skillgrid-cli/internal/mnemonic/service/service.go
git commit -m "feat(mnemonic): observation writes record sync outbox rows"
```

---

### Task 5: Sync manager — lease, loop, push, pull (observation entity only)

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/syncpg/manager.go`
- Create: `skillgrid-cli/internal/mnemonic/syncpg/journal.go`
- Create: `skillgrid-cli/internal/mnemonic/syncpg/apply.go` (observation arm only; other entities in Task 6)
- Create: `skillgrid-cli/internal/mnemonic/syncpg/syncpg.go`
- Create: `skillgrid-cli/internal/mnemonic/syncpg/manager_test.go`
- Modify: `skillgrid-cli/internal/mnemonic/store/sync.go` — add `AcquireLease(ctx, machineID string, ttl time.Duration) (bool, error)`, `ReleaseLease(ctx, machineID string) error`, `GetState(ctx, key string) (string, error)`, `SetState(ctx, key, value string) error` over `sync_state`, and `PendingSyncCount(ctx) (int, error)`.

**Interfaces:**
- Consumes: `*pgxpool.Pool`, `*store.Store`, `*CentralConfig`, Tasks 1-4.
- Produces:
  - `type Manager struct{ ... }` / `func NewManager(cfg *CentralConfig, pool *pgxpool.Pool, st *store.Store, machineID string) *Manager`
  - `(*Manager).Run(ctx context.Context)` — the loop (blocking; called in a goroutine at service boot).
  - `(*Manager).Status() Status` — `type Status struct{ Enabled, Running, Paused bool; ReasonCode, LastPush, LastPull string; OutboxDepth int; MachineID string }`.
  - `(*Manager).PushOnce(ctx) (int, error)` / `(*Manager).PullOnce(ctx) (int, error)` — testable single-cycle primitives.
- SATISFIES: `happy path lease guards single worker per machine`, `error path auth failure backs off and pauses with reason code`, `error path pg down queues in outbox and catches up`

- [ ] **Step 1: Write the failing tests** — `manager_test.go` (PG-backed via `PGX_TEST_DSN`; two-machine scenarios = two `*store.Store` temp files against the same PG project):

```go
func TestLeasePreventsDoubleWorker(t *testing.T) {
	st := testStore(t)
	m1 := NewManager(testCfg(t), testPool(t), st, "m1")
	m2 := NewManager(testCfg(t), testPool(t), st, "m2")
	if ok, _ := m1.AcquireLease(ctx); !ok {
		t.Fatal("m1 should win lease")
	}
	if ok, _ := m2.AcquireLease(ctx); ok {
		t.Fatal("m2 must not hold lease while m1 has it")
	}
	_ = m1.ReleaseLease(ctx)
	if ok, _ := m2.AcquireLease(ctx); !ok {
		t.Fatal("m2 should win after release")
	}
}

func TestLeaseExpiresAfterTTL(t *testing.T) {
	st := testStore(t)
	m1 := NewManager(testCfg(t), testPool(t), st, "m1")
	m2 := NewManager(testCfg(t), testPool(t), st, "m2")
	_, _ = m1.AcquireLease(ctx)
	// simulate m1 crash: expire its lease row
	if err := st.ExpireLease(ctx, "m1"); err != nil { // test helper: sets lease ts to now-70s
		t.Fatal(err)
	}
	if ok, _ := m2.AcquireLease(ctx); !ok {
		t.Fatal("m2 should win after expiry")
	}
}

func TestPushThenPullRoundTripObservation(t *testing.T) {
	stA, stB := testStore(t), testStore(t) // same project, same PG, different machines
	mA := NewManager(testCfg(t), testPool(t), stA, "machineA")
	mB := NewManager(testCfg(t), testPool(t), stB, "machineB")
	memSave(stA, "roundtrip content")
	if _, err := mA.PushOnce(ctx); err != nil {
		t.Fatal(err)
	}
	n, err := mB.PullOnce(ctx)
	if err != nil || n < 1 {
		t.Fatalf("pull: n=%d err=%v", n, err)
	}
	if got := memSearch(stB, "roundtrip"); len(got) != 1 {
		t.Fatalf("B should see A's observation: %+v", got)
	}
}

func TestPushOfflineQueuesAndCatchesUp(t *testing.T) {
	st := testStore(t)
	m := NewManager(testCfg(t), testPool(t), st, "m1")
	m.SetPool(nil) // simulate PG down
	for i := 0; i < 3; i++ {
		memSave(st, fmt.Sprintf("offline %d", i))
	}
	if depth := st.PendingSyncCount(ctx); depth != 3 {
		t.Fatalf("outbox depth = %d, want 3", depth)
	}
	m.SetPool(testPool(t)) // back online
	if _, err := m.PushOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if depth := st.PendingSyncCount(ctx); depth != 0 {
		t.Fatalf("outbox should drain, depth=%d", depth)
	}
}

func TestBackoffPausesAfterTenAuthFailures(t *testing.T) {
	st := testStore(t)
	m := NewManager(testCfg(t), authFailingPool(t), st, "m1") // PG role without LOGIN
	for i := 0; i < 10; i++ {
		m.PushOnce(ctx) // each fails auth
	}
	s := m.Status()
	if !s.Paused || s.ReasonCode != "auth_failed" {
		t.Fatalf("want paused/auth_failed, got %+v", s)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd skillgrid-cli && PGX_TEST_DSN=... go test ./internal/mnemonic/syncpg/ -run 'TestLease|TestPush|TestBackoff' -v`
Expected: FAIL — `NewManager` undefined.

- [ ] **Step 3: Implement manager, journal, apply (observation arm)**

`manager.go` core loop (engram `autosync/manager.go` pattern, transport-agnostic):

```go
func (m *Manager) Run(ctx context.Context) {
	tick := time.NewTicker(m.cfg.PollInterval)
	dirty := make(chan struct{}, 1)
	m.outboxNotifier = func() { select { case dirty <- struct{}{}: default: } }
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			m.cycle(ctx)
		case <-dirty:
			time.Sleep(500 * time.Millisecond) // debounce
			m.cycle(ctx)
		}
	}
}

func (m *Manager) cycle(ctx context.Context) {
	if !m.AcquireLease(ctx) {
		return
	}
	defer m.ReleaseLease(ctx)
	if m.Status().Paused {
		return
	}
	if _, err := m.PushOnce(ctx); err != nil {
		m.noteFailure(err)
		return
	}
	m.noteSuccess()
	if _, err := m.PullOnce(ctx); err != nil {
		m.noteFailure(err)
	}
}
```

`journal.go` — `PushOnce`: `ReadPendingSync(project, cfg.PushBatch)` → per row compute `payload_hash = hex(sha256(payload))` → batch `INSERT INTO mem_<p>.mem_mutations (entity, entity_key, op, payload, payload_hash, occurred_at, created_by) VALUES ... ON CONFLICT (entity, entity_key, payload_hash) DO NOTHING` → count accepted rows; if accepted == sent, `AckSync(sent seqs)`; else leave unacked (retry next cycle — idempotent). Touch `token_meta.last_used_at` (best-effort).

`PullOnce`: `SELECT ... FROM mem_<p>.mem_mutations WHERE seq > lastPulled ORDER BY seq` (cursor from `sync_state.last_pulled_seq:<project>`) → per row: apply into local SQLite per `apply.go` (observation arm: `op=upsert` ⇒ insert-or-update with LWW check `local.occurred_at < pulled.occurred_at` before overwrite; `op=delete` ⇒ soft-delete) → `SetState("last_pulled_seq:<project>", seq)`.

`apply.go` (observation arm; Task 6 adds the rest):

```go
func (a *Applier) ApplyObservation(ctx context.Context, st *store.Store, payload string) error {
	var p observationPayload
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		return err
	}
	// upsert by sync_id: if local row exists and local.occurred_at >= p.OccurredAt, skip (LWW)
	// insert with local id minted fresh; cache sync_state key obs_sync:<sync_id> -> local id
}
```

Backoff: `failCount` 1..10; delay `min(base*2^n, 5m)` ±25% jitter, base 2s; at 10 → `Paused=true`, `ReasonCode` mapped from error class: `pgconn` auth errors / SQLSTATE `28P01` → `auth_failed`; `42501` → `policy_denied`; `42P01`/`3D000` → `schema_mismatch`; else `transport_failed`. Status exposes it for `central status`/`doctor`.

`syncpg.go` — `New(cfg) (*SyncPG, error)`: dial member pgxpool (token file DSN) + admin pgxpool (AdminDSN, lazy), `MigrateAdmin` + `MigrateProject` per enrolled project, construct Manager + Applier + Journal.

- [ ] **Step 4: Wire at service boot**

In `service/service.go` boot path (where MCP/HTTP servers start): if `cfg.Enabled`, build `syncpg.New(cfg)` and `go mgr.Run(ctx)`. Disabled ⇒ nothing dialled.

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd skillgrid-cli && PGX_TEST_DSN=... go test ./internal/mnemonic/syncpg/ -v && go build ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/syncpg/ skillgrid-cli/internal/mnemonic/store/sync.go skillgrid-cli/internal/mnemonic/service/service.go
git commit -m "feat(mnemonic): sync manager with lease, push/pull, LWW apply, backoff"
```

**This completes the tracer thread:** one observation, end-to-end, two machines, live PG.

---

### Task 6: Remaining entity payloads + apply arms + write-site wiring

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/syncpg/payload.go` — per-entity payload builders + local-key minting (same `sync_state` cache pattern as Task 4: `fact:<id>`, `skill:<id>`, `webcache:<id>`, `session:<id>`, `session_event:<id>`, `edge:<id>`, `team:<id>`, `team_member:<id>`, `prompt:<id>`, `obs_version:<id>`)
- Modify: `skillgrid-cli/internal/mnemonic/syncpg/apply.go` — one apply arm per entity
- Modify (one block each, Task 4 pattern): the shared write sites —
  - `skillgrid-cli/internal/mnemonic/prompts/` — prompt save
  - `skillgrid-cli/internal/mnemonic/facts/facts.go` — fact insert
  - `skillgrid-cli/internal/mnemonic/skills/skills.go` — skill insert
  - `skillgrid-cli/internal/mnemonic/webcache/service.go` — web_cache upsert
  - `skillgrid-cli/internal/mnemonic/memory/` — session-end path (session row + session_events flush)
  - the edge writers (`edges`, `memory_relations`, `observation_relations`)
  - `skillgrid-cli/internal/mnemonic/teams/` — team + member changes
- Test: `skillgrid-cli/internal/mnemonic/syncpg/apply_test.go`, `payload_test.go`

**Interfaces:**
- Consumes: Task 4's payload pattern; Task 5's manager (unchanged — it is entity-agnostic).
- Produces: `PromptPayload`, `FactPayload`, `SkillPayload`, `WebCachePayload`, `SessionPayload`, `SessionEventPayload`, `EdgePayload`, `TeamPayload`, `TeamMemberPayload`, `ObservationVersionsPayload` (each `func (ctx, service, localID) string`) and matching `ApplyPromptPayload`, `ApplyFactPayload`, ..., each `func(ctx, st, payload) error` into the local store, LWW-guarded.
- SATISFIES: `happy path each shared write site records its outbox row`

- [ ] **Step 1: Write the failing test** — `payload_test.go` (table-driven, one case per entity: seed local row on store A → build payload → apply into store B → rows match; LWW respected):

```go
func TestEntityRoundTripAllEntities(t *testing.T) {
	cases := []struct {
		name    string
		seed    func(t *testing.T, st *store.Store) string // returns the outbox key
		payload func(ctx context.Context, st *store.Store, key string) string
		apply   func(ctx context.Context, st *store.Store, payload string) error
		match   func(t *testing.T, stA, stB *store.Store, key string)
	}{
		{name: "prompt", seed: seedPrompt, payload: PromptPayload, apply: ApplyPromptPayload, match: assertPromptMatch},
		{name: "fact", seed: seedFact, payload: FactPayload, apply: ApplyFactPayload, match: assertFactMatch},
		{name: "skill", seed: seedSkill, payload: SkillPayload, apply: ApplySkillPayload, match: assertSkillMatch},
		{name: "web_cache", seed: seedWebCache, payload: WebCachePayload, apply: ApplyWebCachePayload, match: assertWebCacheMatch},
		{name: "session", seed: seedSession, payload: SessionPayload, apply: ApplySessionPayload, match: assertSessionMatch},
		{name: "session_event", seed: seedSessionEvent, payload: SessionEventPayload, apply: ApplySessionEventPayload, match: assertSessionEventMatch},
		{name: "edge", seed: seedEdge, payload: EdgePayload, apply: ApplyEdgePayload, match: assertEdgeMatch},
		{name: "team", seed: seedTeam, payload: TeamPayload, apply: ApplyTeamPayload, match: assertTeamMatch},
		{name: "team_member", seed: seedTeamMember, payload: TeamMemberPayload, apply: ApplyTeamMemberPayload, match: assertTeamMemberMatch},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			stA, stB := testStore(t), testStore(t)
			key := c.seed(t, stA)
			p := c.payload(ctx, stA, key)
			if err := c.apply(ctx, stB, p); err != nil {
				t.Fatal(err)
			}
			c.match(t, stA, stB, key)
		})
	}
}

func TestApplyIsLWWGuarded(t *testing.T) {
	stB := testStore(t)
	seedLocalNewer(t, stB) // local fact with occurred_at = now
	if err := ApplyFactPayload(ctx, stB, olderFactPayload()); err != nil {
		t.Fatalf("older payload must be skipped, not an error: %v", err)
	}
	assertFactUnchanged(t, stB)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd skillgrid-cli && go test ./internal/mnemonic/syncpg/ -run 'TestEntityRoundTrip|TestApplyIsLWW' -v`
Expected: FAIL — arms undefined.

- [ ] **Step 3: Implement payloads + apply arms + one-line wiring at each write site**

Apply arms share a helper: `func lwwUpsert(ctx, st, entity, payload) error` — decode JSONB, look up local row by `sync_id` (via the `sync_state` reverse cache `sync:<entity>:<sync_id>` → local id), compare `occurred_at`, insert or update or skip. `op=delete` = `UPDATE <table> SET deleted_at=<pulled occurred_at> WHERE id=<local id>` (soft delete, matching local conventions).

Each write site gets exactly one block (pattern from Task 4), e.g. facts:

```go
	if r := s.outboxRecorder(); r != nil {
		r.Record("fact", outboxKeyFor(t, "fact", id), "upsert", syncpg.FactPayload(ctx, st, id))
	}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd skillgrid-cli && go test ./internal/mnemonic/syncpg/ ./internal/mnemonic/... -count=1`
Expected: PASS, no regressions.

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/
git commit -m "feat(mnemonic): sync outbox + apply for all shared entities"
```

---

### Task 7: LWW race + tombstone + prune semantics (end-to-end against PG)

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/syncpg/semantics_test.go`
- Modify: `skillgrid-cli/internal/mnemonic/syncpg/manager.go` — add `PruneTombstones(ctx) (int, error)`; run once per `cycle` when not paused.

**Interfaces:**
- Consumes: Tasks 5-6.
- Produces: prune pass (used by cycles; surfaced in `central status` count).
- SATISFIES: `happy path lww converges concurrent updates`, `happy path identical re-push is a no-op`, `happy path delete tombstone propagates and revives on re-save`, `happy path tombstone prune keeps re-saved sync ids`

- [ ] **Step 1: Write the failing tests** — `semantics_test.go`:

```go
func TestLWWConvergesConcurrentUpdates(t *testing.T) {
	stA, stB := twoMachines(t) // same PG project, enrolled
	saveOn(t, stA, "obs-x", "v0")
	syncBoth(t) // baseline: both have v0
	// concurrent updates of the same logical row (B's local copy of A's row):
	// A at T-2s, B at T (later)
	updateOn(t, stA, "obs-x", "vA", time.Now().Add(-2*time.Second))
	updateOn(t, stB, "obs-x", "vB", time.Now())
	syncBoth(t)
	if got := contentOn(t, stA, "obs-x"); got != "vB" {
		t.Fatalf("A = %q, want vB (later occurred_at)", got)
	}
	if got := contentOn(t, stB, "obs-x"); got != "vB" {
		t.Fatalf("B = %q, want vB", got)
	}
	assertNoSyncErrors(t)
}

func TestIdenticalRePushIsNoOp(t *testing.T) {
	stA, stB := twoMachines(t)
	saveOn(t, stA, "obs-2", "v0")
	syncBoth(t)
	var before int
	testPool(t).QueryRow(ctx, `SELECT count(*) FROM mem_demo.mem_mutations`).Scan(&before)
	mA := managerFor(t, stA)
	if _, err := mA.PushOnce(ctx); err != nil { // outbox empty; force re-push of same payload via test hook
		t.Fatal(err)
	}
	var after int
	testPool(t).QueryRow(ctx, `SELECT count(*) FROM mem_demo.mem_mutations`).Scan(&after)
	if after != before {
		t.Fatalf("journal grew %d -> %d; identical re-push must be a no-op", before, after)
	}
}

func TestTombstonePropagatesAndRevives(t *testing.T) {
	stA, stB := twoMachines(t)
	saveOn(t, stA, "obs-y", "v0")
	syncBoth(t)
	deleteOn(t, stA, "obs-y")
	syncBoth(t)
	if !isDeleted(t, stB, "obs-y") {
		t.Fatal("B should be soft-deleted")
	}
	resaveOn(t, stA, "obs-y", "v1", sameSyncID(t, "obs-y"))
	syncBoth(t)
	if isDeleted(t, stB, "obs-y") {
		t.Fatal("re-save must revive the row")
	}
}

func TestPruneKeepsResavedSyncIDs(t *testing.T) {
	stA, stB := twoMachines(t)
	saveOn(t, stA, "obs-z", "v0")
	syncBoth(t)
	deleteOn(t, stA, "obs-z")
	syncBoth(t)
	backdateTombstone(t, "obs-z", 40*24*time.Hour) // past 30d retention
	resaveOn(t, stA, "obs-z", "v1", sameSyncID(t, "obs-z"))
	syncBoth(t)
	m := managerFor(t, stA)
	if n, err := m.PruneTombstones(ctx); err != nil {
		t.Fatal(err)
	} else if n != 1 {
		t.Fatalf("pruned %d, want 1 (the stale tombstone only)", n)
	}
	if isDeleted(t, stB, "obs-z") {
		t.Fatal("live row pruned away")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd skillgrid-cli && PGX_TEST_DSN=... go test ./internal/mnemonic/syncpg/ -run 'TestLWW|TestIdentical|TestTombstone|TestPrune' -v`
Expected: FAIL (prune missing; any LWW edge case the apply guard misses).

- [ ] **Step 3: Implement prune + fix any apply edge cases the tests expose**

Prune SQL (per enrolled project):

```sql
DELETE FROM mem_$(p).mem_mutations tm
WHERE tm.op = 'delete'
  AND tm.occurred_at < now() - $1::interval
  AND NOT EXISTS (
    SELECT 1 FROM mem_$(p).mem_mutations newer
    WHERE newer.entity = tm.entity AND newer.entity_key = tm.entity_key
      AND newer.seq > tm.seq AND newer.op = 'upsert'
  );
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd skillgrid-cli && PGX_TEST_DSN=... go test ./internal/mnemonic/syncpg/ -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/syncpg/
git commit -m "feat(mnemonic): LWW race convergence, tombstone revive, prune pass"
```

---

### Task 8: Tokens, grants, audit — issue/revoke

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/syncpg/tokens.go`
- Create: `skillgrid-cli/internal/mnemonic/syncpg/tokens_test.go`

**Interfaces:**
- Consumes: admin pgxpool (from `AdminDSN`), `CentralConfig`.
- Produces:
  - `func IssueToken(ctx context.Context, admin *pgxpool.Pool, env, project, role string, pepper []byte) (token string, err error)` — generates `prefix` (8 hex chars) + 32-byte secret; `CREATE ROLE mem_<prefix> WITH LOGIN PASSWORD '<secret>'`; `GRANT USAGE ON SCHEMA mem_<project>` + `SELECT, INSERT, UPDATE, DELETE` on its tables (role `admin` additionally grants `mem_admin.token_meta` writes); stores `token_meta(prefix, env, hex(HMAC-SHA256(pepper || secret)), role_name)`; returns `egm_<env>_<prefix>_<hex(secret)>`.
  - `func RevokeToken(ctx context.Context, admin *pgxpool.Pool, prefix string) error` — `ALTER ROLE mem_<prefix> NOLOGIN` + `token_meta.revoked_at = now()`.
  - `func TouchLastUsed(ctx context.Context, admin *pgxpool.Pool, prefix string) error` — best-effort, called on push.
  - `func AuditDenial(ctx context.Context, admin *pgxpool.Pool, actor, project, action, reason string)` — inserts into `mem_admin.auth_audit_log`.
- SATISFIES: `happy path issue token grants only named project`, `error path token without grant is denied and audited`, `happy path revoke token locks its role`

- [ ] **Step 1: Write the failing tests** — `tokens_test.go`:

```go
func TestIssueTokenGrantsOnlyNamedProject(t *testing.T) {
	admin := testAdminPool(t)
	_ = pgmigrate.MigrateProject(ctx, admin, "demo")
	_ = pgmigrate.MigrateProject(ctx, admin, "other")
	token, err := syncpg.IssueToken(ctx, admin, "dev", "demo", "member", testPepper(t))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(token, "egm_dev_") {
		t.Fatalf("token shape: %q", token)
	}
	prefix := strings.Split(token, "_")[2]
	member := poolAsRole(t, "mem_"+prefix) // fresh pool logging in as the new role
	var n int
	if err := member.QueryRow(ctx, `SELECT count(*) FROM mem_demo.mem_observations`).Scan(&n); err != nil {
		t.Fatalf("granted project read must work: %v", err)
	}
	err = member.QueryRow(ctx, `SELECT count(*) FROM mem_other.mem_observations`).Scan(&n)
	if !isPermDenied(err) { // SQLSTATE 42501
		t.Fatalf("cross-project read must be denied, got err=%v", err)
	}
}

func TestRevokeTokenRemovesLogin(t *testing.T) {
	admin := testAdminPool(t)
	token, _ := syncpg.IssueToken(ctx, admin, "dev", "demo", "member", testPepper(t))
	prefix := strings.Split(token, "_")[2]
	if err := syncpg.RevokeToken(ctx, admin, prefix); err != nil {
		t.Fatal(err)
	}
	var canLogin bool
	if err := admin.QueryRow(ctx, `SELECT rolcanlogin FROM pg_roles WHERE rolname=$1`, "mem_"+prefix).Scan(&canLogin); err != nil {
		t.Fatal(err)
	}
	if canLogin {
		t.Fatal("revoked role must not have LOGIN")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd skillgrid-cli && PGX_TEST_DSN=... go test ./internal/mnemonic/syncpg/ -run 'TestIssueToken|TestRevokeToken' -v`
Expected: FAIL — `IssueToken` undefined.

- [ ] **Step 3: Implement** (per the Interfaces block; DDL statements executed individually — no multi-statement batches).

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd skillgrid-cli && PGX_TEST_DSN=... go test ./internal/mnemonic/syncpg/ -run 'Token' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/syncpg/tokens.go skillgrid-cli/internal/mnemonic/syncpg/tokens_test.go
git commit -m "feat(mnemonic): token issuance, revocation, grants, auth audit"
```

---

### Task 9: CLI — `mnemonic central` verbs

**Files:**
- Create: `skillgrid-cli/cmd/skillgrid/central.go`
- Create: `skillgrid-cli/cmd/skillgrid/central_test.go`
- Modify: `skillgrid-cli/cmd/skillgrid/main.go` — register the `central` subcommand group.

**Interfaces:**
- Consumes: `syncpg.*` (all of it), existing CLI wiring patterns (see `mem.go`).
- Produces: six verbs with stable output:
  - `central status` — JSON: `{"enabled":bool, "running":bool, "paused":bool, "reason_code":str, "last_push":str, "last_pull":str, "outbox_depth":int, "projects":[{"name":str,"enrolled":bool}]}`.
  - `central enroll <project>` — validates project id, writes `sync_enrollments`, migrates the PG project schema (admin DSN), runs `BootstrapProject` (Task 10).
  - `central unenroll <project>` — clears the enrollment flag (pending rows stay; pushes stop).
  - `central issue-token --project <p> [--role member|admin]` — prints the token once to stdout.
  - `central revoke-token <prefix>`.
  - `central doctor` — checks DSN reachable (admin), role has LOGIN, per-enrolled-project grant present, schema version matches; exit 0 ok / 1 with a named reason otherwise.
- SATISFIES: `happy path central doctor validates connection and grants`, `happy path unenroll stops future pushes and preserves data`

- [ ] **Step 1: Write the failing tests** — `central_test.go` (command-level against test PG + temp data dir):

```go
func TestCentralStatusDisabled(t *testing.T) {
	out := runCLI(t, "central", "status") // central not configured
	var st map[string]any
	if err := json.Unmarshal([]byte(out), &st); err != nil {
		t.Fatal(err)
	}
	if st["enabled"] != false {
		t.Fatalf("want disabled: %s", out)
	}
}

func TestCentralDoctorBadDSNExitsNonZero(t *testing.T) {
	code := runCLIExpectingFail(t, "central", "doctor", withCentralDSN("postgres://bad@nowhere/db"))
	if code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
}

func TestEnrollUnenrollToggle(t *testing.T) {
	runCLI(t, "central", "enroll", "demo")
	if !isEnrolled(t, "demo") {
		t.Fatal("enroll must set flag")
	}
	runCLI(t, "central", "unenroll", "demo")
	if isEnrolled(t, "demo") {
		t.Fatal("unenroll must clear flag")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd skillgrid-cli && go test ./cmd/skillgrid/ -run 'TestCentral' -v`
Expected: FAIL — unknown command `central`.

- [ ] **Step 3: Implement the verbs** (follow `mem.go`'s existing verb style; JSON via `json.MarshalIndent`; doctor prints a per-check table).

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd skillgrid-cli && PGX_TEST_DSN=... go test ./cmd/skillgrid/ -run 'TestCentral' -v && go build ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/cmd/skillgrid/central.go skillgrid-cli/cmd/skillgrid/central_test.go skillgrid-cli/cmd/skillgrid/main.go
git commit -m "feat(mnemonic): central CLI verbs (status/enroll/tokens/doctor)"
```

---

### Task 10: Bootstrap (enroll imports local history) + two-machine end-to-end

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/syncpg/bootstrap.go`
- Create: `skillgrid-cli/internal/mnemonic/syncpg/bootstrap_test.go`
- Create: `skillgrid-cli/internal/mnemonic/syncpg/e2e_test.go`
- Create: `acceptance-tests/features/central-pg.feature` (mirror of `acceptance.feature`)

**Interfaces:**
- Consumes: Tasks 1-9.
- Produces: `func BootstrapProject(ctx context.Context, st *store.Store, project string) (int, error)` — for each shared entity, select all non-deleted local rows, mint/reuse `sync_id` (same `sync_state` key cache as the write path), `Record("upsert")` into the outbox. LWW note: bootstrap loses to anything newer already in PG (enroll prints how many rows were queued).
- SATISFIES: `happy path enroll bootstraps local history`, `happy path two machines share memory end to end`

- [ ] **Step 1: Write the failing tests**:

```go
func TestBootstrapImportsLocalHistory(t *testing.T) {
	st := testStore(t)
	for i := 0; i < 20; i++ {
		memSave(st, fmt.Sprintf("history %d", i))
	}
	n, err := BootstrapProject(ctx, st, "demo")
	if err != nil || n != 20 {
		t.Fatalf("bootstrap: n=%d err=%v", n, err)
	}
	m := NewManager(testCfg(t), testPool(t), st, "m1")
	if _, err := m.PushOnce(ctx); err != nil {
		t.Fatal(err)
	}
	var pgCount int
	testPool(t).QueryRow(ctx, `SELECT count(*) FROM mem_demo.mem_observations`).Scan(&pgCount)
	if pgCount != 20 {
		t.Fatalf("PG has %d, want 20", pgCount)
	}
}

func TestTwoMachinesShareMemoryEndToEnd(t *testing.T) {
	pool := testPool(t)
	stA, stB := testStore(t), testStore(t) // both project "demo", same PG
	mA, mB := NewManager(testCfg(t), pool, stA, "A"), NewManager(testCfg(t), pool, stB, "B")
	_, _ = mA.PushOnce(ctx)
	_, _ = mB.PullOnce(ctx)

	memSave(stA, "e2e shared observation")
	memSaveFact(stA, "e2e fact")
	memSaveSkill(stA, "e2e skill")
	if _, err := mA.PushOnce(ctx); err != nil {
		t.Fatal(err)
	}
	n, err := mB.PullOnce(ctx)
	if err != nil || n < 3 {
		t.Fatalf("pull n=%d err=%v", n, err)
	}
	if len(memSearch(stB, "e2e shared")) != 1 {
		t.Fatal("B must find A's observation")
	}
	assertFactExists(t, stB, "e2e fact")
	assertSkillExists(t, stB, "e2e skill")
	assertNoCodeIndexRows(t, stB) // code index / embeddings never cross
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd skillgrid-cli && PGX_TEST_DSN=... go test ./internal/mnemonic/syncpg/ -run 'TestBootstrap|TestTwoMachines' -v`
Expected: FAIL — `BootstrapProject` undefined.

- [ ] **Step 3: Implement bootstrap + wire into `central enroll` + write the BDD feature mirror**

- [ ] **Step 4: Run the full verification floor**

Run: `cd skillgrid-cli && go build ./... && go test ./... -count=1 && go vet ./...`
Expected: all green (including with no PG — PG tests skip on missing `PGX_TEST_DSN`).

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/syncpg/ acceptance-tests/
git commit -m "feat(mnemonic): enroll bootstrap and two-machine e2e"
```

---

## Plan Review

- Verdict: READY FOR EXECUTION
- Findings: 1 Important (fixed: sync_id identity model spelled out in Task 4 — rows are created on one machine and pulled by the other, so per-machine UUIDv7 mints never collide for the same logical row), 2 Minor (deferred: long-poll push-notify — M3; `central.json` keychain storage — user locked file 0600).
- Failure modes checked: PG down (Task 5), partial push ack (idempotent re-push via payload_hash UNIQUE), lease crash (TTL expiry test), auth revoke mid-run (pause + reason code), concurrent same-key updates (Task 7), interrupt mid-migration (tx per file, ON CONFLICT), offline outbox growth (unbounded by design; depth surfaced in status).
- Scope: matches the briefing's 8 requirements; nothing extra (no dashboard page, no OIDC, no code-index sync — all deferred to M3 in the briefing's out-of-scope list).
- ADR compliance: ADR-0012 (local store unchanged), ADR-0011 (classifier untouched), ADR-0016 (fail-open), ADR-0024 (all locked decisions), "no new deps without an ADR" (pgx named).
- Reviewed: 2026-10-05

## Owed-Decision Gate

Input coverage checked: every value the build produces has a named source — sync_id (Task 4 model), occurred_at (local row timestamp), payload_hash (sha256 of payload bytes), backoff numbers (engram constants, in Task 5), token shape (Task 8), retention (config, Task 3), schema DDL (Task 2), project id charset (Task 2 regex). No owed decisions — all load-bearing choices were locked by the user on 2026-10-05 and recorded in ADR-0024 + the briefing.

## Execution Handoff

10 tasks, dependency chain is strictly linear (each task builds on the previous), so no slicing into waves is needed — execute in order 1 → 10.

Two execution options:

1. **Subagent-Driven (recommended)** — fresh subagent per task, review between tasks.
2. **Inline Execution** — `skillgrid:simple-execution`, batch execution with checkpoints.

T3 note: the final review escalates to `skillgrid:parallel-code-review` (new trust boundary).
