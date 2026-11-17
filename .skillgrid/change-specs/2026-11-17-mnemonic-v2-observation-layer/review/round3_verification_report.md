# Round 3 Verification Report — 2026-11-17-mnemonic-v2-observation-layer

Final verification pass after the round-2 revision (commit
"docs(spec): obs goldfish rounds 1-2"). Scope: (a) verify each round-2 open
item is RESOLVED in the revised text, with code spot-checks on the N-1 fix;
(b) fresh-eyes pass on ONLY the revised deltas (the two pi/omp mapping rows,
the Task 2 Files/Consumes/Step 3 `Txn.Exec` additions, the Task 4 R9 line)
for contradictions elsewhere in the blueprint. Read-only toward the change
docs; no git, no go test.

## Per-item disposition

| ID | Round-2 status | Status | Evidence |
|---|---|---|---|
| F-1 residual | RE-OPENED (Major) | **RESOLVED** | Mapping table now splits pi/omp into three rows: `\| pi, omp \| session_shutdown (quit only) \| session_end \|` (blueprint:185) — `session_shutdown` no longer appears in the "same names" row (line 184 = `session_start / tool_call / tool_result` only). Every raw hook the pi/omp adapter emits is covered by exactly one row: Task 4 adapter source registers exactly `pi.on("session_start")`, `pi.on("before_agent_start")`, `pi.on("tool_call")`, `pi.on("tool_result")`, `pi.on("session_shutdown")` (blueprint:426-431) ↔ rows 184/186/184/184/185 — 5 names, no duplicates. Every produced `action_type` is in the DDL vocabulary (`session_start, session_end, session_error, tool_call, tool_result, prompt, turn_end, compact, agent_response, agent_thought, subagent_start, subagent_stop`); opencode/kilo (5 raw) and cursor (20 raw — re-verified name-by-name against this host's live `~/.cursor/hooks.json`, all 20 present exactly once across the four cursor rows) unchanged from the round-2-verified state. |
| N-1 | Major | **RESOLVED** | Task 2 Files now records the additive extension: "Modify: `mnemonic/internal/core/store/store.go` (add `Exec(ctx context.Context, query string, args ...any) error` to the `Txn` interface), `sqlite.go` (`sqliteTxn`), `postgres_code.go` (`pgCodeTxn`) — additive interface extension … Find and update any test fakes implementing `Txn` (grep `ReplaceChunksForFile`)." Consumes: "The `Txn` surface has no generic statement method today — this task adds `Txn.Exec` (Files above); the ingest tx then composes two raw statements inside one Txn." Step 3: "inside one Txn, using the `Txn.Exec` statement method added in this task's Files step — session upsert (`ON CONFLICT(id) DO NOTHING`), event insert, commit. (The generic `st.Exec`/`ExecDDL` are single-statement non-transactional — NOT the seam.)" **Code spot-checks (all hold):** (1) `Txn` interface at `store.go:146-153` — 5 code-index writes + Commit/Rollback, no generic Exec/Query ✓; (2) `sqliteTxn` at `sqlite.go:2639-2642` holds `tx *sql.Tx` (Begin at 2629-2635) ✓; (3) `pgCodeTxn` at `postgres_code.go:676-679` holds `tx *sql.Tx` (Begin at 666-672, "near :671" holds) ✓; (4) implementer census: repo-wide `grep "Begin(ctx context.Context) (Txn, error)"` → exactly `pgStore` + `sqliteStore`; repo-wide `ReplaceChunksForFile` → the two Txn twins (`sqlite.go:2665`, `postgres_code.go:702`), the two Store-level methods, the two interface declarations, and call sites only — **no test fake implements `Txn`** (test-file hits are `st.ReplaceChunksForFile` on the `Store` interface). The note is sufficient: any `Txn` implementer must define `ReplaceChunksForFile`, so the grep finds them all. |
| Sub-threshold minor (R9) | minor | **RESOLVED** | Task 4 Step 1 status pin now reads: "status reports installed/stale (hash mismatch) per host + **that host's store event count (the briefing R9 sub-clause)**" (blueprint:452-453) — carries briefing R9's "store-side event counts" (`briefing.md:277-279`); composable against the schema (`obs_events.agent` column + per-project store). |

## Fresh-eyes pass on revised deltas

| Check | Result |
|---|---|
| T-1 "in one transaction" vs new seam | Consistent — Step 3 composes both raw statements inside one Txn via `Txn.Exec`; the non-seam alternatives are explicitly excluded. |
| Global Constraints "no new dependencies" | Consistent — `Txn.Exec` adds no import beyond stdlib `context` (already used by the interface) and `*sql.Tx` (already held by both adapters). |
| Task 2 commit scope | Coherent — `feat(obs): store ops (ingest tx, query, prune)` now spans `internal/obs/` + `internal/core/store/`; both are code zone (zone guard is spec-vs-code, no violation), and the extension is explicitly recorded in Files in service of the obs ingest tx. Additive method also fits the extension ADR's own consequence ("adding a method later is additive"). |
| File Structure section ("raw `st.Exec`/`st.Query` with the store's tx seam") | Unchanged, still consistent with the added `Txn.Exec`. |
| Acceptance feature | No scenario was touched by any fix; `obs-live-omp-session-produces-events` expects `session_end` — consistent with the revised mapping row. |
| R9 pin vs other status pins | Task 6 Step 5 "obs status reports cursor installed" — no contradiction (count is additive per host). |

## New findings

### F-14 — MINOR — ADR misattribution in the Task 2 Files note

Task 2 Files cites "(the **ADR-0045** Store-extension precedent; …)" for the
`Txn.Exec` additive extension. ADR-0045 is the **opencode V2 plugin port**
(`.skillgrid/projectwiki/decisions/04-adr-0045-opencode-v2-plugin-port.md`,
"# ADR-0045: OpenCode V2 plugin port") — the blueprint's own header lists
"ADR-0045 (opencode V1+V2 dual entrypoint)". The store-extension precedent is
**ADR-0046** (`04-adr-0058-code-index-store-extension.md`, "# ADR-0046 —
Code-index `Store` interface extension (additive)" — establishes the K1
`Begin`/`Txn` seam this fix extends and the "adding a method later is
additive" consequence). Same class as round-1 F-7 (attribution); no effect on
implementability. Note (out of scope, pre-existing): the decisions vault
itself carries a numbering collision — `04-adr-0057-code-index-onnxer-conditional.md`
also claims the ADR-0045 number.

## Verdict

**CLEAN** — 0 Blocker, 0 Major outstanding. All round-2 open items RESOLVED
(F-1 residual, N-1 + code spot-checks, R9 sub-clause); one new sub-threshold
minor (F-14, ADR-0045→0046 misattribution in the Task 2 note).
