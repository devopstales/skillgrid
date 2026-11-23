# v1 = memory core only (deferred list is named)

---
status: accepted
date: 2026-11-23
supersedes: none
---

## Context and Problem Statement

The predecessor
([[sources/skillgrid-mnemonic-v2]]) bundles memory, a code index, a web
cache, projectwiki generation, `team_*` tools, SDD coupling, and a Postgres
cloud tier. A greenfield v1 that ships all of them inherits all of that
weight before the memory core is proven. The programmer locked (2026-11-23
interview): **memory only** for v1, with five explicit deferrals.

## Considered Options

- **Memory + code index (+ web cache)**: matches the installed
  `skillgrid:mnemonic` skill's surface, but tree-sitter extraction, chunk/
  symbol/edge graphs, and freshness tracking are a second project glued to
  the first.
- **Memory core only**: sessions, observations, FTS5, in-file vectors, hybrid
  recall, trails, ops — a complete, useful memory in its own right; the
  deferred domains slot in via additive migrations (the data model pre-
  validates them: `project` column, `mem_vec` naming).
- **Everything-minus-Postgres**: least cut, most carry-over risk.

## Decision Outcome

Chosen option: **memory core only** for v1.0. Deferred, explicitly named
(each is a future tool or layer, not a dropped idea):

1. `serve` HTTP API + dashboard (v1.1+).
2. projectwiki generation (skill-level concern, not memory's).
3. `team_*` MCP tools.
4. LLM curator (importance scoring, dedup adjudication, consolidation — the
   `llm` seam is kept, the paths are deferred).
5. Importer for old `skillgrid/mnemonic` stores.
6. (Scope decision, same interview) the `code` and `web` tools themselves —
   future tools under the same binary ([[decisions/0010-multi-tool-
   binary]]).

### Consequences

- Good: v1 is shippable and useful alone (persistence + semantic recall);
  every deferred item has a named home, so "deferred" is trackable, not
  "forgotten".
- Good: the data model is pre-validated for the deferred domains without
  paying for them now.
- Bad: the installed `mnemonic` skill's code-ladder steps dangle until the
  `code` tool lands → the skill flip ships **with the `code` tool**, not
  with v1.0 (recorded in [[syntheses/mnemonic-core]]).
- Bad: two memory systems coexist during v1 (this one + the old
  `skillgrid` one) until the skill flip + importer.

## Not adopted

- **Memory + code index in v1** — the code graph is its own build; shipping
  it half-done to match the old skill's surface would make v1 bigger to be
  *less* new.
- **In-place evolution of the old package** — the old package is dialect-
  aware and spec-bound; trimming converges on this greenfield with extra
  archaeology (approaches table in the briefing).

## Related

- Supersedes: none.
- Depends on: [[decisions/0010-multi-tool-binary]] (where the deferred
  tools will live), [[sources/skillgrid-mnemonic-v2]] (what is being cut and
  why).
- Consumed by: [[syntheses/mnemonic-core]], [[entities/mnemonic]] (the v1
  scope line).
