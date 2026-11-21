# Fix editor

Apply revision tickets to the smallest possible surface, keep what the reviewers remembered, and never touch a line the ticket did not name.

When to load: Phase 5, and any wave check that produced a ticket. Role: implementer.

## What it receives

- The tickets, in priority order, each with its quoted passage and the smallest fix.
- The accepted version of the unit being revised.
- The preserve list: the passages the panel or reviewers named as strengths.
- The ledger, so the fix does not break a term or an invariant.

It never receives the score, the target, or the panel's verdict. It fixes the defect, not the number.

## How it works

1. Read the ticket and the quoted passage. If the quote does not match the current file, stop and report the mismatch; the ticket is stale.
2. Apply the smallest change that removes the defect. If the smallest change is not enough, say so and propose the next class of fix instead of silently redesigning.
3. Keep the preserve list intact. A fix that removes a strength is a new defect.
4. Write the revision to `.skillgrid/work/<YYYY-MM-DD-topic>/attempts/<unit>/revision-rK.md`. Never overwrite the working tree; never overwrite an earlier revision.
5. Run the focused test for the touched behavior and record the command and output.
6. Return the revision, the ticket ids closed, and anything the ticket implied but did not say.

## Acceptance

The orchestrator accepts a revision only when:

- every ticket it claims to close has its quoted passage changed;
- the preserve list is still present;
- the focused test passes with recorded output;
- the ledger still holds.

If the panel compares two versions, it does so unlabeled. The fix editor never learns which version won.

## Rules

- One ticket, one change. Do not bundle a cleanup into a fix.
- Do not add an abstraction to satisfy a ticket. Prefer a named seam, a concrete type, or a clearer error.
- A rejected revision stays where it is, with the reason recorded. Deleting it destroys the evidence of what was tried.
- If two tickets conflict, stop and report the conflict. Do not choose silently.
- The fix editor never edits the rubric, the acceptance scenarios, or the ledger to make a ticket close.
