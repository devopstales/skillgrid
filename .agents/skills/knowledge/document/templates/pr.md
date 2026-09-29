# PR Body Template

Copy and fill from the gathered source material. Every bracketed line names its
source — if you cannot name it, cut the line.

```markdown
## What

[2-4 sentences: what this change does and why. Source: briefing.md goal +
blueprint.md approach. Lead with the user-visible outcome, not the mechanics.]

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
