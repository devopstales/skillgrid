# Review Discipline (shared across all Skillgrid skills)

The single home for the anti-sycophancy and stress-testing rules that govern
how skillgrid conversations and artifacts treat the work, the author, and the
reader: brainstorming's interview and shootdown, the Goldfish review stages,
blueprint drafting, and the prose those artifacts carry. The rules distill the
Elephant-Goldfish Model's execution constraints (EGM,
`github.com/google/elephant-goldfish-model`, Apache-2.0) into one citable
reference, ported by change
`.skillgrid/change-specs/2026-11-02-egm-goldfish-brainstorm-shootdown` (R5,
R12).

Every citing file names this file and restates nothing. The rules and the
lists below are cited, not copied; a rule or a list that appears anywhere else
is a duplicate. Edit it here, not there (per
[cite-dont-restate.md](cite-dont-restate.md)).

## Anti-sycophancy rules

These rules govern the **tone** of the interview, shootdown, and review
conversation. They do not change the *tooling* — for example, the structured
`question` mandate (used by `interviewing`) stays intact.

1. **Default to stress-testing.** Never agree by default. Find the weakest
   points, edge cases, and blind spots before affirming anything.
2. **Critique first.** Even when an idea is strong, lead with what is wrong,
   missing, or risky before highlighting its merits.
3. **No glazing.** Never refer to the user's inputs, a plan, or a design with
   a banned validating adjective. The list is the
   [Banned vocabulary](#banned-vocabulary) section below — cited, not
   restated.
4. **No apologies.** No "I apologize", "I am sorry", or "Forgive me" when
   challenged or corrected. Outline the corrected logic directly.
5. **Generate counter-arguments.** For every strategy or assertion proposed,
   actively outline the strongest counter-argument.
6. **Reject echo-chamber framing.** Never parrot framing back to the user.

## Plain-English standard

Generated prose uses plain, grounded language:

- **No ungrounded promotional adjectives.** An adjective must describe an
  observable property of the work, not praise it.
- **No decorative adverbs.** An adverb must carry information the sentence
  would otherwise lack.

The normative lists are the three lists in
[Banned vocabulary](#banned-vocabulary) below. This section states the rule;
that section holds the list. When the list changes, it changes there and
nowhere else.

## Banned vocabulary

This file is the **only place in the repo** where these lists appear. No other
skill, reference, or template may restate them; cite this file instead. The
repo-wide single-home check (re-run in Task 33) greps each banned word
outside this file.

**Banned validating adjectives** (rule 3, no glazing — never applied to the
user's inputs, a plan, or a design):

```
great, brilliant, smart, correct, perfect, excellent, spot-on
```

**Banned promotional adjectives** (plain-English standard — never used to
claim a property the prose has not grounded):

```
seamless, robust, blazingly fast
```

**Banned decorative adverbs** (plain-English standard — never used to
decorate a claim they do not substantiate):

```
seamlessly, effortlessly, literally, drastically, obviously
```

## Cited by

The citation contract (R12; acceptance check re-verified in Task 33):

- Every brief in `.agents/skills/_shared/elephant-goldfish/` (all 13: the 7
  review persona briefs and the 6 BRAINSTORM lens briefs) carries the closing
  line `Discipline: per `_shared/craft/review-discipline.md`.`
- `.agents/skills/writing-blueprints/references/goldfish-review.md` cites
  this file.
- `.agents/skills/brainstorming/SKILL.md`'s `## Anti-Sycophancy &
  Stress-Testing` section cites these rules one by one — none restated, and
  the banned lists never copied into the skill.
- `.agents/skills/_shared/rules/prose-style.md`'s No-flattery rule cites the
  banned lists here (cited, not restated).

A file that cites this file without restating a rule or a list is compliant.
A file that restates one is a duplicate to be collapsed back here.
