# Code Standards (global convention)

Single source of truth for **how code is designed and structured** in this repo —
language-agnostic. Every change, in every language, honors this file. When code
disagrees with it, **this file wins**. Language *idioms* (a Go error-wrapping
rule, a React hooks rule) live in ADRs or per-language notes — this file is the
**global** layer: the principles that hold no matter the syntax.

Companion contracts:
- [commits.md](commits.md) — commit message contract.
- [verification-ladder.md](verification-ladder.md) — L1–L4 evidence floors (the gate runs the tests this file defines).
- [deterministic-boundary.md](../craft/deterministic-boundary.md) — the same "logic that repeats identically belongs in a script, not prose" idea, applied to code: repeatable logic becomes a function, not inline steps.
- `skillgrid:ponytail` — for choosing the simplest implementation *within* these standards.

## Design principles

These are the load-bearing rules. Everything else is a consequence.

1. **Single Responsibility.** A module (package, file, class, component) has one
   reason to change. If you can't name that reason in one sentence, split it.
2. **Separation of Concerns.** Mixing "what to show" with "where the data comes
   from" or "how to persist" is a smell. Keep presentation, orchestration, and
   I/O in separate layers.
3. **Dependency direction is a law.** High-level policy depends on low-level
   detail, never the reverse. The core/leaf imports outward to the edges
   (CLI, web, storage) — **never** the other way. If `A` needs `B` and `B` needs
   `A`, one is in the wrong layer.
4. **Explicit over implicit.** If a dependency, a side effect, or a config value
   isn't visible in the signature/declaration, it's hidden. Pass it in; name it.
5. **Fail fast, fail loud.** Validate at the boundary, surface errors early,
   never swallow. A silent fallback is a bug you find in production, not one you
   catch in review.
6. **YAGNI, then delete.** Don't build the abstraction until the second use
   case exists. When you do, delete the special case — one general path beats
   two with a flag.

## Designing deep modules

Design **deep modules**: a lot of behavior behind a small interface, at a clean
seam, testable through that interface. The aim is **leverage** for callers and
**locality** for maintainers. Use these terms exactly — don't drift into
"component," "service," "API," or "boundary."

**Vocabulary:**
- **Module** — anything with an interface and an implementation. Scale-agnostic:
  a function, class, package, or tier-spanning slice.
- **Interface** — *everything* a caller must know to use the module correctly:
  the type signature, plus invariants, ordering constraints, error modes,
  required config, and performance characteristics. Not just the type-level
  surface.
- **Depth** — leverage at the interface: the behavior a caller (or test) can
  exercise per unit of interface they have to learn. **Deep** = large behavior
  behind a small interface. **Shallow** = the interface is nearly as complex as
  the implementation.
- **Seam** — where you can alter behavior without editing in that place; the
  location where a module's interface lives. Where to put it is its own design
  decision, distinct from what goes behind it.
- **Adapter** — a concrete thing that satisfies an interface at a seam. Names
  the *role*, not the substance.
- **Leverage** / **Locality** — what depth buys: callers get more capability per
  unit learned; maintainers get change, bugs, and verification concentrated in
  one place.

**Rules:**
- **Depth is a property of the interface, not the implementation.** A deep
  module can be internally composed of small, swappable parts; they just aren't
  part of the interface.
- **The deletion test.** Imagine deleting the module. If complexity vanishes,
  it was a pass-through. If complexity reappears across N callers, it was
  earning its keep. (Adjacent to, not the same as, `deterministic-boundary.md`:
  that decides whether *logic* is a function; this decides whether a *module*
  earns its keep.)
- **The interface is the test surface.** Callers and tests cross the same seam.
  If you must test *past* the interface, the module is the wrong shape.
- **One adapter means a hypothetical seam; two means a real one.** Don't
  introduce a seam unless something actually varies across it. A single-adapter
  seam is just indirection.
- **Testable by construction:** accept dependencies, don't create them; return
  results, don't produce side effects; keep the surface small. Fewer methods
  and parameters = fewer tests, simpler setup.

