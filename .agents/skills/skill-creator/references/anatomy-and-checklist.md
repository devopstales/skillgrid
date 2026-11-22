# Anatomy Reference and Verification Checklist

## 10. Anatomy Reference

```
skill-name/
├── SKILL.md              ← required: frontmatter + instructions (< 500 lines)
├── scripts/              ← optional: executable code (no token cost when loaded)
│   └── <name>.mjs
├── references/           ← optional: docs loaded on demand
│   └── <topic>.md
├── templates/            ← optional: fill-in skeletons copied into the
│   └── <type>-template.md   user's workspace (one per output type)
└── assets/               ← optional: static files used in output but not
                            copied (icons, reference data)
```

Frontmatter:
```yaml
---
name: skill-name
description: What it does (third person). Use when {trigger}.
license: MIT
metadata:
  author: name
  version: "1.0"
  part-of: skillgrid
---
```

### Router and bucket index

When user-invoked skills multiply past what a human remembers, the cognitive
load is cured by a **router skill**: one user-invoked skill that names the
others and when to reach for each. It can only *hint*, never *fire* them
(user-invoked skills have no description, so nothing but the human can reach
them). Keep the router's map current whenever a user-reachable skill is added,
removed, renamed, or re-scoped — a skill the router never mentions, or a stale
one it still points to, is a router that lies.

Each domain grouping (a bucket, if the repo organizes skills into one) carries
a one-line-per-skill `README.md` index: the skill name linked to its `SKILL.md`
and a single-line description, grouped by invocation. The index is routing, not
restating — one line per skill, no prose.

### 10a. Smart-Entry Routers (state-aware routing)

A **static router** (§10 "Router and bucket index") lists skills and their
triggers. A **smart-entry router** goes further: it inspects *project state*
and routes to the right next action, so the user never has to decide which
skill to run.

**When to add one:** the library has ≥ 6 model-invoked skills that form a
pipeline (e.g. brainstorm → blueprint → slicing → execute → test → ship) AND
users regularly ask "what should I run now?" The router cures the
*sequencing* problem, not the *discovery* problem — the skill descriptions
already handle discovery; the router handles "given where I am, what's next?"

**The pattern:** one model-invoked skill (e.g. `skillgrid:next`) whose
description fires on "what's next", "where am I", "continue", "resume". Its
body is a **state detector + dispatcher**, not a procedure:

1. **Detect state.** Read the project's working directory (`.skillgrid/`
   artifacts: `STATE.md`, `ROADMAP.md`, `SUMMARY.md`, ticket status) to
   determine what phase the project is in. This is a deterministic step —
   a script that reads the artifacts and prints a state verdict
   (`no-work`, `in-blueprint`, `in-execution`, `in-verification`,
   `ready-to-ship`).
2. **Route.** A Decision Gates table maps each state verdict to the
   appropriate next skill. One row per state; the action is "invoke
   `skillgrid:<name>`".
3. **Dispatch.** The router calls exactly one skill and stops. It never
   does the work itself.

**What stays out of the router:** the actual work. The router is a
*switch statement*, not a workflow. If the router body exceeds ~100 lines,
it has absorbed procedure that belongs in the target skills.

**State detection is a script.** The router's step 1 (read artifacts,
determine state) is deterministic — it gets a script
(`scripts/detect-state.mjs`) with a JSON verdict on stdout. The router's
SKILL.md is then: frontmatter, the script call, the Decision Gates table
(state → skill), and the dispatch instruction. ~50 lines total.

**Anti-pattern:** a router that re-implements the target skill's logic
("if in execution, check the task list and run the next task"). That is the
target skill's job. The router's job is *which skill to invoke*, not *what
the skill should do*.

**Source:** GSD-core's `/gsd-next` — a state-aware front door that reads
project state via `gsd-tools smart-entry --json`, presents a
situation-appropriate menu, and dispatches exactly one GSD command. The
pattern works because the routing logic is separate from the work logic,
so adding a new skill to the pipeline is a one-row Decision Gates addition,
not a router rewrite.

---

## 11. Verification Checklist

- [ ] **TDD Gate:** the eval prompts were written *before* the body (new) or
      the edit was reproduced against the prior version (edit); the agent was
      watched fail without the change and comply with it. No untested skill is
      deployed
- [ ] **Trigger:** invocation mode is a deliberate choice (T1)
- [ ] **Trigger:** description is third-person, trigger-specific, ≤ 1024
      chars, no workflow steps, and passes the pointer-quality rules — leading
      word front-loaded, one trigger per branch, no identity restating the
      body (T2)
- [ ] **Structure:** every step classified as deterministic or judgment (S1)
- [ ] **Structure:** every deterministic step has a tested script (S2)
- [ ] **Structure:** single-branch reference is behind a context pointer (S3)
- [ ] **Structure:** scripts have documented exit-code contracts (S4)
- [ ] **Structure:** operational skill carries the spine sections or
      the skip is deliberate and stated (S5)
- [ ] **Steering:** at least one leading word, repeated consistently, and it
      passes the no-op + pretrain-first tests (St1, per
      `_shared/craft/leading-words.md`)
