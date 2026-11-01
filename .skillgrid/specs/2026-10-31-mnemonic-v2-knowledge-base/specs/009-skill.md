# 009 — Skill (the `kb` management skill, the behavioral layer)

**Status:** DRAFT · **Owner:** knowledge-base change
**Depends on:** `001-structure` (the page conventions it teaches), `005-query` (the
retrieval it routes), `003-generate` (the save it triggers), `006-validate` (the
pin-check it gates on), `007-lint` (the health it checks).
**Reference:** pi-llm-wiki `skills/llm-wiki/SKILL.md` (the full behavioral skill),
adapted and trimmed to the v2 model (no `raw/`, no `outputs/`, no trajectories,
code-first page types).

---

## 1. Capability

The engine's tools do the **mechanical** work (index, generate, validate, lint). The
**skill** does the **judgment** work — it is what the agent loads to know *how and when*
to maintain the KB. This is the behavioral layer: the discipline, not the tools. It is
adapted from pi-llm-wiki's `SKILL.md` (the full 329-line behavioral skill), trimmed to
the v2 model.

**The split is the point:**

| Layer | Does |
|---|---|
| **The engine** (`kb_*` tools) | the mechanical work — rebuild the index, run the graduate-rule, re-tokenize, re-check anchors. Deterministic where possible, LLM where it must be. |
| **The skill** (this spec) | the judgment work — *when* to recall, *when* to save, *how* to cross-reference, *when* to pin-check, *what* is durable enough to be a page. |

A new agent that has the tools but not the skill will index and query but will not
maintain (it will not proactively save a decision it just made, or validate a page
before editing the code it pins). The skill is what closes that loop.