**Deepening a cluster** (deciding where the seam goes, ports & adapters by
dependency category, replace-don't-layer testing) is a process, not a standard —
see the `skillgrid:improve-codebase-architecture` skill.

## Structure

- **Feature-first, not layer-first.** Organize by the *thing* (a feature, a
  domain concept), not by technical layer. `features/decisions/`,
  `internal/mnemonic/memory/` — not `utils/`, `models/`, `controllers/` dumping
  grounds.
- **No catch-all packages.** `util` / `common` / `helpers` / `misc` are where
  cohesion goes to die. Name packages for what they *are*, not what they're
  "not".
- **A clear boundary between entry and logic.** The entry point (CLI `main`,
  HTTP handler, page) is a thin seam: parse input, call the logic, format
  output. Real logic never lives in the seam.
- **Size is a split signal, not a hard limit.** A file growing past ~300 lines,
  or a package doing two unrelated things, is a candidate to split. Smaller is
  always safe; the split is cheap, the tangle isn't.
- **One public surface, deliberate.** Export/`public` only what another unit
  needs. The default is private/unexported. A wide API is a contract you must
  maintain forever.

## Naming

- **Names are the primary documentation.** A name that makes the comment
  redundant is good; a comment needed to explain a name is a bad name.
- **Names state intent, not mechanism.** `retryWithBackoff`, not `loop2`.
  Booleans read as predicates: `hasDrift`, `canPublish` — not `drift` / `flag`.
- **Consistent vocabulary.** One concept, one name, across the whole codebase.
  If the glossary (`01-business-terms.md`) says "observation", the code says
  `observation`, not `record` here and `entry` there.
- **No misleading or ambiguous abbreviations.** If the abbreviation requires
  lookup, it isn't short enough.

## Errors and failure

- **Never swallow.** An error that's caught and ignored must be justified in
  writing; the default is to propagate it to a layer that can act.
- **Carry context.** When re-raising, add *where* it failed, not just *what*.
  "open config: <original>" beats "error".
- **Distinguish recoverable from fatal.** A caller should be able to tell,
  without reading docs, whether the failure is retryable or terminal.
- **Don't use errors for control flow.** If a branch is the *expected* path
  (not an error), don't model it as one.

## Concurrency and state

- **No shared mutable state without a rule.** If two things can touch the same
  state, there's one owner and a defined handoff — a lock, a channel, a single
  writer. Reaching for a global is a smell.
- **Cancellation is a parameter, not a global.** Anything that can block
  receives a way to stop (a context, a signal, a promise).
- **Ownership is explicit.** Who creates it, who closes/frees it — and it's the
  same one. No "someone will clean this up".

## Testing

- **Behavior, not implementation.** Tests assert observable input→output and
  side effects, not private method call counts. A refactor that keeps behavior
  green should not break the test.
- **Table-driven where it fits.** Multiple cases of the same shape → one
  table, one loop, one case per row.
- **The test is the spec.** If a behavior can break, a test covers it. "Too
  small to test" is rationalized away by `skillgrid:ponytail`'s "when NOT to be
  lazy" list — tests are on it.
- **Race-clean.** Concurrency bugs are found with a race detector, not by
  re-running until it looks fine.

## Reviewing

A review checks **two axes in parallel and never reranks them across each
other**:

- **Standards** — does the code follow this repo's documented standards? (this
  file + any per-language notes + ADRs.)
- **Spec** — does the code faithfully implement what the originating issue/spec
  asked for?

A change can pass one and fail the other (correct-but-wrong-thing, or
right-thing-but-wrong-idiom). Keep them separate so one doesn't mask the other.

**Smell baseline.** A review always carries a fixed set of code smells
(Fowler, *Refactoring* ch.3) as **labelled heuristics** — "possible Feature
Envy," never a hard violation. Two rules bind it:

- **The repo overrides.** A documented repo standard always wins; where it
  endorses something the baseline would flag, suppress the smell.
- **Always a judgement call.** Skip anything tooling already enforces.

Each smell: *what it is* → *how to fix*:

| Smell | Fix |
|---|---|
| **Mysterious Name** — a name that doesn't reveal what it does or holds | rename it; if no honest name comes, the design's murky |
| **Duplicated Code** — the same logic shape in more than one place | extract the shared shape, call it from both |
| **Feature Envy** — a method reaching into another object's data more than its own | move it onto the data it envies |
| **Data Clumps** — the same few fields/params keep travelling together | bundle them into one type, pass that |
| **Primitive Obsession** — a primitive standing in for a concept that deserves its own type | give the concept its own small type |
| **Repeated Switches** — the same `switch`/`if`-cascade on the same type recurs | replace with polymorphism, or one map both sites share |
| **Shotgun Surgery** — one logical change forces scattered edits across many files | gather what changes together into one module |
| **Divergent Change** — one module edited for several unrelated reasons | split so each changes for one reason |
| **Speculative Generality** — abstraction/params/hooks added for needs the spec doesn't have | delete it; inline back until a real need shows |
| **Message Chains** — long `a.b().c().d()` navigation the caller shouldn't depend on | hide the walk behind one method on the first object |
| **Middle Man** — a unit that mostly just delegates onward | cut it, call the real target direct |
| **Refused Bequest** — an implementer ignoring or overriding most of what it inherits | drop the inheritance, use composition |

The `skillgrid:requesting-code-review` and `skillgrid:parallel-code-review`
skills apply this two-axis contract and smell baseline.

## What NOT to do

- No god objects / god packages — one concern each.
- No global mutable state; no package-level collections callers mutate.
- No dead code — delete, don't comment out. No `TODO` without a task ID.
- No premature abstraction. No abstraction that only has one use case.
- No hidden dependencies (a global, an env var read mid-function, a date
  "today" baked in) — make them parameters.

## Checklist

- [ ] Each module has one nameable responsibility; size is a split candidate if past ~300 lines
- [ ] Dependency direction is leaf→root; no circular imports
- [ ] Entry point is a thin seam; logic lives in the feature/domain layer
- [ ] No `util`/`common`/`misc`; no public API wider than needed
- [ ] Names state intent and match the glossary vocabulary
- [ ] No swallowed errors; failures carry context and distinguish retryable vs fatal
- [ ] No shared mutable state without a single owner; cancellation is a parameter
- [ ] New behavior has a behavior-level test; no dead code, no `TODO` without a task ID
- [ ] New modules are deep, not shallow: small interface, behavior hidden, testable through the interface
- [ ] Any new seam has ≥ 2 adapters (a real variation), not a hypothetical one
- [ ] Review reports Standards and Spec as separate axes (never cross-reranked); smell findings are labelled heuristics
