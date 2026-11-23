## After the Design (new project / new function path)

**Documentation:**

- **New Project:** create `.skillgrid/projectwiki/reference/00-prd.md` from `templates/PRD.md`,
  `.skillgrid/projectwiki/ASSUMPTIONS.md` from `templates/ASSUMPTIONS.md` (product statement,
  VERIFIED/INFERRED tiers, the in-force table, locked constraints; it links to
  `00-prd.md`), and
  `.skillgrid/projectwiki/ARCHITECTURE.md` from `templates/ARCHITECTURE.md`, filling in every
  section; also copy `templates/briefing.md` →
  `.skillgrid/change-specs/YYYY-MM-DD-<topic>/briefing.md` and fill it in. If
  the global files already exist (a second project in the repo), merge
  rather than overwrite.
- **New Function:** copy `templates/briefing.md` →
  `.skillgrid/change-specs/YYYY-MM-DD-<topic>/briefing.md` and fill it in. The
  `> **STATUS:** \`draft\` (YYYY-MM-DD)` banner on line 3 is required (see
  `_shared/rules/sdd-structure.md` § STATUS Banner). If
  the feature changes `.skillgrid/projectwiki/reference/00-prd.md` (scope, feature, metric), the
  global `.skillgrid/projectwiki/ASSUMPTIONS.md` (new VERIFIED fact, a new ADR file under
  `decisions/`) or `.skillgrid/projectwiki/ARCHITECTURE.md`
  (new component, data store, integration), update those too — only when they exist
  AND the feature actually changes them.
  - (User preferences for spec location override this default)
- If a writing-clarity skill is available, use it for the design document
- Commit the design document (and any updated global docs) to git

**Spec Self-Review:**
After writing the spec (and the global ASSUMPTIONS.md/ARCHITECTURE.md for a new
project), look at it with fresh eyes:

1. **Placeholder scan:** Any "TBD", "TODO", incomplete sections, or vague requirements? Fix them.
2. **Falsifiability check:** Does every requirement in the briefing have all three fields — Current, Target, Acceptance? If any is missing or vague ("improve X" instead of "X becomes Y"), fix it. A requirement without a pass/fail acceptance check is not a requirement.
3. **Clarity gate passed:** Does the briefing's Clarity Report show clarity ≤ 0.20 with all dimensions at or above their minimums? If any dimension is ⚠, it must be listed in Open Questions & Assumptions. If the Clarity Report is missing or incomplete, go back and run the exit check in `skillgrid:questioning`.
4. **Internal consistency:** Do any sections contradict each other? Does the architecture match the feature descriptions?
5. **Scope check:** Is this focused enough for a single implementation plan, or does it need decomposition?
6. **Ambiguity check:** Could any requirement be interpreted two different ways? If so, pick one and make it explicit.
7. **Feasibility verdict present:** Does the spec's Context section contain the 2-line codebase feasibility verdict? If not, add it.

For a **new project**, also verify `.skillgrid/projectwiki/reference/00-prd.md`,
`.skillgrid/projectwiki/ASSUMPTIONS.md`, and `.skillgrid/projectwiki/ARCHITECTURE.md` are complete: no
template placeholders left, the requirements in `00-prd.md` match the product
statement in ASSUMPTIONS.md and the components in ARCHITECTURE.md, and the
documents agree on scope. For a **new function**, verify any global artifacts
updates (`00-prd.md` / `ASSUMPTIONS.md` / `ARCHITECTURE.md`) are consistent with
the briefing.

Fix any issues inline. No need to re-review — just fix and move on.

**User Review Gate:**
After the spec review loop passes, the user must read the written spec
(and `00-prd.md`, `ASSUMPTIONS.md`, and ARCHITECTURE for a new project)
before the next phase. The approval is the user's own edit of the
briefing's STATUS banner — not a chat reply:

> "Spec written and committed to `<path>` (plus `.skillgrid/projectwiki/reference/00-prd.md`,
> `.skillgrid/projectwiki/ASSUMPTIONS.md`, and `.skillgrid/projectwiki/ARCHITECTURE.md` for a new project).
> Please read it, and when you approve it, edit the STATUS line yourself —
> change `draft` to `approved (YYYY-MM-DD)` on line 3."

Wait for the flip. Do not write `approved` yourself — it is the user's value
(per `_shared/rules/sdd-structure.md` § STATUS Banner). Confirm by
re-reading line 3 from disk and quote the banner line in your reply. If the
user requests changes instead, make them, re-run the spec review loop, and
set the banner to `revised` (the user re-reads and re-flips).

Once the banner shows `approved`, set the briefing's `**Intent frozen:**`
line to `yes (YYYY-MM-DD)` and commit — from that line forward, `## Problem / Intent` + `## Purpose & Success Criteria` (including Out of scope) are read-only for agent steps; a later finding that needs them changed is an `intent_gap` (HALT + human decision), per `_shared/craft/review-ledger.md` § Frozen intent.

**Implementation:**

- **Update `state.yaml`:** add `changes.<topic>: { phase: spec, status: in_progress }` to the `changes:` map. If the entry already exists (re-running brainstorming), update its `phase` and `status`.
- **Acquire session lock:** run `node scripts/state-lock.mjs acquire <topic> [session-id]`. Exit 1 (conflict with an active lock for a different change) → surface the warning to the user (advisory, don't block). Exit 0 → continue. The lock is released by `skillgrid:ship` when it removes the `changes.<topic>` entry.
- **Trigger `skillgrid:projectwiki` (Maintain mode)** — the interview wrote terms, ADRs, and (new project) the global wiki files; invoke the projectwiki skill to complete the vault update: same-edit rule (`projectwiki index` after every page write), `log.md` lines for each action, ADR `## Related` blocks where missing, and the ship gate (`projectwiki index` → `validate` → `lint` → `status --json` with no new findings). This runs BEFORE the handoff below and never skips: a vault whose index lags the pages it just received is an incomplete handoff.
- Invoke the skillgrid:blueprint skill to create a detailed implementation plan
- Do NOT invoke any other skill. skillgrid:blueprint is the next step.
