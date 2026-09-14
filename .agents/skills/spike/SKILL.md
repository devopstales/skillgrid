---
name: spike
description: Run a throwaway feasibility experiment to answer a "will this technical approach work?" question. Produces a falsifiable verdict (VALIDATED / INVALIDATED / PARTIAL) with evidence, not an opinion. Use when the design or blueprint rests on an unproven technical claim — a library's real behavior, an integration's feasibility, a performance question, a data-shape question.
# based on gsd-core:gsd-spike + mattpocock-skills:prototype (LOGIC branch — liftable pure module)
---

# Spike

Answer a feasibility question by **building a focused experiment** and reading what
happens. The output is a **verdict with evidence** — `VALIDATED`, `INVALIDATED`, or
`PARTIAL` — not a recommendation you arrived at by reasoning. A spike's value is the
investigation trail (what was tried, what surprised, which edge cases broke), not the
one-line conclusion.

The experiment is **throwaway by design.** It is built to be felt and inspected, not
to ship. The one thing you lift into the real codebase is a **pure module** (see the
Liftable Pure Module section) — the rest stays in the spike directory, labeled as
throwaway.

**Announce at start:** "I'm using the skillgrid:spike skill to test this before we
commit to it."

**When a spike is the wrong tool:** a question that is purely about a *fact* not in the
codebase (a library's current API, a version's behavior, a competitor's offering) is
`skillgrid:research`, not a spike — a spike is for a question that needs code executed
to answer. If the question is "can we build the whole feature this way end-to-end,"
you may want a tracer bullet instead; a spike stays small and single-purpose.

## Config

Read `.skillgrid/config.yaml` before starting. Use `conventions.specs_root` (default
`.skillgrid/specs`) as the base. Spike artifacts live under
`{specs_root}/YYYY-MM-DD-<topic>/spikes/`; the liftable-module location comes from
`spike.module_dir` (default `{specs_root}/{topic}/modules`); the consolidated findings
file comes from `spike.findings` (default `{specs_root}/{topic}/findings.md`). If the
`mnemonic` block is enabled, the findings file you write is cited by
`skillgrid:writing-blueprints` downstream — so keep it factual and specific, not prose.

## The Verdict (three states, evidence-gated)

Every spike ends with exactly one of:

- **`VALIDATED` ✓** — the approach works for the stated purpose. Requires evidence
  beyond a single happy-path test: at least one edge case exercised, OR a comparison
  spike with a head-to-head result. Never declare VALIDATED after one green run.
- **`INVALIDATED` ✗** — the core assumption is false. Name what exactly broke and the
  evidence. An invalidated spike is as valuable as a validated one — it kills a dead
  end before the real build hits it.
- **`PARTIAL` ⚠** — works with constraints (only for X input size, only without
  concurrency, requires a workaround). State the constraint explicitly; the real build
  must know the boundary.

**The evidence rule:** a verdict is a claim with a demonstration. If you cannot point
to specific output, a log line, a measured number, or a screenshot that proves the
verdict, the verdict is `PARTIAL` at best with an `unverified` flag — never
`VALIDATED` by vibes.

## Spike Types

| Type | Use when | Numbering |
|------|----------|-----------|
| **standard** | one approach answering one question | `NNN-name` |
| **comparison** | same question, two approaches — "does A or B work better?" | `NNN-a-name` / `NNN-b-name` (shared number, letter suffix) |

A comparison spike builds both variants back-to-back, then writes a head-to-head
section in the verdict naming the winner and why. A comparison's verdict is `VALIDATED`
on the winner (with the loser's failure or weakness as evidence).

## Risk Ordering

When a session has multiple spikes, the **spike most likely to kill the idea runs
first.** A spike whose invalidation would change the whole design is higher risk than
one whose invalidation only changes a detail. Order by that, not by ease. If you have
one spike, skip this.

## The Process

### Step 1: Frame the hypothesis

Take the idea and write it as a **falsifiable Given/When/Then** before touching code:

> Given [precondition], when [action], then [expected outcome].

If you cannot write the "then" as something observable (a value returned, a state
reached, a latency under a bound, a render that appears), the question is not yet
specific enough — sharpen it. Present the hypothesis in 2-3 sentences to the user and
get a nod before building. (A tiny spike the user already framed in their own message
does not need a separate approval round — the frame IS the approval.)

### Step 2: Set up the spike directory

Create `{specs_root}/YYYY-MM-DD-<topic>/spikes/NNN-descriptive-name/`. The `NNN` is the
next free number in that topic's spikes directory (three digits, zero-padded). The
directory holds:

```
spikes/NNN-name/
├── spike.md       # hypothesis, how to run, investigation trail, verdict (the record)
└── <code>         # the throwaway experiment itself
```

Use the project's language/framework by default. For a spike, **avoid** build tools,
bundlers, transpilers, containers, and config systems — hardcode everything. The goal
is the shortest path to a runnable result, not a realistic setup. (If the spike's
whole point is "does our build config work," the build config is the experiment.)

### Step 3: Research the current state (skip if pure logic)

If the spike depends on an external library, API, or framework, read its **current**
documentation before coding — `context7` (resolve-library-id → query-docs) for
libraries, `exa`/`webfetch` for the rest. A spike built on a stale mental model of a
library's API validates the model, not the approach. Surface competing approaches as a
short table (approach / tool / pros / cons) and note which you chose and why. Skip this
step for pure-logic spikes with no external dependencies.

### Step 4: Build the demo (bias toward something you can feel)

**The default is to build something the user can interact with**, not stdout only the
agent reads. The user wants to *feel* the spike working. This could be:

- a single HTML page that shows the result visually
- a page with a button that triggers the action and shows the response
- a page displaying data flowing through a pipeline
- a minimal interface where the user can try different inputs and see outputs

**Only fall back to stdout/CLI when the spike is genuinely about a fact, not a
feeling:** a pure data transformation ("yes it parses correctly"), a binary yes/no
question ("does this API authenticate?"), or a benchmark number ("how fast is X?").

When in doubt, build the UI. It costs a few extra minutes and produces a spike the
user can demo and trust.

For a **state-machine or reducer spike**, follow the Liftable Pure Module pattern
below — drive the logic from an inline `<script>` block and label it as the
production-liftable unit.

### Step 5: Iterate on findings (depth over speed)

The goal is genuine understanding, not a quick verdict. After the first run:

- **Surprising surface?** Write a follow-up test that isolates and explores it.
- **Answer feels shallow?** Probe edge cases — large inputs, concurrent requests,
  malformed data, network failure, empty state.
- **Assumption wrong?** Adjust. Note the pivot in the investigation trail.

Multiple files per spike are expected for complex questions (`test-basic`,
`test-edge-cases`, `benchmark`). Document each iteration in the investigation trail:
what you tried, what it revealed, what you tried next.

### Step 6: Write the record and verdict

Fill the scaffold at [templates/spike.md](templates/spike.md) →
`spikes/NNN-name/spike.md`. The **investigation trail** is the section that makes the
spike worth keeping — it is the record of the reasoning, not just the conclusion.

Set the verdict. Commit the spike directory (`git add` + `git commit`) — the artifact
is a checkpoint the blueprint cites.

### Step 7: Append to the topic's consolidated findings

If the topic has a `.skillgrid/specs/YYYY-MM-DD-<topic>/findings.md` (created by
`skillgrid:sketch` or a prior spike), **append** a `## Spike: NNN-name` section to it.
If the file does not exist, **create it** with a `# Findings — <topic>` header and the
first spike section. The consolidated file is the single downstream contract that
`skillgrid:writing-blueprints` reads — it must contain every spike's and sketch's
verdict, what's liftable, and the constraints for the build.

The per-spike section in `findings.md` is compact (not a copy of `spike.md`):

```markdown
## Spike: 001-redis-streams

- **Verdict:** VALIDATED ✓
- **What we learned:** Streams handle the reconnect-with-redelivery case; the
  consumer group must use `XAUTOCLAIM` after X seconds of idle, not re-subscribe.
- **What's liftable:** `reducer.js` (pure state machine) at `spikes/001-redis-streams/`
- **Constraints for the build:** reconnection must not re-read the stream from the
  start; use last-delivered-id. The library's `stream.read()` blocks — use the async
  variant.
```

### Step 8: Report

Report: the verdict, the 2-3 findings that drive it (not the verdict alone — the
surprises, the edge cases explored, the constraint boundaries), what's liftable, and
the path to the spike directory. The design (`skillgrid:brainstorming`) or the
blueprint (`skillgrid:writing-blueprints`) reads the file; it does not re-run the
experiment.

If the core assumption was invalidated, present a decision:
**continue with remaining spikes / pivot the approach / abandon** — and wait for the
answer before proceeding.

## Liftable Pure Module (the one thing that survives)

When a spike drives a **state machine, reducer, or pure function** in an inline
`<script>` block, that block is the production-liftable unit. This is the key
difference from a throwaway demo: the demo is discarded, but the pure module is
designed to be dropped into the real codebase unchanged.

Rules for the liftable module:

- **Pure.** No DOM, no fetch, no side effects. Takes state + action, returns new
  state. The `<script>` block that *drives* it (wires up buttons, renders) is
  throwaway; the module it calls is liftable.
- **Labeled.** Mark it in `spike.md` with a `## Liftable Module` section: the file
  path, what it does, its input/output signature, and any dependencies (ideally none).
- **Named for the real codebase.** Use the glossary's vocabulary for its names — a
  module called `reducer.js` with functions named in domain terms lifts cleanly; one
  called `stuff.js` with `doIt()` needs renaming at lift time.
- **No spike-specific constants.** If the module needs a value that's only meaningful
  in the spike (a fake URL, a hardcoded dataset), inject it as a parameter, not a
  literal — so the real build can pass its own.

The blueprint (or a later task) lifts the module by copying the file and wiring the
real I/O around it. The spike's job is to prove the *logic* works; the build's job is
to surround it with the real plumbing.

## Forensic Log Layer (when the spike is hard to judge visually)

If the spike involves runtime behavior that is hard to judge by looking (concurrency,
timing, a sequence of events), add a forensic log layer:

1. An event log array with ISO timestamps and category tags
2. An export mechanism (browser: an Export button; CLI: a JSON file; server: a GET
   endpoint)
3. A log summary (event counts, duration, errors, key metadata)
4. Analysis helpers if the volume warrants them

This lets the user (and the verdict) judge the spike on actual runtime evidence rather
than a single observed frame.

## Common Rationalizations

| Excuse | Reality |
|--------|---------|
| "It works, that's enough" | One happy-path run is not evidence. Exercise an edge case or run a comparison, or the verdict is PARTIAL. |
| "I know this library, no need to read docs" | A spike built on a stale mental model validates the model, not the approach. Read the current docs for the dependency. |
| "I'll keep the code, it's almost production-ready" | A spike's output is an answer + a liftable pure module. Keeping the *demo* is a new request — classify it as a feature and go through the normal flow. |
| "The question is small, I'll skip the file" | The file is what the blueprint cites. Even a two-finding spike gets the scaffold — the verdict and the constraint, in a place that survives the session. |
| "I'll just assert the verdict without showing the output" | The evidence rule: a verdict is a claim with a demonstration. Point to the specific output, log line, number, or screenshot that proves it. |
| "I'll build a full app to test it" | A spike is small and single-purpose. A full end-to-end build is a tracer bullet, not a spike. Keep it to the one question. |

## Red Flags

**Never:**

- Declare VALIDATED after a single happy-path run with no edge case
- Let the demo's throwaway code leak into the "what's liftable" section (only the pure
  module is liftable)
- Hardcode spike-specific constants inside the liftable module (inject them as params)
- Skip the investigation trail because the verdict is obvious (the trail is the record)
- Present a spike as a recommendation without the verdict + evidence (it's an answer,
  not an opinion)