**Where it lives:** the skill is a `SKILL.md` (frontmatter + body) shipped with the
engine, discovered by the agent runtime (OpenCode / pi plugins) like any other skill. It
is **not** a tool — the agent loads it into context and follows it. It references the
`kb_*` tools by name (the agent calls them per the skill's discipline).

## 2. The golden rules (pi's, adapted)

The skill opens with the rules the agent must always hold:

1. **YOU OWN THE PAGES.** Create, update, cross-reference the page-type folders
   (`decisions/ invariants/ workflows/ architecture/ gotchas/ glossary/`). The pages are
   the agent's to maintain — not a read-only artifact.
2. **INDEX IS DERIVED.** Never hand-edit `index/` or the `kb_*` tables. `kb_index` /
   `kb_reindex` rebuild them from the markdown. If the index looks wrong, re-`kb_index`,
   don't edit the rows.
3. **ONE FILE PER THING.** Each decision / invariant / workflow gets its own `.md`.
   Sub-topics are separate pages linked by wikilinks, not sections bolted onto an
   unrelated page.
4. **CROSS-REFERENCE EVERYTHING.** Every page needs ≥2 outgoing wikilinks. A page that
   links to nothing is an orphan (lint will flag it); a page that is never linked is an
   island.
5. **CITE SOURCES.** Every grounded claim cites a `memory:<sync_id>` / `code:<path>#L…`
   / `external:<name>+version+source_url` in `sources:`. No provenance, no grounded page (lint
   errors on a missing `sources:`).
6. **FLAG CONTRADICTIONS.** When sources disagree, document both sides (a
   `## Contradiction` section) and let lint surface it. Do not silently pick one.
7. **PIN THE WHY.** Grounded pages (decision/invariant/workflow) carry `anchors:`.
   Run `kb_validate` before relying on them. If a page is stale, fix the *knowledge*,
   not just the code.

## 3. The session discipline (recall-at-start / save-at-end, adapted to wiki-first)

The skill encodes the **when**. Adapted from pi's recall-at-start / retro-at-end,
pointed at the wiki-first retrieval:

- **At task start** — `kb_query` with task-specific terms (the wiki-first retrieval).
  For large vaults, use **two-stage links-first recall**:
  - **Stage 1:** `kb_query` returns ranked links + 1-line snippets (protects context —
    the agent does not pull 40 page bodies into context).
  - **Stage 2:** `kb_read` the few that actually matter.
  - **Do not trust the snippet as the whole page.** Open it (`kb_read`) before relying
    on it. A snippet is a teaser for ranking, not the content.
- **Before editing grounded code** — `kb_validate` the relevant pages (the pin-check).
  If a page is `stale` / `unresolved`, the code and the documented why disagree — **fix the
  knowledge** (update the page's `anchors:`/`claims:` or its body) *and/or* the code, so
  they agree again. Do not edit the code and leave the page stale.
- **At task end** — if the task produced a durable decision / invariant / gotcha,
  **update the KB proactively** (don't wait to be asked). One atomic change per page
  (a single `kb_generate`-driven edit, not a stream of patches). This is the "retro-at-
  end" — the session's durable learnings become pages.

> **The proactive save is the load-bearing habit.** The KB stays fresh because the
> agent *saves into it* at task end, not because a cron job batch-processes. The
> `SaveEvent` (eager-per-save) handles the *memory → KB* handoff automatically, but the
> *agent's own* durable decisions (the ones it makes in its head, not the ones it
> `mem_save`s) are saved by the agent following this discipline.

## 4. The workflows (pi's, adapted)

The skill gives the agent named workflows for the common flows:

- **Query → Answer → (File if novel).** `kb_query` → `kb_read` the relevant pages →
  synthesize an answer *with citations* (the page's `sources:`) → if the answer is novel
  and durable, **create a page** for it (the "file if novel" step — a good answer that
  is not yet a page becomes one).
- **Task → Save.** Complete a meaningful task → identify the durable decision /
  invariant / gotcha it produced → update the relevant KB page(s) (one atomic change
  per page). This is the retro-at-end made concrete.
- **Generate → Validate → Lint.** `kb_generate` (the write) → `kb_validate` (the
  pin-check: does the new/updated page still match the code?) → `kb_lint` (the health:
  orphans, broken links, coverage). This is the "I just changed the KB, prove it is
  sound" loop. Run it after any manual page edit too.

## 5. Page conventions (the skill teaches the format)

The skill carries the page-format conventions (so the agent authoring a page by hand —
the "file if novel" step — produces a conforming page):

- **Frontmatter is mandatory** and conforms to `001-structure` §5: `title`, `type`
  (one of the six, canonical case), `project`, and `sources:` (non-empty for a grounded
  page). Grounded pages add `anchors:` + `claims:`.
- **Wikilinks are `[[type/slug]]`**, folder-qualified, relative to the vault root.
  Cross-reference ≥2 per page.
- **OKF sections** (`## Why`, `## Where`, etc.) — the body is structured, not a wall of
  prose. A grounded page's `## Why` is the pinned why; `## Where` points at the code
  (`code:<path>#L…` + the relevant wikilinks).
- **One atomic change per page, through the sanctioned path.** The agent creates/edits a
  page either (a) via `kb_generate` (the event-driven or proactive path — the agent's
  `mem_save` → the graduate-rule → synthesis), or (b) by writing the markdown directly
  (the "file if novel" step, when the answer is novel and not yet a fact). Direct page
  authoring is a **sanctioned agent write** into the page folders (`001-structure` §3) —
  it is one coherent edit (a new page, or a new fact integrated, or an anchor updated),
  not a partial stream. After **either** path the page is re-indexed (`kb_index`) and,
  if grounded, re-validated (`kb_validate`). Direct authoring is *not* a write into the
  derived `index/` layer — that stays engine-only (KB-INV-3).
- **Cite, don't copy.** A page cites its sources (`sources:`); it does not paste the
  source's body in. The body is the *synthesis* (the why, the map, the contract), not a
  copy of the fact.

## 6. What the skill is NOT

- **Not a tool.** It is a document the agent loads and follows. It does not run; the
  `kb_*` tools run. The skill *references* the tools by name.
- **Not the engine's spec.** The skill teaches the *discipline* (when/why); the other
  specs (`003-generate`, `006-validate`, `007-lint`) define the *mechanics* (how). If
  the skill and a spec disagree on a mechanic, the spec wins (the skill is the
  behavioral projection of the specs).
- **Not adopted from pi (explicitly):** the agent-trajectory / working-memory tools
  (opt-in in pi, off-by-default — out of scope for v2), the personal + company layered
  vaults (v2 is single-project), `wiki_watch` cron scheduling (the `SaveEvent` already
  drives eager generation — no separate watch loop).
- **Not adopted from openwiki `skills/`:** `mermaid-diagrams` and `write-connector` are
  *authoring aids*, not the management layer. v2 can add them later if page authoring
  needs them; they are not part of the management discipline.

## 7. Invariants

INV-7 (the agent curates — the skill is the agent's curation protocol), INV-1 (citations
are non-negotiable — rule 5), KB-INV-2 (grounded pages are grounded — rule 7), KB-INV-3
(index is derived — rule 2).

## 8. Open items

None. The golden rules, the session discipline, the workflows, and the page conventions
are fixed. The one carried tunable is the two-stage recall threshold (when a vault is
"large enough" to require stage 1 before stage 2) — defaulted to "always use stage 1
(`kb_query`) before `kb_read`" (the cheap, safe default), refined in the review.
