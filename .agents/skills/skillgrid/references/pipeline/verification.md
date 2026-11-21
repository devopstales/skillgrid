# Verification and recorded exceptions

The exit checklist for a genesis run, plus the deliberate deviations this bundle carries. Load before the final inventory. Role: orchestrator.

## Exit checklist

Every box is evidence-gated. "Seems right" is not evidence.

- [ ] `state.yaml` matches the deepest artifact present; `node .agents/skills/qa/scripts/state-drift-check.mjs` exits 0.
- [ ] Every phase in `manifest.yaml` has its listed outputs on disk and was read back after writing.
- [ ] `changes.<slug>.genesis.independence` is filled, and every verdict in `report.md`/`review.md` names the same grade.
- [ ] Phase 4 ran before any score; the audit packet contains no targets, earlier scores or panel verdicts.
- [ ] The QA gate verdict is `PASS` or `WAIVED`, or `CONCERNS`/`FAIL` with a recorded human decision.
- [ ] `review.md` floor is `met` or `met-with-fixes`; no unfixed Critical or Important finding.
- [ ] `ship` moved the folder to `change-archive/` and `diff -r` readback was empty.
- [ ] `report.md` § Genesis was written before the change entry was removed.
- [ ] The final inventory lists files, work units, tests with commands and output, open `WINDOWS.md` entries, next step.

## Recorded exceptions (deliberate, not defects to "fix" silently)

| Exception | Reason | Owner |
|---|---|---|
| The entry file is `SKILL.md` but the folder is `001_skillgrid` | the `001_` prefix is a catalog slot, not part of the skill name; `name: skillgrid` is kept for content continuity | bundle maintainer |
| `references/patterns/glossary.md` is the single glossary source | a live `.skillgrid/` vault inside a skill would drift and would violate "no `.skillgrid/` in the bundle" | bundle maintainer |
| `references/patterns/ai-tells-pt-br.md` is not ported | code is language-neutral; PT-BR prose artifacts follow `skillgrid:_shared/rules/prose-style.md` | bundle maintainer |
| No Phase 6 prompt file | `references/scoring/program-score.md` is the Phase 6 procedure | bundle maintainer |
| `state.yaml` carries an optional `genesis:` block | chosen over a sidecar; requires `sdd-structure.md` to document the block and `ship`/`reflect` to preserve it via `report.md` § Genesis | shared-contract owner |

## What this skill never does

- It never creates `.skillgrid/` inside its own folder. The folder is created only in the programmer's project root.
- It runs onboarding itself (`references/pipeline/onboarding.md`); onboarding is part of this skill, not a separate user-invoked skill.
- It never claims a test passed, a file saved, or a program shipped without the command and its output.
