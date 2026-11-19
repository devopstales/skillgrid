# omp/pi wiki-memory plugin — Design Briefing (requirements & intent)

> **STATUS:** `draft` (2026-11-19)

## Problem / Intent

Coding agents on omp and pi do not retain the knowledge they acquire:
procedures, guardrails, gotchas, and project conventions learned in one
session evaporate at compaction or session end. The user wants a Hermes-style
knowledge-extraction loop (oh-my-pi#2189: significance-gated review,
destination router, semantic peer review, provenance, curator) whose
knowledge substrate is the project's **skillgrid projectwiki vault**
(`.skillgrid/projectwiki/`) — not a plugin-owned memory file and not
host-managed skills. The loop compiles session experience into
skill-shaped wiki pages the agent actually reads and follows.

## Purpose & Success Criteria

- **Purpose:** omp and pi sessions automatically deposit durable,
  reviewable knowledge (procedures / guardrails / gotchas / decisions) into
  the project's existing knowledge base, with a human approval gate before
  knowledge becomes stable, and a bounded orientation injection so the agent
  starts each session knowing what the wiki already knows.
- **Success criteria:**
  - After a substantive session (≥3 file edits or a detected correction),
    a new `status: draft` skill page appears in `skills/` with complete
    provenance frontmatter — without any user command.
  - The agent sees learned skills in the session-start digest and in
    `projectwiki_query` only **after** promotion; drafts are invisible to
    default queries and to `validate --strict`.
  - `/wiki-review` lists pending drafts with evidence; `promote` / `reject`
    are one explicit user action each, with a UI receipt.
  - Every loop failure is a counted drop with a reason — the host session
    is never broken by the loop (fail-open, verified by test).
  - `skillgrid install --family wiki` wires both hosts byte-identically
    (MCP entry + extension), and `uninstall` restores byte-identical state.
  - Analyzer cost is bounded: 0–3 LLM calls per typical session, measured
    via `extract_*` events (significance gate skips below-threshold windows).
