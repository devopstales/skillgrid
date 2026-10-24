# Task 01 — relay-schema report (change 006)

Verdict: `PASS`

## Schema created

`skillgrid-cli/internal/mnemonic/store/migrations/019_session_relay.sql` — slot is **019**, not the brief's `012_*`: the store already ships migrations through `018_layered_distill.sql` (008/010/011/013/017/018 from other changes), so `012_session_relay.sql` would have collided with `012_community_knowledge_graph.sql`. Next free number is 019; confirmed by listing `store/migrations/`.

Both tables are pure `CREATE TABLE IF NOT EXISTS` + `CREATE INDEX IF NOT EXISTS`. No `sessions`/`observations` or other table is altered, dropped, or rewritten.

### session_handoffs
| Column | Type | Notes |
|---|---|---|
| id | INTEGER PK AUTOINCREMENT | surrogate |
| project | TEXT NOT NULL | store partitioning, matches other tables |
| handoff_id | TEXT NOT NULL | operator-facing id = cleave bundle dir name; resume target; `UNIQUE (project, handoff_id)` |
| source_session | TEXT | `REFERENCES sessions(id) ON DELETE SET NULL`, nullable — a handoff survives if its source session row is later removed |
| status | TEXT NOT NULL DEFAULT 'pending' | `pending \| archived`; explicit mutation only |
| cleave_path | TEXT NOT NULL | repo-relative path of the `.skillgrid/.cleave/<handoff_id>/` bundle |
| context_summary | TEXT | optional short note written at handoff time |
| created_at / archived_at | TEXT | RFC3339 |

Indexes: `idx_handoffs_source (project, source_session)`, `idx_handoffs_status (project, status)`.

### session_archives
| Column | Type | Notes |
|---|---|---|
| id | INTEGER PK AUTOINCREMENT | surrogate |
| project | TEXT NOT NULL | |
| session_id | TEXT NOT NULL | `REFERENCES sessions(id) ON DELETE SET NULL`, nullable |
| handoff_id | TEXT NOT NULL | link to the handoff row (app-enforced, see concerns); `UNIQUE (project, handoff_id)` |
| path | TEXT NOT NULL | on-disk archive (tarball or dir) the next session resumes from |
| created_at | TEXT | RFC3339 |

Index: `idx_archives_session (project, session_id)`.

## RED → GREEN

**RED** — `go test ./internal/mnemonic/store/ -run 'SessionRelay' -count=1` (before the migration existed):

```
--- FAIL: TestSessionRelayMigration (0.12s)
    session_relay_schema_test.go:55: expected relay table session_handoffs after open
FAIL
```

Reason: tables don't exist yet — 019 not applied on open.

**GREEN** — same command after creating `019_session_relay.sql`: `ok … 1.523s`.

## Prior-rows-survive + re-open idempotency proof

`TestSessionRelayMigration` (in `session_relay_schema_test.go`) builds a **pre-019** store via `rawSQLDB` + explicit 001-era `sessions`/`observations` DDL, seeds 1 session (`s1`) + 1 observation (`keep-me`), then:

1. `Open` → asserts `session_handoffs` + `session_archives` exist, `019_session_relay.sql` recorded exactly once in `index_meta`.
2. Prior rows survive: `COUNT(*)` on both tables still 1; `sessions.s1.status='active'`; observation `title='keep-me'` and `content='body'` byte-identical → no rewrite, no rebuild.
3. Re-open: second `Open` succeeds, 019 still recorded **once** (not re-applied), both relay tables still present, prior counts still 1/1.
4. Usability smoke: inserts one row per new table and reads it back (steps 02–05 will write these shapes).

## Suite results

Focused + store harness (from `skillgrid-cli/`):

```
$ go test ./internal/mnemonic/store/ -run 'SessionRelay' -count=1
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store	1.523s

$ go test ./internal/mnemonic/store/ -count=1
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store	3.257s
```

Baselines (005/008/010/011/013 owners) all green:

```
$ go test ./internal/mnemonic/memory/... ./internal/mnemonic/route/... ./internal/mnemonic/affected/... ./internal/mnemonic/community/... ./internal/mnemonic/pdg/... ./internal/mnemonic/codeindex/... -count=1
ok  memory            8.615s
ok  memory/layer      2.715s
ok  route             2.461s
ok  affected         12.930s
ok  community         4.662s
ok  pdg               3.936s
ok  codeindex        16.454s
```

## Commits

- `53cc95b` feat(mnemonic): add 019 session relay schema (01.2.e — migration + test; 01.1 and 01.3 folded in, both AFK)

## Concerns / column choices for steps 02–05

- **FK on `session_archives.handoff_id` is app-enforced, not a SQLite FK.** SQLite FKs must target a PK or a unique index, and `handoff_id` uniqueness is scoped to `(project, handoff_id)`, so a cross-project FK would be a "foreign key mismatch". Step 02 must not archive a handoff id that doesn't exist in the same project (fail closed per change.md error table — no orphan rows anyway, since handoff writes fail-closed before the row is written).
- **`source_session` / `session_id` are nullable with `ON DELETE SET NULL`** so relay records survive session-row cleanup; steps 02–05 should treat a NULL source as "session row gone, bundle still authoritative" rather than an error.
- **`status` defaults to `'pending'`** and flips to `'archived'` explicitly — step 03 (`session_status`) can count by this column via `idx_handoffs_status`.
- **`context_summary`** is a free-form optional note; if step 03's `session_status` needs richer stats (cost/context), they live in the cleave bundle on disk, not here — this table stays thin by design.
- Brief said slot `012_*`; shipped as `019` (collision, documented above). `tasks.md`/`acceptance.feature` left untouched per rules — the step verifier may want to note the slot correction.