- [ ] **Steering:** legwork split considered where needed (St2)
- [ ] **Interaction:** gates follow the EGM format — one question, sole tool
      call, no (Recommended) prefix, open-ended vs single-select correctly
      applied, chat summary precedes gate (U1)
- [ ] **Interaction:** multi-step workflows carry a Mermaid diagram with gate
      diamonds, labeled option edges, explicit loop-backs, matching prose
      (U2)
- [ ] **Pruning:** no-ops, sediment, duplication pass the deletion test (P1)
- [ ] **Pruning:** skill-size-budget.mjs exits 0 (P2)
- [ ] **Frontmatter:** check-frontmatter.mjs check . exits 0 — field set,
      order, name=dir (kebab-case), description <= 1024, metadata block with
      quoted semver + part-of: skillgrid (F1)
- [ ] **Audit:** all 10 rules pass (no FAILs)
- [ ] **Model coverage:** if the skill targets multiple models, the
      `## Model Notes` section names per-model behavior differences
      (Haiku / Sonnet / Opus) — or the skill is tested with the
      models it will actually run on and the checklist states which
- [ ] Files > 100 lines have a TOC
- [ ] No domain explanations, no "you are X" preamble
- [ ] Multi-step skills have a copyable checklist with go-back lines
- [ ] Verify-then-repeat loop for consequential outputs
- [ ] Prerequisites section for external tools
- [ ] Critical "never break" rules identified as hook candidates
- [ ] No "English if-statements" in prose
- [ ] No content duplicated from another skill
- [ ] **Lifecycle:** frontmatter `metadata` carries the lifecycle fields the
      library supports — `created_by` provenance, `related_skills` adjacency,
      and (where the harness maintains one) a usage sidecar entry (§12)
- [ ] **Lifecycle:** the skill is findable in the library's curation scope —
      a hand-written skill is registered as curated; an agent-generated skill
      is flagged as agent-created so the curator owns its fate (LC1)

---

## 12. Skill Lifecycle

A skill is alive: it is used, it rots, it is archived. A library of skills
without lifecycle data is a museum — every skill looks equally current
forever, and the catalog bloats until the description index no longer helps
the agent trigger the right one. This phase owns the *library-level* concerns
the four authoring phases don't cover: provenance, adjacency, and usage.
It is the port of the Hermes Agent curator model (Nous Research), applied to
Skillgrid's flat layout.

### Provenance (`created_by`)

Every skill records who made it, in `metadata`:

```yaml
metadata:
  author: devopstales        # hand-written, curated — the default
  # author: agent            # generated by the agent in a session
  # created_by: agent        # explicit machine-provenance marker
  based_on: hermes-agent     # ported/adapted from a named upstream (existing rule)
```

Why it matters: the curation loop must not delete what a human wrote. A
curator that touches a hand-written skill on a stale-usage signal is a
curator that surprises its owner. The rule: **curation transitions
(stale → archive) apply only to agent-created skills.** Bundled and
hand-written skills are off-limits for auto-transition; the human edits or
deletes them deliberately. When porting a skill from another system, keep
the upstream provenance (`based_on:`) — attribution is what makes a port
auditable.

**Decision gate:** a skill with no provenance marker is treated as
hand-written (the conservative default). The curator never archives a
provenance-less skill.

### Adjacency (`related_skills`)

Skills are not isolated nodes. A `related_skills:` list in `metadata` names
the skills that compose with this one, so the agent (and a reviewer) can see
the neighborhood without reading every body:

```yaml
metadata:
  related_skills: [test-driven-development, writing-plans, requesting-code-review]
```

Rules:

- **Name only, no restatement.** The adjacency is a pointer; the *why* of the
  composition lives in the skill's own prose (or `_shared/`), never duplicated
  into the list. A `related_skills` entry that restates a section of the
  target is duplication and will drift.
- **Forward-looking and bidirectional in effect, unidirectional in form.**
  List the skills you call or that call you. Do not require the target to
  reciprocate — a one-directional edge is fine and cheap.
- **Curate the graph with the skill.** When a skill is renamed, removed, or
  re-scoped, sweep `related_skills` entries that pointed at it. A stale
  adjacency is a router that lies (same rule as the router index, §10).
- **Do not over-link.** `related_skills` is for *composition* (A feeds B, B
  gates A), not topical similarity. Two skills both about "testing" are not
  automatically related; they are related if one's output is the other's
  input.

### Usage telemetry (the sidecar)

A library-level `.usage.json` (or per-skill sidecar) records, per skill:

- `use_count` — times the skill was loaded into an agent context
- `view_count` — times it was read without triggering (browsed)
- `patch_count` — times its SKILL.md was edited
- `last_activity_at` — timestamp of the most recent load or patch
- `state` — `active` | `stale` | `archived`
- `pinned` — boolean, exempt from all auto-transitions

The harness writes `use_count` / `last_activity_at` when it loads a skill;
the curator reads them on its interval. **The skill itself does not write
its own telemetry** — that would be self-grading. Telemetry is an external
signal about the library, not a field the skill maintains.

