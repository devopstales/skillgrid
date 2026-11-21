# Host contract

How SkillGrid runs inside the programmer's own agent: start, check in, recover, and finish, with no model runner or background service of its own.

**Programmer** is the person directing the build. **User** is the person who runs the finished program. They are not the same role; see `references/patterns/glossary.md`.

When to load: at the start of every session on a project, and whenever you resume one. Role: orchestrator.

## Start or resume

1. Locate the installed `skillgrid/SKILL.md`. Resolve every `references/...` path relative to that folder, never relative to the `.skillgrid` folder.
2. Confirm you can read this skill and write the programmer's `.skillgrid` folder and working tree. If a tool or permission is missing, say which one. Never claim a file was saved, a commit landed, or a test passed when it was not.
3. If the `.skillgrid` folder has `config.yaml`, read it together with `state.yaml`, `projectwiki/ASSUMPTIONS.md`, `projectwiki/ARCHITECTURE.md`, `projectwiki/index.md` and the files that actually exist. Reconcile before acting: a heading, an empty file, a template or a stale `phase:` is not a finished task. Artifacts beat state — if `state.yaml` disagrees with the deepest artifact in `change-specs/<topic>/`, trust the artifact and patch the state.
4. If the project is new, follow `references/pipeline/onboarding.md`. Onboarding is part of this skill — run it yourself; do not wait for the programmer to invoke a separate skill.
5. If the project exists, find the change entry and its deepest artifact. Load that phase's prompt from `references/pipeline/manifest.yaml` and continue from there.

## Check in

Check in at every phase boundary, at every gate, after the tracer unit, after every wave, and on any `major_rewrite` or third failed pass.

A check-in is a short screen, not a file dump:

1. What was done, with the artifact path and the evidence (command + output).
2. What is next and what it will cost (waves, files, tests).
3. The two or three decisions worth the programmer's attention, with a recommendation.
4. Any assumption that turned out wrong, and what changed because of it.
5. The one question that would change the plan if answered differently.

Wait for the answer. Do not proceed past a gate on silence.

## Modes

- **check-in** (default): stop at each check-in and wait.
- **autonomous**: the programmer granted a run to the next gate. Write the check-in screen to `report.md` instead of stopping, keep going, and still stop on `major_rewrite`, a third failed pass, or the `ship-decision` gate. Autonomy is recorded per change in `state.yaml`; it is never assumed.

## Recover from a failure

When a tool fails, a file does not match, or the state disagrees:

1. Stop. Do not guess and continue.
2. State what failed, what you expected, and what you found.
3. Repair from disk: re-read the artifact, re-run the command, or rebuild the packet.
4. If the repair needs a decision, ask. If it needs a rollback, use the unit's rollback boundary and record it in `report.md`.
5. Resume from the deepest artifact that is actually correct, not from the phase number.

A run that silently skips a broken step is worse than a run that stops.

## Finish

The run ends at Phase 7 with the final inventory, or earlier with a recorded stop. Never end a session by saying "done" without naming:

- the deepest artifact on disk and the phase it belongs to;
- the gates and their verdicts, including any `CONCERNS`/`FAIL`;
- the next action, for the programmer or for the next session.

## What the host does not do

- It does not run a server, a scheduler or a background model. Every step happens in the programmer's session.
- It does not write `.skillgrid/` inside this skill's folder. The folder belongs to the programmer's project root only.
- It runs onboarding itself (`references/pipeline/onboarding.md`); onboarding is part of this skill.
