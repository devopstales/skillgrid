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

8. **PROMOTE ON RECURRENCE.** A candidate concept/invariant earns its *own* page only
   when **>=2 distinct sources** (or >=2 distinct memories) extract it. One mention is a
   fact *about the note*; two is a fact *about the knowledge* (the decodingai
   "graph-ingestion, scaled down" threshold). A one-off mention stays an inline fact in
   the page that noticed it, not a new page. This is the **recurrence sharpening** of the
   graduate-rule (`003-generate` section 2): the graduate-rule asks "is this durable?",
   this rule adds "is it *recurring*?" Together they keep the graph clean (cut noise,
   keep what recurs). The threshold is tunable (`CONVENTIONS.md`), default 2.

## 3. The session discipline (recall-at-start / save-at-end, adapted to wiki-first)

The skill encodes the **when**. Adapted from pi's recall-at-start / retro-at-end,
pointed at the wiki-first retrieval:

- **First, always — `kb_status` (the cheap first-verb).** Before any
  recall/ingest/validate, the agent calls `kb_status` (read-only, zero LLM — the
  pi `wiki_status` / `/wiki-dashboard` pattern): page counts by type,
  `stale`/`unresolved` count, last `kb_index`/`kb_validate` time, coverage-gap count
  (from `kb_lint`'s read-only summary), orphan count, and the `kb_events` activity
  feed tail. It **routes the next action**: a stale page → `kb_validate`; a
  coverage gap → `kb_generate`; a thin/empty vault → ingest. This is the read-only
  half of `007-lint` surfaced as a single glance — parity with the code-index's
  `code_status`. *(If `kb_status` is not yet a named `kb_*` tool when the KB is
  sliced, the skill references `kb_lint`'s read-only summary as the interim
  first-verb — KI note, `009-skill` open items.)*
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

### 3.1 The interactive layer (questions are data — decodingai phase 3)

The KB is not only *fed* by ingest; the agent's **questions are data** — they show what
is not yet clear, and they leave a breadcrumb trail of the knowledge's own gaps. The
skill maintains three lightweight surfaces **beside** the ingest-owned page folders
(`001-structure`):

- **`questions/`** — each asked question is logged as a **slim pointer page** (the query
  + which pages answered it + whether it was answerable). Cheap, append-mostly.
- **`open-questions.md`** — the running list of what the KB **cannot answer yet** (the
  wiki's *own* gaps, surfaced, not hidden). An unanswerable question lands here. This is
  the "the brain knows what it doesn't know" habit — the coverage-gap lint (`007-lint`)
  surfaces the *structural* gaps; `open-questions.md` surfaces the *query* gaps.
- **`notes/`** — a second page layer *on top of* the page-type folders. An answer that
  **synthesizes >=2 pages** (or that the user asks to save) becomes a **note**; a
  single-concept answer **stays an answer** (not a note — creating a note out of every
  answer makes the wiki useless). A repo/code question the `architecture`/`workflows`
  pages can't answer makes the agent read the code and write a **code note**, which
  re-enters the graduate-rule and can push a concept over the >=2 threshold (rule 8).

> **This is a *skill* convention, not a new engine table.** `questions/`,
> `open-questions.md`, and `notes/` are plain markdown the agent writes into the vault
> (sanctioned direct authoring, section 5); the index picks them up via the usual
> `kb_index`. If the KB later wants an *engine* `kb_questions` index (to surface
> open-questions in `kb_status`), that is a separate engine decision — flagged here, not
> assumed. The promotion of a note/concept over the >=2 threshold is the **recurrence**
> rule (golden rule 8) applied to the interactive layer.

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

### 5.1 The claims authoring contract (the authoring side of the pin-check)

The skill teaches the agent **how to author + reconcile claims** when it writes a
grounded page by hand (the "file if novel" path). The engine (`kb_validate`,
`006-validate`) *checks* claims; this is the *writing* discipline (the openwiki Claims
contract, adapted):

- **One falsifiable truth per claim.** A claim is *one substantive, independently
  checkable* system truth. Prefer **behavior / responsibility / architecture /
  relationships / flow / invariants / lifecycle / failure / config / security** claims.
  **Do not create a claim merely because a symbol/path/parameter/return-type exists** —
  "this function exists" is not a claim; "this function retries 3x then escalates" is.
- **Cite the code span, not the file.** Every claim cites `code:<path>#L<a>-<b>`
  (the KB's `repo://` equivalent) — a **bounded span**, whole-file only when the whole
  file is genuinely the evidence. This is what `kb_validate`'s `symbol:` / `line:` +
  version-token checks re-read against live code.
- **Reconcile deliberately — never a silent re-point.** When `kb_validate` reports a
  claim `stale` or `unresolved`, that is a **requirement to re-check the live code**
  (via `code_resolve` / `code_read`), **not** an instruction to auto-retract. The agent
  makes one explicit decision per stale claim: **confirm** (re-checked, still true),
  **revise** (change the statement/anchor, keep the claim's identity), or **retract**
  (the truth no longer holds — fix the *prose*, then drop the claim). This is
  `006-validate` section 2.1's fail-closed stance, stated as an authoring rule: an
  ambiguous re-point is `unresolved`, never a guess.
- **Reconcile the *page*, not just the claim.** A stale claim usually means the page's
  *why* moved too — fix the body and the claim together (one atomic edit, section 5).

### 5.2 The engine/agent ownership boundary (KB-INV-3, generalized)

The split is total and explicit (the openwiki "non-negotiable boundaries"):

- **The agent owns** the page **prose + the claims** it authors (the page-type folders,
  the `notes/`/`questions/` layer, the `sources:`/`anchors:`/`claims:` frontmatter).
- **The engine owns** the **derived** state: the `kb_pages` / `kb_pages_fts` /
  `kb_claims` / `kb_events` **rows**, `index/`, the `kb_fts_pending` ledger, and the
  pin-check machinery. The agent **writes pages; it never writes the derived tables
  directly.** If the index looks wrong, re-`kb_index` — don't edit the rows (golden
  rule 2). This is KB-INV-3 generalized from "index is derived" to "everything the
  engine rebuilds is engine-owned."

### 5.3 The mermaid authoring sub-skill (`architecture` / `workflows` pages)

Diagrams are **part of high-quality generation, not decoration** (the openwiki
`mermaid-diagrams` skill, adopted here). For `architecture/` + `workflows/` pages, the
agent embeds a ` ```mermaid ` fence where a flow / lifecycle / data-model is clearer
visually than as prose:

- **Type selection:** `sequenceDiagram` for runtime/request flows; `stateDiagram-v2`
  for lifecycles/state machines; `erDiagram` for the data model; `flowchart TD` for
  control flow.
- **Grounded in source.** Never invent a participant/state/entity the code doesn't
  support (verify via `code_resolve` / `code_read`). Prefer **a few strong diagrams**
  over decorating every page; one accurate diagram on the page that needs it beats a
  diagram forced onto every page. A one-line caption directly below each.
- **Syntax safety** (the render-breakage rules — a fence that fails to parse degrades
  to a text fence, a quality failure): no `;`/`|` in labels; escape `<…>` (`returns
  Promise of User`, not `returns Promise<User>`); quote labels with punctuation in
  `flowchart` (`A["calls foo(bar)"]`); never use a reserved word (`note`, `end`,
  `loop`, `alt`, `opt`, `class`, `state`, …) as a node/participant id (alias a
  space-y participant: `participant AS as Auth Service`).

> **The full rules live in `MERMAID.md`** (a companion the skill references on demand —
> progressive disclosure). This subsection is the *when* (which pages, which type);
> `MERMAID.md` is the *how* (the syntax-safety checklist).

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
- **Adopted from openwiki `skills/`:** `mermaid-diagrams` **is adopted** (section 5.3 +
  `MERMAID.md`) — it is an *authoring sub-skill* the management skill references on
  demand for `architecture`/`workflows` pages, not a separate management layer.
- **Not adopted from openwiki `skills/`:** `write-connector` is a *deferred authoring
  aid* — it is about **ingest connectors** (Notion/YouTube/etc.), which is the KB's
  `002-sources` *engine* concern, not the management skill. The KB's four sources
  (memory / code / git / external) are already defined; a connector-authoring skill is a
  future authoring aid, not the management discipline.

## 7. Invariants

INV-7 (the agent curates — the skill is the agent's curation protocol), INV-1 (citations
are non-negotiable — rule 5), KB-INV-2 (grounded pages are grounded — rule 7), KB-INV-3
(index is derived — rule 2).

## 8. Open items

The golden rules, the session discipline, the workflows, and the page conventions are
fixed. Open items (defaults recorded, not blocking — resolved at the KB slice):

- **SK-O1 (the `kb_status` first-verb):** is `kb_status` a named `kb_*` engine tool, or
  does the skill use `kb_lint`'s read-only summary as the interim first-verb?
  **Default: add `kb_status`** (parity with the code-index's `code_status` — it is the
  cheap always-first verb). If not added, the skill's `kb_status` references `kb_lint`.
- **SK-O2 (the question/notes layer, engine-indexed or not):** are `questions/` /
  `notes/` / `open-questions.md` plain vault pages (the default) or engine-indexed
  (a `kb_questions` table surfaced in `kb_status`)? **Default: plain pages** (no engine
  table, lowest risk; the index picks them up via the usual `kb_index`).
- **SK-O3 (the >=2 promotion threshold, skill or engine):** is the recurrence rule a
  *skill* convention the agent applies (default) or a *deterministic* `003-generate`
  graduate-rule parameter? **Default: skill convention** — the engine's graduate-rule
  already gates "is this durable?"; the skill adds the "is it *recurring*?" sharpening.
  If the KB wants it deterministic, it is a `003-generate` tuning.
- **SK-O4 (mermaid, engine-validated or pass-through):** does the KB engine validate
  mermaid fences (a broken fence is a `kb_lint` finding) or pass them through (the
  syntax-safety rules prevent a broken fence)? **Default: pass-through** (the engine
  renders markdown; the sub-skill's syntax rules are the guard).
- **SK-O5 (two-stage recall threshold):** when a vault is "large enough" to require
  stage 1 before stage 2 — **defaulted to "always use stage 1 (`kb_query`) before
  `kb_read`"** (the cheap, safe default), refined in the review.

---

## 6. The `kb` management skill (TICKET-09 amendment)

The `kb` management skill ships as five files plus this amendment:

- `SKILL.md` — modes (read/maintain), commands, rules.
- `CONVENTIONS.md` — conventions.
- `PAGES.md` — page-format conventions.
- `CLAIMS.md` — claims authoring contract.
- `MERMAID.md` — diagram conventions.

Amendment: the engine verbs (`kb bootstrap`/`index`/`query`/`validate`/`lint`/`status`/`cite-source`/`eval`) are now implemented in `mnemonic-v2/internal/kb`; the skill text above is the authoring-side discipline and remains authoritative for how agents write/validate pages.
