# Gap analysis: superpowers:brainstorming → skillgrid:brainstorm

**Date:** 2026-11-24
**Sources read:** `/data/git/AI/superpowers/skills/brainstorming/` (SKILL.md 316 lines, builder-check-prompt.md, visual-companion.md, scripts/) and `.agents/skills/brainstorm/` (SKILL.md 464 lines, 5 references, 4 templates, evals) + `.agents/skills/questioning/SKILL.md` (the delegated interview loop) + `_shared/craft/review-discipline.md`, `_shared/skill-size-budget.json`, `templates/briefing.md`.

## Framing

The two skills are different *kinds* of skill, and that's the key insight:

| | superpowers:brainstorming | skillgrid:brainstorm |
|---|---|---|
| **Nature** | Behavioral — *how the agent talks* (first message, conversation texture, attribution) | Pipeline — *phases, files, gates, artifacts* (paths, ratchet, spec-zone, ADRs, shootdown, resume) |
| **Strength** | Conversation craft: echo-back, playback, "options steer", just-in-time offers | Structure: 4-path classification, clarity gate + confidence stop, 7-lens shootdown, machine-checkable file contracts |
| **Weakness** | No artifact contract, no resume, one generic doc location | The behavioral layer is delegated to `questioning`, and that layer is thinner than theirs — ours grills in *frontier rounds*; theirs also *plays back, sketches, and attributes* |

Everything valuable in superpowers lives in the behavioral layer, which is exactly where our `questioning` skill is weakest. Nothing in their file touches the pipeline layer — we don't need to port any of the doc-location/gate mechanics.

---

## Tier 1 — high leverage, port these

### 1. Builder check (fresh-context document test) — the single best find

Mechanism: after the design doc is written, dispatch **one fresh subagent** with a persona prompt: *"You are a talented [domain builder]… handed a design document… you will not be able to talk to the person who wrote it. Report only the questions you'd still need answered before you could plan this well: places where you'd have to guess and a wrong guess would matter. Rank them."* Then sort answers into three piles: **already answered** (put it in the doc) / **minor** (settle it yourself, mark as *your call*) / **theirs** (one message to the user, most important first). One round only. Fallback without subagents: read the doc as that builder yourself.

Why it's a gap: our `after-the-design.md` has a **spec self-review — same context, author's head**. Structurally it cannot find *intent that was discussed but never written down*, because the author's context still contains it. The Goldfish shootdown attacks the *design* (adversarial, 7 lenses, pre-manifest); the builder check tests the *document's self-sufficiency*. Different failure classes, complementary. Our infra already dispatches 7 subagents for the shootdown — one more small one is cheap, and the 3-pile disposition keeps it from becoming a second interview with the user.

**Change:** new step in `references/after-the-design.md` between self-review and the user review gate + a small `references/builder-check-prompt.md` (same dispatch-prompt pattern as `shootdown.md` → `goldfish.md`). The "theirs" pile is the only round of questions it may produce.

### 2. Play It Back (progressive playback with attribution)

Mechanism: *"As soon as you understand one part well enough to describe it… describe that part back in about 200–300 words, then return to questions about the next part. Playback and questions alternate through the whole conversation."* With strict attribution: *"Only what they said goes in as theirs. Anything you filled in goes in as your proposal, with your reason. End each chunk by asking what's wrong or missing."*

Why it's a gap: we confirm understanding **only at the exits** — hypothesis validation round, clarity gate, confidence stop. All of those are *late* confirmation: a distorted model is caught after the full interview, approaches, and feasibility check. Theirs catches it *incrementally*, while it's still cheap, and the theirs/your-proposal split is exactly the discipline that keeps the briefing honest (validated vs. assumed). It maps cleanly onto our structure: **after each frontier round, when a subtree settles, play back that subtree**. It also feeds our `ASSUMPTIONS.md` tiers directly — what was confirmed is VERIFIED, what we filled in is INFERRED, with the attribution already collected.

**Change:** "Play it back" subsection in `questioning/SKILL.md` (round structure), 200–300-word chunks, attribution rule, "what's wrong or missing" close, and the visual interplay: chunk about something they'll *see* → text sketch next to it (see #8); look is theirs to decide and sketch offer not yet made → send the offer, hold the chunk.

