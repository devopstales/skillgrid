# 003 — Generate (the pipeline, the graduate-rule, incremental entity-targeted)

**Status:** DRAFT · **Owner:** knowledge-base change
**Depends on:** `001-structure` (page format), `002-sources` (the adapters),
`004-index` (the `kb_run_state` checkpoint + `kb_page_baselines` it advances — note:
`003` ↔ `004` are co-designed and intentionally circular on the checkpoint; they share
`kb_run_state`), the memory change (`SaveEvent`, `importance`), `003-intelligence`
(the LLM seam).
**Reference:** openwiki incremental update (no-op clean runs, resumable checkpoint),
pi-llm-wiki `wiki_generate`, the eager-per-save handoff (OPEN-11).

---

## 1. Capability

Generation is how **source material becomes pages**. This spec defines the pipeline,
the **graduate-rule** (which facts become pages), the LLM synthesis, and the two
properties that make it safe to run eagerly on every save: **incremental** (touches only
what changed) and **a no-op when clean** (writes nothing if the input is unchanged).

The trigger is the memory `SaveEvent`. The pipeline is:

```
SaveEvent (memory, eager)
   │
   ▼
1. RESOLVE     read the fact live by sync_id (+ its judged relations, conflicts,
               supersedes) from the memory store. Read the affected code spans /
               graph nodes live.
   ▼
2. GRADUATE    the graduate-rule (003 §2): does this fact graduate to a page?
               (importance + type + novelty). If not → NO-OP (changed: 0).
   ▼
3. TARGET      entity-targeting: which EXISTING pages does this fact affect?
               (by topic_key/entity, by anchors, by wikilinks). This is the
               "incremental" step — only these pages are regenerated.
   ▼
4. SYNTHESIZE  LLM synthesis: for each target page, merge the new fact into the page
               body, update sources:/anchors:/claims: as needed, preserve the page's
               existing still-valid content. [degraded: append-only, no LLM merge]
   ▼
5. REINDEX     advance the index checkpoint; rebuild the affected pages' FTS rows in
               the shared DB (004-index).
   ▼
6. NO-OP CHECK if no target page's content actually changed → changed: 0, no write,
               checkpoint still advances (so the run is recorded).
```

**Steps 1–6 are local and idempotent.** The LLM call (step 4) is the only expensive
step; everything else is cheap reads + writes. A flaky LLM degrades to append-only
(INV-3, `degraded: true`) — it never blocks the save (the event bus isolates failures,
`002-sources` §2.1).

## 2. The graduate-rule (which facts become pages)

The graduate-rule is the **filter between the raw store and the brain**. It answers:
"is this fact durable, important, and novel enough to be a page?" It reads:

- **`importance`** (memory `002-schema` §2.2.3, carried on the `SaveEvent`). Durable
  types (decision / architecture / convention / config) score higher than transient
  (session_summary / discovery). This is the primary gate.
- **`type`** (memory `002-schema` §2.2.1). A `session_summary` or `preference` is
  rarely a page; a `decision` / `architecture` / `bugfix` (with root-cause) is a
  strong candidate. The type maps to a page type:
  - `decision` → `decisions/`
  - `architecture` → `architecture/` (or `workflows/` if it describes a flow)
  - `bugfix` (with root-cause) → `gotchas/`
  - `convention` / `config` → `invariants/` (a must-hold contract)
  - `discovery` / `learning` → `gotchas/` or `glossary/` (depending on content)
- **novelty.** A fact that updates an existing page's topic (same `topic_key` / entity)
  does not create a new page — it **updates** the target (step 3). A genuinely new
  entity (new `topic_key` family, no existing page) may create a new page.
- **soft budget** (KB-O5): the graduate-rule consults `kb_status`'s coverage so that
  generation does not unboundedly fan out; it prefers updating over creating.

**The rule is deterministic, the merge is LLM.** The graduate-rule (step 2) is a
deterministic gate (importance threshold + type map + novelty check). The *synthesis*
(step 4) is LLM. This is INV-3 applied: the cheap, safe decision is deterministic; the
expensive, judgment call is LLM with an explicit degraded floor.

**Degraded floor:** when the LLM seam is absent, the graduate-rule still runs (it is
deterministic), but step 4 degrades to **append-only** — the new fact is appended to the
target page as a dated, cited entry rather than merged. The page stays correct (the fact
is there, cited); it is not as clean as a merged page. `degraded: true` is set on the
result.

## 3. Incremental, entity-targeted (the eager property)

The KB does **not** regenerate the whole vault on every save. It regenerates **only the
pages the fact affects**:

- **By entity / topic_key:** the `SaveEvent` carries `entity` + `topic_key`. Pages whose
  frontmatter `sources:` cite that `sync_id`, or whose slug/aliases match the entity, are
  the targets.
- **By anchor:** pages whose `anchors:` point at code spans the fact touched are targets.
- **By wikilink:** pages that link to an affected page are *re-checked* (their
  cross-references may need a touch), but not fully regenerated unless their own content
  changed.

