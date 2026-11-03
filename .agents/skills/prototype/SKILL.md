---
name: prototype
description: Run a throwaway feasibility experiment to answer a "will this technical approach work?" question. Produces a falsifiable verdict (VALIDATED / INVALIDATED / PARTIAL) with evidence, not an opinion. Use when the design or blueprint rests on an unproven technical claim — a library's real behavior, an integration's feasibility, a performance question, a data-shape question.
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
  based_on: gsd-core:gsd-spike + mattpocock-skills:prototype (LOGIC branch — liftable pure module)
---

# Prototype

**Announce at start:** "I'm using the skillgrid:prototype skill to test this before we
commit to it."

## Overview

Answer a feasibility question by **building a focused experiment** and reading what
happens. The output is a **verdict with evidence** — `VALIDATED`, `INVALIDATED`, or
`PARTIAL` — not a recommendation you arrived at by reasoning. A prototype's value is the
investigation trail (what was tried, what surprised, which edge cases broke), not the
one-line conclusion.

The experiment is **throwaway by design.** It is built to be felt and inspected, not
to ship. The one thing you lift into the real codebase is a **pure module** (see the
Liftable Pure Module section) — the rest stays in the prototype directory, labeled as
throwaway.

## When to Use

- When a technical risk must be de-risked before committing to an approach — the design or blueprint rests on an unproven technical claim (a library's real behavior, an integration's feasibility, a performance question, a data-shape question).
- When an unknown (API, library, integration) needs a time-boxed hands-on answer — code must actually run to produce the answer.

