## Checklist

Classify first, announce the path, then create a task for each item on
your path and complete them in order.

**Prototype (tiny — inline):**
1. **Explore project context** — enough to frame the probe
2. **Present question + probe plan** — 2-3 sentences
3. **Get approval** — a nod is enough
4. **Investigate** — as cheaply as correctness allows
5. **Report findings in chat** — a recommendation; label anything built as throwaway

**Prototype (real probe — delegate):**
1. **Explore project context** — enough to frame the probe
2. **Delegate to `skillgrid:prototype`** — it frames the falsifiable hypothesis, gets the nod, builds the experiment, writes the verdict + investigation trail, and appends to the topic's consolidated `findings.md`. Report back the verdict and what's liftable.

**Bounded:**
1. **Explore project context** — check files, docs, recent commits
2. **Ask clarifying questions** — one at a time, the ones that matter
3. **Present short design in chat** — approach, files touched, testing
4. **Get approval** — STOP and wait for an explicit yes; presenting the design and starting in the same breath is skipping the gate
5. **Implement** — proceed with the normal development workflow (TDD applies); no plan document

**New Project:**
 1. **Explore project context** — check existing files, docs, commits (often minimal for greenfield)
  2. **Offer a sketch just-in-time** — NOT upfront, and only when the design has **2+ meaningfully different layout or interaction options** whose choice depends on *feeling* it, not reading a description. The first time that is true, offer it then (its own message): "This has a few different layout options — I can build throwaway interactive mockups so you can feel which one works. Want me to?" On approval, invoke the `skillgrid:sketch` skill; the marked winner + constraints land in the topic's consolidated `findings.md`, which the blueprint reads. If the design never has 2+ genuinely different visual options, never offer it.
  3. **Research external facts just-in-time** — NOT upfront. The first time the design depends on a fact not in the codebase (a library's current API, a version's behavior, a domain constraint, a competitor's offering), dispatch `skillgrid:research` (one inline pass) or `skillgrid:code-research` (wide or high-stakes) and let the design read the resulting `## Research:` section in `.skillgrid/change-specs/YYYY-MM-DD-<topic>/findings.md`. The research answers the question; the interview still owns the decisions. If no external fact ever comes up, never offer it.
    4. **Interview the user** — run `skillgrid:questioning`. Work the design
       tree in rounds: map the design tree, ask the whole frontier per round
       (numbered, with your recommended answer), look up facts yourself, put
       decisions to the user. Repeat until the frontier is empty AND the
       clarity gate passes (see the exit check in `skillgrid:questioning`).
       Before the interview closes, run the hypothesis validation round:
       put the core understanding you are about to encode into documents —
       the problem as understood, the intended outcome, the key
       assumptions, the approaches you will weigh — to the user through the
       structured answer tool (options with trade-offs, one at a time, each
       with your recommendation). The user accepts, overrides, or amends
       each one; nothing is written to a file until this round passes. In
       the same step, every validated decision that clears the
       ADR-sparingly test (`_shared/adr-records/README.md`) is written to
       `{projectwiki}/decisions/NNNN-slug.md` (monotonic number,
       frontmatter, body per the config's `adr_style` template) plus its
       pointer row in the `## LOCKED` ADR index in
       `{projectwiki}/ASSUMPTIONS.md` — ADR capture is part of this round,
       not a step after it.
       This replaces one-at-a-time questioning — the frontier rounds are the
       structure. As the interview runs, write VERIFIED facts to `.skillgrid/projectwiki/ASSUMPTIONS.md` and INFERRED hypotheses as they are confirmed, and `_shared/adr-records/` is maintained during the interview — the terms files (`glossary/01-business-terms.md` / `02-technical-terms.md`) are updated as terms resolve and ADRs are offered (`decisions/NNNN-slug.md`, body per the config's `adr_style` template, plus a path row in `ASSUMPTIONS.md`); the paper trail is written during the interview, not after.
    5. **Propose 2-3 approaches** — with trade-offs and your recommendation
    6. **Present design** — in sections scaled to their complexity, get user approval after each section
     7. **Codebase feasibility check** — read the repo's existing structure and `.skillgrid/projectwiki/ARCHITECTURE.md` / `.skillgrid/projectwiki/ASSUMPTIONS.md` (if present); verify the proposed architecture fits existing patterns; record a 2-line verdict in the spec's Context section. If the design doesn't fit and there's no justification, revise before writing the spec.
     8. **Goldfish shootdown** — per [shootdown.md](shootdown.md) (readiness gate → 7 parallel fresh lenses → synthesizer; all 🔴 Blockers dispositioned before the ADR manifest is written)
     9. **Reconcile the domain model + write the ADR manifest** — before writing the spec: (a) confirm the terms files (`glossary/01-business-terms.md` / `02-technical-terms.md`) reflect every term the interview resolved; (b) run the per-change ADR Review Manifest per `_shared/adr-records/README.md` — read the `## LOCKED` ADR index in `.skillgrid/projectwiki/ASSUMPTIONS.md` for the in-force set, author any qualifying ADRs as `decisions/NNNN-slug.md` (4-digit, monotonic, frontmatter, body per the config's `adr_style` template) plus a pointer row in the index, and write `<specs_root>/YYYY-MM-DD-<topic>/adr.md` from `adr-manifest.md` (this skill, references/) listing the in-force ADRs reviewed + the new ADRs created (or "none"). Commit the terms, any new ADRs, and the manifest with the spec.
     10. **Write the acceptance contract** (BDD is always on; full protocol: `skillgrid:blueprint` Phase 5) — copy `skillgrid:blueprint/templates/acceptance.feature` → `.skillgrid/change-specs/YYYY-MM-DD-<topic>/acceptance.feature`. One `Rule:` (`### Requirement:`) per briefing requirement, with happy/edge/failure scenarios in domain language (use the terms files). Happy-path scenarios must cover the Definition of Done. Commit the spec before any code. This is the first write into the change folder — create the folder now, and if `ticketing.enabled: true` in `.skillgrid/config.yaml`, create the change's milestone in the same step (title = the folder name `YYYY-MM-DD-<topic>` verbatim, via the resolved tracker's mechanism — `slicing/references/ticketing/` formatting reference; one milestone per SDD change, see `_shared/rules/sdd-structure.md` § Writing Rules).
     11. **Write the global artifacts** — create `.skillgrid/projectwiki/reference/00-prd.md` from `templates/PRD.md`, `.skillgrid/projectwiki/ASSUMPTIONS.md` from `templates/ASSUMPTIONS.md`, and `.skillgrid/projectwiki/ARCHITECTURE.md` from `templates/ARCHITECTURE.md`, filling in every section; also copy `templates/briefing.md` → `.skillgrid/change-specs/YYYY-MM-DD-<topic>/briefing.md` and fill it in (including the Clarity Report from the interview). If the global files already exist, merge rather than overwrite. Commit.
     12. **Spec self-review** — quick inline check for placeholders, contradictions, ambiguity, scope, falsifiability (see [after-the-design.md](after-the-design.md)); run it on `reference/00-prd.md`, ASSUMPTIONS.md, ARCHITECTURE.md, and the briefing
     13. **User reviews the artifacts** — ask user to review `.skillgrid/projectwiki/reference/00-prd.md`, `.skillgrid/projectwiki/ASSUMPTIONS.md`, `.skillgrid/projectwiki/ARCHITECTURE.md`, the spec, the acceptance contract, and the updated terms/ADRs before proceeding. The approval is the user's own hand-edit of the briefing's line-3 STATUS banner — `draft` → `approved (YYYY-MM-DD)`; the agent never writes `approved` (approval gate per `_shared/rules/sdd-structure.md` § STATUS Banner). Confirm by re-reading line 3 from disk, then set the briefing's `**Intent frozen:**` line to `yes (YYYY-MM-DD)` (the frozen-intent contract, per `_shared/craft/review-ledger.md` § Frozen intent — the `## Problem / Intent` + `## Purpose & Success Criteria` sections are read-only for agent steps from that line forward) and commit the briefing.
    14. **Transition to implementation** — trigger `skillgrid:projectwiki` (Maintain mode) to complete the vault update (same-edit rule, `log.md` lines, ADR `## Related` blocks, ship gate with no new findings), then invoke skillgrid:blueprint skill to create implementation plan

**New Function:**
1. **Explore project context** — check files, docs, recent commits; read `.skillgrid/projectwiki/ASSUMPTIONS.md`, `.skillgrid/projectwiki/reference/00-prd.md`, and `.skillgrid/projectwiki/ARCHITECTURE.md` if present
  2. **Offer a sketch just-in-time** — NOT upfront, and only when the design has **2+ meaningfully different layout or interaction options** whose choice depends on *feeling* it, not reading a description. The first time that is true, offer it then (its own message): "This has a few different layout options — I can build throwaway interactive mockups so you can feel which one works. Want me to?" On approval, invoke the `skillgrid:sketch` skill; the marked winner + constraints land in the topic's consolidated `findings.md`, which the blueprint reads. If the design never has 2+ genuinely different visual options, never offer it.
   3. **Interview the user** — run `skillgrid:questioning`. Work the design
       tree in rounds: map the decision tree, ask the whole frontier per round
       (numbered, with your recommended answer), look up facts yourself, put
       decisions to the user. Repeat until the frontier is empty AND the
       clarity gate passes (see the exit check in `skillgrid:questioning`).
       Before the interview closes, run the hypothesis validation round:
       put the core understanding you are about to encode into documents —
       the problem as understood, the intended outcome, the key
       assumptions, the approaches you will weigh — to the user through the
       structured answer tool (options with trade-offs, one at a time, each
       with your recommendation). The user accepts, overrides, or amends
       each one; nothing is written to a file until this round passes. In
       the same step, every validated decision that clears the
       ADR-sparingly test (`_shared/adr-records/README.md`) is written to
       `{projectwiki}/decisions/NNNN-slug.md` (monotonic number,
       frontmatter, body per the config's `adr_style` template) plus its
       pointer row in the `## LOCKED` ADR index in
       `{projectwiki}/ASSUMPTIONS.md` — ADR capture is part of this round,
       not a step after it.
       This replaces one-at-a-time questioning — the frontier rounds are the
       structure. As the interview runs, write VERIFIED facts to `.skillgrid/projectwiki/ASSUMPTIONS.md` and INFERRED hypotheses as they are confirmed, and `_shared/adr-records/` is maintained during the interview — the terms files (`glossary/01-business-terms.md` / `02-technical-terms.md`) are updated as terms resolve and ADRs are offered (`decisions/NNNN-slug.md`, body per the config's `adr_style` template, plus a path row in `ASSUMPTIONS.md`); the paper trail is written during the interview, not after.
   4. **Propose 2-3 approaches** — with trade-offs and your recommendation
   5. **Present design** — in sections scaled to their complexity, get user approval after each section
   6. **Codebase feasibility check** — read the files and flows the feature touches; verify the design follows existing patterns (naming, layering, data access, error handling, testing); record a 2-line verdict in the spec's Context section. If the design doesn't fit and there's no justification, revise before writing the spec.
     7. **Goldfish shootdown** — per [shootdown.md](shootdown.md) (readiness gate → 7 parallel fresh lenses → synthesizer; all 🔴 Blockers dispositioned before the ADR manifest is written)
     8. **Reconcile the domain model + write the ADR manifest** — before writing the spec: (a) confirm the terms files (`glossary/01-business-terms.md` / `02-technical-terms.md`) reflect every term the interview resolved; (b) run the per-change ADR Review Manifest per `_shared/adr-records/README.md` — read the `## LOCKED` ADR index in `.skillgrid/projectwiki/ASSUMPTIONS.md` for the in-force set, author any qualifying ADRs as `decisions/NNNN-slug.md` (4-digit, monotonic, frontmatter, body per the config's `adr_style` template) plus a pointer row in the index, and write `<specs_root>/YYYY-MM-DD-<topic>/adr.md` from `adr-manifest.md` (this skill, references/) listing the in-force ADRs reviewed + the new ADRs created (or "none"). Commit the terms, any new ADRs, and the manifest with the spec.
     9. **Write the acceptance contract** (BDD is always on; full protocol: `skillgrid:blueprint` Phase 5) — copy `skillgrid:blueprint/templates/acceptance.feature` → `.skillgrid/change-specs/YYYY-MM-DD-<topic>/acceptance.feature`. One `Rule:` (`### Requirement:`) per briefing requirement, with happy/edge/failure scenarios in domain language (use the terms files). Happy-path scenarios must cover the Definition of Done. Commit the spec before any code. This is the first write into the change folder — create the folder now, and if `ticketing.enabled: true` in `.skillgrid/config.yaml`, create the change's milestone in the same step (title = the folder name `YYYY-MM-DD-<topic>` verbatim, via the resolved tracker's mechanism — `slicing/references/ticketing/` formatting reference; one milestone per SDD change, see `_shared/rules/sdd-structure.md` § Writing Rules).
     10. **Write the spec** — copy `templates/briefing.md` → `.skillgrid/change-specs/YYYY-MM-DD-<topic>/briefing.md`, fill it in (including the Clarity Report from the interview), and commit. If the feature changes the global `.skillgrid/projectwiki/ASSUMPTIONS.md` (new VERIFIED fact, metric, a new ADR file under `decisions/`) or `.skillgrid/projectwiki/ARCHITECTURE.md` (new component/store/integration), update those too — only when they exist AND the feature actually changes them.
     11. **Spec self-review** — quick inline check for placeholders, contradictions, ambiguity, scope, falsifiability (see [after-the-design.md](after-the-design.md))
     12. **User reviews the spec** — ask user to review the spec file, the acceptance contract, the updated terms/ADRs, and any global updates before proceeding. The approval is the user's own hand-edit of the spec's line-3 STATUS banner — `draft` → `approved (YYYY-MM-DD)`; the agent never writes `approved` (approval gate per `_shared/rules/sdd-structure.md` § STATUS Banner). Confirm by re-reading line 3 from disk, then set the briefing's `**Intent frozen:**` line to `yes (YYYY-MM-DD)` (the frozen-intent contract, per `_shared/craft/review-ledger.md` § Frozen intent — the `## Problem / Intent` + `## Purpose & Success Criteria` sections are read-only for agent steps from that line forward) and commit the briefing.
     13. **Transition to implementation** — trigger `skillgrid:projectwiki` (Maintain mode) to complete the vault update (same-edit rule, `log.md` lines, ADR `## Related` blocks, ship gate with no new findings), then invoke skillgrid:blueprint skill to create implementation plan

## Process Flow

```dot
digraph brainstorming {
    "Classify: prototype / bounded / new project / new function" [shape=diamond];
    "Present question + probe (2-3 sentences)" [shape=box];
    "Ask clarifying questions (bounded)" [shape=box];
    "Present short design in chat" [shape=box];
    "Human approves?" [shape=diamond];
    "Investigate; report recommendation" [shape=doublecircle];
    "Implement via normal workflow (no plan doc)" [shape=doublecircle];
    "Explore project context" [shape=box];
    "Interview (skillgrid:questioning)\ndesign tree → rounds → frontier empty\n+ clarity gate passes" [shape=box];
    "Propose 2-3 approaches" [shape=box];
    "Present design sections" [shape=box];
    "User approves design?" [shape=diamond];
    "Codebase feasibility check\n(fits existing patterns?)" [shape=diamond];
    "Goldfish shootdown\n(readiness gate → 7 lenses → synthesizer)\nuser disposes all Blockers" [shape=diamond];
    "Write .skillgrid/projectwiki/ASSUMPTIONS.md (VERIFIED/INFERRED\n+ LOCKED on user OK) + ARCHITECTURE.md\n+ briefing.md (new project)" [shape=box];
    "Write briefing.md (+ update global\nartifacts if changed) (new function)" [shape=box];
    "Spec self-review\n(fix inline)" [shape=box];
    "User reviews artifacts?" [shape=diamond];
    "Trigger skillgrid:projectwiki (Maintain)\nindex + log + Related + ship gate" [shape=box];
    "Invoke skillgrid:blueprint skill" [shape=doublecircle];
    "Hidden complexity? Upgrade path" [shape=box];

    "Classify: prototype / bounded / new project / new function" -> "Present question + probe (2-3 sentences)" [label="prototype"];
    "Classify: prototype / bounded / new project / new function" -> "Ask clarifying questions (bounded)" [label="bounded"];
    "Classify: prototype / bounded / new project / new function" -> "Explore project context" [label="new project / new function"];
    "Present question + probe (2-3 sentences)" -> "Human approves?";
    "Ask clarifying questions (bounded)" -> "Present short design in chat";
    "Present short design in chat" -> "Human approves?";
    "Human approves?" -> "Investigate; report recommendation" [label="prototype: yes"];
    "Human approves?" -> "Implement via normal workflow (no plan doc)" [label="bounded: yes"];
    "Hidden complexity? Upgrade path" -> "Classify: prototype / bounded / new project / new function";
    "Explore project context" -> "Interview (skillgrid:questioning)\ndesign tree → rounds → frontier empty\n+ clarity gate passes";
    "Interview (skillgrid:questioning)\ndesign tree → rounds → frontier empty\n+ clarity gate passes" -> "Propose 2-3 approaches";
    "Propose 2-3 approaches" -> "Present design sections";
    "Present design sections" -> "User approves design?";
    "User approves design?" -> "Present design sections" [label="no, revise"];
    "User approves design?" -> "Codebase feasibility check\n(fits existing patterns?)" [label="yes"];
    "Codebase feasibility check\n(fits existing patterns?)" -> "Present design sections" [label="no, revise design"];
    "Codebase feasibility check\n(fits existing patterns?)" -> "Goldfish shootdown\n(readiness gate → 7 lenses → synthesizer)\nuser disposes all Blockers" [label="yes"];
    "Goldfish shootdown\n(readiness gate → 7 lenses → synthesizer)\nuser disposes all Blockers" -> "Codebase feasibility check\n(fits existing patterns?)" [label="resolved: revise design, re-check"];
    "Goldfish shootdown\n(readiness gate → 7 lenses → synthesizer)\nuser disposes all Blockers" -> "Write .skillgrid/projectwiki/ASSUMPTIONS.md (VERIFIED/INFERRED\n+ LOCKED on user OK) + ARCHITECTURE.md\n+ briefing.md (new project)" [label="new project"];
    "Goldfish shootdown\n(readiness gate → 7 lenses → synthesizer)\nuser disposes all Blockers" -> "Write briefing.md (+ update global\nartifacts if changed) (new function)" [label="new function"];
    "Write .skillgrid/projectwiki/ASSUMPTIONS.md (VERIFIED/INFERRED\n+ LOCKED on user OK) + ARCHITECTURE.md\n+ briefing.md (new project)" -> "Spec self-review\n(fix inline)";
    "Write briefing.md (+ update global\nartifacts if changed) (new function)" -> "Spec self-review\n(fix inline)";
    "Spec self-review\n(fix inline)" -> "User reviews artifacts?";
    "User reviews artifacts?" -> "Write .skillgrid/projectwiki/ASSUMPTIONS.md (VERIFIED/INFERRED\n+ LOCKED on user OK) + ARCHITECTURE.md\n+ briefing.md (new project)" [label="changes requested"];
    "User reviews artifacts?" -> "Write briefing.md (+ update global\nartifacts if changed) (new function)" [label="changes requested"];
    "User reviews artifacts?" -> "Trigger skillgrid:projectwiki (Maintain)\nindex + log + Related + ship gate" [label="approved"];
    "Trigger skillgrid:projectwiki (Maintain)\nindex + log + Related + ship gate" -> "Invoke skillgrid:blueprint skill";
}
```

**Terminal states are path-bound.** New Project / New Function: the
ONLY skill you invoke after brainstorming is skillgrid:blueprint — never
or any other implementation skill.
Bounded: after approval, implementation proceeds directly through the
normal development workflow; no plan document. Prototype: the terminal
state is a reported recommendation.