### 3. The first-message recipe

Their first message is a **fixed ~200-word shape**, not a style guideline:

1. Echo the request back in your own words — *what, who it's for, why, as far as they've said* (so they can see whether you've got its shape).
2. Add what you found that bears on it: how the relevant parts work today, what the change would touch.
3. Flag plainly anything that would change the ask: a constraint, a conflict, **a cheaper path**.
4. One line inviting corrections.
5. Then exactly one of: open question about the *why* (in their own words, no options, say what the answer would change) → or the one open question whose answer changes what gets built most (sketch each choice concretely with trade-offs and your pick) → or, if nothing is open, the short design itself + approval ask.
6. "Specs are often incomplete, so **confirm before you build on one**."

Why it's a gap: ours jumps from classification to design-tree rounds. No echo-back means the user can't verify the *shape* of our understanding before spending an interview on it — the core risk the skill names in its first sentence ("that belief is usually thinner than it feels") gets no early check. And we have no rule for the case where the user hands us a spec and we silently build on it.

**Change:** "Your first message" section in `brainstorm/SKILL.md` (~30 lines), including the spec-confirm rule. It belongs in SKILL.md because it happens *before* the interview exists.

### 4. "How it gets made" — the how/what split + agent-call attribution

Mechanism: *"Read from how they talk which of those choices they want to make. Someone who raises the how or asks about trade-offs wants the options with your pick. Someone who talks about outcomes is handing you the how: make those calls and state each in a line. When there's existing work, build on what it already uses. **Any choice you make goes in the design as your call, never as something they agreed to.**"*

Why it's a gap: our architecture-option gate ("no ranking, no default, no pick") is strong but trigger-blind — it fires on *load-bearingness*, not on whether the user signaled they care. A user who talks purely about outcomes has already delegated the how; forcing a gate turn on them is friction, and their "just pick one" → restate-and-regate loop can spin. Superpowers adds the missing trigger: **who decides depends on their talk, and every agent-made call is stated as such** — which our design presentation lacks (we log *rejected* approaches; we don't surface *the calls I made on your behalf* for checking).

**Change:** edit "Exploring approaches" + "Presenting the design" in `SKILL.md`: put-to-user rule = load-bearing **AND** (user signals how-care OR user didn't delegate); otherwise make the call, one line each, labeled *agent call*. Add a "Calls I made" block to the design presentation (user checks it while reading) and a `decided by: user / agent` column in the briefing's interview log.

### 5. "Get them describing" — options steer

Mechanism: *"Options steer. Ask them to describe it first." / "Their own words carry intent that your options can't."* Until the *why* is known: open questions only, no example answers attached. "Just guess" → then propose something concrete.

Why it's a gap: we *have* phase scoping (open-ended problem-drilling before structured rounds), but the principle is not stated as an anti-pattern anywhere, and our own rule *"Every question has options. Even free-text questions get a short list of realistic choices"* (frontier rounds) is one mis-applied round away from menu-steering in the early phase. Their red-flag row — *"I'll offer a menu of what they might want"* — is the steering surface we're missing.

**Change:** rule + red-flag row in `questioning/SKILL.md` (no option lists before the design tree exists; options belong to *decisions*, never to *what they want*), one red-flag row in `brainstorm/SKILL.md`.

---

## Tier 2 — real value, medium effort

### 6. The opt-out

*"Or if you'd rather I just dive in, say so."* — a light one-line offer, **only in the first message, only when they didn't ask for brainstorming**. If taken: the skill is done, *the gate goes with it*; state any significant how-call in one plain line; do the work through the normal workflow. We have no exit. Their HARD-GATE lists exactly two exceptions — opted-out questions and approved spikes — we have the spike, not the opt-out. **Change:** one line in the first-message recipe + gate exception in the `<HARD-GATE>` + record the opt-out (commit context / interview log) so it's a decision, not a silence.

### 7. The "genuinely one sensible way" exit

*"When there's genuinely one sensible way, say so in that message and explain why the others don't fit; the design comes in the next one."* Ours mandates "Propose 2–3 approaches" always. Manufacturing alternatives to look thorough is itself a failure, and the architecture-option gate (no pick) is incoherent when there's only one way. **Change:** single-approach exit in "Exploring approaches": present the one way + what was considered and rejected with reasons (which still fills `Alternatives Weighed` / the briefing's Approaches table).

