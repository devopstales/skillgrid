# 012 — Trajectories (agent working-memory) + recall/health deltas

**Status:** READY FOR EXECUTION · **Owner:** knowledge-base change
**Depends on:** `001-structure` (packet dir, skill/case types, the config sentinel),
`002-sources` (the `sources:` form it extends), `004-index` (`kb_trajectories`),
`005-query` (the stage-1 carve-out), `007-lint` (the checks it implements),
`008-transport` (the surface it wires).
**Reference:** zosmaai/pi-llm-wiki trajectories feature (`raw/trajectories/TRJ-*`,
`wiki_capture_trajectory` / `wiki_distill_skills` / `wiki_recall_skill`, the
skill/case split, `recallSkillInlineMax`, the opt-in issue #80) + its
`wiki_dashboard` + lint check set — read 2026-11-06 against `skills/llm-wiki/SKILL.md`.

## 1. What this is

Two halves, one slice.

**Half A — trajectories (a third knowledge class).** The KB captures *what*
(decisions/invariants) and *why* (claims, grounding) — but not *how*: the process of
solving a task is lost after the session (the memory family keeps prose session
summaries, not the structured tool-call record). Trajectories add an **immutable,
replayable packet** of a completed task's tool-call sequence, distilled into **skill**
pages (generalized "how I do X") and **case** pages (one concrete run). A completed
task is just another kind of source — it flows through the same cite/distill pipeline
(`001` §5.4, `002-sources`), not a parallel one.

**Opt-in, off by default** (pi's issue-#80 discipline): the three tools are *absent
entirely* when the flag is off — no system-prompt cost (the `### KnowledgeBase` MCP
section and the AGENTS.md block document trajectory routing only when enabled).

**Half B — recall/health deltas.** The reference's recall + health behavior, applied
where the KB already has the substrate: the 005 stage-1 **inline carve-out** (agents
skip bare links; "apply-immediately" pages inline their short body), the 007 lint
checks that are **specified but unimplemented** (orphans, broken wikilinks, dup
aliases, contradiction markers, … — the shipped `lint.go` has 2 of the 12), the 008
`kb_status` **dashboard-grade** extension, and the 003 **per-run model override**.

## 2. Design map (pi-llm-wiki → kb)

| pi-llm-wiki artifact | kb artifact | Notes |
|---|---|---|
| `raw/trajectories/TRJ-*/` (`manifest.json`, `packet.json`, `extracted.md`) | `trajectories/TRJ-YYYY-MM-DD-NNN/` **directly under the vault root** | Flat layout (001 KB-1): NOT the `raw/` layer — KB-8 is about *external* sources (cite, don't store); a trajectory packet is a **KB-generated input**, git-tracked, engine-owned, immutable. Re-capture = a new id, the old packet is never edited |
| `wiki_capture_trajectory` (one call, self-contained summary, no skeleton) | `kb_capture_trajectory` (`title`, `steps`/session log) | A single lightweight **write**. `packet.json` = the replayable tool-call sequence (tool, args, result summary, error); `extracted.md` = the self-contained summary. No to-be-fleshed skeleton page at capture |
| `wiki_distill_skills` (returns undistilled) | `kb_distill_skills` (`expected_project`) | Lists `kb_trajectories` rows with `distilled_at IS NULL`. **Distilled promotion is derived, not a call**: when `kb_index` indexes a page whose `sources:` cites `trajectory:TRJ-…`, the engine sets `distilled_at` (INV-5 event `trajectory.distilled`) |
| `wiki_recall_skill` (recall filtered to skill/case, bodies inlined) | `kb_recall_skill` (`query`) | `kb_query` with `type ∈ {skill, case}` + the §2.3 inline carve-out. "Have I done this before?" at task start |
| `skill` / `case` page types (frontmatter `trajectories:`, `trajectory_id`, `outcome`) | `skill` / `case` page types (001 §4, **contextual** — no anchors) | No new sidecar fields: provenance rides the existing `sources:` as `trajectory:TRJ-*` (001 §5.4, the 4th form). `case` adds one agent-authored field: `outcome: success \| failure \| partial`. A `skill` page cites every TRJ it generalizes; a `case` page cites its one |
| `recallSkillInlineMax` (1600) + `recallLinksThreshold` (50) | `config.json` `recall: { inline_types: [invariant, gotcha, skill, case], inline_max: 1600, links_threshold: 50 }` | 001 §8 (the sentinel carries it); 005 §2.3 (the behavior). Page count read from `kb_pages` `COUNT(*)` — O(1), no body I/O (pi's `registry.json` trick) |
| opt-in `llm-wiki.trajectories` toggle | `config.json` `"trajectories": false` | Gates **tool registration** (absent when off), CLI subcommand presence (clean exit-2 message, not a crash), and the conditional docs (half A) |
| `wiki_dashboard` (counts by type, stale 30d+, 7-day activity, zero-backlink, all from on-disk state) | `kb_status` extension (008 §3) | Pure read over `kb_pages` / `kb_events` / `in_degree` — no LLM, no new state |
| lint orphans / contradictions / gaps | **already specified** in 007 §2 (12 checks) — unimplemented (shipped `lint.go`: 2) | Tickets KB-T12/13 implement what 007 already says; one **new** check: `weak_cross_reference` (001 §6's golden rule — ≥2 **outgoing** links — is stated but not enforced) |
| `/wiki-model` per-call override (override > configured > session, graceful degrade) | `kb_generate` optional `model` param (003 §4) | Unknown ref → the existing append-only degraded floor (INV-3 unchanged) |

## 3. Deliverables (this slice)

Engine (`mnemonic-v2/internal/kb/`), over the existing `store.Store` seam:

1. `trajectory.go` — `CaptureTrajectory` (writes the packet dir; id =
   `TRJ-YYYY-MM-DD-NNN`; `manifest.json` `{id, project, title, created_at,
   format: "trajectory"}`, `packet.json`, `extracted.md`), `ListUndistilled(project)`,
   the distilled-promotion hook (index path, §2).
2. `frontmatter.go` — the `skill` / `case` types (contextual, no `anchors:`), the
   agent-authored `outcome` field on `case`, the `trajectory:` source-form parse.
3. `cite.go` — the `trajectory:TRJ-*` citation (resolves to the packet dir's
   `extracted.md`; a missing packet = `broken_citation` lint finding — a packet is in
   the vault, so it must exist once captured).
4. `index_inc.go` — index skill/case pages like other contextual pages; parse
   `trajectory:` citations; **set `kb_trajectories.distilled_at`** when a cited page
   is indexed (emit `trajectory.distilled`).
5. Migration — `kb_trajectories` (`id` TRJ-… PK with project, `project`, `title`,
   `created_at`, `distilled_at` NULL). Packet `extracted.md` is **not** FTS-indexed as
   a page (raw input; reachable via its citing pages + `kb_read` of the packet).

Surface:

6. CLI (`cmd/mnemonic/kb.go`) — `kb trajectory capture|distill|recall`. Flag off →
   clean `trajectories disabled (config.json)` error, exit 2 (usage/operational), not a
   crash.
7. MCP (`tools_kb.go`, 008 §3) — 3 tools, **agent profile**, registered **only when
   `config.json.trajectories` is true** (runMCP reads the sentinel before
   `RegisterKbTools`). Write queue: `kb_capture_trajectory` + `kb_distill_skills`
   (distilled promotion) are KB writes; `kb_recall_skill` is a read. `expected_project`
   on the writes (008 §4).
8. **Conditional docs** — the `### KnowledgeBase` MCP instructions section + the
   AGENTS.md block gain a trajectory-routing line **only when enabled** (both are
   marker-managed — a re-bootstrap / config flip refreshes them).
9. 005 carve-out — `query.go`: stage-1 inlines the body for `recall.inline_types`
   pages whose body ≤ `inline_max` (an optional `body` field on the row);
   **small-vault mode** (page count < `links_threshold`) inlines previews for **all**
   types (the two-step contract becomes optional). The "snippet is a teaser, not the
   page" discipline holds for everything not inlined.
10. 007 lint — implement the specified checks: KB-T12 structural (orphans, broken
    wikilinks, duplicate aliases, **weak_cross_reference** [new], contradiction
    markers, `trajectory:` resolution); KB-T13 citation/grounding (broken citations
    incl. the neuledge staleness gate, ungrounded grounded-page, weak pin, past
    `stale_after`, missing `sources:`, orphan citations, coverage gaps). 007 §4's
    `--fix` scope is unchanged.
11. 008 `kb_status` — add: counts by type, stale count (`updated_at` vs
    `stale_after` / a 30-day default), 7-day activity by `event_type` (`kb_events`),
    zero-backlink count (`in_degree`). 003 `kb_generate` — optional `model` param
    (override > config > session; unknown → degraded floor).
12. kb `SKILL.md` — a **Working Memory** section (capture at task end, distill in a
    batch, `kb_recall_skill` at task start; skill vs case; one-packet-per-task) +
    the recall/health doc updates.

## 4. Tickets

| # | Ticket | Scope |
|---|---|---|
| KB-T8 | trajectory capture (packet + `kb_trajectories` + migration + `trajectory:` source form + skill/case taxonomy + `outcome`) | 1, 2, 3, 5 |
| KB-T9 | distill + recall (undistilled list, derived distilled-promotion, `kb_recall_skill` = type-filtered recall) | 4 (promotion), 1 |
| KB-T10 | opt-in surface (config flag, CLI subcommands, MCP registration, conditional docs, clean-off error) | 6, 7, 8 |
| KB-T11 | 005 recall carve-out (inline types + budget + small-vault mode + `body` row field) | 9 |
| KB-T12 | 007 lint structural checks (orphans, broken wikilinks, dup aliases, weak_cross_reference, contradiction markers, `trajectory:` resolution) | 10a |
| KB-T13 | 007 lint citation/grounding checks (broken citations + neuledge gate, ungrounded, weak pin, stale_after, missing sources, orphan citations, coverage gaps) | 10b |
| KB-T14 | 008 `kb_status` dashboard-grade + 003 `kb_generate` model override | 11 |

## 5. Invariants (restated + new)

KB-INV-1 (pages are truth — packets are **input**, not truth: the skill/case page is
the durable artifact, the packet its evidence); KB-INV-3 (the derived layer is
engine-owned — and now so is `trajectories/**`: the agent never edits a packet,
re-capture instead); INV-3 (explicit degradation — capture with no session tool log
is an explicit error, never a silent empty packet); INV-5 (`kb_events`:
`trajectory.captured` / `trajectory.distilled`); **opt-in discipline** (flag off →
tools absent from `tools/list`, clean CLI error, docs silent — zero prompt cost).

## 6. Verification floor

- `go test ./internal/kb/...` green, including: re-capture immutability (new id, old
  packet bytes unchanged); distill returns only undistilled; promotion-on-index (cite
  TRJ → index → `distilled_at` set, event emitted); recall type-filter + inline; the
  carve-out matrix (small/large vault × type in/budget out); the KB-T12/13 check
  matrix (each check: fires on its fixture, silent on the clean vault); status metrics
  from fixture state; model-override precedence; **flag-off proof** (tools absent from
  the MCP `tools/list` frame, CLI exit 2, no trajectory text in the served
  instructions).
- Smoke: on a temp vault — capture → distill → write a `skill` page citing
  `trajectory:TRJ-…` → `kb_index` → `kb_recall_skill`; then `kb_status` +
  `kb_lint --strict` on this repo's vault (the KB-T12/13 checks run clean against the
  live 8-page vault).
