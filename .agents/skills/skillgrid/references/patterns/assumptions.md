# ASSUMPTIONS.md template

`.skillgrid/projectwiki/ASSUMPTIONS.md` is the durable project understanding. Four tiers, nothing else. Operational rules belong in `AGENTS.md` § Rules, not here.

```markdown
# ASSUMPTIONS

## VERIFIED
<!-- Facts confirmed against the code, a command output, or a source with a date. -->
- [fact] — evidence: [file:line | command output | URL, checked YYYY-MM-DD]

## INFERRED (HYPOTHESIS)
<!-- Guesses the programmer can correct in one reply. One line each, with the reason. -->
- [inference] — because [the words or evidence that suggested it]

## LOCKED
<!-- The ADR index only. One pointer row per in-force decision.
     status / supersedes / date live in the ADR frontmatter, not the row. -->
| Decision | ADR |
|---|---|
| [short description] | `decisions/NNNN-slug.md` |

## Open Questions
<!-- Disagreements or unknowns waiting on the programmer. -->
- [question] — asked YYYY-MM-DD
```

Rules:

- On a fresh project the tiers start empty; they fill as the work produces evidence.
- A verified fact cites its evidence. Without evidence it stays INFERRED.
- A locked decision is a pointer. The body is the ADR file; never inline it.
- When the programmer corrects an inference, move the line (INFERRED → VERIFIED or delete) and note the date.
