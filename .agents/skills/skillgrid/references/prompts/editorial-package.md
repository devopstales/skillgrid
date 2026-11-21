# Phase 7: Editorial Package

Package the change so someone who did not build it can run it, review it and find out what changed. This phase edits documents and moves the change folder; it does not add features.

When to load: Phase 7. Role: orchestrator, with the release specialist and the prior-art map from Phase 0.

## Steps

1. **Document.** Use `skillgrid:document` to write from the real diff and commits, never from memory:
   - `README.md` — what the program does, how to install and run it, one worked example, the supported versions, and where the docs live. A reader must reach a working command without reading source.
   - `CHANGELOG.md` — one entry for this change under the current version, in the repo's existing style.
   - The PR body — the problem, the change, the evidence (tests run, gates), the risks, the rollback.
   - The release note — the user-facing summary, with the breaking changes first.
2. **Check the package.** Every command in `README.md` was run in this tree and its output recorded. Every path exists. Every version number matches the manifest. The prior-art map from Phase 0 is updated with what this program now does that the comps do not.
3. **Decide.** The `ship-decision` gate is the programmer's, always. Present the QA verdict, the review floor, the score, the scope gate and the open windows in one screen, and ask. Never merge on your own.
4. **Ship.** Use `skillgrid:ship`. Before it removes the change entry, copy the final `changes.<slug>.genesis` block into `report.md` § Genesis — the entry is gone afterwards and `reflect` reads the report. Then let `ship` move `.skillgrid/change-specs/<topic>/` to `.skillgrid/change-archive/<topic>/` and verify with an empty `diff -r` readback.
5. **Reflect.** Use `skillgrid:reflect` for the retro half and `skillgrid:environment-retro` for the environment half. The record covers what worked, what cost the most, which assumption was wrong, and the one change to the workflow worth keeping.
6. **Final inventory.** List the files created and changed, the work units, every test run with its command and result, the gates and their verdicts, the open `WINDOWS.md` entries, and the next step. This is the last thing the programmer reads.

## Rules

- The archive folder is immutable. A defect found after `ship` becomes a new change, not an edit to the archived one.
- A release note that claims a fix must point at the test that proves it.
- If the `ship-decision` gate is `no`, stop cleanly: leave the folder in `change-specs/`, record the reason in `report.md`, and list what would need to change.
