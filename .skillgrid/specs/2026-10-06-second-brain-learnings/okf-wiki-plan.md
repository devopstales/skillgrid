# Proposal: `.skillgrid` as an OKF v0.2 llm-wiki (decisions / invariants / workflows split)

> **STATUS:** `draft` (2026-10-10; restructure promoted to goal 2026-10-10)
> **Scope:** The RESTRUCTURE portion is a GOAL of this change (briefing R6) —
> pending its spec-zone delta (blueprint task + acceptance Gate + sliced
> ticket) and a re-gate; it does not execute before that lands. The SKILLS
> portion (`knowledge-authoring/-retrieval/-lint/-diagrams`) remains a
> follow-up proposal. The 5 frozen tickets (TICKET-01..05, R1–R5 ↔ blueprint
> Tasks 1–5 ↔ acceptance G1–G7) are NOT modified by this file.
> **Relation to findings.md:** this proposal revisits the `## Research: repocodex`
> "Explicitly NOT borrowed — storage layout" line. The mechanisms (anchors,
> retrieval, ratchet, shadow posture) stay as recorded there; this file proposes
> adopting the *layout + format* as well, for `.skillgrid/` only.

## Q&A decisions (user-confirmed 2026-10-10)

1. **Format = OKF v0.2** (not "OFK" — that was a typo). Google Open Knowledge
   Format v0.2, the same bundle format openwiki emits and repocodex stores at
   `.context/`.
2. **SoT = hybrid.** `.skillgrid/` OKF bundle is the durable human-readable
   knowledge source of truth; Mnemonic SQLite is the retrieval index (FTS +
   vectors over the same files). Rationale: most `.skillgrid/` content has no
   reason to live *in* SQLite rows.
3. **Restructure scope = DISSOLVE `artifacts/`, rename `archive/`.**
   Per user decision 2026-10-10 (supersedes the earlier "`artifacts/`-split"
   scope): no new folders go inside `artifacts/` — `artifacts/` is dissolved
   and replaced by top-level knowledge folders; `archive/` is renamed to
   `xarchive/`. `specs/` stays where it is.
4. **Pins = borrow the pattern, no new dependency.** No `repocodex` pip package,
   no Python/`rg` engine in this repo. Reimplement the cheap halves natively
   (Node scripts like the existing drift guards + skill/hook discipline).

## Why (problem / intent)

- `artifacts/` is a flat mixed zone: PRD, two glossaries, 36 ADR files,
  constraints, research, roadmaps share one directory with only numeric prefixes
  (`00-`, `01-`, `04-`, `06-`) as structure.
- Current ADR frontmatter (`status`/`supersedes`/`date`) is not machine-checkable
  for freshness: an ADR is stale when its anchors break, not N days after writing
  (cf. findings.md repocodex item 1 — freshness as attestation, not TTL).
- Mnemonic indexes `.md` as opaque chunks; there is no type/tag/status-aware
  recall ("which decisions pin these files") and no deterministic catalog
  (`index.md` / reverse index).
- Skills update knowledge ad hoc; no same-edit rule keeps catalogs in sync.

## Target layout

```
.skillgrid/
  ASSUMPTIONS.md              # keeps 4 tiers; LOCKED points at decisions/ (ADR-0019)
  ARCHITECTURE.md             # gains OKF frontmatter, type: Reference
  config.yaml / state.yaml    # EXCLUDED from OKF conversion (machine files)
  index.md                    # OKF root catalog (okf_version: "0.2"), one section per folder (was artifacts/README.md)
  log.md                      # chronological writes, ## YYYY-MM-DD groups (repocodex journal pattern)
  reverse-index.md            # GENERATED, committed (path → concept map)
  decisions/                  # was artifacts/04-adr-*.md → type: TechnicalDecision (+ per-dir index.md)
  invariants/                 # split from artifacts/05-locked-constraints.md → type: InvariantContract + claims[]
  workflows/                  # SDD pipeline + per-domain flows → type: BusinessWorkflow
  glossary/                   # was artifacts/01-, 02-terms → type: Reference
  reference/                  # was artifacts/00-prd, 06-*, 07/08/09-* → type: Reference
  specs/YYYY-MM-DD-<topic>/   # ACTIVE: unchanged location; pages gain OKF types
  xarchive/YYYY-MM-DD-<topic>/# CLOSED (renamed from archive/): status: deprecated + successor links, immutable
```

`artifacts/` is DISSOLVED (nothing new goes inside it; its contents move out
and the directory is removed). `specs/`, `prototypes/`, `sdd/`,
`brainstorm/` keep their locations. The rename `archive/` → `xarchive/`
changes the ship target — every in-repo reference to either path must be
swept (see migration map).

**Conventions adopted (SeedX lesson, 2026-10-11):** the root `index.md`
carries a reader-class paragraph (concepts + catalog for agent and human;
`sdd/` scratch for the agent only; `state.yaml` for machines); `_` marks
internal-only for any future folder (no new mixed folders); index entries
are paths + one-line statuses, never body content (enforced by `okf-lint.mjs`).
A running human run-log per change (`_run/` lesson) stays a follow-up —
`report.md` at close plus `session_events` live cover it until a visualizer
need forces the issue.

