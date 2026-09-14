# Codebase Design (deep-module vocabulary)

Design **deep modules**: a lot of behaviour behind a small interface, placed at
a clean seam, testable through that interface. Use this language and these
principles wherever code is being designed or restructured in a blueprint. The
aim is leverage for callers, locality for maintainers, and testability for
everyone.

The terms below are also seeded into the project glossary at
`.skillgrid/glossary/technical.md` (see `skillgrid:architectural-decision-records`),
so blueprints, briefs, and reviews all name things the same way. Use them
exactly: don't substitute "component," "service," "API," or "boundary."
Consistent language is the whole point.

## Glossary

- **Module**: anything with an interface and an implementation. Deliberately
  scale-agnostic: a function, class, package, or tier-spanning slice.
  _Avoid_: unit, component, service.
- **Interface**: everything a caller must know to use the module correctly: the
  type signature, but also invariants, ordering constraints, error modes,
  required configuration, and performance characteristics. _Avoid_: API,
  signature (too narrow, they refer only to the type-level surface).
- **Implementation**: what's inside a module, its body of code. Distinct from
  **Adapter**: a thing can be a small adapter with a large implementation
  (a Postgres repo) or a large adapter with a small implementation (an
  in-memory fake). Reach for "adapter" when the seam is the topic;
  "implementation" otherwise.
- **Depth**: leverage at the interface. The amount of behaviour a caller (or
  test) can exercise per unit of interface they have to learn. A module is
  **deep** when a large amount of behaviour sits behind a small interface,
  **shallow** when the interface is nearly as complex as the implementation.
- **Seam** (Michael Feathers): a place where you can alter behaviour without
  editing in that place; the *location* at which a module's interface lives.
  Where to put the seam is its own design decision, distinct from what goes
  behind it. _Avoid_: boundary (overloaded with DDD's bounded context).
- **Adapter**: a concrete thing that satisfies an interface at a seam.
  Describes *role* (what slot it fills), not substance (what's inside).
- **Leverage**: what callers get from depth. More capability per unit of
  interface they learn. One implementation pays back across N call sites and M
  tests.
- **Locality**: what maintainers get from depth. Change, bugs, knowledge, and
  verification concentrate in one place rather than spreading across callers.
  Fix once, fixed everywhere.

## Deep vs shallow

**Deep module** = small interface + lots of implementation. **Shallow module**
= large interface + little implementation (avoid). When designing an
interface, ask:

- Can I reduce the number of methods?
- Can I simplify the parameters?
- Can I hide more complexity inside?

## The gates (falsifiable invariants)

These are the two checks that turn the vocabulary into a gate an agent answers
yes/no at blueprint or review time. They are the same mechanism as a
`#### Gates` `CHECK`/`EXPECT` oracle: each produces a verdict, not a vibe.

- **The deletion test.** Imagine deleting the module. If complexity vanishes,
  it was a pass-through. If complexity reappears across N callers, it was
  earning its keep. A blueprint task that creates a module MUST name, in its
  `Interfaces` field, what complexity reappears if the module is removed.
- **The one-adapter rule.** One adapter means a hypothetical seam; two adapters
  mean a real one. Don't introduce a seam (port) unless at least two adapters
  are justified (typically production + test). A single-adapter seam is just
  indirection. This is the design-side twin of ponytail's "no interface with
  one implementation" — both say *the same thing from opposite sides*.

## Principles

- **Depth is a property of the interface, not the implementation.** A deep
  module can be internally composed of small, mockable, swappable parts; they
  just aren't part of the interface. A module can have **internal seams**
  (private to its implementation, used by its own tests) as well as the
  **external seam** at its interface.
- **The interface is the test surface.** Callers and tests cross the same
  seam. If you want to test *past* the interface, the module is probably the
  wrong shape.

## Designing for testability

Good interfaces make testing natural:

1. **Accept dependencies, don't create them.**

   ```typescript
   // Testable
   function processOrder(order, paymentGateway) {}

   // Hard to test
   function processOrder(order) {
     const gateway = new StripeGateway();
   }
   ```

2. **Return results, don't produce side effects.**

   ```typescript
   // Testable
   function calculateDiscount(cart): Discount {}

   // Hard to test
   function applyDiscount(cart): void {
     cart.total -= discount;
   }
   ```

3. **Small surface area.** Fewer methods = fewer tests needed. Fewer params =
   simpler test setup.

## Deepening a cluster (dependency categories)

When assessing a candidate for deepening, classify its dependencies. The
category determines how the deepened module is tested across its seam.

- **1. In-process** — pure computation, in-memory state, no I/O. Always
  deepenable: merge the modules and test through the new interface directly.
  No adapter needed.
- **2. Local-substitutable** — dependencies that have local test stand-ins
  (PGLite for Postgres, in-memory filesystem). Deepenable if the stand-in
  exists; test with the stand-in in the suite. The seam is internal; no port at
  the module's external interface.
- **3. Remote but owned (Ports & Adapters)** — your own services across a
  network boundary. Define a **port** (interface) at the seam; the deep module
  owns the logic, the transport is injected as an **adapter**. Tests use an
  in-memory adapter; production uses HTTP/gRPC/queue.
- **4. True external (Mock)** — third-party services you don't control
  (Stripe, Twilio). The deepened module takes the external dependency as an
  injected port; tests provide a mock adapter.

**Seam discipline.** Internal seams stay private to the implementation; don't
expose them through the interface just because tests use them. **Replace,
don't layer:** once tests at the deepened module's interface exist, the old
unit tests on the shallow modules are waste; delete them.

## Design It Twice

When a blueprint task introduces a non-trivial module and the interface is
genuinely uncertain, don't guess once. This is `skillgrid:sketch` with a
technical brief instead of a UI:

1. **Frame the problem space** for the user: the constraints any new interface
   must satisfy, the dependencies and their category (above), a rough code
   sketch to ground the constraints (not a proposal). Show it, then proceed.
2. **Spawn 3+ sub-agents in parallel.** Each produces a **radically different**
   interface for the module, under a different constraint: minimize the
   interface (1-3 entry points); maximize flexibility; optimize the most common
   caller; design around ports & adapters (if a cross-seam dependency exists).
   Each outputs: the interface (types, methods, params, invariants, ordering,
   error modes), a usage example, what the implementation hides behind the
   seam, the dependency strategy + adapters, and the trade-offs (where leverage
   is high, where it's thin).
3. **Compare and recommend.** Contrast on **depth** (leverage at the
   interface), **locality** (where change concentrates), and **seam
   placement**. Give your own opinionated pick (or a hybrid) — the user wants a
   strong read, not a menu.

## Rejected framings

- **Depth as a ratio of implementation-lines to interface-lines** (Ousterhout):
  rewards padding the implementation. We use depth-as-leverage instead.
- **"Interface" as the TypeScript `interface` keyword or a class's public
  methods:** too narrow; interface here includes every fact a caller must know.
- **"Boundary":** overloaded with DDD's bounded context. Say **seam** or
  **interface**.
