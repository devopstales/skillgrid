# Postmortem Template

Write to `.skillgrid/postmortems/<date>-<slug>.md` (directory created on first
use). Audience: the team that must not repeat this. Blameless: name the
conditions, not the person.

```markdown
# Postmortem: [incident name] — [date]

**Status:** [draft | final]
**Severity:** [S1 | S2 | S3]
**Detected:** [when + how it was noticed — the source: a user report, an
alert, a test, a review]
**Resolved:** [when + how]

## Impact

[Who was affected, for how long, what was broken. Source: the incident facts
the user supplied + the change folder. Be specific — "checkouts failed for
~40 minutes for users on the EU deployment" not "some users had issues"].

## Timeline

All times in [timezone]. Every line names its source (an alert, a commit, a
log line, a message):

- [time] — [what happened] (source: [alert/commit/log])
- [time] — [what happened] (source: ...)

Include: when it started, when it was noticed (and the gap), what was tried,
when it was resolved. The notice gap is a finding, not an embarrassment.

## Root Cause

[The proven cause, with its evidence. Source: the debug state file
(.skillgrid/sdd/debug/<date>-<slug>/state.md) if the fix ran
skillgrid:structured-debugging, or the ADR / change folder. State the cause,
then the evidence that confirmed it. If the cause is a decision, not code, say
so — name the decision and where it is recorded.]

## What went well

- [detection that worked, a rollback that was clean, a test that caught it
  early — name the thing]

## What went wrong

- [detection that was slow, a missing test, a config that drifted — name the
  condition, not the person]

## Actions

Each action: what, owner, due date, and whether it is done. No action without
an owner is a wish.

| Action | Owner | Due | Status |
|---|---|---|---|
| [concrete, verifiable action] | [name] | [date] | [open | done] |

## Lessons

[1-3 factual sentences the team should carry forward. These are the
mem_save-shaped learnings — they also feed the Mnemonic capture.]
```

Rules:
- Blameless: "the deploy pipeline had no canary step" not "X forgot the canary
  step".
- Every timeline line and every "what went wrong" bullet names its source.
  An unsourced claim in a postmortem is a rumor.
- If the root cause is a bad decision, the Actions must include the decision
  review — not just the code patch. Patching around a bad decision is how the
  same incident ships twice.
- The Lessons section is what survives: concrete, falsifiable, carry-forward.
