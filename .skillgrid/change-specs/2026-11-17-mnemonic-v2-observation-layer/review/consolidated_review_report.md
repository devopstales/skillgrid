# Consolidated Review Report — 2026-11-17-mnemonic-v2-observation-layer

Preset: **Consolidated** (Step=Consolidated: New Function, ≥3 tasks → full
pipeline, no pauses). One round. Stage reports: `zoom_out_report.md`,
`comprehension_report.md`, `critic_report.md`, `intermediate_zoom_out_report.md`,
`implementation_readiness_report.md` (this directory).

Target: `blueprint.md` (9 tasks). Reading list: `briefing.md`,
`acceptance.feature`, and the cited codebase files (all read and verified —
see the Stage 9 verification record).

## Stage / reviewer status

| Stage | Reviewer(s) (brief) | Status | Findings |
|---|---|---|---|
| 5 Initial zoom out | zoom-out | ran | 0 (NO FINDINGS) |
| 6 Comprehension | comprehension | ran | 1 Nit |
| 7 Critic | design-critic + structural + style | ran (3 briefs, merged) | 7 (4 Major, 2 Minor, 1 Nit) |
| 8 Intermediate zoom out | zoom-out (fresh instance) | ran (fold-state noted in report) | 0 (NO FINDINGS) |
| 9 Implementation readiness | implementation-readiness + codebase-details + style | ran | 5 (3 Major, 2 Minor) |

No reviewer was `not run`; no stage capped at "not verified".

## Severity breakdown (normalized at the record boundary)

Blocker→CRITICAL, Major→WARNING, Nit/Minor→SUGGESTION:
**CRITICAL 0 · WARNING 4 · SUGGESTION 9** (raw reviewer vocabulary: 4 Major,
6 Minor, 3 Nit).

## Consolidated findings (cross-stage deduplicated)

Dedup note: F-3/F-4 were raised by Stage 7 (design-critic) and re-verified
against code in Stage 9 (codebase-details) — carried once, Stage 7 is the
origin stage. F-7 likewise (Stage 7 origin, Stage 9 verification). F-13 was
re-scanned in Stage 9 style (no new hits) — recorded once in Stage 7.

