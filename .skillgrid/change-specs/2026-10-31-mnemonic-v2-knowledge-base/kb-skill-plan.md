# KB Management Skill — Plan

**Status:** COMMITTED (spec zone — the source-citation doc backing the `009-skill.md` amendments)
**Scope:** the `kb` management skill (the behavioral layer) of the
`2026-10-31-mnemonic-v2-knowledge-base` change — the agent-facing discipline that
teaches the agent *when/why* to maintain the KB, on top of the mechanical `kb_*`
engine tools.
**References reviewed (2026-11-01):**
1. decodingai, *Build an LLM Wiki From First Principles* (2026-10-20) — the 3-phase
   skill (vanilla → multi-source → interactive), 3-md+2-scripts layout, progressive
   disclosure, deterministic-in-scripts, ≥2-source promotion, questions-as-data.
2. zosmaai/pi-llm-wiki `skills/llm-wiki/SKILL.md` — the 329-line behavioral skill,
   the direct ancestor of our `009-skill.md` (golden rules, two-stage recall,
   recall-at-start/retro-at-end, `wiki_status`, `/wiki-model`, trajectories).
3. langchain-ai/openwiki `skills/` — the resumable page-job lifecycle, the claims
   contract, the engine/agent ownership boundary, the mermaid-diagrams authoring
   sub-skill.

**Sequencing:** the KB change folder currently has **only `specs/`** (no
blueprint/tasks — it is queued behind code-index, which is queued behind memory).
This doc produces (a) the source-citation plan, (b) the spec amendments to
`specs/02-knowledge-base/009-skill.md`, and (c) the actual skill artifacts. The skill
authoring lands as **one task in the future KB blueprint** (a docs/skill deliverable,
not Go code — lower tier than the engine).

---

## 1. The current state (what `009-skill.md` already has)

The spec is solid on the **management discipline**:

- **7 golden rules** (you own the pages / index is derived / one file per thing /
  cross-reference everything / cite sources / flag contradictions / pin the why).
- **Session discipline** (recall-at-start via two-stage `kb_query`→`kb_read`;
  `kb_validate` before editing grounded code; proactive save-at-end).
