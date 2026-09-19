# Release Note Template

Write to `.skillgrid/releases/<version>.md` (directory created on first use).
Audience: end users. Plain language. No internal names — no ticket IDs, no
module names, no "the API layer". Lead with what they can now do.

```markdown
# [Version] — [Date]

[1-2 sentences: what this release is about, in the user's words. Source: the
change topics this version contains, translated to outcomes.]

## What's new

- [what the user can now do, one line, plain words]
- [what the user can now do, one line, plain words]

## What changed

- [behavior that changed, and what the user should do about it if anything,
  one line each]

## What was fixed

- [what used to go wrong and now works, one line each, in the user's words —
  "signing in on a slow network no longer drops you" not "fixed timeout in
  auth handler"]

## Known issues

- [anything the user may still hit, one line each. Omit if none.]

## Upgrade notes

[Anything the user must do to upgrade — a new setting, a renamed page, a data
change. Omit if the upgrade is automatic.]
```

Rules:
- Every line traces to a shipped change (the source is the change folder /
  changelog entry; you are translating it, not inventing it).
- "Fixed" lines describe the user's symptom, not the code. "The list no longer
  loses your scroll position" — not "fixed state hydration in the list
  component".
- No version-to-version diff talk. The user does not care what commit 412
  changed; they care what the product does now.
- The QA verdict is NOT in the release note (that is the PR/changelog record).
  A known-issue line is the honest way to surface a limitation.