| ID | Sev | Origin stage(s) | Blueprint line | One line |
|---|---|---|---|---|
| F-1 | Major (WARNING) | 7 dc, 9 | 173; 212-219; 442; 455-460 | Envelope contract "event ∈ normalized action_type vocabulary" contradicts the pinned adapter calls (per-host hook names for opencode/kilo/cursor + pi/omp `session_shutdown`); the agent-scoped event→action mapping is unpinned (opencode `session.error` has no target; no mapping rows in Task 1's table tests) |
| F-2 | Major (WARNING) | 7 dc | 316; 174-175 | `_ingest` "missing args → exit 2" (CLI convention) collides with "exit 2 = reserved block" + the pi/omp adapter's exit-2→`{block:true}` — a usage error on the pre-tool path would block the agent (R8 hole) |
| F-3 | Major (WARNING) | 7 dc, 9 cd/ir | 395 | "the built-in copy" source of the 4 adapter files is undefined (v1 `stagedPluginPath` pattern has no v2 staging owner; no `//go:embed` in any Files list) — a Zero-Decision choice every host installer depends on |
| F-4 | Major (WARNING) | 7 dc, 9 cd | 257 | Pinned `path.Match` for `--file` cannot match `src/**` recursively (`path.Match("src/**","src/a/b.go")` == false, verified); gryph's `--file` is recursive (`gryph query --help` example `src/**/*.ts`) — the pinned matcher fails the pinned acceptance T-3 for nested paths |
| F-5 | Minor (SUGGESTION) | 9 cd | 126 | Data-dir env override pinned as `SKILLGRID_MNEMONIC_DATA_DIR`; v2 config reads `MNEMONIC_DATA_DIR` (`config.go:35`); the pinned name exists nowhere |
| F-6 | Minor (SUGGESTION) | 7 st | 232; 252 | `Ingest(ctx, st, ev Event)` must upsert `sessions.directory` but `obs.Event` ("the row struct") carries no cwd; v2 precedent writes `''` — value + carrier both unpinned |
| F-7 | Minor (SUGGESTION) | 7 dc, 9 cd | 212-219 | "the file is the spec; port, don't re-derive" misattributes 3 items: v1 JS `sensitivePath` lacks `.env`/`*.pem`/`*.key`/`*secret*` (R5/gryph union), v1 `contentHash` is 32-bit djb2 (not sha256), v1 truncation marker is `"\n[truncated]"` (not `…[+N]`) — the per-item pinned values are right, the attribution line invites a regression |
| F-8 | Minor (SUGGESTION) | 9 ir | 333-378 | Task 4 claims `obs-fail-open-missing-binary` "(unit half)" but has no pi/omp adapter unit test (opencode has one) — the pi-family fail-open contract is live-backstop-only |
| F-9 | Minor (SUGGESTION) | 7 dc | 505 | `obs sessions` column "agent" has no source (sessions table has no agent column; ingest sets no title) and "last-active" is undefined — derivation unpinned |
| F-10 | Minor (SUGGESTION) | 7 dc | 339; 75 | `obs install all` on a host-absent machine is unpinned (create all 5 vs detect); scenario `obs-live-hosts` implies "detected hosts" with no detection rule defined |
| F-11 | Nit (SUGGESTION) | 7 dc | 589; 123-125 | Threat-matrix "the hard timeouts per host (Global Constraints)" overstates cursor (no adapter cap pinned; bound is the DSN `busy_timeout(5000)` + unbounded `gitToplevel` shell-out) |
| F-12 | Nit (SUGGESTION) | 6 | 276; 324 | "K1"/"K6" spec-zone ids used without definition or link — unresolvable for a zero-context reader |
| F-13 | Nit (SUGGESTION) | 7 style (re-checked 9) | 568 | "correct agent label" — "correct" is on the banned validating-adjective list |

## Proposed dispositions (CRITICAL/WARNING, for the ledger)

- **F-1** — Revise line 173: "`<event>` is the host hook name (per-host
  vocabulary, pinned by the briefing's evidence table); `action_type` is the
  normalized output of the agent-scoped mapping". Pin the full mapping table
  in the blueprint (cursor mapping already in briefing R1; add opencode
  `tool.execute.before→tool_call`, `tool.execute.after→tool_result`,
  `session.created→session_start`, `session.idle→turn_end`,
  `session.error→?` (DECISION: `session_end` with the message in `error`, or a
  new `session_error` value), and pi/omp `session_shutdown→session_end`,
  `before_agent_start→prompt`). Add the mapping rows to Task 1 Step 1 table
  tests.
- **F-2** — In the envelope exit-code table, pin "_ingest usage/parse errors
  exit 1 (stated exception to the CLI usage=2 convention); exit 2 is emitted
  only by the phase-2 policy plane"; change Task 3 Step 1 "missing args → exit
  2" to exit 1; optionally pin positionals-as-canonical vs the JSON `agent`/
  `event` fields.
- **F-3** — Pin one sentence in Task 4 Step 3: "the obs package
  `//go:embed`s `plugins/obs/**` (the built-in copy; repo truth at build
  time); `--repo <checkout>` (or cwd inside a checkout via `gitToplevel`)
  re-stages `$DATA_DIR/obs-plugins/` and installs from there (dev flow)" + add
  the embed location to Task 4's Files list.
- **F-4** — Pin in Task 2 Interfaces: a small `**`-aware string matcher in
  `internal/obs` (segment-wise Glob semantics: `**` = zero or more segments)
  with table cases (`src/**` vs `src/a/b.go` true, vs `src/b.go` true, vs
  `src/a.tsx` false); or, if single-segment is intended, re-pin T-3 + the
  acceptance scenario to `src/*` (the former is the gryph-parity choice).

SUGGESTIONS (F-5…F-13): each has a one-to-three-line fix stated in its stage
report; no ledger disposition required beyond the fold.

## Traceability matrix

SATISFIES ↔ acceptance scenarios ↔ tasks (all names resolve; reverse coverage
complete — no orphan scenario, no dangling SATISFIES name):

| Acceptance scenario | Task(s) |
|---|---|
| obs-ingest-normalizes | 1 |
| obs-privacy-contract | 1 |
| obs-ingest-atomically-registers-session | 2 |
| obs-prune-scoped-to-obs-events | 2, 7 |
| obs-logs-and-query-read-stored-events | 3 (F-4 bears on its `--file` check) |
| obs-ingest-exit-codes-fail-open | 3 (F-2 refines its exit-code pin) |
| obs-install-omp-and-pi | 4 |
| obs-fail-open-missing-binary | 4 (unit — F-8), 9 (live) |
| obs-live-omp-session-produces-events | 4 (backstop) |
| obs-install-opencode-kilo | 5 |
| obs-install-cursor-preserves-foreign-entries | 6 |
| obs-sessions-session-stats-export | 7 (F-9 bears on its sessions check) |
| obs-live-hosts | 9 (F-10 bears on "detected hosts") |

Must-Have truths → tasks: T-1 → 1,2,3 · T-2 → 4,9 · T-3 → 3 · T-4 → 1 · T-5 →
3,4,9 · T-6 → 6 · T-7 → 2,7 · T-8 → 7 · T-9 → 1. One-way doors: migration 300
is the only true one-way door (additive-only migration discipline); the
"Assessed, not one-way" entries (installed-file ownership, envelope v:1, CLI
family) are defensible — the CLI family is renamable via the existing
`renamedCommands` did-you-mean mechanism (v2 itself renamed `mem`→`memory`).
Threat-matrix N/A reasons hold row by row (doc-paths, commit/push/PR, tool
surface, shared-convention drift); the "Git repository selection" row is
applicable with the test pinned. No placeholder patterns (TBD/TODO/"similar to
Task N") found. Task ordering is topological; every Consumes resolves to an
earlier task or existing, verified code.

## Verdicts (one line per stage)

- **Stage 5 (zoom out):** clean — scope, reuse, altitude hold; the one
  deliberate non-reuse (self-contained installer) is a documented decision.
- **Stage 6 (comprehension):** substantially clean — document paraphrases
  zero-context; one Nit (K1/K6 ids unresolvable).
- **Stage 7 (critic):** 4 contract-level Majors (event-mapping contradiction,
  exit-2 collision, built-in-copy mechanism, `path.Match` vs `src/**`) + 2
  Minors + 1 Nit — all fixable in-place, none structural to the architecture.
- **Stage 8 (intermediate zoom out):** clean — no pre-existing drift; the
  round-1 folds are altitude-safe if each lands in its pinned location
  (drift watch items noted in the report).
- **Stage 9 (implementation readiness):** codebase claims verify almost
  completely (checks a–h + sessions schema, Detector, WAL DSN, ADRs, test
  patterns — evidence in the report); the actionable gaps are the wrong env
  var name, the v1-port attribution, and the missing pi/omp adapter unit test.