- **Out of scope:**
  - Cross-project promotion (issue #2189's remaining gap — revisit criterion 4).
  - Hermes Agent v2 distill implementation (`plugins/hermes/` seam noted; the verb is shaped for it, Hermes v2 itself is a separate change).
  - llm-wiki companion tooling (Obsidian audit plugin, web preview server, external `raw/` ingestion — the captured prompt store is the immutable raw layer for session-derived knowledge).
  - Auto-writing to any zone outside `skills/`, any `origin: user` page, or any `status: accepted` page (hard invariant, R2).
  - A Go-side LLM client for the analyzer (seam exists, unused in v1 — revisit criterion 1).
  - pi-hermes-style memory aging / secret scanning / min-tool-call gating for the *memory* family (that change's territory).

## Context

- **The vault:** `.skillgrid/projectwiki/` is a Karpathy-LLM-wiki-shaped KB
  already: `PROJECTWIKI_SCHEMA.md` (schema layer), `index/` (derived
  catalog), `projectwiki_events` (the action log), frontmatter-typed pages
  with wikilinks, pin-checked grounded pages, and `validate --strict`
  (drift gate). Vault files are truth; the FTS index is derived (PW-INV-1/3).
- **The engine:** `mnemonic/internal/projectwiki/` exposes
  `projectwiki_*` over one shared handler (CLI + MCP, ADR-0061); family
  server `mnemonic-projectwiki` v0.2.0; agent tool profile currently
  12 defs / agent 8 (test-pinned, D1).
- **Pre-wired plumbing (verified 2026-11-19):** the `collectPages` whitelist
  (`index.go`, `pageDirs`) already carries `skills` + `cases` as opt-in
  (spec 012); `recall.inline_types` already lists `skill`/`case` (stage-1
  recall inlines short actionable pages). What is missing: `PageTypeDirs`
  (bootstrap scaffolding), the vault `config.json` `page_types`,
  `shortTypeFor` / type-map registration, and `GroundedTypes` — all
  additive Go (M3).
- **Blocking gotcha (live):** `projectwiki generate` runs degraded —
  no Synthesizer wired (gotcha `projectwiki-generate-has-no-synthesizer.md`)
  — so extraction does NOT route through generate (approach B rejected).
- **In-flight changes this change sequences after (serial development):**
  `2026-11-17-mnemonic-v2-host-memory-plugins` (ADR-0067: host-MCP
  transport, adapter-owns session ID, `(host, family)`-keyed tracked-set
  installer boundary, `memory save-prompt` prompt capture the analyzer
  consumes) and `2026-11-17-skillgrid-host-plugins` (its pi-omp extension
  seam this change generalizes).
- **Reference shapes (user-validated):** pi-memory (KV-cache-stable
  snapshot injection, fail-open capture), pi-hermes-memory (per-turn
  background review, correction detection, procedural skills,
  auto-consolidation), oh-my-pi#2189 (significance gate, destination
  router, peer review, provenance, curator, approval), Karpathy
  llm-wiki (llm-wiki-skill / hermes-agent `research-llm-wiki`:
  orientation-first session start, page thresholds, contradiction
  marking, lint-surfaced weak claims).
- **Rules in force:** Go 1.22+ floor (ADR-0031); no new dependencies
  without an ADR (none added); conventional commits; spec-zone commit
  before code-zone; repo is the source of truth (ADR-0013).

## Approaches Considered

- **Chosen:** Direct page writes into a new `skills/` vault zone (draft
  status), Go owns frontmatter/index/events, analyzer on a host
  side-turn, human gate via `/wiki-review`. Rationale: the vault's
  ownership model, the degraded generate gotcha, and pre-wired index
  plumbing all point the same way.
- **Rejected:** Route through `projectwiki generate` (blocked by the
  missing Synthesizer; generated pages bypass the draft contract).
  Plugin-owned memory files under `~/.omp/agent/` (substrate the user
  rejected; forks a fourth knowledge location). Staging-directory
  `.proposed/` + promote (issue #2189's shape — user chose direct
  `status: draft` writes; drafts are query-excluded + validate-skipped
  rather than path-hidden). Analyzer in Go with a second LLM client (two
  LLM seams, extra config surface, unused in v1).

## Requirements

1. **R1 — skills/ zone with skill-shaped artifacts:** The vault gains a
   `skills/` page zone; every page there is `type: Skill`, SKILL.md-shaped
   (name / description / when-to-use / procedure), with a `kind`
   frontmatter field (`procedural|guardrail|gotcha|decision`).
   - **Current:** `skills` is in the `pageDirs` index whitelist (opt-in,
     spec 012) but has no `PageTypeDirs` entry, no `config.json
     page_types` entry, no `shortTypeFor`/type-map registration, and no
     `Skill` type anywhere in the engine.
   - **Target:** `Skill` registered (non-grounded — `GroundedTypes`
     unchanged); `skills` added to `PageTypeDirs` (bootstrap scaffolding)
     and to existing vaults' `config.json` `page_types`; frontmatter
     `type: Skill` is authoritative (shortTypeFor is only the generate
     fallback).
   - **Acceptance:** A new vault bootstraps a `skills/` scaffold; a
     `skills/<slug>.md` with `type: Skill` indexes (FTS row) and is
     served by `projectwiki_read`; `isGrounded("Skill") == false`.
   - **Acceptance scenario:** `happy path skills-zone-registration` → `acceptance.feature`

2. **R2 — auto-write confinement (hard invariant):** The extraction loop
   may only create or modify `status: draft` pages inside `skills/`.
   - **Current:** No write path for the loop exists; ADR-0060's
   autolearning writes `sources/` + draft `syntheses/` (different zones).
   - **Target:** The `extract` verb's write path is structurally confined:
     destination folder is fixed to `skills/`, status is fixed to `draft`
     (new page) or preserved (existing draft update); `origin: user`
     pages and `status: accepted`/`stable` pages are rejected by the
     verb, not by convention.
   - **Acceptance:** A unit test drives extract against a fixture vault
     containing a stable page, an accepted ADR, and a user-origin page in
     `skills/` — none are modified; only a draft is created/updated.
   - **Acceptance scenario:** `happy path extract-cannot-escape-drafts` → `acceptance.feature`

3. **R3 — significance-gated extraction triggers:** Cadence (every 10
   turns / 15 tool calls, configurable) schedules candidate windows only;
   the analyzer LLM runs only when the window is substantive: correction
   detected, OR ≥3 file edits in window, OR ≥8 tool calls across ≥3
   distinct tool types, OR an explicit remember/learn cue in a prompt.
   Additional triggers: correction (immediate), `session_before_compact`,
   `session_shutdown`, agent-initiated.
   - **Current:** No trigger machinery exists for the wiki loop (pi-hermes
     has its own per-host loop; this repo's memory family triggers belong
     to the in-flight memory change).
   - **Target:** Extension-side trigger table (TS) with the heuristic
     above; below-threshold windows emit `extract_skipped{reason}` and
     make no LLM call; the analyzer runs on the host side-turn API
     (`runEphemeralTurn` on omp — live-verified; pi — execution-time
     assumption, degrades to loop-off per R4).
   - **Acceptance:** An E2E on a quiet read-only session (20 turns, no
     edits/corrections/cues) produces zero analyzer calls and N
     `extract_skipped` events; a session with 3 file edits in a window
     produces exactly one analyzer call.
   - **Acceptance scenario:** `happy path significance-gate-skip` → `acceptance.feature`

4. **R4 — host side-turn analyzer:** The analyzer prompt (session
   window + wiki digest + output schema) runs as a side-turn on the
   session model; the result is strict JSON (`significance`, `kind`,
   `title`, `content`, `conflicts: [rel_path]`, `evidence: [session/turn
   refs]`); the extension spawns `mnemonic projectwiki extract
   --session <id> --result -`. No LLM client in the extension; no Go LLM
   call in v1.
   - **Current:** pi-hermes calls its own provider from TS; this repo's
     Go LLM client (ADR-0023, `llmMode: smol`) is memory-family-only.
   - **Target:** One prompt-override assembly in the shared extension
     base (memory family reuses the same seam); JSON parse failure →
     drop receipt `llm_unavailable`; pi hosts without the side-turn API
     get loop-off (skip + drop-count + one `ui.notify` per session).
   - **Acceptance:** Malformed JSON from the side-turn path yields a
     counted drop, no partial write, host session intact; omp E2E shows
     the side-turn is not appended to session history.
   - **Acceptance scenario:** `happy path analyzer-strict-json` → `acceptance.feature`

5. **R5 — `projectwiki extract` verb (peer review + contradiction marking + drop receipt):**
   The verb performs semantic peer review over the vault (FTS including
   drafts): update an existing covered draft/skill, or create new; when
   knowledge conflicts with an existing page, the draft carries
   `contradictions: [rel_path]` (no silent overwrite of stable content);
   page thresholds apply (create only when central or 2+ evidence
   sessions; add when covered; split at ~200 lines is the curator's, not
   the verb's). Any internal failure (store-busy exhaustion, timeout,
   write conflict, LLM unavailable) exits 0 with a structured drop receipt
   on stdout: `{"dropped":true,"reason":"store_busy|timeout|write_conflict|llm_unavailable"}`.
   - **Current:** No extract verb; projectwiki writes go through the raw
     `Exec` seam (no `withWriteRetry` — the memory family's retry wrapper
     is memory-side only).
   - **Target:** Verb-level failure boundary (ADR-0068 decision 6): the
     extension counts drops **with reasons**; optionally the verb wraps
     its own writes in the existing retry wrapper (blueprint picks one).
   - **Acceptance:** A busy-store fixture (locked DB / busy timeout) makes
     the verb exit 0 with the drop receipt; the extension's drop counter
     increments with the reason; a conflicting draft carries
     `contradictions:` and the stable page is untouched.
   - **Acceptance scenario:** `happy path extract-drop-receipt` → `acceptance.feature`

6. **R6 — one draft contract:** Drafts (status `draft`) are FTS-indexed
   (peer review must find them) but excluded from `projectwiki_query`
   by default (new `include_drafts` opt-in) and skipped by
   `validate --strict`; promote = status flip to `stable` (same
   primitive ADR-0060 uses); reject = delete + event. `StatusResult`
   gains a separate `Drafts` field (`Pending` stays plan jobs).
   - **Current:** ADR-0060 has contextual drafts (draft `syntheses/`
     finalization) but no query exclusion or `Drafts` status field;
     `StatusResult.Pending` is plan-job semantics.
   - **Target:** One draft semantics both lifecycles share;
     `projectwiki status` reports draft count + slugs.
   - **Acceptance:** A draft page is found by raw FTS, absent from
     `projectwiki_query` (default) and present with `include_drafts:
     true`; `validate --strict` passes with drafts present; promote
     makes the page visible to default query; reject deletes it and
     records the event.
   - **Acceptance scenario:** `happy path draft-lifecycle` → `acceptance.feature`

7. **R7 — MCP surface, agent profile 8→9:** `projectwiki_extract` joins
   the agent profile (agent-initiated learning with the same draft
   contract). `projectwiki_review`, `projectwiki digest`,
   `projectwiki curate` are **CLI-only** (human-gated curation out of the
   agent working set). The test-pinned D1 contract (12 defs / agent 8 /
   all 12) is re-baselined to (13 / 9 / 13).
   - **Current:** Agent profile = 8 tools (pinned in tests); review/
     digest/curate verbs do not exist.
   - **Target:** Pinned contract re-baselined in the same change; the
     working-set/admin split (D1) preserved in shape.
   - **Acceptance:** The pinned-contract test passes at 13/9/13; the
     agent-profile tool list contains `projectwiki_extract` and not
     `projectwiki_review`.
   - **Acceptance scenario:** `happy path agent-profile-rebaseline` → `acceptance.feature`

8. **R8 — /wiki-review review affordances:** `/wiki-review` lists
   pending drafts (id · kind · title · one-liner · evidence);
   `/wiki-review <n>` shows full text (new page) or unified diff (update);
   `promote <n>` / `reject <n>` are explicit user actions, each followed
   by a `ui.notify` receipt; error texts defined for `no pending drafts` /
   `not found` / `already promoted`; undo = rejecting a promoted auto-page
   demotes it to draft (deletion stays a human git act).
   - **Current:** No review surface exists for skill drafts.
   - **Target:** Host command (TS) over the `review` verb (Go:
   list/show-diff/promote/reject subcommands) — one verb, both hosts.
   - **Acceptance:** E2E: a session producing one draft ends with the
     pending list in the digest; `/wiki-review` renders the list;
     `/wiki-review 1` renders the page; `promote 1` flips status +
     receipt; `reject 1` on a promoted draft demotes to draft; each
     error case renders its defined text.
   - **Acceptance scenario:** `happy path wiki-review-flow` → `acceptance.feature`

9. **R9 — bounded digest injection:** `projectwiki digest` (CLI, ≤2K
   chars) = stable skill pages (name + one-liner) + flagged pending
   draft list + recent-activity tail (last 5 `projectwiki_events`).
   Injected via `before_agent_start` system-override; byte-stable until
   compaction (KV-cache pattern), re-rendered at `session_before_compact`.
   - **Current:** The memory family injects a bounded snapshot (ADR-0067);
   no wiki digest exists.
   - **Target:** Extension reads the digest at agent start and
   re-render at compaction; a dead MCP server at start degrades to no
   injection (fail-open, counted).
   - **Acceptance:** Two consecutive agent-starts with no vault change
   produce byte-identical injections; a promoted page between compactions
   appears after the re-render; MCP-down at start injects nothing and the
   session runs.
   - **Acceptance scenario:** `happy path digest-stable-until-compaction` → `acceptance.feature`

10. **R10 — curator composition:** `projectwiki curate` (CLI-only)
    composes `lint` (duplicate / orphan / weak-pin) + `validate` (pin
    drift) + new near-duplicate (FTS self-similarity among skills) and
    unused-draft (event-based age) checks; reports findings; destructive
    merges/archives require explicit user confirmation; hard invariant:
    never touches `origin: user` pages or `status: accepted` pages.
    - **Current:** `lint` and `validate` exist separately; no
    near-duplicate or usage checks; no curator.
    - **Target:** One command, one semantics home per check (no
    re-implementing lint/validate logic); `absorbed_into` lineage for
    archives.
    - **Acceptance:** Two near-duplicate skill drafts are flagged with
      merge suggestion; an `origin: user` page present in the fixture is
      never reported as mergeable/deletable; confirmation is required
      before any archive executes.
    - **Acceptance scenario:** `happy path curator-composition` → `acceptance.feature`

11. **R11 — installer family wiki:** `skillgrid install|uninstall|status
    --family wiki [omp|pi|all]` extends the ADR-0067 (host, family)-keyed
    tracked-set boundary: MCP entries (key `mnemonic-projectwiki` →
    `[<abs-bin>, "projectwiki", "mcp", "--profile", "agent"]`) +
    extension files; manifest `~/.skillgrid/mnemonic/wiki-install.json`;
    tracked-set-only semantics (foreign files/entries never adopted,
    never clobbered, never deleted); uninstall restores byte-identical
    state.
    - **Current:** ADR-0067's installer boundary is defined by the
      in-flight memory change (family `memory`).
    - **Target:** Family `wiki` on the same boundary; no installer code
      duplication (the memory change exports the seam).
    - **Acceptance:** Install on a fixture home writes the tracked
      entries; a foreign MCP entry and a foreign extension file in the
      same config survive install + uninstall byte-identically;
      uninstall removes exactly the tracked set (manifest sha256 check).
    - **Acceptance scenario:** `happy path installer-family-wiki` → `acceptance.feature`

12. **R12 — generalized pi-omp extension (ADR-0067 revisit #2):** This is
    the second thin extension on the omp+pi pair; this change
    generalizes the in-flight pi-omp extension into a shared multi-family
    base (session registration, fail-open spawn + drop counting,
    compaction re-render, prompt-override assembly) and adds the wiki
    family to it. One protocol assembly, not two.
    - **Current:** The pi-omp extension (memory family, in flight) has its
      own protocol assembly and spawn/fail-open logic.
    - **Target:** `plugins/wiki/pi-omp/` (shared source, both families)
      installing to `~/.omp/agent/extensions/` and `~/.skill-pi/agent/
      extensions/`; node builtins only (no third-party deps); adapter-owns
      session id (`wiki-<uuid>` / memory family's existing id scheme).
    - **Acceptance:** Both families load from one source; the memory
      family's behavior is unchanged by the refactor (its E2E still
      passes); a failing spawn in either family is a counted drop, host
      intact.
    - **Acceptance scenario:** `happy path shared-extension-base` → `acceptance.feature`

13. **R13 — shipped operating-manual skill:** The existing `projectwiki`
    agent skill (`.agents/skills/projectwiki/`) is updated with the loop's
    knowledge: orientation-first behavior (schema → index → recent
    activity), the draft/pending concept, when to call
    `projectwiki_extract` explicitly, `/wiki-review` dispositions, and the
    contradiction + page-threshold conventions (user-validated Q9:
    "shipped skill + loop-written skill pages").
    - **Current:** The skill covers engine usage (query/read/index/
      validate/lint/status) but knows nothing of the extraction loop,
      drafts, or the review flow.
    - **Target:** Skill updated in the same change; it remains the agent's
      operating manual for the KB (Karpathy llm-wiki pattern: the skill
      is how the agent works the KB; the loop deposits into it).
    - **Acceptance:** The skill text names the loop's trigger
      conditions, the draft contract, and the review flow; the
      instructions-block size budget for the skill is respected (the
      pre-existing 2,596→≤2,048 rune trim is included — shootdown nit).
    - **Acceptance scenario:** `happy path operating-manual-update` → `acceptance.feature`

14. **R14 — extract_* event family:** Events gain session/trigger
    dimensions and failure visibility: `extract_run`, `extract_dropped
    {reason}`, `extract_skipped{reason}`, `extract_promoted`,
    `extract_rejected` (carried in `event_key` or an additive column —
    blueprint picks one; no breaking schema change).
    - **Current:** `projectwiki_events` has (project, event_type,
      subject_kind, rel_path, event_key, occurred_at) — no session,
      trigger, or failure dimension.
    - **Target:** The loop is fully auditable from the event trail alone;
      `projectwiki status` surfaces drop/skip counts.
    - **Acceptance:** A session with one skipped window, one dropped run,
      and one promoted draft produces exactly those events, queryable by
      session id; `status` reports the counts.
    - **Acceptance scenario:** `happy path extract-events-audit-trail` → `acceptance.feature`

## Implementation Decisions

- **Modules to build/modify:**
  - `plugins/wiki/pi-omp/` (new) — generalized shared extension source
    (family base + wiki family + memory family folded onto it per R12).
  - `mnemonic/internal/projectwiki/` (changed) — verbs `digest`,
    `extract`, `review`, `curate`; MCP tool `projectwiki_extract`;
    `include_drafts` query filter; `StatusResult.Drafts`; `Skill` type
    registration (`shortTypeFor`, `PageTypeDirs`, type map,
    `GroundedTypes` untouched); additive `Frontmatter` extension
    (`origin`, `kind`, `contradictions`, `learned_at`, `generated_by` —
    struct-based `EmitFrontmatter` drops unknown fields, so the struct
    must carry them; `generated_by` is a new field, NOT
    `Stamp.By`-overload); lint rules (auto-page missing provenance;
    contested/low-confidence surfacing); `extract_*` event family.
  - `skillgrid-cli/internal/install/` (changed) — family `wiki`.
  - `.agents/skills/projectwiki/` (changed) — operating-manual update.
  - Vault `config.json` `page_types` (changed, per-vault) + `skills/`
    scaffold for existing vaults (additive).
- **Interfaces:**
  - Verb: `projectwiki extract --session <id> --result <json|->`
    (stdin JSON = analyzer result) → stdout: write receipt or drop
    receipt (always exit 0 on loop-recoverable failure).
  - Verb: `projectwiki review list|show <n>|promote <n>|reject <n>` →
    list rows / full text or unified diff / receipt.
  - Verb: `projectwiki digest [--max-chars 2048]` → bounded text.
  - Verb: `projectwiki curate [--suggest] [--apply <n>...]` → findings;
    `--apply` after user confirmation.
  - MCP: `projectwiki_extract` (agent profile) — same result contract as
    the verb, for agent-initiated learning.
  - Analyzer JSON: `{significance: bool, kind, title, content,
    conflicts: string[], related: string[], evidence: string[]}`.
- **Data flow:** session window (prompts via `memory save-prompt`
  capture, in-flight) + digest + schema → trigger heuristic (TS, no
  LLM) → side-turn (session model, ephemeral) → JSON → `extract` verb →
  peer review (FTS incl. drafts) → draft upsert (skills/ only) +
  provenance frontmatter + event → (compaction) digest re-render →
  (human) `/wiki-review` promote/reject.
- **Error handling:** every loop failure path terminates in either a
  write receipt or a drop receipt (R5); extension spawn failure →
  counted drop + host intact; MCP-down mid-session → loop-off for the
  session (no respawn — host owns lifecycle, ADR-0067), one `ui.notify`;
  concurrent extract spawns (compaction + shutdown + correction racing)
  → single-flight per project, latest-wins, timer cancellation on later
  trigger (M9).
- **Dependencies:** none new (Go stdlib; node builtins in the
  extension). Sequencing: after `2026-11-17-mnemonic-v2-host-memory-
  plugins` + `2026-11-17-skillgrid-host-plugins` (M15).
- **Cost model (M11, priced):** analyzer call ≈ context-sized on the
  session model; significance gate targets 0–3 calls/session (measured);
  digest ≤2K chars (~500 tokens) once + re-render at compaction;
  index rebuild is O(V) per extract — incremental index, on-demand
  curator, `status` surfaces draft count; the pre-existing
  instructions-block trim (2,596→≤2,048 runes) is included.

## Shootdown Findings

**Anchored hypothesis:** omp/pi gain a Hermes-style extraction loop (pi-hermes
capability shape on the projectwiki substrate) via the established ADR-0067
conventions: host-MCP transport to the existing `mnemonic-projectwiki`
family server, a thin hooks-only TS extension, Go-side write authority
(drafts in a new `skills/` zone), human approval via `/wiki-review`,
fail-open with counted drops.

**Deviation (recorded):** the `reviewer` agent's model
(`9router/ag/gemini-3.8-flash`) was quota-exhausted (429
`RESOURCE_EXHAUSTED`, 5/5 first attempts + 1 retry failed) — all six
lenses re-ran on the `flint` agent (local NVFP4 40B, quota-free). All
six returned reports; zero `not run`. All lens code claims
spot-checked against the tree.

| Severity | Lens | Description | Disposition | Note |
|----------|------|-------------|-------------|------|
| 🔴 Blocker | Cost (also SRE) | Pre-gate is a raw cadence counter with no significance condition — read-only exploration sessions still fire the analyzer | resolved | R3: cadence schedules candidate windows; defined heuristic gates the LLM; below-threshold windows emit `extract_skipped{reason}`, no LLM call; yield measured via events |
| 🔴 Blocker | UX | `/wiki-review` approval gate is a backend op with no user-facing affordance (no diff preview, error text, notification, undo) | resolved | R8: list / show-diff / promote / reject with defined error texts, `ui.notify` receipts, demote-as-undo, pending list in session-start digest |
| 🔴 Blocker | SRE | Fail-open breaks on the new write path: projectwiki writes use raw `Exec` (no `withWriteRetry`), so busy-exhaustion is a hard error with no home in the drop taxonomy | resolved | R5: verb-level failure boundary — any internal failure exits 0 with a structured drop receipt; extension counts drops with reasons (also closes the untriageable-drop major); optional retry-wrapper wrap named for the blueprint |
| 🟡 Major | TechDebt | Second thin extension before generalization (ADR-0067 revisit criterion #2) | resolved-by-design | R12: this change generalizes the pi-omp extension into a shared multi-family base (also kills R10 protocol-assembly duplication) |
| 🟡 Major | TechDebt | `skills/` half-wired: index whitelist yes; `PageTypeDirs` / `config.json page_types` / `shortTypeFor` / type map / `GroundedTypes` have no `Skill` entry | resolved-by-design | R1: additive registration; frontmatter `type: Skill` authoritative; non-grounded |
| 🟡 Major | UX | Provenance: file-frontmatter-only in v1 (PW-INV-1); `by` must not overload `Stamp.By` semantics | accepted-with-note | R5/R1: new `generated_by` field; struct extension (not parser tolerance); no DB column in v1 (curator reads files) |
| 🟡 Major | TechDebt | Two draft lifecycles (ADR-0060 contextual drafts vs skills/ drafts) | resolved-by-design | R6: one draft contract (status draft→stable, FTS-indexed, query-excluded by default, validate-skipped); promote is the shared status-flip primitive |
| 🟡 Major | TechDebt | `StatusResult.Pending` overloaded (plan jobs) for pending drafts | resolved-by-design | R6: separate `Drafts` field |
| 🟡 Major | TechDebt | D1 contract cardinality test-pinned (12/agent-8); `projectwiki_review` placement | resolved-by-design | R7: re-baseline to 13/9/13; review is CLI-only (human-gated curation out of the agent working set); extract vs generate gate separation documented |
| 🟡 Major | TechDebt | Curator vs lint/validate overlap | resolved-by-design | R10: curator = composition — one semantics home per check |
| 🟡 Major | SRE | Concurrent extract spawns race (compaction + shutdown + correction) | resolved-by-design | R5/R14 data flow: single-flight per project, latest-wins, timer cancellation |
| 🟡 Major | SRE | Mid-session MCP death: no behavior named | resolved-by-design | R9 error handling: loop-off for the session, counted, one `ui.notify`; no respawn (host owns lifecycle — ADR-0067) |
| 🟡 Major | Cost | Cost numbers unpriced | resolved-by-design | Implementation Decisions: priced cost model (0–3 calls/session target, measured; digest ≤2K chars; O(V) incremental index; draft-retention via on-demand curator) |
| 🟡 Major | MigrationCompat | pi `runEphemeralTurn` unverified in skill-pi 1.1.0 | accepted-with-note | R4: execution-time assumption; degrade = loop-off on pi (not session-breaking); omp is the live-verified host; verification is a blueprint task |
| 🟡 Major | SRE | Event trail lacks session/trigger/failure dimensions | resolved-by-design | R14: `extract_*` event family with session id + trigger + reason |
| 🟡 Major | Contrarian | Hermes convergence: `plugins/hermes/` already carries v2 distill hooks ("The distill pass is a v2 feature") | accepted-with-note | ADR-0068 decision 14: `projectwiki extract` is the engine-side distill the Hermes provider's v2 seam will shell out to; Hermes v2 out of scope |
| 🟡 Major | MigrationCompat | Sequencing on unsettled seams (host-plugins still draft; plans against v1 `index.ts` the memory change rewrites) | accepted-with-note | R11/R12 data flow: serial order pinned — memory → host-plugins → this change; shared seam lands with this change |
| 🟢 Nit | UX | No `projectwiki_digest` / `projectwiki_curate` MCP tools | accepted (by design) | CLI-only curation surface (R7); digest consumed by the extension, not the agent |
| 🟢 Nit | UX | `by` frontmatter naming | resolved-by-design | `generated_by` (new field) |
| 🟢 Nit | Cost | Instructions block 2,596 runes exceeds the ≤2,048 budget (pre-existing) | resolved-by-design | R13: trim included in the operating-manual update |
| 🟢 Nit | UX | Auto-writes appear as untracked git files; no git affordance linking event → pending draft | accepted-with-note | Mitigated: pending list in digest + `status` drafts field + `extract_*` events; git commit stays a human act |
| 🟢 Nit | UX | Digest pending list goes stale until compaction (snapshot vs freshness) | accepted-with-note | R9: the accepted cost of KV-cache stability; compaction re-render is the freshness bound |

**Overall verdict:** survived-with-notes.

Gate: all 🔴 Blocker findings dispositioned (user gate, 2026-11-19: B1
resolved / B2 resolved / B3 resolved) before the spec write. ✅

## Testing Decisions

- **What makes a good test:** external behavior at the verb seam (CLI
  stdout/exit + vault diff + event rows), the pinned MCP contract, and
  extension E2E (spawn, drop counting, digest stability) — not
  frontmatter byte-layout or FTS internals.
- **Modules to test:**
  - Go: extract/review/digest/curate verbs (fixture vaults); write-path
    confinement (R2); draft visibility rules (R6); Skill registration
    (R1); contract re-baseline (R7); lint provenance/contested rules;
    event emission (R14). Prior art: the existing projectwiki verb tests
    (fixture-vault + `projectwiki status` assertions) and the memory
    family's store-busy drop tests.
  - CLI: installer family wiki (fixture home, tracked-set, byte-identical
    restore). Prior art: ADR-0067's installer tests (in flight).
  - TS: extension unit (trigger heuristic table, drop counting, digest
    assembly) + one E2E on omp (live host) covering trigger → side-turn
    → draft → review promote → digest visibility.
- **Edge cases:** busy store / locked DB (drop receipt), malformed
  analyzer JSON, MCP-down at start and mid-session, concurrent trigger
  race (single-flight), 200+-line skill page (split deferred to curator,
  not the verb), existing user-origin page in `skills/` (untouched),
  `validate --strict` with drafts present (passes), promote then reject
  (demote, not delete).

## Impact on Global Docs

- `.skillgrid/projectwiki/reference/00-prd.md`: None this round (the PRD
  remains installer-first; the wiki plugin is engine + host plugin
  surface, tracked in ADR-0068).
- `.skillgrid/projectwiki/ASSUMPTIONS.md`: ADR-0068 row added to § LOCKED;
  "Highest sequence in use" note → 0068.
- `.skillgrid/projectwiki/ARCHITECTURE.md`: component note — the
  `plugins/wiki/pi-omp/` generalized extension + the four new
  projectwiki verbs (digest/extract/review/curate) as an optional
  addendum; exact section decided at blueprint time (file is a
  component map; the verbs live inside the existing projectwiki
  component).
- `.skillgrid/projectwiki/glossary/02-technical-terms.md`: new terms —
  Wiki Extraction Loop, Significance Gate, Draft Skill Page, Drop
  Receipt, Wiki Peer Review, Operating-Manual Skill.

## Clarity Report

| Dimension           | Score | Min  | Status | Notes                              |
|---------------------|-------|------|--------|------------------------------------|
| Goal Clarity        | 0.90  | 0.75 | ✅     | Substrate + artifact model + loop pinned across 12 rounds |
| Boundary Clarity    | 0.85  | 0.70 | ✅     | Scope in/out explicit; non-goals listed; serial order pinned |
| Constraint Clarity  | 0.80  | 0.65 | ✅     | ADR-0068 conventions + rules in force; pi side-turn assumption flagged |
| Acceptance Criteria | 0.85  | 0.70 | ✅     | 14 falsifiable requirements, each with pass/fail check + BDD pointer |
| **Clarity**         | 0.20  | ≤0.20| ✅     | 1 − min dimension; boundary case on Constraint (pi assumption tracked) |

**Interview log:**

| Round | Question summary         | Decision locked                    |
|-------|-------------------------|------------------------------------|
| 1     | Substrate + artifact model | Projectwiki KB; "a skill that works into the knowledgebase" (confirmed R13 in round 12) |
| 2     | Transport | host-MCP (ADR-0067 convention; existing `mnemonic-projectwiki` server) |
| 3     | Triggers | periodic + compaction + shutdown + correction + agent-initiated (user-validated) |
| 4     | Invariants | auto-writes: draft-only, skills/ only (user-confirmed) |
| 5     | Approval | `/wiki-review` user gate |
| 6     | Review semantics | confirm with diff before promote |
| 7     | Artifact shape | everything becomes a skill page (`kind` field distinguishes) |
| 8     | Injection | session-start digest (stable pages + flagged pending) |
| 9     | References: pi-memory / pi-hermes / #2189 | pi-hermes capability shape (background review, correction detection, procedural skills, consolidation) on the wiki substrate |
| 10    | References: llm-wiki-skill / hermes research-llm-wiki | E1 contradiction marking, E2 page thresholds, E3 digest = orientation read, E4 shipped operating-manual skill, E5 non-goals (companion tooling) |
| 11    | Shootdown gate (B1/B2/B3) | all three resolved by design revision (significance gate; review affordances; verb-level drop boundary) |
| 12    | Q9 shipped skill | yes — shipped operating-manual skill + loop-written skill pages |

## Open Questions & Assumptions

- **Assumption A1:** pi (skill-pi 1.1.0) exposes the same side-turn API
  as omp (`runEphemeralTurn`-equivalent). Unverified; degrade = loop-off
  on pi (drop-counted, one `ui.notify`). Verification is a blueprint
  task (R4).
- **Assumption A2:** side-turn prefix-cache behavior (does the ephemeral
  call reuse the session's cached prefix?) is unverified on omp; the
  cost model (ADR-0068 consequences) prices the call as context-sized
  worst-case; a local session model makes this near-zero.
- **Assumption A3:** serial ordering — this change executes after both
  in-flight changes (their pi-omp seam, installer boundary, and prompt
  capture are prerequisites). If those slip, the shared-base refactor
  (R12) lands on the then-current extension source.
- **Assumption A4:** git commits of auto-written drafts remain a human
  act (accepted-with-note, shootdown UX).
- **Assumption A5:** the digest's pending list may be stale until
  compaction (accepted cost of KV-cache stability, R9).
- **Question:** none open — all interview questions answered, all
  blockers dispositioned.

## Decisions (ADR)

- `.skillgrid/projectwiki/decisions/04-adr-0068-omp-wiki-hermes-extraction.md` —
  the architecture: single auto zone (skills/, draft-only, skill-shaped),
  one draft contract aligned with ADR-0060, significance-gated
  triggers, host side-turn analyzer, verb-level fail-open boundary,
  generalized pi-omp extension (ADR-0067 revisit #2 fired), MCP 8→9,
  curator-as-composition, file-frontmatter provenance, installer
  family `wiki`, shipped operating-manual skill, serial sequencing.
  User-accepted at the shootdown gate (2026-11-19).

## Terms

Pointers to `.skillgrid/projectwiki/glossary/02-technical-terms.md`
(new entries added by this change):

- **Wiki Extraction Loop** — the Hermes-style session→wiki extraction
  pipeline (trigger → significance gate → side-turn analyzer → extract
  verb → draft → review).
- **Significance Gate** — the heuristic that decides whether a
  cadence-scheduled window is substantive enough to spend an LLM call
  on (correction / ≥3 edits / ≥8 calls of ≥3 tool types / remember-cue).
- **Draft Skill Page** — a `status: draft`, `type: Skill` page in
  `skills/` awaiting `/wiki-review` disposition.
- **Drop Receipt** — the structured `{"dropped":true,"reason":…}`
  stdout contract of the extract verb: the fail-open boundary.
- **Wiki Peer Review** — the extract verb's FTS-based check for existing
  coverage / conflicts before writing (update vs create vs
  `contradictions:`).
- **Operating-Manual Skill** — the shipped `projectwiki` agent skill:
  how the agent works the KB (Karpathy llm-wiki pattern).
