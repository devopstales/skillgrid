# Deferred Work (shared across all Skillgrid skills)

The home for findings that are real but not now. A deferred-work ledger is the
append-only record of work that was deferred with a ruling — the cross-change
counterpart of the per-change review ledger's `parked` disposition (per
`_shared/craft/review-ledger.md`). The ledger makes deferred work visible,
verifiable, and closeable: an entry that says "open" but is already fixed in
the code is debt data, not debt.

**Core principle:** A deferred entry without evidence is a rumor. Evidence is a
`file:line` or a commit hash — "probably fixed" is not evidence.

**Config:** read `.skillgrid/config.yaml` before recording or sweeping —
`conventions.specs_root` (default `.skillgrid/change-specs/`) for the
per-change sink, `ticketing.enabled`/`ticketing.type` for tracker publication
of bundled work.

## The ledger

One file per project: `.skillgrid/deferred-work.md`, in the spec zone (commit
it with the change that added entries, per
`_shared/work-unit-commits/README.md`). Create it on the first entry.

```markdown
# Deferred Work

## DW-1
- source: `.skillgrid/change-specs/<change>/review-ledger.md` (R<N>-F<NNN>)
- summary: <one sentence>
- evidence: <why it is real — file:line, test name, or commit>
- status: open
- parked: <YYYY-MM-DD> by <named human> — <ruling, one line>
```

**Entry rules:**

- **Append-only.** New entries continue the `DW-<n>` numbering past the
  highest existing number. Never edit, reorder, or delete an existing entry —
  the ledger is an audit trail, like the review ledger.
- **Every entry carries all five fields.** An entry without `evidence` is a
  red flag; an entry without a ruling and a named human is not deferred, it is
  dropped.
- **One entry per finding.** Do not merge two findings into one entry, even
  when they share a file; they may close at different times.

**The two sinks of deferred work:**

| Sink | Where | Who writes |
|---|---|---|
| Review triage "defer" (real, pre-existing or out of scope) | the change's `review-ledger.md` `parked` row **and** a new `DW-<n>` entry | the review triage step (`test` → review-and-fix-loop) |
| Sweep "build" decisions the human defers again | the sweep's triage report | the sweep step, at close-out |

A `parked` review finding that is *this change's problem to fix later in the
same change* stays in the review ledger alone. It earns a `DW-<n>` entry when
it leaves the change open — i.e. at every review triage that disposition-defers
it, regardless of change boundary.

## The sweep

A **sweep** verifies every `open` entry against the current codebase and
partitions it into a machine-checkable triage plan. It runs on demand — a human
says "sweep the deferred work" or "triage the backlog" — or as a sub-step of
`skillgrid:ship` Step 2 (reflect) when open entries exist. The sweep is
**read-only against the codebase**: it edits exactly one file, the ledger, and
only the `status:` line of entries it is closing.

### Step 1 — Locate and read

Read `.skillgrid/deferred-work.md` in full. Open entries are `## DW-<n>`
blocks whose `status:` line is `open`. If the ledger is missing or has zero
open entries, the sweep is done: report "no open deferred work" and stop.

### Step 2 — Verify every open entry against the code

Ledger statuses are known-unreliable: entries are often resolved by later work
but never marked done. For EACH open entry:

1. Read its `evidence:` location in the current tree.
2. Check whether the described issue still exists — read the code, grep for
   the symptom, check `git log` for commits that touched the area since the
   entry's `parked:` date.
3. Record concrete evidence either way: a `file:line` that now handles it, or
   the commit that fixed it. "Probably fixed" is not evidence.

Use subagents for parallel verification when available (per
`_shared/rules/chain-strategy.md`); a sweep over many entries is a fan-out,
not a serial read.

### Step 3 — Partition

Classify each open entry into exactly ONE category:

| Category | Meaning | Required field |
|---|---|---|
| **resolved** | The issue no longer exists in the code | `evidence` — file:line or commit that resolved it |
| **bundle** | Buildable now: every file the fix needs exists, no future change must land first | `intent` — 2–6 sentences describing the one cohesive goal |
| **blocked** | Meaningful only after a named change lands | `blocker` — the change name verbatim |
| **skip** | Superseded, moot, or tied to a scenario the project excludes | `reason` — one line |
| **decision** | A human must choose | `question` + 2–4 `options` (each: label, effect, and `intent` when effect = build) + `recommendation` |

**Bundle rules:** group entries that share a touchpoint (same file, same
subsystem, same fix pattern) into cohesive single-goal bundles sized for one
fresh context window (the sizing rule in `skillgrid:slicing` applies — ≤3
files, ≤2 subsystems, ≤500 lines changed). A bundle that stands alone is a
one-entry bundle. Bundle names are kebab-case, ≤40 characters.

**Decision rules — conservative on human territory.** These are ALWAYS
decisions, never bundles: renegotiating a decision recorded in a change's
`briefing.md` (scope, approach, acceptance), reversing a deliberate
human-approved scope ruling, and any API-shape change that ripples to unbuilt
consumers. Each option's `effect` is one of `build` / `close` / `keep-open`.
When in doubt between bundle and decision, choose decision.

### Step 4 — Apply and report

