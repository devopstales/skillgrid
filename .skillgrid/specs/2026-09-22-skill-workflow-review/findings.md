# Findings — skill-workflow-review

## Research: What do we think about the skillgrid workflow in this repo?

**Run:** deep preset — 6 parallel reviewers (pipeline/gates, skills hygiene, orchestration/memory, cold-start, evidence/TDD, adversarial failure modes) + fresh-context verifier + red-teamer. 2026-09-22.

### Executive summary

The workflow is **better-architected than most agent pipelines** — the gate polarity (hard FAIL vs. advisory review vs. human-owned CONCERNS), the file-first source-of-truth hierarchy, and the mechanical archive move are genuinely load-bearing and well-specified. The problems are not conceptual; they are **drift**: the repo is mid-migration (v1→v2 skills, Handoff Hub → session-events layer) and the config, the resume protocol, and the tasks.md format each froze at a different point in that migration.

**Top 3 findings that drive the verdict:**

1. **The newest change is the canary, and it's stuck.** `2026-09-21-session-events-layer` has 11/11 backlog tickets `done`, zero `- [x]` checkboxes in tasks.md (so subagent-execution's Status Guard can never compute `all_done`), and no `qa-report.md` (so ship's hard gate has nothing to read). The pipeline is currently deadlocked between execution and QA, and only a human reading `.backlog/` can see why. [C3, C6 verified]
2. **config.yaml is a stripped v1 fossil against v2 skills.** The v2 onboarding template has `commands`, `security.trivy`, `bdd`, `mnemonic`, `research`, `rules.tiers` blocks; the live config has none. Skills read those keys and fall to defaults — so `security.trivy.command` unset silently disables the security gate, and `rules.tiers.default` makes the tier machinery run on inference. One re-onboard (merge-mode) fixes most of it. [C2 verified, T1 red-team: SURVIVED with "least load" overstated — ticketing/testing/paths keys ARE consumed]
3. **The TDD default has three textual answers.** Config: `tdd: false` ("Standard default, strict opt-in"); skills: "`testing.tdd` is always `true`" / "TDD is always on"; onboarding: "TDD is non-negotiable." Red-team's fair reading: config's `false` means *strict*-TDD off, skills' "always on" means *basic* RED-GREEN-REFACTOR on — two different questions conflated. The `n` token claim did NOT survive verification (stale revision) — the live contradiction is the un-cross-referenced axis, not a typo. [C4 verified, T3 red-team: SURVIVED, revised]

**Biggest caveat:** much of the damage is migration debt, not design. The session-events layer (the change that fixed resume) is itself mid-flight — TICKET-08/09 landed, table drops uncommitted, `checkpoint.json` zombie still read by `gate-state.sh:19,65`. Several "bugs" close when this change ships.

### Dimension: Pipeline & gates

- **Fast-track gate semantics conflict across four files.** `config.yaml:104` says fast-track verify is `advisory-on-missing`; `ship:335` says CONCERNS → NO-GO by default; `qa:246-249`'s machine table has no fast-track exception — a waived change with a missing `acceptance.feature` renders CRITICAL → FAIL. A human at ship is the only release valve.
- **Three review gates, no single owner.** subagent-execution's internal Final Review (two-axis, escalate at 50+ lines), requesting-code-review (50+ lines → parallel-code-review *instead*), and ship Step 9 (its own fan-out if review was waived, with its own "≤2 files, <50 lines" skip). A branch can be reviewed twice; no document says which one ship's "Review Gate" reads.
- **The `rules:` block in config is v1 vocabulary** (propose/spec/verify phases no v2 skill names) and its `rules.archive` "PASS WITH WARNINGS" contradicts the live four-state gate (no such state exists; that word belongs to the archived `skillgrid.verify-result/v1` schema still found in archive reports).
- **User Gate is merged with the executor-choice in practice.** `slicing:186-192` and `writing-blueprints:500-508` both present "subagent vs inline" as the handoff; nothing records slice approval; subagent-execution:27 then says "do not pause." Red-team revision: the **fast-track waiver IS verified** (QA honors it at fast-track.md:40-41, ship renders the light variant) — only the User Gate itself is prose-only, confirmed by zero `user_gate` references in `hooks/`.
- **Stalled changes have no exit.** Two spec dirs carry `STATUS: stalled` (2026-09-04-hermes-memory, 2026-09-17-compact-search-output); no resume row handles stalled, no stale-folder detection. They will poison every future "newest dir" resume.
- **Overhead:** a standard change pays ~13 distinct steps (interviews, 2-3 approaches, ADR manifest, acceptance.feature, blueprint, Plan Review subagent, slicing, ticketing lifecycle, per-task loops, 11-step QA, internal review, requesting-code-review, possible third fan-out at ship, reflect). Ticketing's full discovery/duplicate/privacy/label lifecycle runs on markdown files in the *same* repo — four state stores (tracker, tasks.md, ledger, git blocks) for one change.

### Dimension: Skills hygiene

- **32/33 skills have an explicit "When NOT to use" naming the sibling that owns the excluded case** — the strongest disambiguation mechanism in the set, and it mostly works. 28/33 conform to the self-governed `skill-anatomy.md` spine with line budgets.
- **Duplication is the tax.** "50+ lines → parallel-code-review" in exactly 5 files [C8 verified]; "fix loop capped at 3 rounds" in 7 files (with a second, different 5-round cap for task fixes); "BDD is always on" re-announced in 14 files; the config-read paragraph in ~15; the `mem_session_summary` 6-section structure copy-pasted in mnemonic AND reflect while its declared source of truth (`_shared/conventions/mnemonic-memory.md`) carries none of it — and both copies say "all 5 sections" while enumerating 6.
- **Dangling references:** `skillgrid:code-review` (receiving-code-review:29, means requesting-code-review), `skillgrid:planning` (simple-execution:30, means writing-blueprints) — evidence of a rename that skipped a full-text sweep; `_shared/SKILL.md:16-23` omits `rigor-tiers.md` from its pointer list although the file exists [C10 verified]. No file-level broken refs otherwise (all `references/`, `researchers/`, `templates/` verified present).
- **Three competing "trivial" definitions:** ship (≤2 files AND <50 lines), fast-track.md (≤3 / ≤10 files), writing-blueprints classification (≤3 / ≤10). A 3-file change with behavior change is "small/full pipeline" to two of them and trivially skippable to ship.
- **No repo/global mirror drift** (disjoint skill sets), but two parallel orchestration vocabularies (Orca global skills vs using-skillgrid's anti-patterns) with zero cross-reference, and `archify` is a foreign object in the repo's skills root that fails 6 of 10 anatomy checklist items.

### Dimension: Orchestration & memory

- **Source-of-truth hierarchy is the most consistent rule in the grid** — in-repo file wins, mnemonic is a fallback index, dual-write with a `Degraded:` contract, stated verbatim in every place it could be needed. This is the workflow's backbone and it never drifts.
- **Compaction protocol is double-booked and contradictory.** mnemonic:212 says after compaction FIRST `mem_session_summary` (a write); resume:29,85 says re-orient from files first and explicitly distrusts summaries. Both claim the "right after compaction" slot. No precedence rule.
- **The documented producer of the `execution-progress` memory slot doesn't write it.** `memory.md:55` assigns it to simple/subagent-execution; neither contains a `mem_save`. Only resume (opportunistic) and parallel-execution (wave summaries) mirror it — so resume's fallback can be empty in exactly the mid-execution-death scenario it exists for.
- **Topic-key slot lists diverged in four copies** (6 / 8 / 7 / ~13 slots); research keys are three-way split (`/research`, `/findings`, `/deep-research`); `sdd-structure.md` claims "this file wins" and disagrees with everything.
- **simple-execution is the weakest link of the execution trio** — described as "the separate-session skill" in its description while its own When-NOT-to-use abandons it whenever subagents exist, and subagent-execution routes away to it precisely for separate sessions. Both skills point at each other.
- **Depth-1 and no-paraphrasing rules are stated in exactly one place** (using-skillgrid) and cited by none of the skills that must obey them; the per-task 15-line-summary loop is exactly the paraphrasing the rule forbids, salvaged only by on-disk report files.

### Dimension: Evidence & TDD

- **Verification has real teeth, and it's the best part.** `gate-stop.sh` (Stop hook, exit-2 block, hash-keyed 6-block loop guard) + `gate-state.sh --reverify` + `gate-lint.sh` make "evidence before claims" mechanical, not vibes. The QA four-state gate (PASS/CONCERNS/FAIL/WAIVED) is internally consistent; thresholds from config; "threshold 0 = disabled, cannot FAIL"; ABANDON = handoff, never pass.
- **But all 24 `EVIDENCE:` fields in the in-flight specs are `pending`** — the gate contract requires evidence bound to a run, and nothing blocks on binding (hooks check CHECK/EXPECT exit codes, not EVIDENCE state). The "fresh run this session" link is the weakest in the chain.
- **Ponytail's "runs automatically on every coding task" is a soft guarantee** — no hook, plugin, or SessionStart wiring mentions it (session-start.sh injects only the router). It's reliably active only inside simple-execution / the implementer prompt.
- **The spike→blueprint promotion path is fully specified and has zero in-repo precedent** — no `spikes/` dir, no `## Spike:` section ever written. Unproven contract.
- **`deep-research` is the most autonomous research skill and the only one missing the firewall + untrusted-input epistemics** [C9 verified: 0 matches for "firewall|untrusted" in its tree] — the longest-running agent has the biggest prompt-injection surface. One-line fix: inherit them the way code-research already does.

### Dimension: Cold start (fresh-agent walkthrough)

- **Router loading is the single point of failure — and the patch is unwired in this repo.** `hooks/session-start.sh` exists and hardcodes the router path, but there is no `.claude/` dir, no `.pi/`, no codex config in the repo [C7 verified — weaker than the original claim: not "wired only for Claude Code" but "wired for none of them in-repo"]. On any harness that doesn't auto-load the skill from its description, the agent starts router-less and just edits code.
- **`checkpoint.json` (in gitignored `sdd/`) says `remaining: "ship"`** — it is the actual live resume pointer, yet no skill reads it (the hook that derived it was removed in the events-layer change; `gate-state.sh` still reads it for spec discovery). Docs still call it "the resume pointer."
- **The session-events CLI resume path has a gap:** `resume:65` says run `skillgrid session <session-id>`, but a fresh session has no way to know the old session's UUID. The `[skillgrid-context]` commit block is the layer that actually works without it — yet resume treats both as complementary ("use both") and the "use both" step is the one that fails.

### Cross-dimension insights

- **One migration, three freeze points.** config.yaml froze at v1, the resume protocol froze at v1 tasks.md (checkboxes), the execution protocol froze at "ledger + session events" — and the repo currently contains v1-format and v2-format spec folders side by side. The "compound resume failure" is three single-file mismatches, each one-line fixable, that only look systemic because they coexist.
- **The enforcement split is the real design:** machine-checkable invariants (spec-before-code zone guard, `G<n>` gate satisfaction, TDD evidence) are hook-enforced; human checkpoints (User Gate, decision-before-apply) are prose-only. That's a defensible split — it just isn't named as one, which is why "mandatory but unenforced" reads as a bug.
- **The archive is the pipeline's graveyard with no reaper:** 4 of 16 archive folders are missing the artifacts (tasks.md, qa-report.md, report.md) that the pipeline's own reflect/resume gates read — so `reflect`'s terminal state would loop on half the archive.

### Contrary evidence (red-team, survived)

- **T1 (config fossil):** "least load" overstated — `ticketing.*`, `testing.runner`, `conventions.scratch_dir`, `rules.fast_track` ARE consumed and are the source of truth for those surfaces. Drift concentrates in optional sections where absence = disabled (trivy, mutation, coverage) — intended minimal-config semantics. Survived as "stripped variant, one re-onboard from fixed."
- **T2 (resume deadlock):** the repo is mid-migration, not mis-designed — the change that fixes resume (session-events-layer) is itself mid-flight; `checkpoint.json` is a zombie the in-flight TICKET-07 is deleting; single-in-flight-per-branch is the implicit contract. Survived as "real today, closes when the current change ships."
- **T3 (TDD muddle):** the `n` token was a misread of a stale revision; the live issue is an un-cross-referenced axis (basic-TDD-on vs strict-TDD-off), a documentation gap, not a three-way contradiction. Survived, narrowed.
- **T4 (unforced gates):** false for the fast-track waiver (QA + ship both read it) and the verification gates (hook-blocked); true only for the User Gate itself. Revised as above.

### Recommendations

1. **Finish the canary, in order:** run `qa` on `2026-09-21-session-events-layer` → commit its `qa-report.md` → ship → archive. This unblocks the pipeline and commits the session-events migrations (currently uncommitted per git status). Confidence: high — the blocker is procedural, not technical.
2. **Re-onboard in merge-mode** so the v2 template's missing blocks (`commands`, `security.trivy`, `bdd`, `mnemonic`, `research`, `rules.tiers`) land in config.yaml; set `rules.tiers.default: T2` explicitly. Confidence: high — onboarding's merge-mode exists for exactly this.
3. **Pick ONE done-signal.** Either tasks.md checkboxes (flip the Status Guard) or backlog status (repoint the Status Guard to `.backlog/`); retire the other from the guard text. Confidence: high — C3/C6 are verified deadlocks today.
4. **Add a stalled/stale row to the resume table** and archive (or explicitly re-open) the two stalled spec dirs. Confidence: high — two dirs verified stalled with no exit path.
5. **One-paragraph fixes:** cross-reference the TDD axis in config comment ↔ skills; add firewall/untrusted-input rules to deep-research (mirror code-research); fix `skillgrid:code-review` → `requesting-code-review`, `skillgrid:planning` → `writing-blueprints`; add `rigor-tiers.md` to the `_shared` pointer; move `mem_session_summary` structure into `_shared/conventions/mnemonic-memory.md` and fix "5 sections"→6. Confidence: high — all verified, all cheap.
6. **Name the enforcement split** (machine invariants = hooks; human checkpoints = prose) in using-skillgrid, and decide whether the User Gate earns a gate-stop-style hook or stays explicitly advisory. Confidence: medium — the hook version needs a "slice approved" marker artifact that doesn't exist yet.
7. **Deduplicate the 50-line threshold and fix-loop caps into `_shared/conventions/`** (rigor-tiers.md already maps tier→review shape). Confidence: medium — kills the 5-site/7-site drift surface but touches 12 files.
8. **Wire `session-start.sh` into at least one live harness config in-repo** (or delete it and document the description-based loading as the contract). Confidence: medium — C7 shows it's currently load-bearing nowhere.

### Open questions

- Does `gate-state.sh --reverify` ever bind `EVIDENCE:` back into `acceptance.feature`, or is "evidence bound to the run" unenforceable and permanently `pending`? (Decides whether recommendation 5 needs a new hook.)
- Is the "newest spec dir" tie-breaker in resume an intentional single-in-flight contract or a leftover? If intentional, state it; if not, it fires the moment two changes are genuinely concurrent.
- Who flips tasks.md checkboxes under the current heading format — is the checkbox format deprecated and the Status Guard reading a dead format?
- Is `checkpoint.json` the dead artifact (TICKET-07 deletes it) or the live pointer no skill reads yet? `docs/user-guide/08-concepts.md` still calls it "the resume pointer."
- Is the backlogmd ticketing investment deliberate for an in-repo tracker, or v1 leftover? `.backlog/` holds 11+ tasks from the v1 era.

### Source appendix

| # | Source | Kind |
|---|--------|------|
| 1 | Reviewer A digest — pipeline & gates (fresh context) | subagent |
| 2 | Reviewer B digest — skills hygiene (fresh context) | subagent |
| 3 | Reviewer C digest — orchestration & memory (fresh context) | subagent |
| 4 | Reviewer D digest — cold-start walkthrough (fresh context) | subagent |
| 5 | Reviewer E digest — evidence & TDD (fresh context) | subagent |
| 6 | Reviewer F digest — adversarial failure modes (fresh context) | subagent |
| 7 | Verifier digest — C1..C10 independent checks (fresh context) | subagent |
| 8 | Red-team digest — T1..T4 bear cases (fresh context) | subagent |
| 9 | `.skillgrid/config.yaml:5-112` | file, verified this run |
| 10 | `.agents/skills/onboarding/templates/config.yaml` (v2 template) | file, verified this run |
| 11 | `.skillgrid/specs/2026-09-21-session-events-layer/tasks.md` (0 checkboxes, 11 TICKET headings) | file, verified this run |
| 12 | `.backlog/tasks/task-019*` (12/12 `status: done`) | file, verified this run |
| 13 | `hooks/session-start.sh`, `hooks/gate-stop.sh`, `hooks/gate-state.sh:19,65`, `hooks/precommit-zone-guard.sh` | files, verified this run |
| 14 | `.skillgrid/sdd/checkpoint.json` (`remaining: "ship"`) | file, verified this run |
| 15 | `subagent-execution/SKILL.md:22,166`, `simple-execution/SKILL.md:48,112`, `qa/SKILL.md:32-41,242-259`, `ship/SKILL.md:43-67,249-275,326`, `resume/SKILL.md:44-66,85`, `using-skillgrid/SKILL.md:43,49-55,75-82,112-116` | files, verified this run |
