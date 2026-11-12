# Harnesses — Portability + Freshness (R36/R38)

This file is the single source for subagent capability per harness.
Dispatching skills name the protocol, not a specific harness — no
harness-specific string (`Agent`, `invoke_subagent`, `omitClaudeMd`)
appears in `SKILL.md` or persona briefs, only here and in the adapter
files.

## Freshness rule (R36)

A harness without fresh-context subagents does NOT degrade — it STOPS.
State verbatim:

> *"If your harness has no subagent tool, stop and tell the user: a
> review in the same context is not a Goldfish review."*

The check applies at each track's first dispatch point:

- shootdown: before the six lenses spawn
- plan review: before Stage 5 is dispatched
- drift: before the first `codebase-details` reviewer dispatches

The stop is not a suggestion: the skill halts, tells the user a
same-context review is not a Goldfish review, and names the EGM rule it
is enforcing. No partial run, no silent degradation, no "we'll count
this one as a review anyway."

## Harness table (R38)

A port of EGM's `docs/harnesses.md`, scoped to the four supported
harnesses.

| Harness | Subagent tool | Read-only mechanism | Project-instruction exclusion (R37 part 3) | Partial-enforcement note |
|---------|---------------|---------------------|---------------------------------------------|--------------------------|
| **Claude Code** | `Agent` (Task); agents at `.claude-plugin/agents/<persona>.md`; `model: inherit` | `tools: [Read, Grep, Glob]` | `omitClaudeMd: true` | none — all four parts enforced; nesting depth 3, so the main context may dispatch reviewers directly |
| **OpenCode** | `subagent`; agents at `.opencode/agents/<persona>.md` | `permissions: [edit: deny, shell: deny]` | `mode: subagent` | part 3 is not enforced without the deferred Goldfish plugin (below) — a harness that cannot enforce part 3 says so in the report header; no silent pass |
| **Cursor** | Cursor's Task subagent; agents at `.cursor/agents/<persona>.md` | read-only via agent permission settings | `rules/` files are not loaded for subagents by default — verify at install | verify part 3 at install; state the result in the report header; no silent pass |
| **Pi** | native delegation (MCP + agent delegation since v0.99), detected at runtime | per the delegation configuration at runtime | per the delegation configuration at runtime | if the current session cannot spawn a fresh-context subagent, R36's stop rule applies — do not degrade |

## Contributing a new harness

The adapter is a file; the skill is the protocol. Adding a harness =
adding an adapter file (the persona briefs at the harness's agent
location + its isolation settings) + one row in the harness table
above. A fifth harness must not require changes to the dispatching
skills — each dispatch section names the protocol, and this table is the
single lookup.

## Deferral (not built in this change)

The OpenCode Goldfish plugin — `.opencode/plugins/goldfish/index.ts`
(a `context` hook stripping instruction sources, `tool.transform` denying
edit/write/shell, `session.hook("prompt")` denying web tools) — is the
spec's Open Questions deferral candidate. It is the native-enforcement
path for OpenCode: with it, part 3 moves from partial enforcement to
full (the isolation mandate in each persona brief stays as
defense-in-depth — the plugin is the enforcement, the mandate the
fallback if the plugin fails to load).