- **3 workflows** (query→answer→file-if-novel; task→save; generate→validate→lint).
- **Page conventions** (OKF frontmatter, `[[type/slug]]` wikilinks, ≥2 outgoing
  links, cite-don't-copy, one atomic change per page).
- **The engine/skill split** (engine = mechanical `kb_*` tools; skill = judgment).

It explicitly defers two openwiki authoring aids (`mermaid-diagrams`,
`write-connector`) as "can add later" and the pi trajectory working-memory as
"out of scope for v2."

## 2. The 4 gaps (all additive, none touch the engine or one-way doors)

Against the three references, the current spec is missing four things. Each is
grounded in a named source and is additive to the behavioral layer.

### Gap 1 — the interactive / continual-learning layer (decodingai phase 3)

**Source:** decodingai "Interactive LLM wikis" — *"Your questions are data."*

decodingai adds a second page layer **beside** the ingest-owned pages:

- `questions/` — each asked question logged as a **slim pointer page**.
- `open-questions.md` — the running list of what the wiki **can't answer yet**
  (the wiki's own gaps, surfaced, not hidden).
- `notes/` — a second layer of pages *on top of* sources/entities/concepts; an answer
  that synthesizes ≥2 pages (or that the user asks to save) becomes a note. A
  single-concept answer stays an answer; an unanswerable question goes to
  `open-questions.md`.

Plus the **≥2-source promotion threshold** (decodingai's "graph ingestion, scaled
down"): a concept/entity earns its own page **only when ≥2 distinct sources extract
it** — "one note mentioning something is a fact about the note; two is a fact about
your knowledge." The threshold is what keeps the graph clean (cuts noise, keeps what
recurs).

**Fold into our KB:** our KB has six page-type folders (`decisions/ invariants/
workflows/ architecture/ gotchas/ glossary/`) but **no question-logging / open-question
surface** and the graduate-rule (`003-generate` §2) is per-fact, not a recurrence
threshold. Add to the skill:
- a `questions/` + `open-questions.md` surface (the "the wiki's own gaps are visible"
  habit) — a *skill* convention, not a new engine table;
- the **≥2-source promotion rule** as an explicit golden-rule companion to the
  graduate-rule (a candidate page is *promoted* only on recurrence; one mention stays
  an inline fact). This sharpens `003-generate`'s graduate-rule from "is this durable?"
  to "is this durable **and recurring**?"

> **Boundary:** this is a *behavioral* convention the skill teaches. It does **not**
> add an engine table or a `kb_*` tool — `open-questions.md` is a plain page, `notes/`
> is a page-type folder. (If the KB blueprint later wants an *engine* `kb_questions`
> index, that's a separate engine decision — flagged, not assumed.)

### Gap 2 — the claims authoring contract (openwiki)

**Source:** openwiki `integrations/openwiki/SKILL.md` — the **Claims contract**
+ the "non-negotiable boundaries" (engine vs agent ownership).

openwiki's claim model:
- A **Claim** is *one substantive, independently falsifiable* system truth. Prefer
  behavior / responsibility / architecture / relationships / flow / invariants /
  lifecycle / config / security / persistence / operations / extension seams. **Do not
  create a Claim merely because a symbol/path/parameter/return-type/inheritance exists.**
- Each claim **cites one or more repository resources** (`repo://src/auth.ts#L20-L48`),
  a bounded span, whole-file only when the whole file is the evidence.
- **Reconcile deliberately:** `stale`/`unresolved` is a *requirement to re-check current
  source*, **not** an instruction to auto-retract. Issue-free claims are retained
  automatically (don't re-submit). A stale claim gets one explicit decision:
  confirm-after-recheck / revise-with-same-id / retract-after-fixing-the-prose.
- **Ownership:** the engine owns run state, the page queue, claims validation/
  persistence, indexes, provenance. **The agent owns semantic research + the prose of
  the single assigned page.** "Never edit the engine-managed files."

**Fold into our KB:** we **already have** `kb_claims` (KB `004-index`) and the pin-check
(`006-validate`, `symbol:`/`line:`/`all_of`/`claim` checks). What the skill is **missing
is the authoring side** — how the agent *writes* and *reconciles* claims when it
authores a grounded page by hand (the "file if novel" path). Add to the skill:
- a **claims authoring contract**: one falsifiable truth per claim; cite the code via
  `code:<path>#L<a>-<b>` (our `repo://` equivalent); prefer behavioral/invariant claims
  over "this symbol exists"; when `kb_validate` reports `stale`/`unresolved`, **re-check
  the live code** (via `code_resolve` / `code_read`) and make an explicit
  confirm/revise/retract decision — never a silent re-point (this *is* `006-validate`
  §2.1's fail-closed stance, stated as an authoring rule).
- the **engine/agent ownership boundary** made explicit (the KB-INV-3 "index is derived"
  rule, generalized: the agent owns the page *prose + claims*; the engine owns
  `kb_pages`/`kb_pages_fts`/`kb_claims` rows, `index/`, and the pin-check machinery —
  the agent writes pages, it never writes the derived tables directly).

### Gap 3 — a cheap `kb_status` first-verb (pi)

**Source:** pi `wiki_status` ("Instant stats") + `/wiki-dashboard` (read-only vault
health, zero LLM calls, computed from on-disk state).

pi's discipline: the agent has a **cheap, always-first, zero-LLM** health verb it calls
*before* deciding to recall or ingest — page counts by type, stale count, last-touch,
pending queue, zero-backlink count, embedding coverage. It is the "is the vault even in
a state I should touch?" check.

**Fold into our KB:** we have `code_status` for the code-index, but the skill doesn't
name a **`kb_status`** as the *first* verb. Add to the skill:
- **`kb_status` is the first verb.** Before recall/ingest/validate, the agent calls
  `kb_status` (read-only, zero LLM): page counts by type, `stale`/`unresolved` count,
  last `kb_index`/`kb_validate` time, coverage-gap count (from `007-lint`), orphan count.
  It routes the next action (a stale page → `kb_validate`; a coverage gap → `kb_generate`;
  a thin vault → ingest). This is the read-only half of `007-lint`'s output surfaced as a
  single glance, mirroring `code_status`'s capabilities report.

> **Note for the KB blueprint:** `kb_status` may already be implied by the engine's
> `kb_*` surface; if it's not a named tool yet, this is a *candidate engine verb* to add
> when the KB is sliced (parity with `code_status`). The skill assumes it exists; the
> blueprint confirms or adds it.

### Gap 4 — the mermaid-diagrams authoring sub-skill (openwiki)

**Source:** openwiki `skills/mermaid-diagrams/SKILL.md` — a disciplined, syntax-safe
diagram authoring aid.

openwiki treats diagrams as **part of high-quality generation, not decoration**:
- **Type selection:** `sequenceDiagram` for runtime/request flows; `stateDiagram-v2`
  for lifecycles; `erDiagram` for data models; `flowchart TD` for control flow.
- **Grounded in source:** never invent participants/states/entities the code doesn't
  support; prefer a few strong diagrams over decorating every page; one-line caption
  below each.
- **Syntax safety** (the render-breakage rules): no `;`/`|` in labels; escape `<…>`;
  quote labels with punctuation in `flowchart`; never use reserved words (`note`,
  `end`, `loop`, …) as node/participant ids; alias space-y participants.

**Fold into our KB:** our `architecture/` + `workflows/` page types are *exactly* where
flows / lifecycles / data-models live. The current spec defers `mermaid-diagrams` as
"can add later." Add it as a **named authoring sub-skill** the `kb` skill references:
- a `mermaid.md` companion (or a section in `PAGES.md` for `architecture`/`workflows`
  pages) carrying the type-selection + grounded-in-source + syntax-safety rules.
- the KB engine already validates/renders page content (it's OKF + markdown); a
  mermaid fence that fails to parse degrades to a text fence — so the syntax rules
  prevent diff noise (a degraded diagram is a quality failure, openwiki's framing).

## 3. What does NOT change (the resisted pulls)

- **`write-connector` (openwiki)** stays deferred — it's about *ingest connectors*
  (Notion/YouTube/etc.), which is the KB's `002-sources` engine concern, not the
  management skill. The KB's four sources (memory / code / git / external) are
  already defined; a connector-authoring skill is a future authoring aid, not the
  management discipline.
- **pi trajectories / working-memory** stays out of scope for v2 (the skill spec
  already says so) — `wiki_capture_trajectory` / `wiki_distill_skills` /
  `wiki_recall_skill` are an opt-in extension. **But** the *distinction* is folded in
  (below) because it sharpens the proactive-save discipline without the machinery.
- **decodingai's per-origin ETL adapter scripts** (`fetch_article.py`, etc.) are the
  *engine's* ingest concern (`002-sources`), not the skill. The skill teaches the
  agent to *use* the adapters, not to write them.

## 4. The target skill artifact layout (the deliverable)

Matching decodingai's lean 3-md layout, adapted to our v2 model (no `raw/`, no
`outputs/`, code-first page types, engine-backed like pi):

```
mnemonic-v2/skills/kb/            # shipped with the engine, discovered by the runtime
├── SKILL.md            # LEAN. frontmatter + the 8 golden rules + session discipline +
│                       #   the 3 workflows + when-to-call which kb_* tool. Progressive
│                       #   disclosure: tells the agent to read CONVENTIONS/PAGES/CLAIMS/
│                       #   MERMAID only when it needs to write.
├── CONVENTIONS.md      # the data contract: the 6 page-type folders, OKF frontmatter,
│                       #   [[type/slug]] wikilinks, sources: (memory:/code:/external:),
│                       #   the ≥2-source promotion threshold, the question/notes layer.
├── PAGES.md            # one template per page type (decision/invariant/workflow/
│                       #   architecture/gotcha/glossary + question/answer/note).
├── CLAIMS.md           # the claims authoring contract (Gap 2): one falsifiable truth,
│                       #   code:<path>#L… citation, the confirm/revise/retract rule.
└── MERMAID.md          # the diagram authoring sub-skill (Gap 4): type selection +
                        #   grounded-in-source + syntax-safety rules.
```

**Frontmatter (the trigger):**

```yaml
---
name: kb
description: Maintain the project knowledge base (the LLM-wiki "brain") over the
  shared store. Call kb_status first (vault health), kb_query at task start
  (wiki-first recall), kb_validate before editing grounded code, and save durable
  decisions at task end. Engine tools do the mechanical work; this skill is the
  discipline.
---
```

The body opens with the **8 golden rules** (the existing 7 + the ≥2-source promotion
rule from Gap 1), then the session discipline (with `kb_status` as the first verb from
Gap 3), then the 3 workflows, then progressive-disclosure pointers to
CONVENTIONS/PAGES/CLAIMS/MERMAID.

## 5. The spec amendments (to `specs/02-knowledge-base/009-skill.md`) — APPLIED

| § | Amendment | Gap |
|---|---|---|
| §2 | Add **rule 8: "PROMOTE ON RECURRENCE."** A candidate concept/entity earns a page only when ≥2 distinct sources (or 2 distinct memories) extract it; one mention stays an inline fact. Sharpens the graduate-rule. | 1 |
| §3 | Add **`kb_status` as the first verb** in the session discipline (before recall/ingest/validate). | 3 |
| §3 (new sub-§) | Add the **interactive layer**: a `questions/` pointer-page convention + `open-questions.md` (the wiki's visible gaps) + the `notes/` second layer (a ≥2-page answer, or a user-requested save, becomes a note; a single-concept answer stays an answer; an unanswerable question → `open-questions.md`). | 1 |
| §5 (new sub-§) | Add the **claims authoring contract**: one falsifiable truth per claim; `code:<path>#L…` citation; prefer behavioral/invariant claims over "symbol exists"; `stale`/`unresolved` → re-check live code → explicit confirm/revise/retract (never silent re-point). | 2 |
| §5 (new sub-§) | Make the **engine/agent ownership boundary** explicit: agent owns page prose + claims; engine owns `kb_pages`/`kb_claims`/`index/` + the pin-check machinery (KB-INV-3 generalized). | 2 |
| §5 (new sub-§) | Add the **mermaid authoring sub-skill** reference (`MERMAID.md`): type selection + grounded-in-source + syntax safety, for `architecture`/`workflows` pages. | 4 |
| §6 | Update "Not adopted from openwiki": `mermaid-diagrams` is now **adopted** (as an authoring sub-skill); `write-connector` stays deferred. | 4 |
| §8 | Add open items SK-O1…O5 (whether `kb_status` is a named engine tool; whether the question/notes layer is engine-indexed; the ≥2-source threshold skill-vs-engine; mermaid engine-validated vs pass-through; the two-stage recall threshold). | 1, 3 |

## 6. How it wires to the future KB blueprint (sequencing)

The KB change folder has only `specs/`. When it is sliced (after code-index ships),
the retrieval-eval (`kb_eval`, `005-query` §8, KB-INV-5) becomes **a tracked work-unit
alongside the engine** — a frozen mem/KB case set (`.skillgrid/knowledge/_eval/cases.yaml`)
+ the `kb_eval` CLI/MCP verb that replays cases through the live fused search, reusing the
v1 ablation-harness machinery (bootstrap CI + permutation + shipped-config gate), and
gating any change to the ranking / `thin` threshold / semantic-leg fallback. It is the
measurement leg for the retrieval model (the RAG-tutorial "add evals" lesson); it is NOT
answer-quality grading (that stays `kb_validate`'s pin-check). See `005-query` §8 +
overview open items KQ-O1…O4.

The skill itself becomes **one task/ticket**:

- **Task: "Author the `kb` management skill"** — deliverable = the 5 files in §4
  under `mnemonic-v2/skills/kb/` + the `009-skill.md` amendments in §5.
- **Tier:** low (docs/skill deliverable, no Go code, no migration, no one-way door).
- **Acceptance:** the skill files exist; frontmatter triggers on the right requests;
  each of the 8 golden rules + the 3 workflows + the claims contract + the mermaid
  sub-skill is present and cross-references the correct `kb_*` tool names; the
  `009-skill.md` spec and the shipped `SKILL.md` agree (spec = the *what*, the
  artifact = the *projection*); a lint pass confirms no dangling tool references
  (every `kb_*` the skill names exists in `008-transport`).
- **Dependency:** `kb_status` (Gap 3) must be confirmed as a named `kb_*` tool in
  `008-transport` when the KB is sliced — if not, the skill task either adds it
  (small engine verb) or the skill references `kb_lint`'s read-only summary as the
  interim first-verb.

## 7. The verification questions (to answer before the blueprint slice)

1. **`kb_status`** — is it already a named tool in `008-transport`? If not, does the
   KB blueprint add it (parity with `code_status`), or does the skill use `kb_lint`'s
   read-only summary as the first-verb? *(default: add `kb_status` — it's the cheap
   always-first verb and mirrors the code-index.)*
2. **question/notes layer** — engine-indexed (`kb_questions` table) or plain pages?
   *(default: plain pages — the skill convention, no engine table, lowest risk.)*
3. **≥2-source threshold** — is it a *skill* convention (the agent applies it) or an
   *engine* graduate-rule parameter (`003-generate`)? *(default: the engine's
   graduate-rule already gates "is this durable?"; the skill adds the recurrence
   sharpening. If the KB wants it deterministic, it's a `003-generate` tuning.)*
4. **mermaid** — rendered by the KB engine, or just a markdown fence the engine passes
   through? *(default: the engine passes it through; the syntax-safety rules prevent a
   broken fence. If the engine validates mermaid, a degraded fence is a `kb_lint`
   finding.)*

## 8. Out of scope (explicitly)

- The **engine** `kb_*` tools (that's `003-generate`/`004-index`/`005-query`/
  `006-validate`/`007-lint`/`008-transport` — a separate, larger blueprint).
- **`write-connector`** (ingest connectors — the KB `002-sources` engine concern).
- **pi trajectories / working-memory** (opt-in extension, out of v2 scope).
- The **code-index** (a different function; the KB skill *references* `code_*` tools
  but doesn't own them).
- **OKF spec conformance** beyond what `001-structure` already defines.

## 9. Decision record (what "plan it properly" means here)

The plan is **complete** when:
- [x] This plan doc is committed to the KB change folder (spec zone).
- [x] `009-skill.md` carries the 8 amendments in §5 (spec zone, commits before code).
- [ ] The 5 skill artifact files in §4 are drafted (the *projection* of the spec).
- [x] The 4 verification questions in §7 have defaults (recorded, not blocking).
- [ ] The future KB blueprint has the **one skill task** (tier low) + the `kb_status`
  confirmation, so the skill authoring is a tracked, sliced unit — not an afterthought.
