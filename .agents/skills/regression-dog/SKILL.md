---
name: regression-dog
description: 'Review code for regressions. TRIGGER when: reviewing changes, checking for breakage, or auditing commits (e.g., "did I break anything", "review last commit").'
---

Review code changes for behavioral differences between the before and after code.

Important:
- Do NOT run tests, typechecks, linters, or build commands. CI already handles those. Focus your context budget entirely on reasoning about the code changes and its implications on logic/behavior/data/etc.
- Enumerate behavioral differences: "this used to do X, now it does Y." Do not judge whether the old or new behavior is correct — just surface the delta. Do not flag pre-existing issues or suggest improvements.

Output format:
- List regressions first, numbered, with severity.
- After the regressions section, add a "Cleared" section listing items that were reviewed and found to have no issues. Prefix each cleared item with a ✅ emoji.

Scope (pass as arguments):
- No arguments: review the most recent commit
- `main`: review all commits since the last merge from main
- `HEAD~3`: review the last 3 commits
- Any git revision range (e.g., `abc123..HEAD`, `main..HEAD`)
