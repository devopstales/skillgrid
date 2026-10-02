## After the Design (new project / new function path)

**Documentation:**

- **New Project:** create `.skillgrid/ASSUMPTIONS.md` from `templates/PRD.md` and
  `.skillgrid/ARCHITECTURE.md` from `templates/ARCHITECTURE.md`, filling in every
  section; also copy `templates/briefing.md` →
  `.skillgrid/specs/YYYY-MM-DD-<topic>/briefing.md` and fill it in. If
  the global files already exist (a second project in the repo), merge
  rather than overwrite.
- **New Function:** copy `templates/briefing.md` →
  `.skillgrid/specs/YYYY-MM-DD-<topic>/briefing.md` and fill it in. The
  `> **STATUS:** \`draft\` (YYYY-MM-DD)` banner on line 3 is required (see
  `_shared/rules/sdd-structure.md` § STATUS Banner). If
  the feature changes the global `.skillgrid/ASSUMPTIONS.md` (new VERIFIED fact,
  changed metric, a new `### ADR-NNNN` entry) or `.skillgrid/ARCHITECTURE.md`
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
3. **Clarity gate passed:** Does the briefing's Clarity Report show clarity ≤ 0.20 with all dimensions at or above their minimums? If any dimension is ⚠, it must be listed in Open Questions & Assumptions. If the Clarity Report is missing or incomplete, go back and run the `skillgrid:interviewing` skill's exit check.
4. **Internal consistency:** Do any sections contradict each other? Does the architecture match the feature descriptions?
5. **Scope check:** Is this focused enough for a single implementation plan, or does it need decomposition?
6. **Ambiguity check:** Could any requirement be interpreted two different ways? If so, pick one and make it explicit.
7. **Feasibility verdict present:** Does the spec's Context section contain the 2-line codebase feasibility verdict? If not, add it.

For a **new project**, also verify `.skillgrid/ASSUMPTIONS.md` and
`.skillgrid/ARCHITECTURE.md` are complete: no template placeholders left, the
feature/product facts in ASSUMPTIONS.md match the components in ARCHITECTURE.md,
and the two documents agree on scope. For a **new function**, verify any global
artifacts updates (`ASSUMPTIONS.md` / `ARCHITECTURE.md`) are consistent with the
briefing.

Fix any issues inline. No need to re-review — just fix and move on.

**User Review Gate:**
After the spec review loop passes, ask the user to review the written
spec (and the global PRD/ARCHITECTURE for a new project) before
proceeding:

> "Spec written and committed to `<path>` (plus `.skillgrid/ASSUMPTIONS.md` and
> `.skillgrid/ARCHITECTURE.md` for a new project). Please review and let me
> know if you want to make any changes before we start writing out the
> implementation plan."

Wait for the user's response. If they request changes, make them and re-run the spec review loop. Only proceed once the user approves.

**Implementation:**

- **Update `state.yaml`:** set `pipeline.current_change: <topic>` and `pipeline.current_phase: spec`.
- **Acquire session lock:** run `node scripts/state-lock.mjs acquire <topic> [session-id]`. Exit 1 (conflict with an active lock for a different change) → surface the warning to the user (advisory, don't block). Exit 0 → continue. The lock is released by `skillgrid:ship` when it clears `current_change`.
- Invoke the skillgrid:writing-blueprints skill to create a detailed implementation plan
- Do NOT invoke any other skill. skillgrid:writing-blueprints is the next step.

## Visual Companion

A browser-based companion for showing mockups, diagrams, and visual options during brainstorming. Available as a tool — not a mode. Accepting the companion means it's available for questions that benefit from visual treatment; it does NOT mean every question goes through the browser.

> **The decision bridge is the primary companion now.** Interview questions — and the
> visual options that belong to them — are posted to the **Mnemonic decision bridge**:
> the agent `mem_save`s each frontier question as a `type: decision` observation
> (JSON `content` + `state: pending`, `recommended`, and a `visual` prototype id when
> the question is clearer shown than told) and shares it to `team`; the user answers
> in the **Decisions** view of the running `skillgrid serve` dashboard; the agent reads
> the answer back by polling `mem_search`. This is durable, governed (version history),
> and needs no second process — see `docs/user-guide/10-decision-companion.md` and the
> `skillgrid:interviewing` skill's "Post each question to the decision bridge".
> The standalone Node companion below (`visual-companion.md` — WebSocket + fs.watch +
> `?key=` gate) is the older, weaker path: use it only when there is no running
> `skillgrid serve` to host the Decisions view.

**Offering the companion (just-in-time):** Do NOT offer it upfront. Wait until a question would genuinely be clearer shown than told — a real mockup / layout / diagram question, not merely a UI *topic*. The first time that happens, offer it then, as its own message:
> "This next part might be easier if I show you — I can put together mockups, diagrams, and comparisons in a browser tab as we go. It's still new and can be token-intensive. Want me to? I'll open it for you."

**This offer MUST be its own message.** Only the offer — no clarifying question, summary, or other content. Wait for the user's response. If they accept, start the server with `--open` so their browser opens to the first screen automatically. If they decline, continue text-only and don't offer again unless they raise it.

**Per-question decision:** Even after the user accepts, decide FOR EACH QUESTION whether to use the browser or the terminal. The test: **would the user understand this better by seeing it than reading it?**

- **Use the browser** for content that IS visual — mockups, wireframes, layout comparisons, architecture diagrams, side-by-side visual designs
- **Use the terminal** for content that is text — requirements questions, conceptual choices, tradeoff lists, A/B/C/D text options, scope decisions

A question about a UI topic is not automatically a visual question. "What does personality mean in this context?" is a conceptual question — use the terminal. "Which wizard layout works better?" is a visual question — use the browser.

If they agree to the companion, read the detailed guide before proceeding:
`visual-companion.md`