1. For every **resolved** entry: set its `status:` line to
   `done <YYYY-MM-DD> — <evidence>` (the evidence is the reason, on the same
   line).
2. For **skip** entries: set `status:` to
   `skipped <YYYY-MM-DD> — <reason>`.
3. **Bundle**, **blocked**, and **decision** entries stay `open`.
4. Report the partition in chat — counts per category, one line per entry
   (id, category, its required field) — and, for each bundle, the `intent`.
   For each decision, present the options with the recommendation and get the
   human's choice (zero-tool turn-yield, per
   `_shared/craft/review-discipline.md`):
   - `build` → the human may start it now (route to `skillgrid:brainstorm` or
     `skillgrid:blueprint` with the `intent` as the idea) or defer it again —
     record `note: deferred again <date> — <why>` under the entry.
   - `close` → set `status:` to `done <YYYY-MM-DD> — <resolution>`.
   - `keep-open` → no status change; the ruling is recorded as a `note:` line.
5. Commit the ledger (spec zone) when any status line changed.

**The sweep never:** edits code, edits a change folder, publishes tracker
tickets, or invents an entry that is not in the ledger.

### The validation rules

A sweep is valid only when ALL hold (the report names each check's result):

1. Every `status: open` entry read in Step 1 appears in exactly one category —
   no misses, no duplicates, no invented ids.
2. Every `resolved` entry has non-empty evidence; every `blocked` a named
   blocker; every `skip` a reason.
3. Bundle names are unique, kebab-case, ≤40 characters; every bundle's
   `intent` is non-empty.
4. Every `decision` has ≥2 options with unique labels, a valid `effect`, and a
   `recommendation` naming one of the options.
5. No `status:` line of a `done`/`skipped` entry was modified by the sweep.

A sweep that fails its own validation is not a sweep — redo the partition, do
not patch the report.

## Staleness

A `DW-<n>` entry open across 3+ archived changes (count the
`change-archive/` folders dated after its `parked:` line) is **stale**. The
`skillgrid:test` debt-surfacing step and the ship-time sweep both flag stale
entries: "DW-<n> open since <date>, 3+ changes shipped. Address or expire."
Staleness is a flag, not a gate — the entry stays open until a sweep or a
human closes it.

## How to measure it

Per `_shared/craft/measurement.md`.

| | Indicator | Data source | Direction |
|---|-----------|-------------|-----------|
| Leading | % of sweep partitions that pass all 5 validation rules on the first run | the sweep's triage report (`Validation:` lines) | should rise |
| Lagging | Open `DW-` entries that are later found already-resolved at their next sweep (status was stale, code had moved on) | `.skillgrid/deferred-work.md` `resolved` entries vs. their `parked:` age | should fall |

## Common Rationalizations

| Excuse | Reality |
|--------|---------|
| "I'll just fix it when I see it again" | "When I see it again" is how findings become permanent. The ledger + sweep force the decision on a schedule, not on memory. |
| "Parking it in the review ledger is enough" | The review ledger is per-change; it dies in the archive. The deferred-work ledger is the cross-change view. A parked row without a `DW-` entry is debt the next change will never see. |
| "The entry says open, so it's open" | Entries are written at parking time by a context that no longer exists. Later commits resolve them silently. Verify against the code; the status line is a hypothesis, not a fact. |
| "I'll merge the two entries, they're the same area" | They may close at different times. Merged entries can't be partially closed. One entry per finding. |
| "This decision is obviously right, I'll just build it" | The sweep proposes, the human disposes. A bundle that quietly renegotiates a recorded human decision is scope creep wearing a ledger entry. |
| "The sweep should fix things while it's at it" | A sweep that edits code is a change. It gets a change folder, a review, a verdict. The sweep's only write is status lines on entries it verified. |

## Red Flags

- A `parked` review finding with no matching `DW-<n>` entry (debt invisible to the next change)
- A `DW-` entry without `evidence` or without a ruling + named human
- An entry closed as `resolved` with "probably fixed" or no file:line / commit
- A sweep report whose categories don't partition the open set exactly (missed or double-counted entry)
- A bundle that renegotiates a `briefing.md` decision (it is a decision, not a bundle)
- The ledger's numbering skipping or restarting (an entry was edited or deleted)
- A stale flag that is waved through without a recorded ruling

## Verification

- [ ] `.skillgrid/deferred-work.md` exists when any `DW-` entry exists; every entry has all five fields
- [ ] Every review-triage "defer" produced both a `parked` ledger row and a `DW-` entry
- [ ] A sweep verified every open entry against the code (evidence recorded either way)
- [ ] The sweep's partition passed all 5 validation rules; the report shows the checks
- [ ] Only `status:` lines (and `note:` lines) were edited by the sweep — no code, no change folders
- [ ] Stale entries (3+ archived changes) are flagged with the age and the ruling
- [ ] The ledger is committed in the spec zone with the change that touched it

## Final Rule

```
defer (review triage) → parked row + DW-<n> (evidence + ruling + human)
sweep → verify against code → resolved / bundle / blocked / skip / decision
resolved → status: done <date> — <evidence>
decision → human chooses build / close / keep-open
→ the ledger is append-only; the code is the truth; the status is a hypothesis
```
