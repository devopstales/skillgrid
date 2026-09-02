---
name: design-spike
description: "Build a throwaway prototype to answer a design question — state machine, data shape, or UI look. Use when the user wants to sanity-check whether a state model or logic feels right, or explore what a UI should look like before committing; not for production code or feature work."
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
---

# design-spike

A prototype is **throwaway code that answers a question**. The question decides the shape.

## Pick a branch

Identify which question is being answered, using the user's prompt, the surrounding code, or by asking if the user is around:

- **"Does this logic / state model feel right?"** → [logic.md](references/logic.md). Build a single shareable HTML file (free-play buttons plus tabbed guided walkthroughs) that pushes the state machine through cases that are hard to reason about on paper, and that a non-developer can drive.
- **"What should this look like?"** → [ui.md](references/ui.md). Generate several radically different UI variations on a single route, switchable via a URL search param and a floating bottom bar.

The two branches produce very different artifacts, so getting this wrong wastes the whole prototype. If the question is genuinely ambiguous and the user isn't reachable, default to whichever branch better matches the surrounding code (a backend module → logic; a page or component → UI) and state the assumption at the top of the prototype.

## Rules that apply to both

1. **Throwaway from day one, and clearly marked as such.** Locate the prototype code close to where it will actually be used (next to the module or page it's prototyping for) so context is obvious, but name it so a casual reader can see it's a prototype, not production. For throwaway UI routes, obey whatever routing convention the project already uses; don't invent a new top-level structure.
2. **Trivial to run.** A UI prototype starts from one command in the project's task runner: `pnpm <name>`, `python <path>`, `bun <path>`, etc. A logic demo is a single HTML file the user double-clicks. Either way, no thinking required to start it.
3. **No persistence by default.** State lives in memory. Persistence is the thing the prototype is *checking*, not something it should depend on. If the question explicitly involves a database, hit a scratch DB or a local file with a clear "PROTOTYPE, wipe me" name.
4. **Skip the polish.** No tests, no error handling beyond what makes the prototype *runnable*, no abstractions. The point is to learn something fast.
5. **Surface the state.** After every action (logic) or on every variant switch (UI), print or render the full relevant state so the user can see what changed.
6. **Capture it when done.** See [Capture](#capture-when-done) below.

## Where it fits in SDD

A prototype is a **spike for a pending decision**, not a change of its own. It produces a *validated answer* that lands in an existing SDD artifact:

- **Before `sdd-propose`** — prototype to resolve an open question the intent should own (e.g. "is this state model even feasible?"). The answer goes into the intent's step blueprint or a `## Decisions` section.
- **Before or during `sdd-design`** — prototype to resolve an architecture question that the plan must answer (e.g. "does this reducer shape actually handle the illegal transition?"). The answer goes into the plan's Architecture Decisions (Choice/Alternatives/Rationale).
- **Not during `sdd-apply`** — by the time you are applying, the plan is committed; a "prototype" during apply is just production code, so it is not a prototype. Stop and surface to the user that this needs to go back through design.

The prototype folder itself does **not** live under `docs/skillgrid/changes/<NNN-slug>/`; it lives next to the code it's answering. The *answer* does.

## Capture (when done)

1. **Fold the validated decision into the real code** (or into the SDD artifact that owns it, per [Where it fits](#where-it-fits-in-sdd)).
2. **Commit the prototype to a throwaway branch**, out of `main`, so it stays re-runnable as a primary source. Name the branch `prototype/<NNN-slug>/<branch-keyword>` if tied to an SDD change, else `prototype/<topic>`.
3. **Leave a context pointer** to that branch on the implementing issue (Backlog.md ticket, GitHub PR, or — for an SDD change — the owning step's `tasks.md` or the plan's Decisions section).
4. **Capture the answer** in the SDD artifact and, when the change is in flight, persist a Mnemonic observation:

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
     **Branch**    prototype/<NNN-slug>/<branch-keyword>
     **Branch files** <paths in the branch>
     **Landed in** <intent.md §X / plan.md §Architecture Decisions / <issue id>>
     **Date**      <ISO>
     """
   )
   ```

   This is the recovery point — a future `mem_search("sdd/<NNN-slug>/prototype")` finds the branch, the verdict, and where it landed, even after the branch is deleted.
5. **Record the commit chain** per the shared commit convention ([commits.md](../_shared/conventions/commits.md)): the prototype branch's commits are checkpoint commits with the question in the subject; the decision landing in `main` is a `feat:` or `decision:` that references the prototype branch in the footer (`Refs: prototype/<NNN-slug>/<branch-keyword>`).
6. **Main branch keeps only the validated decision.** The prototype does not merge into `main`.

## References

- [references/logic.md](references/logic.md) — shareable-HTML state-model demo (single file, free-play buttons, guided walkthroughs).
- [references/ui.md](references/ui.md) — multi-variant UI on one route with a floating switcher.
- [mnemonic-memory.md](../_shared/conventions/mnemonic-memory.md) — save shape and `sdd/<NNN-slug>/…` topic keys.
- [commits.md](../_shared/conventions/commits.md) — prototype-branch and decision commits.

## Gotchas

- A prototype that adds tests, DB migrations, abstractions, or "while we're at it" production work is not a prototype. Stop and switch to `sdd-apply`.
- Variants that differ only in colour or copy are wallpaper, not a UI prototype. Real disagreement = different structure.
- Do not promote the throwaway HTML shell or variant components into `main`. The decision is the only thing that promotes; the artifact stays on the throwaway branch.
- If the question turned out to be "no, that model doesn't work," the **verdict is still the answer**. Record the rejection in the same Mnemonic shape — a rejected prototype is a valid primary source too.
- `mem_search` returns 300-char previews only; always `mem_get_observation(id)` to read the full answer before citing it.
- At session end: `mem_session_summary` then `mem_session_end`.
