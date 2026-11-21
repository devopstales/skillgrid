# Phase 0: Intake

Turn a one-line idea into a brief the whole program can stand on, with every inference written down so the programmer can correct it in one reply.

When to load: Phase 0. Role: orchestrator, with the prior-art specialist.

## Input

The only required input is the idea, however short. Infer the rest: language, stack, program class, audience, scope boundary, risk class, Definition of Done, comparable projects. Write every inference to `ASSUMPTIONS.md` as one line with the reason, and to a `projectwiki/concepts/` page when the inference cluster justifies one ("CLI, because the idea says 'a tool I can pipe into'").

If `.skillgrid/` does not exist, run onboarding yourself (`references/pipeline/onboarding.md` + `references/pipeline/onboarding-write.md`) before continuing.

## Steps

1. **Project and stack.** Reuse what onboarding detected. Record language, package manager, test/build/lint commands, and the repo's existing conventions. Follow `references/pipeline/onboarding.md`.
2. **Program class and scope.** Choose the shape (CLI, service, library, UI, job) and state what is out of scope in one sentence. A program with no boundary grows until it never ships.
3. **Risk class.** Classify as `trivial | small | standard | risky | high-risk` and record the verification floor L1–L4 per `skillgrid:_shared/rules/verification-ladder.md`. Name any trust boundary, migration, public API or money path now.
4. **Prior-art map.** Two to four comparable projects, the patterns they share, and the gap this program can own, as `references/specialists/prior-art.md` describes. Without browsing, comps are provisional and the market gate stays unverified.
5. **Definition of Done.** Write the contract: acceptance scenarios green, build passes, QA gate rendered, package exists. Add any project-specific line (a migration, a benchmark, a doc page).
6. **User panel.** Build the primary, hostile and adjacent-user cards from the brief and comps (`references/roles/panel.md`) and add the fixed casual user.
7. **Independence.** Find the strongest critic setup the host allows (`references/scoring/evaluator-protocol.md`, `references/hosts.md`). If another model family's CLI is installed, propose it in the check-in: "I found Codex on this machine. Can it read the code blind as a second opinion? It uses your Codex quota." Run the isolation check where the host needs it. Record the result under `changes.<slug>.genesis.independence`.
8. **Quality target.** Keep `quality_target` at 8.5 unless the programmer names another number.

## Outputs

- `.skillgrid/projectwiki/ASSUMPTIONS.md` — the four tiers, filled with what is known.
- `.skillgrid/projectwiki/index.md` — the catalog, updated with every new page.
- `.skillgrid/projectwiki/concepts/` — one page per inference cluster that justifies a dedicated page.
- `.skillgrid/change-specs/<topic>/briefing.md` — the idea verbatim, inferred direction, stack, program class, scope, risk class, Definition of Done, the reader promise in one sentence, and the "Panel" section with the four user cards.
- `.skillgrid/change-specs/<topic>/adr.md` — the ADR manifest of pointers.
- `.skillgrid/change-specs/<topic>/findings.md` — the prior-art map with a source and checked date for every factual claim, or the label "unverified hypothesis".
- `.skillgrid/state.yaml` — the change entry with `phase: spec`, `genesis.independence` and `genesis.quality_target` filled.

## Check-in

The intake screen is the programmer's main chance to steer. Show the one-sentence promise, stack and program class, risk class and floor, the comps, the independence proposal, and the five assumptions most worth correcting. Do not bury the programmer in files; point to `briefing.md` for the rest.
