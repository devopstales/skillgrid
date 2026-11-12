---
type: TechnicalDecision
status: accepted
accepted: 2026-11-12
supersedes: none
date: 2026-11-12
---

# ADR-0062 — Goldfish brainstorm shootdown: before-design idea validation is a fresh-context 6-lens subflow of brainstorming, not a new phase and not a new top-level skill

Change: 2026-11-02-egm-goldfish-brainstorm-shootdown

## Context

skillgrid's `brainstorming` skill is a structured 4-path classifier with a hard approval gate, but it has no adversarial review of the idea itself before the design is committed. The author and the user design together in the same context, which is a sycophancy risk: the agent affirms the user's premise rather than stress-testing it. The Elephant-Goldfish Model (EGM, `github.com/google/elephant-goldfish-model`) solves this with `egm-brainstorm`: six parallel fresh "Goldfish" subagents attack the anchored hypothesis from six conflicting lenses before the design is written. The question was where to port that review in skillgrid.

The insertion point already exists in `brainstorming`'s New Project / New Function paths: interview → approaches → design → codebase feasibility check → ADR manifest → acceptance contract → spec. A same-context review of the premise (the failure mode) is exactly what the existing Self-Review-style passes do; a review that counts must run in fresh context (R36: a harness without fresh-context subagents stops, it does not degrade).

## Decision

Before-design idea validation is a fresh-context 6-lens Goldfish shootdown subflow of brainstorming; it validates the premise before the spec is written.

- **Six parallel fresh-context lens subagents** (R1), one per lens: Contrarian/YAGNI (can existing code/libs solve 90% without new code?), Cost/Quota/Token Economics, UX & Developer Ergonomics, Migration & Compatibility, Observability & SRE, Tech Debt & Maintenance. Each is a freshly spawned `reviewer` (fallback `general`) given ONLY the anchored hypothesis + the relevant existing code/files — no conversation history, no author narrative. A review run in the author's own context is not a Goldfish shootdown.
- **Insertion point** (R2): New Project / New Function paths only, after the codebase feasibility check passes and before the ADR manifest is written. Bounded and prototype paths skip it. The path ratchet is unchanged: a bounded task that upgrades to New Function mid-task makes the shootdown mandatory.
- **Blocker-disposition gate** (R3): after the six lenses report, the main agent (the Elephant) synthesizes inline (dedupe, group by severity 🔴 Blocker / 🟡 Major / 🟢 Nit). Every 🔴 Blocker is dispositioned — resolved (design revised, loop back to feasibility), accepted-with-note (recorded as a known risk), or downgraded-to-non-goal (moved to Out of Scope) — before the spec is written. 🟡/🟢 findings are recorded but do not block.
- **Findings land in `briefing.md`** (R4): a new `## Shootdown Findings` section (anchored hypothesis + severity table with dispositions + one-line verdict). No separate artifact — `briefing.md` is the single source of truth.
- **Anti-sycophancy & stress-testing rules** (R5) govern the tone of the interview and shootdown conversation, cited from `_shared/craft/review-discipline.md` (single home of the banned-adjective list); the structured `question` tool mandate is untouched.

## Alternatives considered

- **A new top-level `goldfish-brainstorm` skill** — rejected: duplicates the seam and grows the skill roster (the owner's constraint: "we have too many skills"). The shootdown is a subflow of `brainstorming`'s existing New Project / New Function paths, not a new pipeline phase.
- **Add the shootdown to `qa`** — rejected: wrong time and wrong target (code, not idea); would conflate the before-design, before-code, and after-code failure classes.
- **Port EGM's `concepts_brief.md` as a separate artifact** — rejected: skillgrid already has `briefing.md` as the spec; a second artifact would be a second source of truth.
- **Port EGM's `synthesizer` subagent** — rejected: the main agent synthesizes inline; a subagent-of-subagents layer adds a dispatch round-trip without adding value.
- **Register new named OpenCode subagents per lens** — rejected: the harness does not make that cheap; EGM's documented fallback (a general-purpose subagent that reads a persona brief file first) is the faithful, lower-friction port.

## Consequences

- Good, because the premise is validated at the cheap point (before any design is committed) by six conflicting fresh-context perspectives; a wrong idea dies before spec-writing cost is spent.
- Good, because it is additive: the 4-path classifier, the feasibility check, and the path ratchet are unchanged; `brainstorming`'s New Project / New Function paths gain one step between the feasibility check and the ADR manifest.
- Good, because the blocker-disposition gate makes the review load-bearing: an undispositioned 🔴 Blocker means no committed spec, so the shootdown cannot be rubber-stamped into silence.
- Bad, because New Project / New Function brainstorms pay six extra subagent dispatches plus an interactive disposition round before the spec is written.
- Neutral, because the anti-sycophancy rules change tone, not tooling — the structured `question` tool mandate stays intact.
- Neutral, because a harness without a subagent tool halts the shootdown and says so (R36) — no silent same-context degradation.
