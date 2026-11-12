## Goldfish Shootdown

The Goldfish stress test for the New Project / New Function paths: six
fresh-context lenses attack the anchored hypothesis after the codebase
feasibility check passes and before the ADR manifest is written. This file is
the dispatch protocol — where the shootdown sits, how the lenses spawn, and how
the findings are dispositioned. The tone rules it enforces live in
`_shared/craft/review-discipline.md` (cited, not restated).

## Dispatch point (R2)

Runs on the **New Project** and **New Function** paths only, **after the
codebase feasibility check passes and before the ADR manifest is written**.
The brainstorming flow is interview → approaches → design → feasibility →
ADR manifest → briefing; the shootdown sits between feasibility and the ADR
manifest, so the manifest is written against a shootdowned design.

The **Bounded** and **Prototype** paths skip it — their path descriptions do
not name the shootdown. Path ratchet unchanged: if a bounded task upgrades to
New Function mid-task, the shootdown becomes mandatory at the new path.

## Freshness check (R36)

Before the six lenses spawn, check the harness:

> *If your harness has no subagent tool, stop and tell the user: a review in
> the same context is not a Goldfish review.*

This is a STOP, not a degrade. The skill halts and names the EGM rule it is
enforcing — no partial run, no silent degradation, no "we'll count this one as
a review anyway."

## The six lenses (R1)

All six are dispatched **in parallel** (not sequentially): one freshly spawned
`reviewer` subagent (fallback `general`) per lens brief, all under
`_shared/elephant-goldfish/`:

- `_shared/elephant-goldfish/lens-contrarian-yagni.md`
- `_shared/elephant-goldfish/lens-cost-economics.md`
- `_shared/elephant-goldfish/lens-ux-ergonomics.md`
- `_shared/elephant-goldfish/lens-migration-compat.md`
- `_shared/elephant-goldfish/lens-observability-sre.md`
- `_shared/elephant-goldfish/lens-tech-debt.md`

Dispatch prompt = lens brief path + the anchored hypothesis + the explicit
reading list (the relevant existing code/files) + the R37 four-part isolation
contract re-asserted verbatim (below) + `Mode: BRAINSTORM`. Each subagent is
given ONLY the anchored hypothesis + relevant code — no conversation history,
no author narrative. A review run in the author's own context MUST NOT count
toward R1.

The R37 four-part isolation contract, re-asserted verbatim in the dispatch
prompt:

1. **Fresh context** — no conversation history, no author narrative, no
   parent deliberation.
2. **Read-only** — no file edits, no shell, no writes. A reviewer that edits
   the target has not reviewed; its findings are discarded.
3. **No project instructions** — the harness's per-project instruction files
   (Claude Code: `CLAUDE.md` → `omitClaudeMd: true`; OpenCode: project
   instructions block excluded; Cursor: `rules` files not loaded; Pi:
   `AGENTS.md` not injected) must not reach the reviewer. The reviewer sees
   ONLY the persona brief, the target file path(s), the explicit reading
   list, and the `Mode: BRAINSTORM|REVIEW` flag.
4. **No network, no MCP** — the reviewer cannot open URLs, fetch web pages,
   or call MCP tools. Documentation the review needs is saved as a file in
   the workspace and passed by path.

## Error handling

A reviewer that times out or returns no report: retry once, then record
`not run` — never zero findings. A lens that is `not run` caps its
contribution at "not verified," not "clean."

## Synthesis + finding gate (R3)

After the six lenses report, the main agent (the Elephant) synthesizes
inline: dedupes overlapping findings, groups by severity (🔴 Blocker /
🟡 Major / 🟢 Nit).

The user is shown the 🔴 Blockers and disposes of each:

- **resolved** — the design is revised; loop back to the feasibility check.
- **accepted-with-note** — recorded in the briefing's `## Shootdown Findings`
  as a known risk.
- **downgraded-to-non-goal** — moved to the spec's Out of Scope.

🟡/🟢 are recorded but do not block. The spec is NOT written until all
🔴 Blockers are dispositioned — the dispositions land in the briefing
template's `## Shootdown Findings` section. The synthesis tone follows
`_shared/craft/review-discipline.md` (cited, not restated).

## Consolidated chat response

The synthesis turn ends with:

- the severity table (normalized vocabulary at the record boundary),
- every 🔴 finding with its disposition options,
- the path where the findings will land.

A turn that ends with only a file path does not satisfy this.
