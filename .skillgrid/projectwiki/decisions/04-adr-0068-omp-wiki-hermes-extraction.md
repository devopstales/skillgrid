---
type: TechnicalDecision
title: "omp/pi wiki-memory plugin: Hermes-style extraction into the projectwiki skills/ zone — generalized pi-omp extension, Go-owned draft lifecycle, significance-gated session-model analyzer, verb-level fail-open boundary"
status: "accepted"
supersedes: none
date: 2026-11-19
---

# omp/pi wiki-memory plugin: architecture

## Context and Problem Statement

The change `2026-11-19-omp-wiki-memory` gives two coding-agent hosts (omp, pi)
a Hermes-style knowledge-extraction loop (oh-my-pi#2189) whose knowledge
substrate is the skillgrid projectwiki vault (`.skillgrid/projectwiki/`)
instead of a plugin-owned memory file or host-managed skills. A six-lens
Goldfish shootdown (2026-11-19; `reviewer`-agent model quota-exhausted —
lenses ran on the `flint` agent, deviation recorded in the briefing) attacked
the anchored design; three Blockers were user-dispositioned at the gate and
three design revisions followed. The decisions below are hard to reverse
(vault zone, draft-lifecycle semantics, MCP contract re-baseline, the
generalized pi-omp extension seam the in-flight memory family also adopts)
and they trade against each other, so they clear the ADR bar as one
architecture decision.

The reference shapes (user-validated): pi-memory (KV-cache-stable snapshot
injection, fail-open capture), pi-hermes-memory (per-turn background review,
correction detection, procedural skills, auto-consolidation), oh-my-pi#2189
(significance gate, destination router, semantic peer review, provenance,
curator, approval), and the Karpathy llm-wiki pattern (llm-wiki-skill /
hermes-agent `research-llm-wiki`: orientation-first session start, page
thresholds, contradiction marking, lint-surfaced weak claims).

## Considered Options

- **A. Direct page writes into the vault (chosen).** The extraction loop
  writes skill pages directly (draft status) into a new `skills/` vault zone;
  Go owns frontmatter, index, and events. Rationale: the vault's own
  ownership model ("You own the page markdown; the engine owns derived
  state"), the live gotcha that `projectwiki generate` runs degraded
  (no Synthesizer → append-only fact-dump mode that pollutes a curated
  vault), and the pre-wired plumbing (`pageDirs` whitelist already carries
  `skills`/`cases` opt-in (012); `recall.inline_types` already lists
  `skill`/`case`).
- **B. Route through `projectwiki generate`** (extraction → observations →
  generate materializes pages). Reuses the most machinery but is currently
  blocked by the missing Synthesizer (gotcha); the generated pages also
  bypass the draft/approval contract.
- **C. Plugin-owned memory files (pi-hermes shape: own MEMORY.md + SKILL.md
  under `~/.omp/agent/`)** — the substrate the user explicitly rejected in
  favor of the projectwiki; would fork a fourth knowledge location.
- **D. Staging directory + promote** (issue #2189's `skill_propose` shape:
  `.proposed/` dir, invisible until promoted) — user chose direct `status:
  draft` writes instead (git-visible pending set; drafts excluded from
  default query + `validate --strict` rather than path-hidden).

Secondary decisions (same gate): **analyzer LLM placement** (host side-turn
vs engine LLM client vs TS custom provider); **significance gate shape**
(shootdown B1); **review surface affordances** (shootdown B2); **fail-open
boundary** (shootdown B3, verb-level vs engine-level retry wrap); **agent
MCP profile growth** (8→10 vs 8→9); **extension generalization** (ADR-0067
revisit criterion #2).

## Decision Outcome

Chosen option: **A**, plus the secondary decisions below.

1. **Single auto zone, skill-shaped artifacts.** All extracted knowledge —
   procedures, guardrails, gotchas, decisions — becomes a page in
   `.skillgrid/projectwiki/skills/<slug>.md`: `type: Skill`, SKILL.md-shaped
   body (name / description / when-to-use / procedure), a `kind` frontmatter
   field (`procedural|guardrail|gotcha|decision`) for curation. Hard
   invariant (code-level, not config): auto-writes create/modify
   `status: draft` pages in `skills/` ONLY — never stable/accepted pages,
   never outside the zone. `decisions/`, `invariants/`, `gotchas/`,
   `glossary/`, `ASSUMPTIONS.md` stay human-only in the auto loop; formal
   ADR promotion of a learned decision is a human move (IRON RULE: an index
   row needs an explicit user OK — preserved).
2. **One draft contract, aligned with ADR-0060.** Drafts (status `draft`)
   are FTS-indexed (peer review must find pending drafts) but excluded from
   `projectwiki_query` by default (new `include_drafts` opt-in) and skipped
   by `validate --strict`. Promote = the same status-flip primitive
   ADR-0060's contextual-draft lifecycle uses; reject = delete + event.
   `StatusResult.Pending` (plan jobs) is NOT overloaded — a separate
   `Drafts` field.
3. **Significance-gated triggers (shootdown B1, resolved).** Cadence
   (every 10 turns / 15 tool calls, configurable) only schedules *candidate
   windows*; the LLM runs only when the window is substantive per a defined
   heuristic: correction detected, OR ≥3 file edits in window, OR ≥8 tool
   calls across ≥3 distinct tool types, OR an explicit remember/learn cue.
   Other triggers: correction (immediate), `session_before_compact`,
   `session_shutdown`, agent-initiated `projectwiki_extract` MCP tool.
   Below-threshold windows emit `extract_skipped{reason}` events — no LLM
   call.
4. **Analyzer on the host side-turn (user-validated).** TS pre-gate →
   `runEphemeralTurn(analyzerPrompt)` on the session model (side-turn: not
   appended to session history, may run while the main turn is active) →
   strict-JSON result (significance, kind, content, `conflicts:
   [rel_path]`, related pages) → spawn `mnemonic projectwiki extract
   --session <id> --result -`. No LLM client in TS, no Go LLM call in v1
   (the engine LLM client stays the fallback seam, unused in v1).
   Execution-time assumption: pi (skill-pi 1.1.0) exposes the same
   side-turn API; if not, the loop degrades to loop-off on pi (skip +
   drop-count, fail-open) — omp is the live-verified host.
5. **Contradiction marking (llm-wiki pattern, ADR-0060-aligned).** Peer
   review (FTS over the vault including drafts) prefers updating an existing
   covered page; when new knowledge conflicts with an existing page, the
   draft carries `contradictions: [rel_path]` frontmatter (never a silent
   overwrite of stable content); `projectwiki lint` surfaces contested /
   low-confidence pages for review. Page thresholds (llm-wiki, now
   concrete): create only when central or 2+ evidence sessions corroborate;
   add when covered; split at ~200 lines (curator); archive when superseded
   (curator, `absorbed_into` lineage).
6. **Verb-level fail-open boundary (shootdown B3, resolved).** The
   `extract` verb owns the failure boundary: any internal failure
   (store-busy exhaustion — projectwiki writes use the raw `Exec` seam
   without `withWriteRetry`, unlike memory writes — timeout, write
   conflict, LLM unavailable) exits 0 with a structured drop receipt on
   stdout (`{"dropped":true,"reason":"store_busy|timeout|write_conflict|llm_unavailable"}`);
   the extension counts drops **with reasons** (per-drop triage). Optional
   blueprint-level refinement: wrap the verb's own writes in the existing
   retry wrapper; the receipt boundary is the contract either way.
7. **Generalized pi-omp extension (ADR-0067 revisit #2 fired).** This is the
   second thin extension on the omp+pi pair; per ADR-0067's criterion #2
   ("generalize before a third family, not after"), this change generalizes
   the pi-omp extension into a shared multi-family base (session
   registration, fail-open spawn + drop counting, compaction re-render,
   prompt-override assembly) so the code-index family (the third) reuses
   instead of re-deriving. One protocol assembly, not two (shootdown:
   R10 honored at engine level only).
8. **MCP contract re-baseline, 8→9.** `projectwiki_extract` joins the agent
   profile (agent-initiated learning); `projectwiki_review` stays
   **CLI-only + host command** — human-gated curation is out of the agent
   working set (the D1 working-set/admin split is test-pinned; the pin is
   re-baselined). `projectwiki digest` / `projectwiki curate` are CLI-only.
9. **Curator = composition.** `projectwiki curate` composes `lint`
   (duplicate/orphan/weak-pin checks) + `validate` (pin drift) + new
   near-duplicate (FTS self-similarity) and unused-draft (event-based)
   checks — one semantics home per check; user confirms before destructive
   merges. Hard invariant: never touches `origin: user` pages or
   `status: accepted` ADRs.
10. **Provenance: file-frontmatter-only in v1 (PW-INV-1).** Every auto page
    carries `origin: auto`, generated-by (host/session — a new field; does
    NOT overload `Stamp.By`'s generator-label semantics), `learned_at`,
    `evidence: [session/turn refs]`, `kind`, `contradictions?`. The
    `Frontmatter` struct is extended additively (struct-based
    `EmitFrontmatter` drops unknown fields); `Skill` is registered in the
    type maps (`shortTypeFor`, non-grounded — `GroundedTypes` unchanged);
    `skills` joins `PageTypeDirs` (bootstrap scaffolding) and the vault
    `config.json` `page_types`. No new DB/FTS columns in v1 (the file is
    truth; the curator reads files). New `projectwiki lint` rule:
    auto-origin page missing provenance → finding. Events gain
    session/trigger dimensions (`extract_*` event family, incl.
    `extract_dropped{reason}` and `extract_skipped{reason}`).
11. **Review affordances (shootdown B2, resolved).** `/wiki-review` lists
    pending drafts (id · kind · title · one-liner · evidence);
    `/wiki-review <n>` shows full text (new page) or unified diff (update);
    `promote <n>` / `reject <n>` are the explicit user actions, each
    followed by a `ui.notify` receipt; defined error texts for `no pending
    drafts` / `not found` / `already promoted`; undo = rejecting a promoted
    auto-page demotes it to draft (deletion stays a human git act).
    Session-start digest carries the flagged pending list (user-validated
    Q8: stable pages + flagged pending).
12. **Transport / installer / sequencing (ADR-0067 conventions).** Host-MCP:
    each host's MCP config gains key `mnemonic-projectwiki` →
    `[<abs-bin>, "projectwiki", "mcp", "--profile", "agent"]` (existing
    server `mnemonic-projectwiki` v0.2.0). One shared TS source
    `plugins/wiki/pi-omp/` (generalized per 7) installing to
    `~/.omp/agent/extensions/` and `~/.skill-pi/agent/extensions/`; node
    builtins only; adapter-owns session id `wiki-<uuid>`. skillgrid-cli
    gains family `wiki` on the (host, family)-keyed tracked-set manifest
    boundary the memory change exports (`~/.skillgrid/mnemonic/
    wiki-install.json`): MCP entries + extension files; uninstall restores
    byte-identical state. The shipped `projectwiki` agent skill is updated
    (operating-manual update: orientation-first, drafts/pending, when to
    call `projectwiki_extract`, `/wiki-review` dispositions, contradiction +
    threshold conventions — user-validated Q9). Serial development: this
    change executes after `2026-11-17-mnemonic-v2-host-memory-plugins` and
    `2026-11-17-skillgrid-host-plugins` (their pi-omp seam, installer
    boundary, and `memory save-prompt` prompt capture are prerequisites).
13. **Digest = bounded orientation read.** `projectwiki digest` (CLI,
    ≤2K chars): stable skill pages (name + one-liner) + flagged pending
    draft list + recent-activity tail (last 5 `projectwiki_events`).
    Injected via `before_agent_start` system-override; byte-stable until
    compaction (KV-cache pattern; re-render at `session_before_compact`),
    the accepted cost trade that the pending list can be stale mid-session
    (shootdown UX major, recorded).
14. **Hermes convergence noted.** `plugins/hermes/` (Hermes Agent
    MemoryProvider) already carries v2 distill hooks ("The distill pass is
    a v2 feature" — `on_session_end`/`on_pre_compress`); `projectwiki
    extract` is the engine-side distill that seam will shell out to. Hermes
    v2 itself is out of scope; the verb is shaped to be its implementation.

### Implementation

- `plugins/wiki/pi-omp/` — generalized pi-omp extension source (shared
  family base + wiki family), installed for omp + pi.
- `mnemonic/internal/projectwiki/` — verbs `digest`, `extract`, `review`,
  `curate`; MCP tool `projectwiki_extract` (agent profile 8→9); `include_drafts`
  filter on query; `Drafts` field on status; `Skill` type registration;
  `Frontmatter` additive extension; lint rules (provenance, contested);
  `extract_*` event family.
- `skillgrid-cli/internal/install/` — family `wiki` on the exported
  (host, family) boundary.
- `.agents/skills/projectwiki/` — operating-manual update.
- No new third-party dependencies.

### Consequences

- Good, because the loop compounds into the project's existing
  knowledge substrate (one KB, one index, one review surface) instead of
  forking a plugin-owned memory file — the pi-hermes capability without a
  fourth knowledge location.
- Good, because the hard invariants are code-level: the auto loop can only
  touch drafts in `skills/`, can never touch `origin: user` or
  `status: accepted` content, and every auto page is provable (provenance
  lint rule + events).
- Good, because the ADR-0067 convention set is preserved end-to-end:
  host-MCP transport, thin hooks-only TS, Go write authority, one-shot CLI
  verbs, tracked-set installer — and the generalization means the
  third family (code-index) does not pay the second-extension tax.
- Bad, because the analyzer runs on the session model: per-call cost is
  context-sized and the prefix-cache behavior of the side-turn is unverified
  (omp partially). Mitigation: the significance gate bounds calls (target
  0–3/session, measured via events); on a local session model (the common
  case here) the cost is near-zero; a Go LLM-client fallback seam exists
  for a cheap-tier switch later.
- Bad, because two draft lifecycles (ADR-0060 contextual drafts + skills/
  drafts) and two page generators (generate + extract) now share the vault
  with different gates. Mitigation: one draft contract (decision 2),
  extract confined to skills/+draft (decision 1), gate separation
  documented in the blueprint.
- Bad, because fail-open on pi depends on an unverified side-turn API; a
  quiet machine may have a loop that never fires. Mitigation: drop-counted
  with reasons, surfaced by `projectwiki status` + `/wiki-status`;
  execution-time verification is a blueprint task.
- Bad, because the `validate --strict` CI gate now skips drafts: a
  mis-scoped write (an auto page that escaped `skills/`) is invisible to
  CI. Mitigation: the provenance lint rule + the write-path confinement
  test (extract can only produce `skills/` drafts).

### Revisit criteria

Revisit this ADR (write a superseding ADR) when **any** of:
1. pi is verified not to expose the side-turn API and pi users demand the
   loop — the analyzer moves to the Go LLM-client seam (cheap local tier).
2. The Synthesizer is wired for `projectwiki generate` (the blocking
   gotcha retires) and the memory→wiki route needs a second page generator
   with the same draft contract — re-derive the dual-generator gates.
3. Draft volume makes on-demand curation insufficient (measured via
   `extract_*` events + `status` draft counts) — a scheduled curator pass
   or an auto-archive policy.
4. Cross-project promotion is requested (issue #2189's remaining gap) —
   global promotion needs the independent-context threshold the thread
   converged on, and a new ADR.
5. The third family (code-index) lands and the generalized extension base
   still shows family-specific seams — generalize further or split.