### 8. Show, don't tell — text sketches

*"Whenever seeing would help more than reading, sketch it in text: a few lines of what they'd see."* Their trigger list is broader than ours and includes non-UI: *something they'll look at / a layout or arrangement / a flow or sequence / options that would look different / a structure that's easier to see than read (a diagram, a timeline, a map)*. Ours: the sketch skill is the only visual path, and it's heavyweight (interactive mockups). A text sketch costs nothing and works on the **Bounded path** (whose output is "a few sentences to a few short paragraphs" — today nothing visual) and in playback chunks. **Change:** one rule in `questioning` (sketchable chunk → text sketch next to it) + a line in the Bounded checklist in `paths.md` + the non-UI trigger list.

### 9. Mid-conversation spike (just-in-time Prototype)

*"Agentic work is waterfall, but very, very fast. A quick throwaway build is often the cheapest way to learn what they want or whether something works. Offer one when it would help. When they just want to spike, get out of the way."* Our Prototype path is an **upfront classification**; we have just-in-time *research* (external facts) but no just-in-time *spike* for mid-interview questions where a 30-minute throwaway answers better than any description ("which of these two flows feels right?" — build both, click them). Answer comes back into the tree; artifacts stay throwaway; keeping them is a new request (we already have that rule). **Change:** a note in `questioning`'s rules: when a decision would be better resolved by a short throwaway than by description, offer the spike; don't block the rest of the frontier on it.

### 10. Ground to cover (territory list)

Six areas, explicitly *"territory, not a script. Never read these out as questions"*: what kind of thing / why now + success / goals, non-goals, **anti-goals** / **prior art: what they've seen elsewhere, loved or hated** / current state: what exists, **what they've tried** / the details they care about. Ours seeds the design tree from nothing; the clarity-gate dimensions *measure* coverage but don't *suggest* it. The two sharpest items we lack: **anti-goals** ("what would make this a failure even if it technically works" — distinct from out-of-scope, which is "not building this time") and **prior art / what they've tried** (a rich intent source). **Change:** territory list in `questioning/SKILL.md`, marked non-exhaustive.

---

## Tier 3 — small, cheap, do them in the same pass

