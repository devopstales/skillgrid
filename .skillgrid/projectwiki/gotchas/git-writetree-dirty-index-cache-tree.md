---
type: Gotcha
title: "git write-tree on a dirty/staged index persists the cache-tree extension — the index file is no longer byte-identical even when the tree SHA is"
status: verified
sources:
  - ".skillgrid/change-archive/2026-11-22-pi-worktree-plugin/report.md (QA finding D)"
---

# write-tree on a dirty/staged index persists the cache-tree extension

`git write-tree` writes a tree object and — as a side effect — may update the
on-disk index: it persists the **cache-tree** extension (the cached tree SHAs
per directory) so the next status/commit is faster. Consequence:

- After `git add` (staging) or any `write-tree` on a dirty/staged index, the
  **index file bytes change** (cache-tree extension + staged entries) even
  though the **tree SHA that `write-tree` returns is unchanged**.
- So "the tree SHA is the same" is NOT the same as "the index file is
  byte-identical". A test that restores `HEAD^{tree}` into the index to prove
  a before-restore invariant must not also assert the index *file* is
  byte-identical to a pre-stage snapshot — it will fail on the cache-tree
  extension.

## The check

- To assert "the tree is unchanged" compare the **tree SHA** (`git write-tree`
  output / `git rev-parse HEAD^{tree}`), not the index file bytes.
- To assert "the working tree / index content is unchanged" use a content
  query (`git diff --quiet`, `git status --porcelain` empty), not a byte
  comparison of `.git/index`.
- Symptom of the bug: a before-restore test that staged files, then restored
  the tree, then compared the index file to a pre-stage snapshot — fails on
  the cache-tree extension while the tree SHA is identical.