Identity folders mirror repocodex (`decisions/`, `invariants/`, `workflows/`);
`type` is author intent (orthogonal — one change may write more than one type).

## OKF profile (minimal, SPEC §4/§5 + repocodex extension)

```yaml
---
type: TechnicalDecision | InvariantContract | BusinessWorkflow | Reference | Briefing | Blueprint | Tasks | Report
title: <display name>
description: <one line, feeds index.md>
status: draft | stable | deprecated
generated: { by: human:<id> | skillgrid/<skill-version>, at: <ISO8601Z> }
verified: { by: human:<id>, at: <ISO8601Z> }   # optional doc-review; NOT the pin verdict
sources: [{ id, resource, title }]              # optional provenance
verification: { engine: ripgrep, anchors: [{ path, all_of: [...] }] }  # only when pinning live text
claims: [{ subject, literal }]                  # REQUIRED for InvariantContract
---
```

- Markdown links between concepts are the graph (OKF §6); `index.md`/`log.md`
  are reserved, never concepts (§3.1).
- Unanchored pages are valid knowledge (reports, retrospectives) — they load,
  they just don't arm the pin check (findings.md repocodex item 5).
- `verified` records definition review against `sources`; liveness
  (`LIVE`/`DRIFT`/`CLAIM_BROKEN`) is a runtime verdict, not a field
  (repocodex `memory.md`: "verified is not a gate stamp").

## Migration map (IRON-RULE safe: `git mv`, never delete)

| From | To | Notes |
|---|---|---|
| `artifacts/04-adr-*.md` (36 files) | `decisions/04-adr-*.md` | `git mv` (history preserved). Add OKF keys (keep `status/supersedes/date`, add `type/title/description/generated`); update every `ASSUMPTIONS.md` LOCKED row path + `index.md` + `03-adr-index.md` successor; keep `0014/0015-removed.md` rows frozen |
| `artifacts/05-locked-constraints.md` (18 bullets) | `invariants/<slug>.md` (one per constraint) + `invariants/index.md` catalog | Each invariant gets `claims[]` where a literal exists (e.g. `go 1.22`); un-literal constraints are valid unanchored knowledge (no pin) |
| SDD pipeline (sdd-structure.md §Phase Order) + per-domain flows | `workflows/sdd-pipeline.md`, `workflows/<domain>.md` | One anchor per participating site; thin ordering/boundaries, not a dump |
| `artifacts/01/02-*-terms.md`, `00-prd.md`, `06/07/08/09-*.md` | `glossary/`, `reference/` | `type: Reference`; split only pages >200 lines (llm-wiki page-size rule) |
| `artifacts/README.md` | `.skillgrid/index.md` (root catalog) | Was the topic index; becomes the OKF root (`okf_version: "0.2"`) |
| `artifacts/03-adr-index.md` | folded into `decisions/index.md` | Per-dir catalog replaces the flat-file index |
| `artifacts/` (directory itself) | REMOVED | Dissolved after all contents move out; verify empty before `git rm` |
| `archive/` (whole dir, all closed changes) | `xarchive/` | `git mv archive xarchive` (history preserved); closed changes stay immutable |
| `specs/*/briefing\|blueprint\|tasks\|report\|review.md`, `ASSUMPTIONS.md`, `ARCHITECTURE.md`, `WINDOWS.md` | in-place OKF frontmatter | `specs/` = `status: draft/stable`; `xarchive/` = `status: deprecated` + link to successor; per-change `adr.md` stays a pointer manifest |
| Every in-repo reference to `artifacts/` or `archive/` | new paths | Reference sweep (mechanical, reviewed): `config.yaml conventions.artifacts`, `_shared/rules/sdd-structure.md`, all skills citing `artifacts/…`/`specs/…→archive/`, `ship`/`reflect`/`resume`/`qa` scripts + hooks, `ARCHITECTURE.md` §9 state-zone diagram, dashboard `/plans` + docs endpoints if they hardcode paths |

## Indexing (two layers, no new dep)

1. **Deterministic layer (new, Node, mirrors existing guards):**
   `scripts/okf-lint.mjs` + `scripts/okf-reverse-index.mjs` — frontmatter/`type`
   validation, index completeness (every concept in `index.md`), broken-link
   check, orphan report, `claims`-requires-`verification` check, reverse-index
   regen + sync check. Lands advisory-first (repocodex `shadow` posture): WARN
   in `qa` report `## OKF Lint` section; blocking only by a later user OK.
   **Telemetry:** `okf-lint.mjs` appends a JSONL event per run to
   `.skillgrid/okf-metrics.jsonl` (gitignored; `OKF_METRICS_SINK` override).
   Event: `{"event":"lint","at":<iso>,"drift":<n>,"findings":[...],"false_drift_rate":<f>}`.
   `false_drift_rate` = prior-run findings absent this run / prior total
   (repocodex `metrics.jsonl` pattern). This is the measured data trail for
   promoting the gate from shadow (WARNING) to ratchet (blocking) — the
   promotion decision is made on numbers, not gut feeling.