| Port | Where | One line |
|---|---|---|
| Red-flag rows | `brainstorm/SKILL.md` | "I'll offer a menu" / "I'll fill in sensible defaults — Good. Say them out loud. A default they never heard is one they couldn't reject" / "I should explain the process first — ask your question. The process shows itself" / "'sounds good' to the idea — approval covers what you showed them; a doc you haven't written isn't approved" |
| Register rule | `brainstorm/SKILL.md` | Plain language, *pitched just a little above where they are*, match their vocabulary, **no process jargon in chat** (frontier/ratchet/Elephant/Goldfish stay in files), never quote the skill to the user |
| Findings discipline | `brainstorm/SKILL.md` | "give them the most important findings and offer the rest" (recon reports) |
| Process steps are your calls | `brainstorm/SKILL.md` | "writing things up, where files go, looking further — do them and say so" (config answers where-files-lands; don't ask) |
| Size the work | `brainstorm/SKILL.md` | one plain-words size line at design presentation, user can override (our ratchet is the override mechanism) |
| Visual-offer protocol | `paths.md` | declined → don't offer again unless they raise it; "a question *about* a UI topic is not automatically a visual question" |
| "Left to the builder" | `templates/briefing.md` | explicit intentional-delegation framing in Open Questions & Assumptions — the builder check skips deliberate delegations, so they must be named |

**Also worth porting as evals** (our `evals.json` has 4 scenarios; this format makes new behaviors testable): user hands a detailed spec → expect echo + confirm-before-building; user says "just dive in" → opt-out honored, no interview; one obviously-right approach → no manufactured alternatives; design written → builder check dispatched, 3-pile disposition.

---

## What we should NOT port (we're already stronger)

| Theirs | Ours |
|---|---|
| Binary sizing (small / project), "step up a size" | 4-path classification + decision table + one-way ratchet, with the shootdown riding the ratchet |
| "Several independent projects → agree on order, brainstorm one at a time" | Scope Pushback: list the pile, recommend first slice, hold/cut/split, **pick recorded** before blueprint |
| Doc at `docs/superpowers/specs/…`, "pick a sensible place" | Spec-zone only, config-driven paths, filenames = phase signals for `resume` (machine-checkable) |
| Hard gate: no building before approval | Same + No-Code boundary + hypotheses-validated-before-any-document + spec-zone gate |
| Approval of the doc, then plans | Self-review → shootdown → user hand-edits STATUS banner → **intent frozen** contract |
| No state | Spec-zone artifacts ARE the phase state; resume re-enters at first incomplete step |
| Visual-companion **server** (port, session key, HTML fragments, click events) | We have the `skillgrid:sketch` mechanism — port only the protocol lines above, not their infra |

Keep the builder check and the shootdown **separate**: one tests the document, the other attacks the design. Merging them into "review" loses both.

---

## Cross-cutting issues found while reading (fix in the same pass)

1. **`brainstorm/references/questioning.md` is orphaned AND diverged.** The standalone `questioning` skill (frontmatter: "questioning reference extracted") is *thinner* than the reference it was extracted from: the reference carries the clarity **weights/minimums table** (0.75/0.70/0.65/0.70), the **confidence stop**, the **three-question test**, the **out-of-scope line**, the **domain-model completeness check**, and the mnemonic posting bridge — **none of which are in the skill**. Nothing links to `references/questioning.md` by name, yet the briefing template's Clarity Report cites the minimums that are defined *only* in that orphaned file. Fix: merge the exit checks into `questioning/SKILL.md` (it's the exit contract), turn the reference into a one-line pointer (cite-don't-restate).
2. **`skillgrid:sketch` is referenced but doesn't exist in this repo** — cited in `brainstorm/references/paths.md` (both full paths), `blueprint/references/codebase-design.md`, `blueprint/templates/blueprint.md`, and `skillgrid/references/patterns/config.yaml`, but there's no `.agents/skills/sketch/`. The just-in-time sketch-offer step points at a missing skill (or it lives only globally — worth deciding).
3. **A spec-zone violation is on disk:** `.skillgrid/change-specs/2026-11-23-mnemonic-core/brainstorm.md` (13 KB, "pre-decision record") — exactly the filename the spec-zone gate forbids. Historical or live, it's evidence the filename rule does real work (or that `skillgrid-guards` should check for it).
4. **Budget headroom:** `brainstorm` is 27.5 KB of the 38 KB "large" ceiling. The changes above fit, but the placement rule matters: first-message + tone in `SKILL.md` (short), conversation texture in the `questioning` skill (not in the size-budget map — confirm it should be), builder-check prompt as a small new reference (mirrors the `shootdown.md` → `goldfish.md` pattern).

## Proposed file map

```
brainstorm/SKILL.md                  + Your first message (§, ~30 lines, incl. opt-out + spec-confirm)
                                     ~ Exploring approaches: one-sensible-way exit, how/what split
                                     ~ Presenting the design: Calls-I-made list, text sketch, size line
                                     + 4 red-flag rows, 3 register lines, process-steps line
questioning/SKILL.md                  + Play it back (§), Ground to cover (§), options-steer rule,
                                       mid-conversation spike note, text-sketch line
                                     ~ merge orphaned reference's exit checks (weights, confidence
                                       stop, three-question test, out-of-scope line, domain-model)
brainstorm/references/after-the-design.md   + builder-check step (between self-review and user gate)
brainstorm/references/builder-check-prompt.md  NEW (~20-line dispatch prompt, 3-pile disposition)
brainstorm/references/questioning.md   → pointer to skillgrid:questioning
brainstorm/references/paths.md         ~ declined→no-reoffer line; Bounded: text sketch when output is visible
brainstorm/templates/briefing.md       ~ "left to the builder" framing; decided-by column in interview log
brainstorm/evals/evals.json            + 2-3 scenarios for the new behaviors
```

**Suggested execution order:** two commits — (1) the behavioral ports + the questioning drift fix, (2) the builder check (new prompt file + after-the-design step + evals), so each is reviewable.
