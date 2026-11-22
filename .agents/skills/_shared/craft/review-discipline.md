# Review Discipline (anti-sycophancy + stress-testing)

The single home for the tone and vocabulary rules the Goldfish review applies —
the shootdown lenses (brainstorm), the 5-stage plan review (blueprint), and the
prose audit (execute style guide). Cite this file; never restate the list in a
skill body. A banned word has exactly one home (here) and appears nowhere else.

## The rules

A reviewer — subagent or main agent synthesizing — reports what it found, at the
severity it earned. Softening is a defect:

- **A Blocker stays a Blocker until the user disposes of it.** Dedupe and group
  without diluting. Never drop a 🔴 Blocker to make the verdict look cleaner,
  and never invent a finding a lens did not report.
- **Verdicts are not vibes.** "Looks good" with no named check is no verdict.
  Name the check that passed, or say what is unverified.
- **Contest with evidence, not opinion.** "This was intentional" is not
  evidence; a citation to a line, a test, or an ADR is.
- **No sycophancy in either direction.** Do not praise to please, do not attack
  to seem rigorous. The finding is the finding; the relationship is irrelevant.
- **Artifact-first at a gate.** Never present a large report, plan, or proposal
  in the chat pane at a decision gate. Write it to a file, give the user the
  path, then ask for review. Chat carries the path + the question, not the
  artifact.
- **Zero-tool turn-yield.** At a human decision gate, the turn ends with zero
  tool calls — go idle and wait for the human. During an item-by-item decision
  walk, no file-editing tool runs until every item is decided: collect the
  decisions, then apply them in one batch.

## Banned vocabulary (the glazing list)

These carry no information and are deleted on sight:

- **Flattery / self-aggrandizing:** *elite, world-class, cutting-edge,
  game-changing, robust, seamless, elegant, powerful, sophisticated,
  best-in-class, state-of-the-art.*
- **Hedges that add nothing:** *simply, just, essentially, effectively,
  fundamentally, quite, very, really, arguably.*
- **Additive-only transitions:** *Additionally, Furthermore, Moreover* opening
  a sentence the reader did not need.
- **Corporate filler:** *leverage, methodology, synergy, unlock, empower,
  circle back, touch base.*

The escape hatch is the Orwell rule 6 in `_shared/rules/prose-style.md`: break
the ban when a sentence reads better for the word — but the default is cut.

## Cited, not restated

The banned list and the rules above live only in this file. A skill that
names a banned adjective or re-explains "a Blocker stays a Blocker" is
restating — replace the restatement with `per _shared/craft/review-discipline.md`.
