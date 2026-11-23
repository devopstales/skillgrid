---
type: Gotcha
title: "git C-quotes non-ASCII paths in --porcelain output; JSON.parse throws on the octal escapes"
status: verified
sources:
  - ".skillgrid/change-archive/2026-11-22-pi-worktree-plugin/report.md (QA finding B1)"
---

# git C-quotes non-ASCII paths in --porcelain output

When `core.quotePath` is on (its **default**), git C-quotes any path that is
not pure printable ASCII: `git diff --name-status --porcelain` returns
`R100 "caf\xc3\xa9.md" "na\xefme.md"` — a double-quoted string whose non-ASCII
bytes are escaped as octal (`\xxx`). Two traps:

1. **`JSON.parse` throws.** The C-quoted form is *not* valid JSON (octal
   escapes are a git extension, not a JSON escape). `JSON.parse('"caf\xc3\xa9.md"')`
   is a SyntaxError. The non-ASCII case was previously "covered" only by a test
   that asserted the *un-quoted* name and never exercised the parser.
2. **The name must be unquoted, then decoded.** Spawn the git command with
   `-c core.quotePath=false` so git emits the raw UTF-8 path (no quotes, no
   octal), and keep a small unquote fallback for any path that still arrives
   C-quoted.

## The check

- Any code that parses `git diff --name-status --porcelain` (or another
  `--porcelain`/quoted output) for filenames must spawn with
  `-c core.quotePath=false` **and** unquote a `"…"`-wrapped field before use.
- Regression test: create + rename a non-ASCII file (`café.md`), run the
  parser, assert the decoded name is present and the rename is detected.
- Symptom of the bug: non-ASCII paths throw or are absent from the parsed
  list, while ASCII paths work — the happy-path test passes, the Unicode one
  fails only when the octal form is actually produced.