2. **Retrieval layer (existing Mnemonic):** confirm `.skillgrid/**/*.md` is
   indexed (`config.d/indexing.yaml` excludes must not drop it;
   `ARCHITECTURE.md` §6.3 claims `md` is included — verify, don't assume),
   then add OKF-aware recall: filter by `type/tags/status`, one link-hop of
   titles (repocodex `context` pattern), staged reads (index → snippet → full
   body). No schema change — frontmatter is plain text the chunker already
   captures; the upgrade is query-side. (Pairs with findings.md follow-up items
   1+2: anchor liveness + file-path-keyed retrieval.)

## Skill reinforcement (order matters)

1. `_shared/rules/sdd-structure.md` FIRST (it wins all disagreements): new
   `## Knowledge (OKF)` section — layout, type table, index/log/reverse-index
   rules, STATUS↔`status` mapping.
2. `architectural-decision-records`: write to `decisions/`, offer
   `InvariantContract`/`BusinessWorkflow` when the shape fits (one concept per
   why), update `index.md`+`log.md`+reverse-index in the same edit.
3. `brainstorming`/`interviewing`: VERIFIED/INFERRED → concepts; LOCKED → new
   ADR file + index row + catalog entries (same edit).
4. `writing-blueprints`/`slicing`: consume `decisions|invariants|workflows` as
   constraints; `tasks.md` Files column stays the edit-scope guard.
5. `qa`: run `okf-lint.mjs` as a new step (WARNING-only, never blocks — same
   treatment as state-drift/ship-drift); `reflect`: lift durable findings to
   `reference/` + `log.md`.
6. `onboarding` (+`references/process.md`): scaffold
   `index.md`/`log.md`/`decisions|invariants|workflows|glossary|reference` on
   new projects; merge-mode migrates old flat `artifacts/`.
7. `mnemonic`: document the split — `mem_save` for episodic memory, concepts
   for durable knowledge; recall-via-index rule.
8. `resume`/`using-skillgrid`: orient via `.skillgrid/index.md` + `log.md` tail
   (cheap) before `mem_context`.
9. Hooks: extend `pre-commit.js` zone guard with OKF lint (advisory);
   reverse-index desync = WARNING until promoted.

## ADR / gating requirements

- **New ADR (next free number at authoring time — NOT 0037/0038 (0037 taken by the embedded-ANN index, 0038 reserved by
  TICKET-03 for the knowledge-class dimension; expect 0039):** amends (never
  deletes/supersedes-destructively) ADR-0012 — records Hybrid SoT (OKF bundle
  durable, SQLite retrieval index), the OKF profile, and the no-new-dep
  decision. Requires explicit user OK (LOCKED rule).
- **Operational rule (AGENTS.md `### Rules`):** one bullet —
  "`.skillgrid/` knowledge is OKF v0.2; skills keep index/log/reverse-index in
  the same edit." Full schema lives in `sdd-structure.md`, not the prompt block
  (block stays lean per onboarding rule).
- **`ASSUMPTIONS.md` stays path-only** (ADR-0019); `index.md` summaries are
  one-liners, not body copies.

## Follow-up change shape (when opened)

- **Spec zone:** briefing (falsifiable: lint-clean on current tree; recall
  latency before/after) → blueprint → `acceptance.feature` Gates: G1
  frontmatter-valid, G2 index-complete, G3 links-unbroken, G4
  reverse-index-synced, G5 `state.yaml` drift-clean, G6 mnemonic recalls a
  moved ADR → user gate.
- **Apply waves:** W1 schema + scripts (`okf-lint`, `reverse-index`,
  fixtures); W2 `git mv` ADRs → `decisions/` + index/ASSUMPTIONS updates +
  `archive/` → `xarchive/` rename; W3 invariants/workflows/glossary/reference
  split + `artifacts/` dissolution; W4 specs/xarchive frontmatter backfill +
  reference sweep (`sdd-structure.md`, skills, hooks, `config.yaml`,
  dashboard); W5 skill + hook + `sdd-structure.md` + `AGENTS.md` Rules updates.
- **Verify:** `okf-lint` clean, `state-drift-check` + `ship-drift-check` clean,
  `go test ./...`, index-status fresh + spot `code_search` for a relocated ADR,
  `/plans` STATUS parsing unaffected.

## Risks

- **Churn vs IRON RULE:** 36 ADR moves touch the LOCKED index — mitigate with
  mechanical `git mv` + `diff -r` readback (ship pattern) and one review pass
  over the index table.
- **AGENTS.md bloat:** one Rules bullet + pointers; schema in
  `sdd-structure.md` + `.skillgrid/index.md`.
- **openwiki overlap:** openwiki `.claims/` sidecars solve the same problem as
  `verification.anchors` — pick anchors (code-pin), skip sidecars; revisit only
  if openwiki generation is adopted later.
- **Scope guard:** this proposal must not leak into the frozen TICKET-01..05.
  It executes only as its own change after the user gate.
