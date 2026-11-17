# Stage 7 — Critic Review (target: `blueprint.md`)

Personas merged: `design-critic.md` + `structural.md` + `style.md` (three fresh
passes, findings merged below, tagged by persona). Mode: REVIEW.
Harness isolation: part 1 enforced by fresh subagent dispatch; part 2 enforced
(writes confined to `review/`); part 3 partial — per-project instruction files not
loaded as instructions (stated, not silent); part 4 enforced (local files + local
binaries; no network).

## Findings

### design-critic

Severity: Major
Affected scope: Ingest envelope section (line 173); Task 1 Step 1 (lines 212-219);
Task 5 Step 1 (line 442); Task 6 template (lines 455-460)
Description: The envelope contract line "`event` ∈ the normalized `action_type`
vocabulary" is contradicted by the blueprint's own pinned adapter calls. The
positional `<event>` is the per-host hook name for opencode/kilo (Task 5 line 442
pins `["obs","_ingest","opencode","tool.execute.before"]`) and cursor (Task 6 pins
`obs _ingest cursor <hookType>` over the 20 cursor hook names), and for pi/omp's
`session_shutdown` — none of which are in the normalized vocabulary
(`session_start|session_end|tool_call|tool_result|prompt|turn_end|compact|
agent_response|agent_thought|subagent_start|subagent_stop`). The normalized
vocabulary is the *output* of an agent-scoped event→action mapping (briefing R1),
but that mapping is not pinned in the blueprint: Task 1's table tests (lines
212-219) cover mapToolType/sensitivePath/stripPrivateTags/truncation/hash/
private-tools but no event→action case; briefing R1 pins the cursor mapping fully
but leaves opencode `session.error`'s target unpinned (the vocabulary has no
`session_error`; only `session.created → session_start` and `session.idle →
turn_end` are stated).
Rationale: Architectural-gap + unclear-interface vectors: the whole audit layer
rests on this seam, and Task 3's "unknown `event`/`agent` → exit 1" rule is
unimplementable without the per-host event vocabulary. A fresh implementer must
invent a mapping table and possibly a vocabulary extension. Fix: restate line 173
("`<event>` is the host hook name; `action_type` is the normalized output of the
agent-scoped mapping below"), pin the complete mapping table in the blueprint
(including `session.error`'s target and pi/omp `session_shutdown`/
`before_agent_start`), and add the mapping rows to Task 1's table tests.

Severity: Major
Affected scope: Task 3 Step 1 (line 316); Ingest envelope exit codes (lines 174-175);
Task 4 Adapter source (pi `tool_call` exit-2 handling)
Description: Exit-code collision on the hidden verb: Task 3 pins "missing args →
exit 2" (the CLI-wide usage-error convention, `main.go` `usageError` → exit 2),
while the envelope section pins "2 reserved (block, phase 2)" and Task 4's adapter
maps exit 2 → `{block: true, reason: stderr}` on the pre-tool `tool_call` path.
Combined: any usage/parse path that yields exit 2 on the `tool_call` path
converts a capture error into a *block* of the agent's tool call (with a usage
message as the block reason) — directly against R8 ("a capture bug can never
freeze an agent") and this change's own "risky — untrusted agent processes exec
the binary" tier.
Rationale: Failure-mode vector at the change's core trust boundary. Today the
shipped adapters always pass exactly two positionals, so the hole is narrow — but
fail-open must absorb adapter bugs, and a wrong argv is exactly such a bug. Fix:
pin `_ingest` usage/parse errors to exit 1 (an explicit, stated exception to the
CLI convention in the envelope exit-code table), so exit 2 is emitted only by the
phase-2 policy plane. While there: pin whether the JSON `agent`/`event` fields
must match the positionals (the acceptance says "an envelope with an unknown agent
exits 1", implying JSON validation; make the positionals canonical or state the
cross-check).

Severity: Major
Affected scope: Task 4 Step 3 (line 395)
Description: The adapter source-of-truth mechanism is left to the implementer:
"staged source dir = `$DATA_DIR/obs-plugins/` synced from the repo `plugins/obs/`
at install time when `--repo` points at a checkout, else the built-in copy".
"the built-in copy" is undefined — go:embed in the obs package? a copy staged by
some other step? The referenced v1 pattern (`stagedPluginPath`,
`skillgrid-cli/internal/setup/setup.go:196`) resolves `~/.skillgrid/plugins/<rel>`
which is owned by v1's `skillgrid install` hook-tree staging — a staging owner that
does not exist for these new obs plugins. The choice propagates: an `//go:embed`
directive appears in no task's Files list, and the resolution affects the install
behavior of all five hosts (all four source files are read through this path).
Rationale: Zero-Decision Rule applied because the choice is a design decision, not
a detail: embed-free (requires a repo checkout at install time → breaks a
globally-installed binary), embed (truth frozen at build time), or external
staging (new owner needed). Fix: pin one mechanism, e.g. "the obs package
`//go:embed`s the four source files (the built-in copy; repo truth at build time);
`--repo <checkout>` (or running with cwd in a checkout, detected via
`gitToplevel`) re-stages `$DATA_DIR/obs-plugins/` and installs from there (dev
flow)". This is the briefing's own open item ("source-of-truth for plugin files");
the blueprint's decision note resolves the installer-machinery question but
leaves this half open.

Severity: Major
Affected scope: Task 2 Interfaces — Produces (line 257)
Description: Task 2 pins `--file` filtering to Go stdlib `path.Match` ("`--file`
uses `path.Match` in Go, not SQL"), but `path.Match` cannot express the recursive
glob the blueprint itself pins: T-3 and scenario `obs-logs-and-query-read-stored-
events` require `--file "src/**"` to glob-match `path`, and gryph's own help
(verified: `gryph query --help`) documents `--file "src/**/*.ts"` — recursive.
Empirically verified: `path.Match("src/**", "src/a/b.go")` == false (`*` in
`path.Match` does not cross `/`; `**` has no recursive meaning there — it is two
consecutive single-segment wildcards, so `src/**` silently behaves as `src/*`).
Only `path.Glob`/`filepath.Glob` (filesystem matchers, not string matchers) treat
`**` as multi-segment.
Rationale: Faulty-assumption + architectural-gap vectors: the pinned matcher
deterministically fails nested paths (the common case for a file audit trail), so
T-3's acceptance can only pass with shallow fixture paths — a silent
implementation escape hatch that diverges from gryph parity. Fix: pin a small
`**`-aware string matcher in `internal/obs` (segment-wise: `**` = zero or more
segments, `*`/`?`/`[]` per segment, Glob semantics) in Task 2's Interfaces, and
pin table cases including `src/**` vs `src/a/b.go` (true) and vs `src/b.go` (true);
or, if single-segment semantics are intended, re-pin T-3 and the acceptance
scenario to `src/*`.

Severity: Minor
Affected scope: Task 7 Interfaces (line 505)
Description: `obs sessions` listing column "agent" has no source: the `sessions`
table has no agent column (verified: `migrations/001_schema.sql` — id, sync_id,
project, ownership_mode, directory, title, started_at, ended_at, summary, …) and
obs ingest does not set `title`. The listing's agent must be derived from
`obs_events` (which agent — latest event? first? distinct set? one session id can
even carry multiple agents, e.g. the shared `"ephemeral"` id), and "last-active"
is not defined anywhere.
Rationale: Unclear-interface vector: the acceptance ("each session lists with
agent, title, event count, last-active") is checkable only once the derivation is
pinned. Fix: pin the derivation in Task 7, e.g. agent = agent of the session's
most recent event (comma-join distinct when >1), last-active =
`max(occurred_at)` over the session's events.

Severity: Minor
Affected scope: Task 4 Interfaces (line 339); Must-Have T-6 (line 75)
Description: `obs install all` semantics on a host-absent machine are not pinned:
create all five targets unconditionally (MkdirAll) or detect which hosts are
present and skip absent ones? Scenario `obs-live-hosts` says "obs installed for all
detected hosts", implying a detection concept no task defines, and T-6's
`install all` on a fixture HOME defines only the cursor-merge case.
Rationale: Missing-edge-case vector (low blast radius: `all` on a laptop without
kilo). Fix: pin `all` = "writes all five targets, creating parent directories; a
host's absence is not an error" (simplest, matches gryph's install surface), or
define detection (host config dir exists).

Severity: Nit
Affected scope: Threat Matrix subprocess paragraph (line 589); Global Constraints
(lines 123-125)
Description: The threat-matrix mitigation "the hard timeouts per host (Global
Constraints)" overstates the cursor case: Global Constraints pin caps for pi/omp
(30s `tool_call`) and opencode (5s `tool.execute.before`) only. Cursor's command
strings run directly under cursor's own harness with no pinned adapter cap; the
binary's latency there is bounded by the store DSN's `busy_timeout(5000)`
(verified: `sqlite.go:304-311`) plus the `gitToplevel` shell-out, which has no
deadline (verified: `main.go` `runGit` = bare `exec.Command("git", …).Output()`).
Rationale: The claim as written does not hold for one of the five hosts; the
residual risk is small (worst case ≈5s store wait; git hang is a rare,
pre-existing CLI-wide property) but the threat row should state the actual bound.
Fix: extend Global Constraints with the cursor bound ("no adapter cap; worst case
bounded by the 5s busy_timeout; git-hang residual accepted") or add a
self-bounded deadline inside `_ingest` (e.g. 5s context) which covers all hosts
at once.

### structural

Severity: Minor
Affected scope: Task 1 Interfaces — Produces (line 232); Task 2 Interfaces —
Produces (line 252)
Description: Interface inconsistency between Task 1's produced `obs.Event` and
Task 2's consuming `Ingest(ctx, st, ev Event)`: Task 2's tx upserts
`INSERT INTO sessions (id, project, directory) VALUES (?,?,?)` — the `directory`
value (the envelope's `cwd`) is not in any pinned interface, because `obs.Event`
is defined as "the row struct" of `obs_events`, which has no `directory` column.
Note v2's own auto-register precedent writes `directory = ''`
(verified: `store/sqlite.go:623` `ensureSessionTx`).
Rationale: The consuming task needs a value the producing task's interface does
not carry, and the two viable values (envelope cwd vs `''` per v2 precedent) are
different data. Fix: pin `Event` to carry the envelope's `cwd` as a non-persisted
field (or extend the `Ingest` signature) and state which value is written.

Verified (no findings): template conformance — every task carries Files /
Interfaces (Consumes-Produces-Seam-Deletion test-Adapters) / SATISFIES / Steps
with checkboxes; heading hierarchy consistent. Task ordering is topological:
every Consumes resolves to an earlier task or existing code (Task 2 ← Task 1;
Task 3 ← 1/2; Task 4/5/6 ← Task 4's install.go; Task 7 ← 2; Task 8/9 ← 1-7). All
SATISFIES names resolve to existing acceptance scenarios, and the reverse coverage
is complete (all 13 scenarios are claimed by ≥1 task; no orphan scenario, no
dangling name). Cross-references resolve: `store.go:139` is exactly
`Begin(ctx) (Txn, error)`; `run_test_helpers_test.go` exists with the
`installRunEnv`/`captureRun` fixtures described; `plugins/opencode/test/` exists
(plain-node pattern); ADRs 0008/0013/0021/0031/0032/0043/0045 all exist with the
cited semantics.

### style

Severity: Nit
Affected scope: Task 9 Step 1 (line 568)
Description: "rows present with correct agent label + action sequence" —
"correct" is on the banned validating-adjective list
(`_shared/craft/review-discipline.md`, Banned vocabulary).
Rationale: Per the style persona, banned vocabulary is a finding even where the
sentence is technically accurate. Fix: "rows present with the host's agent label
+ action sequence".