Skillgrid does not ship a curator daemon; the sidecar is *optional* and the
format is the contract. If the library has no harness writing the sidecar,
`last_activity_at` is the only honest field (update it on patch), and the
stale/archive transitions below are manual, run when the library is reviewed.

### Curation transitions (never delete)

The lifecycle moves a skill through states. **The maximum destructive action
is archive, never delete.** Archive means move to a known location (e.g.
`_archive/`) or set `state: archived` — the skill is restorable.

| Transition | Signal | Applies to | Guard |
|---|---|---|---|
| `active` → `stale` | `last_activity_at` older than `stale_after_days` (default 90) | agent-created only | pinned skills exempt |
| `stale` → `archived` | `stale` for longer than `archive_after_days` (default 180) | agent-created only | pinned skills exempt; backup first |
| `archived` → `active` | human restores, or re-use detected | any | bump `last_activity_at` |

**Before any archive: a backup.** A tar.gz (or git tag) of the skill
directory taken immediately before the transition, so rollback is one step.
This is the invariant that makes "never delete" honest — you can archive
aggressively if the backup exists, and you must not archive if it doesn't.

**Pinned skills are exempt from every auto-transition and every review
pass.** Pinning is the owner's explicit "keep this even if it looks dead"
signal — a skill that is rarely used but load-bearing (a setup skill, a
safety skill) is pinned, not deleted.

### The curation interval

When the library grows past what a human reviews in one sitting (rule of
thumb: > 30 skills, or a single review takes > 30 minutes), the curation
becomes a scheduled pass, not an ad-hoc one:

1. **Status.** List every skill with its `state`, `last_activity_at`,
   `use_count`, provenance. The stale candidates jump out.
2. **Review candidates.** For each `active` skill that crossed the stale
   threshold and is agent-created, decide: still needed (bump or pin),
   genuinely dead (→ `stale`), or superseded (→ `archived`, name the
   replacement in `related_skills`).
3. **Backup, then transition.** Tar.gz, then move the state.
4. **Report.** One line per transition: skill, from→to, reason, backup path.

The interval is deliberate (weekly/monthly), not a background daemon. A
daemon that archives skills while you work is a curator that surprises you;
a scheduled pass you trigger is one you can watch.

### Lifecycle red flags

- A skill with `created_by` agent but no `last_activity_at` ever set (never
  used since creation — the RED eval never went GREEN)
- `related_skills` naming a skill that no longer exists
- An agent-created skill with 0 `use_count` that is not pinned and not
  archived (the curator never ran, or the skill is a zombie)
- A hand-written skill that the curation pass is about to auto-archive
  (provenance was not recorded — the curator is about to surprise its owner)
- An archive without a recorded backup path

### 12b. Surface Profiles (token-budget curation)

When the library's *description index* (every model-invoked skill's
`description` string, loaded into the agent's context on every request)
exceeds the context budget the agent can spend on routing, the fix is not
shorter descriptions — it is **fewer skills in the surface**.

**Surface = the set of skills whose descriptions are active in the agent's
context.** A skill outside the surface is invisible to the model (it can
still be invoked by name, but the model won't reach for it unprompted).

**The mechanism:** a profile file (e.g. `.agents/skills/.surface.json`)
declares which skill *clusters* are enabled. The harness (or a one-time
install script) reads the profile and stages only the enabled skills'
descriptions into the agent's context directory. Disabled skills remain on
disk but are not surfaced.

**Clusters are the unit of toggle, not individual skills.** A cluster is a
named group of related skills (e.g. `core-loop`: brainstorm, blueprint,
slicing, execute, test, ship; `research`: code-research, debug, prototype;
`security`: trivy, nuclei, wapiti, owasp). Clustering keeps the toggle
coarse — you disable a *capability area*, not a single skill. Individual
skills that are load-bearing regardless of profile are always-on (exempt
from clusters).

**Profile tiers:**

| Profile | Clusters enabled | Use case |
|---|---|---|
| `core` | core-loop only | Minimal footprint; the SDD pipeline |
| `standard` | core-loop + research + quality | Default; adds debugging and testing |
| `full` | all clusters | Maximum capability; token budget is not a concern |

**The token budget rule:** the sum of all active descriptions should not
exceed ~1% of the agent's context window (e.g. ~2000 tokens for a 200K
context). When it does, the agent is spending context on *routing* instead of
*doing work*. Check the budget at curation interval; drop clusters that are
rarely triggered (low `use_count` in the usage sidecar).

**What this is NOT:** an install/uninstall mechanism. The skills are all
present on disk; the profile controls *which ones the model sees*. Switching
profiles is a context change, not a file change — no reinstall, no
`npm i -g`.

**When to add this:** the library has ≥ 15 model-invoked skills AND the
description index exceeds the token budget. Below that, the overhead of a
profile mechanism exceeds the savings.

**Source:** GSD-core's `gsd-surface` — toggles 70+ skills across 10 clusters
via profiles (`core`, `standard`, `full`), with a token-cost summary per
profile. The state file (`.gsd-surface.json`) records the active profile and
disabled clusters; re-staging is in-place (no reinstall).

---

