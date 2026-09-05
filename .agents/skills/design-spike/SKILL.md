---
name: design-spike
description: "Optional throwaway prototype before locking change.md — taste/UI, architecture, or external smoke. Use when orchestrator/propose needs a concrete answer first; commit marked PROTOTYPE; list path as Prototype: in change.md. Not production; not during sdd-apply."
license: MIT
metadata:
  author: devopstales
  version: "1.1"
  part-of: skillgrid
---

# design-spike

A prototype is **throwaway code that answers a question**. The question decides the shape. Optional **before** locking `change.md` — never production work during apply.

## Pick a branch

Identify which question is being answered, using the user's prompt, the surrounding code, or by asking if the user is around:

- **"Does this logic / state model / arch shape feel right?"** → [logic.md](references/logic.md). Build a single shareable HTML file (free-play buttons plus tabbed guided walkthroughs) that pushes the state machine through cases that are hard to reason about on paper, and that a non-developer can drive.
- **"What should this look like?"** (taste / UI) → [ui.md](references/ui.md). Generate several radically different UI variations on a single route, switchable via a URL search param and a floating bottom bar.
- **"Does this external API / unknown shape even work?"** (external smoke) — smallest runnable call or harness against the real dependency; print raw responses; no product UI. Still throwaway and marked PROTOTYPE.

These branches produce different artifacts, so getting this wrong wastes the whole prototype. If the question is ambiguous and the user isn't reachable, default to whichever branch better matches the surrounding code (backend → logic; page/component → UI; third-party API → external smoke) and state the assumption at the top of the prototype.

## Rules that apply to all branches

1. **Throwaway from day one, and clearly marked as such.** Locate the prototype code close to where it will actually be used (next to the module or page it's prototyping for) so context is obvious, but name it so a casual reader can see it's a prototype, not production. For throwaway UI routes, obey whatever routing convention the project already uses; don't invent a new top-level structure.
2. **Trivial to run.** A UI prototype starts from one command in the project's task runner: `pnpm <name>`, `python <path>`, `bun <path>`, etc. A logic demo is a single HTML file the user double-clicks. Either way, no thinking required to start it.
3. **No persistence by default.** State lives in memory. Persistence is the thing the prototype is *checking*, not something it should depend on. If the question explicitly involves a database, hit a scratch DB or a local file with a clear "PROTOTYPE, wipe me" name.
4. **Skip the polish.** No tests, no error handling beyond what makes the prototype *runnable*, no abstractions. The point is to learn something fast.
5. **Surface the state.** After every action (logic) or on every variant switch (UI), print or render the full relevant state so the user can see what changed.
6. **Capture it when done.** See [Capture](#capture-when-done) below.

## Where it fits in SDD

Optional **before locking `change.md`** (orchestrator / propose pre-gate). Typical triggers: taste/UI uncertainty, architecture shape, external smoke.

- **Before / while shaping `change.md`** — answer the open question; fold Choice/Alternatives/Rationale (or Step Blueprint) into `change.md`; list the path as `Prototype: <path>`.
- **Not during `sdd-apply`** — apply is production against a locked change. A "prototype" mid-apply is production code. Stop; send the work back through propose (and re-spike if needed).

The prototype itself does **not** live under `docs/skillgrid/changes/<NNN-slug>/`; it lives next to the code it's answering. The *answer* and the `Prototype:` pointer live in `change.md`.

## Capture (when done)

1. **Fold the validated decision** into `change.md` (Architecture Decisions / Step Blueprint).
2. **Commit marked PROTOTYPE** is OK — subject or body must say `PROTOTYPE` so nobody mistakes it for production. Prefer a throwaway branch `prototype/<NNN-slug>/<keyword>` (or `prototype/<topic>`); keep it re-runnable.
3. **List the path in `change.md`**: `Prototype: <relative-path>` (and `Research:` when explore ran). Context pointer only — do not paste the prototype into the change folder.
4. **Capture the answer** in Mnemonic when a change is in flight:

   ```
   mem_save(
     title:      "sdd/<NNN-slug>/prototype-<branch-keyword>",
     topic_key:  "sdd/<NNN-slug>/prototype-<branch-keyword>",
     type:       "discovery",
     scope:      "project",
     session_id: {sid},
     content: """
     ## Prototype answer — <branch-keyword>

     **Question**  <the exact question this prototype settled>
     **Verdict**   <one sentence: what was validated / rejected>
     **Path**      <Prototype: path>
     **Branch**    prototype/<NNN-slug>/<branch-keyword>
     **Landed in** <change.md §Architecture Decisions>
     **Date**      <ISO>
     """
   )
   ```

5. **Commit chain** per ([commits.md](../_shared/conventions/commits.md)): PROTOTYPE checkpoint commits answer the question; the decision that lands in production is a separate `feat:` / `docs:` that references the prototype (`Refs: Prototype: <path>` or branch name).
6. **Do not merge the throwaway prototype as production.** Only the validated decision promotes.

## References

- [references/logic.md](references/logic.md) — shareable-HTML state-model demo (single file, free-play buttons, guided walkthroughs).
- [references/ui.md](references/ui.md) — multi-variant UI on one route with a floating switcher.
- [mnemonic-memory.md](../_shared/conventions/mnemonic-memory.md) — save shape and `sdd/<NNN-slug>/…` topic keys.
- [commits.md](../_shared/conventions/commits.md) — prototype-branch and decision commits.

## Gotchas

- Spiking during `sdd-apply` is not a prototype — return to propose.
- Missing `Prototype:` in `change.md` after a kept spike leaves apply blind.
- A prototype that adds tests, DB migrations, abstractions, or "while we're at it" production work is not a prototype.
- Variants that differ only in colour or copy are wallpaper, not a UI prototype. Real disagreement = different structure.
- Do not promote the throwaway HTML shell or variant components as production. The decision is the only thing that promotes.
- If the question turned out to be "no, that model doesn't work," the **verdict is still the answer**. Record the rejection the same way.
- `mem_search` returns 300-char previews only; always `mem_get_observation(id)` before citing.
- At session end: `mem_session_summary` then `mem_session_end`.
