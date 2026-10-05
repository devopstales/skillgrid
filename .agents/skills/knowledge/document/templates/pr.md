# PR Body Template

Copy and fill from the gathered source material. Every bracketed line names its
source — if you cannot name it, cut the line.

```markdown
## What

[2-4 sentences: what this change does and why. Source: briefing.md goal +
blueprint.md approach. Lead with the user-visible outcome, not the mechanics.
Pick the SMALLEST visual that makes the change clear, then the prose: pseudocode
for logic, a call tree for runtime control flow, a component/file tree for
structure, Mermaid for interaction, or a diff sketch when the point is what
changes. Match the visual to the topic — one or two, not all of them.]

[optional: the smallest visual — pseudocode / call tree / file tree / Mermaid /
diff sketch. Source: the real diff. Keep only the calls, files, and boundaries
that make the point.]

## Changes

[Per-ticket or per-area bullets. Source: git diff --name-status BASE...HEAD +
tasks.md. Each bullet names what changed and where.]

- [area/file] — [what changed, one line]
- [area/file] — [what changed, one line]

## Evidence

[Source: report.md → ## Gate Decision verdict + its named evidence (test command, exit code,
coverage/mutation numbers). Name the gate, not a feeling about it.]

- QA gate: [PASS | WAIVED | CONCERNS + override] — [evidence: command + result]
- [named truth → named test that proves it, 1-3 of the most important]

## Merge Danger

[Is this a one-way or two-way door? A two-way door can be walked back cheaply —
low risk. A one-way door is hard to reverse (a migration, a data change, a public
contract, a delete) — call it out and say why it can or can't be rolled back.
Then the blast radius: one word for scope (e.g. "auth", "all of checkout",
"layout") plus any real ramifications — breaking consumers, deploy ordering,
reversal cost. Source: the diff + tasks.md.]

**Door:** [one-way | two-way]
**Blast Radius:** [one word] — [optional: what could go wrong / cost to reverse]

## Decisions

[Source: the ADRs this change touched. One line per decision: what was chosen
and the one-line why. Omit if the change made no design decisions.]

## Notes for Review

[Anything a reviewer must know that is not in the diff: a migration that runs
on deploy, a config flag that must be set, a known limitation that is
intentional. Omit if none.]
```

Rules:
- Title: `<type>(<scope>): <subject>` matching the work-unit commit style
  (`feat`, `fix`, `chore`, …).
- If the repo has a PR template (`.github/PULL_REQUEST_TEMPLATE*`), follow its
  structure and fill it from the same sources.
- No marketing. No "this PR implements the epic". State what is true.