**When NOT to use:** when the answer is already known from the codebase or a quick lookup — a prototype is for genuine unknowns, not confirmation. A question that is purely about a *fact* not in the codebase (a library's current API, a version's behavior, a competitor's offering) is `skillgrid:research`, not a prototype. If the question is "can we build the whole feature this way end-to-end," you may want a tracer bullet instead; a prototype stays small and single-purpose.

## Config

Read `.skillgrid/config.yaml` before starting. Prototype directories live under
`{prototype.dir}` (default `.skillgrid/prototypes`) — numbered `NNN-descriptive-name/`,
**permanently retained** (not under the spec zone, so they are not pruned when a change
ships). The liftable-module location comes from `prototype.module_dir` (default
`{specs_root}/{topic}/modules`); the consolidated findings file comes from
`prototype.findings` (default `{specs_root}/{topic}/findings.md`). If the `mnemonic` block
is enabled, the findings file you write is cited by `skillgrid:writing-blueprints`
downstream — so keep it factual and specific, not prose.

## The Verdict (three states, evidence-gated)

Every prototype ends with exactly one of:

- **`VALIDATED` ✓** — the approach works for the stated purpose. Requires evidence
  beyond a single happy-path test: at least one edge case exercised, OR a comparison
  prototype with a head-to-head result. Never declare VALIDATED after one green run.
- **`INVALIDATED` ✗** — the core assumption is false. Name what exactly broke and the
  evidence. An invalidated prototype is as valuable as a validated one — it kills a dead
  end before the real build hits it.
- **`PARTIAL` ⚠** — works with constraints (only for X input size, only without
  concurrency, requires a workaround). State the constraint explicitly; the real build
  must know the boundary.

**The evidence rule:** a verdict is a claim with a demonstration. If you cannot point
to specific output, a log line, a measured number, or a screenshot that proves the
verdict, the verdict is `PARTIAL` at best with an `unverified` flag — never
`VALIDATED` by vibes.

## Prototype Types

| Type | Use when | Numbering |
|------|----------|-----------|
| **standard** | one approach answering one question | `NNN-name` |
| **comparison** | same question, two approaches — "does A or B work better?" | `NNN-a-name` / `NNN-b-name` (shared number, letter suffix) |

A comparison prototype builds both variants back-to-back, then writes a head-to-head
section in the verdict naming the winner and why. A comparison's verdict is `VALIDATED`
on the winner (with the loser's failure or weakness as evidence).

## Risk Ordering

When a session has multiple prototypes, the **prototype most likely to kill the idea runs
first.** A prototype whose invalidation would change the whole design is higher risk than
one whose invalidation only changes a detail. Order by that, not by ease. If you have
one prototype, skip this.

## The Process

### Step 1: Frame the hypothesis

Take the idea and write it as a **falsifiable Given/When/Then** before touching code:

> Given [precondition], when [action], then [expected outcome].

If you cannot write the "then" as something observable (a value returned, a state
reached, a latency under a bound, a render that appears), the question is not yet
specific enough — sharpen it. Present the hypothesis in 2-3 sentences to the user and
get a nod before building. (A tiny prototype the user already framed in their own message
does not need a separate approval round — the frame IS the approval.)

### Step 2: Set up the prototype directory

Create `{prototype.dir}/NNN-descriptive-name/` (default `.skillgrid/prototypes/NNN-name/`). The
`NNN` is the next free number in the prototypes directory (three digits, zero-padded).
Prototypes are project-scoped and permanently retained — the directory is a checkpoint the
blueprint cites, and it survives the change that motivated it. The directory holds:

```
.skillgrid/prototypes/NNN-name/
├── prototype.md       # hypothesis, how to run, investigation trail, verdict (the record)
└── <code>         # the throwaway experiment itself
```

Use the project's language/framework by default. For a prototype, **avoid** build tools,
bundlers, transpilers, containers, and config systems — hardcode everything. The goal
is the shortest path to a runnable result, not a realistic setup. (If the prototype's
whole point is "does our build config work," the build config is the experiment.)

### Step 3: Research the current state (skip if pure logic)

If the prototype depends on an external library, API, or framework, read its **current**
documentation before coding — `context7` (resolve-library-id → query-docs) for
libraries, `exa`/`webfetch` for the rest. A prototype built on a stale mental model of a
library's API validates the model, not the approach. Surface competing approaches as a
short table (approach / tool / pros / cons) and note which you chose and why. Skip this
step for pure-logic prototypes with no external dependencies.

### Step 4: Build the demo (bias toward something you can feel)

**The default is to build something the user can interact with**, not stdout only the
agent reads. The user wants to *feel* the prototype working. This could be:

- a single HTML page that shows the result visually
- a page with a button that triggers the action and shows the response
- a page displaying data flowing through a pipeline
- a minimal interface where the user can try different inputs and see outputs

**Only fall back to stdout/CLI when the prototype is genuinely about a fact, not a
feeling:** a pure data transformation ("yes it parses correctly"), a binary yes/no
question ("does this API authenticate?"), or a benchmark number ("how fast is X?").

When in doubt, build the UI. It costs a few extra minutes and produces a prototype the
user can demo and trust.

For a **state-machine or reducer prototype**, follow the Liftable Pure Module pattern
below — drive the logic from an inline `<script>` block and label it as the
production-liftable unit.

### Step 5: Iterate on findings (depth over speed)

The goal is genuine understanding, not a quick verdict. After the first run:

- **Surprising surface?** Write a follow-up test that isolates and explores it.
- **Answer feels shallow?** Probe edge cases — large inputs, concurrent requests,
  malformed data, network failure, empty state.
- **Assumption wrong?** Adjust. Note the pivot in the investigation trail.

Multiple files per prototype are expected for complex questions (`test-basic`,
`test-edge-cases`, `benchmark`). Document each iteration in the investigation trail:
what you tried, what it revealed, what you tried next.

### Step 6: Write the record and verdict

Fill the scaffold at [templates/prototype.md](templates/prototype.md) →
`prototypes/NNN-name/prototype.md`. The **investigation trail** is the section that makes the
prototype worth keeping — it is the record of the reasoning, not just the conclusion.

Set the verdict. Commit the prototype directory (`git add` + `git commit`) — the artifact
is a checkpoint the blueprint cites.

### Step 7: Append to the topic's consolidated findings

If the topic has a `.skillgrid/specs/YYYY-MM-DD-<topic>/findings.md` (created by
`skillgrid:sketch` or a prior prototype), **append** a `## Prototype: NNN-name` section to it.
If the file does not exist, **create it** with a `# Findings — <topic>` header and the
first prototype section. The consolidated file is the single downstream contract that
`skillgrid:writing-blueprints` reads — it must contain every prototype's and sketch's
verdict, what's liftable, and the constraints for the build.

The per-prototype section in `findings.md` is compact (not a copy of `prototype.md`):

```markdown
## Prototype: 001-redis-streams

- **Verdict:** VALIDATED ✓
- **What we learned:** Streams handle the reconnect-with-redelivery case; the
  consumer group must use `XAUTOCLAIM` after X seconds of idle, not re-subscribe.
- **What's liftable:** `reducer.js` (pure state machine) at `prototypes/001-redis-streams/`
- **Constraints for the build:** reconnection must not re-read the stream from the
  start; use last-delivered-id. The library's `stream.read()` blocks — use the async
  variant.
```

### Step 8: Report

Report: the verdict, the 2-3 findings that drive it (not the verdict alone — the
surprises, the edge cases explored, the constraint boundaries), what's liftable, and
the path to the prototype directory. The design (`skillgrid:brainstorming`) or the
blueprint (`skillgrid:writing-blueprints`) reads the file; it does not re-run the
experiment.

If the core assumption was invalidated, present a decision:
**continue with remaining prototypes / pivot the approach / abandon** — and wait for the
answer before proceeding.

## Liftable Pure Module (the one thing that survives)

When a prototype drives a **state machine, reducer, or pure function** in an inline
`<script>` block, that block is the production-liftable unit. This is the key
difference from a throwaway demo: the demo is discarded, but the pure module is
designed to be dropped into the real codebase unchanged.

Rules for the liftable module:

- **Pure.** No DOM, no fetch, no side effects. Takes state + action, returns new
  state. The `<script>` block that *drives* it (wires up buttons, renders) is
  throwaway; the module it calls is liftable.
- **Labeled.** Mark it in `prototype.md` with a `## Liftable Module` section: the file
  path, what it does, its input/output signature, and any dependencies (ideally none).
- **Named for the real codebase.** Use the terms files' vocabulary (`artifacts/01-business-terms.md` /
  `02-technical-terms.md`) for its names — a
  module called `reducer.js` with functions named in domain terms lifts cleanly; one
  called `stuff.js` with `doIt()` needs renaming at lift time.
- **No prototype-specific constants.** If the module needs a value that's only meaningful
  in the prototype (a fake URL, a hardcoded dataset), inject it as a parameter, not a
  literal — so the real build can pass its own.

The blueprint (or a later task) lifts the module by copying the file and wiring the
real I/O around it. The prototype's job is to prove the *logic* works; the build's job is
to surround it with the real plumbing.

## Forensic Log Layer (when the prototype is hard to judge visually)

If the prototype involves runtime behavior that is hard to judge by looking (concurrency,
timing, a sequence of events), add a forensic log layer:

1. An event log array with ISO timestamps and category tags
2. An export mechanism (browser: an Export button; CLI: a JSON file; server: a GET
   endpoint)
3. A log summary (event counts, duration, errors, key metadata)
4. Analysis helpers if the volume warrants them

This lets the user (and the verdict) judge the prototype on actual runtime evidence rather
than a single observed frame.

## Common Rationalizations

| Excuse | Reality |
|--------|---------|
| "It works, that's enough" | One happy-path run is not evidence. Exercise an edge case or run a comparison, or the verdict is PARTIAL. |
| "I know this library, no need to read docs" | A prototype built on a stale mental model validates the model, not the approach. Read the current docs for the dependency. |
| "I'll keep the code, it's almost production-ready" | A prototype's output is an answer + a liftable pure module. Keeping the *demo* is a new request — classify it as a feature and go through the normal flow. |
| "The question is small, I'll skip the file" | The file is what the blueprint cites. Even a two-finding prototype gets the scaffold — the verdict and the constraint, in a place that survives the session. |
| "I'll just assert the verdict without showing the output" | The evidence rule: a verdict is a claim with a demonstration. Point to the specific output, log line, number, or screenshot that proves it. |
| "I'll build a full app to test it" | A prototype is small and single-purpose. A full end-to-end build is a tracer bullet, not a prototype. Keep it to the one question. |

## Red Flags

**Never:**

- Declare VALIDATED after a single happy-path run with no edge case
- Let the demo's throwaway code leak into the "what's liftable" section (only the pure
  module is liftable)
- Hardcode prototype-specific constants inside the liftable module (inject them as params)
- Skip the investigation trail because the verdict is obvious (the trail is the record)
- Present a prototype as a recommendation without the verdict + evidence (it's an answer,
   not an opinion)

## Verification

- [ ] The prototype produced a verdict — one of the three states (`VALIDATED` / `INVALIDATED` / `PARTIAL`) — that is evidence-gated, not vibes.
- [ ] The verdict cites the concrete evidence that produced it: specific output, a log line, a measured number, a screenshot, or a head-to-head comparison result (not a single happy-path run for `VALIDATED`).
- [ ] The investigation trail in `prototype.md` records what was tried, what surprised, and which edge cases broke — not just the conclusion.
- [ ] Any liftable pure module was extracted, labeled in `prototype.md` (`## Liftable Module`), and preserved at the `prototype.module_dir` path.
- [ ] The prototype artifacts (prototype directory + consolidated `findings.md` section) were saved and committed — the record the blueprint cites survives the session.
- [ ] If the runtime behavior was hard to judge visually, the forensic log layer (event log + export + summary) captured the evidence the verdict rests on.