This is what makes **eager-per-save** (OPEN-11) affordable: a save that touches one
module regenerates only that module's pages, not the vault.

**A clean run is a no-op (KB-INV-4) — and it is *proven*, not assumed.** A `kb_generate`
(incremental) run skips the LLM synthesis and reports `changed: 0` only when **all four
layers** hold (openwiki's layered no-op proof, adapted):

1. **Source-clean (already-processed).** The triggering `SaveEvent` fact has not already
   been processed. "Processed" is keyed on **the set of `sync_id`s** (the
   `kb_run_state.processed` set, §5.1) — *not* on a sequence number, because the memory
   `SaveEvent` carries no `seq` (only `sync_id`/`project`/`scope`/`type`/`topic_key`/
   `entity`/`importance`/`degraded`). For an **upsert** of an existing fact (same
   `sync_id`, new content), "processed" additionally requires the fact's current
   `normalized_hash` to match the hash recorded when it was last processed — a changed
   fact is *not* already-processed. *And* the set of facts the graduate-rule would act
   on is empty (nothing new, nothing changed).
2. **No claim issues (read-only).** Re-resolve every pinned anchor/claim on the affected
   pages (the `006-validate` pin-check, **read-side**). Zero `stale` / `unresolved`.
   This step is **read-only: it does NOT append a `verified` stamp** (stamping happens
   only on an explicit `kb_validate` call, §`006-validate` §4). A clean source set
   cannot hide a stale pin — the pin-check runs *before* the no-op decision.
3. **Full baseline coverage.** Every affected page has a `kb_page_baselines` entry (§5.2)
   whose `source_fingerprint` matches the **current** source fingerprint (a page with no
   baseline, or a stale baseline, is forced to re-derive, never a no-op).
4. **Source-fingerprint re-check.** Take the **source fingerprint** (SHA-256 over the
   *source inputs only* — the affected memory facts + code spans; **excluding** the
   vault's own `index/**` files and `kb_*` rows, so the run's own writes can never
   perturb it), run the run's metadata writes (checkpoint, baseline manifest), then
   re-take the source fingerprint **twice**. If the re-taken fingerprint differs from
   the first (a source fact or code span moved concurrently), the no-op is **void**: the
   run falls through to synthesis (step 4) with the fresh source state. If it is
   byte-stable, the no-op holds.

Only then does the run write `changed: 0` and advance the checkpoint (writing the
`sync_id` into `processed`). This is what makes "no-op" *provable*: the vault is not
churned by idle runs, git sees no spurious diffs, and a no-op is never the result of
*skipping* the checks — it is the result of the checks passing. If any layer fails, the
run proceeds to synthesis (step 4).

> **Why the source fingerprint excludes the vault's own writes:** the no-op proof must
> be able to distinguish "the *source* is unchanged" from "the *KB* changed." If the
> fingerprint covered the KB's own `index/**`/`kb_*` rows, every `kb_generate` that
> writes anything would invalidate every page's baseline (making layer 3 vacuous) and
> the re-check (layer 4) would be trivially flaky. Excluding them keeps the fingerprint
> a pure *source* identity — the same stance as openwiki, which excludes `openwiki/`
> from its source fingerprint.

## 4. The LLM synthesis (step 4)

For each target page, the LLM receives:

- the page's current body + frontmatter,
- the new fact (content, type, importance, `sync_id`),
- the related judged relations (conflicts, supersedes — so the LLM can flag a
  contradiction, golden rule 6),
- the relevant code spans (live) for grounding.

The LLM returns an **updated page** (body + frontmatter). The contract:

- **Preserve still-valid content.** The LLM merges, not rewrites. Existing claims that
  still hold are kept; the new fact is integrated.
- **Update provenance.** The new `sync_id` is appended to `sources:`. If the fact
  supersedes an old claim, the old claim is moved to a `## Superseded` section (or the
  `superseded_by` relation is cited) — never silently deleted (INV-1, additive).
- **Flag contradictions.** If the new fact conflicts with existing page content, the LLM
  documents both sides (golden rule 6) and sets a `## Contradiction` marker that
  `kb_lint` surfaces.
- **One atomic change per page** (session discipline). A single generation run updates
  a page in one write; it does not emit a stream of partial edits.

**Determinism guard:** the LLM output is validated against the page template (frontmatter
present, valid `type`, `sources:` well-formed, wikilinks resolvable). A malformed output
is rejected and the run degrades (append-only) for that page — it does not write a
corrupt page.

**Per-run model override (012):** `kb_generate` takes an optional `model` parameter
(`provider/id`). Precedence: explicit param > configured model > session model. An
unknown ref degrades gracefully (the run continues on the configured/session model —
the append-only floor above is unchanged, INV-3). The choice is per-run, never
persisted into `config.json`.

## 5. Resumability: the run checkpoint + the per-page baseline manifest

Two durable, derived artifacts cooperate (openwiki's `.run.json` + `.page-manifest.json`
model, adapted). Both are under the derived `index/` layer (KB-INV-3) — rebuilt, never
hand-edited. Neither is the FTS index (that is `kb_pages_fts`, `004-index`).

### 5.1 The run checkpoint (`index/last-update.json` / `kb_run_state`)

`kb_generate` and `kb_index` share a **resumable run checkpoint**: it records, per run,
the `runId`, the phase (`planning`/`generating`), the source fingerprint, **the set of
processed `sync_id`s** (keyed on `sync_id` + the fact `normalized_hash` at processing
time — see layer 1, §3; there is no `seq` because the `SaveEvent` carries none), and the
ordered page-job queue (each page with `status: pending|skipped|complete`). If a
generation run is interrupted, the next run **resumes** from the checkpoint (it does not
reprocess already-handled facts).

**Resume rules** (openwiki):
- A pending job whose **page baseline** (§5.2) proves completion *for this runId* is
  promoted to `complete` — this is the "crash after the page write, before the
  checkpoint" window. A `complete` job missing its baseline proof is re-proven or fails.
- If the **source fingerprint drifted** since the checkpoint (a fact or code span moved
  mid-run), the phase resets to `planning` and the plan is deleted; completed pages stay
  durable. A no-drift resume keeps the plan and reconciles.
- **On a clean finish the checkpoint row persists** (it is the `kb_status` "last run"
  record) but its **active plan is cleared** (`phase → idle`, `plan` emptied). On a
  failure the checkpoint **survives intact** so the next `kb_generate` can reconstruct
  and retry — never discard resumable work. The *active-run* marker (the non-idle
  `phase`) is what is "deleted last"; the row itself is durable history.

> **Single authoritative store: `kb_run_state` (the DB row).** `index/last-update.json`
> is the **file projection** of the same checkpoint (written by `kb_index`/`kb_generate`
> for git-diff visibility and human inspection). The **DB row is authoritative**; the
> file is regenerated from it and is never read back to decide a no-op. This removes the
> dual-copy consistency question — there is one source of truth, one projection.

### 5.2 The per-page baseline manifest (`index/.page-manifest.json` / `kb_page_baselines`)

Each committed page earns a **baseline entry** recording the exact source it was
checked against:

```json
{
  "invariants/enterprise-grace": {
    "sourceFingerprint": "sha256:…",   // the facts + code spans this page was derived from
    "pageVersion": "sha256:…",          // hash of the exact markdown bytes verified
    "completedBy": "mnemonic/2.0",
    "completedRunId": "…"
  }
}
```

A page entry is recorded **only** when the page's markdown and its pin-check are
**mutually consistent** (the pin-check passed and `pageVersion` matches the written
bytes) — a page cannot earn a baseline it hasn't proven. On a future run, a page whose
baseline fingerprint no longer matches the live source is **re-derived**; a page whose
baseline still matches is a **no-op candidate** (layer 3 of the no-op proof, §3).

**Fast-forward:** because the baseline `source_fingerprint` covers **only source inputs**
(memory facts + code spans, §3 layer 4 note), a change to the vault's *own* generated
state (the `index/**` files, the `kb_*` rows, or the engine-owned frontmatter fields
`generated`/`verified`/`stale_after`) does **not** change a page's source fingerprint.
Such a change therefore neither invalidates the baseline nor requires a re-derive — the
baseline simply remains valid. (This is what keeps KB-only churn from accumulating
review debt; it is a direct consequence of the source-only fingerprint, not a separate
rule.)

> **The two artifacts are distinct.** The run checkpoint makes a *run* resumable
> (where the run got to); the page manifest makes a *page* provably current (what it was
> derived from). Both are derived and rebuildable; neither is the FTS index.

## 6. Full vs incremental

| Mode | Trigger | What it does |
|---|---|---|
| **incremental** (default, eager) | a `SaveEvent` | regenerates only the affected pages (steps 2–6, targeted). |
| **full** (`kb_generate --full`) | explicit (CLI / `kb_generate` with `full: true`) | re-derives the **entire** vault from all sources. Used after a schema change, a manual page edit that desynced the index, or a rebuild. |

A **full** run still respects the graduate-rule (it does not create pages for
non-graduate facts) and is still a no-op if nothing changed (KB-INV-4). It is the
"resync" path: if a user hand-edited a page and the index drifted, `kb_generate --full`
reconciles.

## 7. Invariants

INV-3 (LLM-first, degraded floor), INV-1 (provenance preserved on merge), INV-4 (files
are truth — generation writes files; the index follows), KB-INV-2 (grounded pages stay
grounded — synthesis updates `anchors:`/`claims:`), KB-INV-4 (no-op when clean).

## 8. Open items

None new. The graduate-rule thresholds (the importance cutoff per type) are the one
tunable — defaulted in `008-transport` config, refined during the review.
