# The user panel

Four users, built per program, not one generic reader. They judge whether people keep using the program, which no test and no score can tell you.

When to load: Phase 0 (build the cards), Phase 3 (unit and wave checks), Phase 5 (compare mode and gate read). Role: reviewer.

## The four users

| User | Built from | Cares about | Votes |
|---|---|---|---|
| Primary | the brief's target user and the prior-art comps | does the main job get done, quickly, without surprises | keep / fix / quit |
| Hostile | the failure modes in the threat matrix and the comps' issue trackers | what breaks, what is slow, what is unsafe, what is undocumented | keep / fix / quit |
| Adjacent | the nearest neighbouring use case the program does not target | can I stretch it to my case without forking it | keep / fix / quit |
| Casual | fixed: a competent developer who read the README only | can I run it in ten minutes and understand the first error | keep / fix / quit |

Each card is one file in the change folder's `evaluations/` and carries: who they are, what they tried, what they expected, what happened, the one thing they would change, and their verdict. The card names the surface it judged.

## Modes

- **Unit read** — one work unit, judged on whether it earns its place. Used after the tracer.
- **Wave read** — the latest user-visible surface after a wave.
- **Compare** — two versions of the same unit, unlabeled and in random order. Used to accept or reject a revision. The panel never learns which is the revision.
- **Gate read** — the whole change at Phase 5, sampling the surfaces that changed most. This is the vote the review gate reads.

## How to run it

1. Give each user only what that user would have: the surface, the README or help text, and their own card. Never the blueprint's targets, the score, or the audit findings.
2. Ask each for a verdict and the one defect that decided it.
3. A defect raised by two or more users is a **two-vote defect**. It is a ticket in the next revision pass; one vote is a note.
4. Record the verdicts in `evaluations/` and summarize them in `review.md`.

## Reading the gate

The review gate passes when every sampled surface has a majority of users choosing keep, the `review.md` floor is met, and no high-severity defect remains. If a surface fails, name the users who quit and the defect that decided it. Do not average the users into a single score; a program that half the users abandon is not half good.

## Rules

- The panel judges use, not craft. It never sees the code's structure and never grades it.
- A user who cannot state what they tried did not use the program. Re-run the card.
- The panel is stable across a run. Do not swap a hostile user for a friendly one after a bad verdict.
- The panel never overrides the audit, and the audit never overrides the panel. Both gates must pass.
