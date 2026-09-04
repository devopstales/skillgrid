---
name: codebase-design
description: Shared vocabulary for designing deep modules. Use when designing or improving a module's interface, finding deepening opportunities, deciding where a seam goes, making code more testable, or when another skill needs the deep-module vocabulary.
license: MIT
metadata:
  author: skillgrid
  version: "1.0"
  source: derived from Matt Pocock's codebase-design skill, language-agnostic port
---

# Codebase Design

Design **deep modules**: a lot of behaviour behind a small interface, placed at a clean seam, testable through that interface. Use this language and these principles wherever code is being designed or restructured. The aim is leverage for callers, locality for maintainers, and testability for everyone.

## Glossary

Use these terms exactly. Do not substitute "component," "service," "API," or "boundary." Consistent language is the whole point.

- **Module**: anything with an interface and an implementation. Deliberately scale-agnostic — a function, class, package, or tier-spanning slice. _Avoid_: unit, component, service.
- **Interface**: everything a caller must know to use the module correctly — the type signature, but also invariants, ordering constraints, error modes, required configuration, and performance characteristics. _Avoid_: API, signature (too narrow — they refer only to the type-level surface).
- **Implementation**: what's inside a module, its body of code. Distinct from **Adapter**: a thing can be a small adapter with a large implementation (a Postgres repo) or a large adapter with a small implementation (an in-memory fake). Reach for "adapter" when the seam is the topic; "implementation" otherwise.
- **Depth**: leverage at the interface. The amount of behaviour a caller (or test) can exercise per unit of interface they have to learn. A module is **deep** when a large amount of behaviour sits behind a small interface, **shallow** when the interface is nearly as complex as the implementation.
- **Seam** _(Michael Feathers)_: a place where you can alter behaviour without editing in that place; the _location_ at which a module's interface lives. Where to put the seam is its own design decision, distinct from what goes behind it. _Avoid_: boundary (overloaded with DDD's bounded context).
- **Adapter**: a concrete thing that satisfies an interface at a seam. Describes _role_ (what slot it fills), not substance (what is inside).
- **Leverage**: what callers get from depth. More capability per unit of interface they learn. One implementation pays back across N call sites and M tests.
- **Locality**: what maintainers get from depth. Change, bugs, knowledge, and verification concentrate in one place rather than spreading across callers. Fix once, fixed everywhere.

## Deep vs shallow

**Deep module** = small interface + lots of implementation. **Shallow module** = large interface + little implementation (avoid).

When designing an interface, ask:

- Can I reduce the number of methods?
- Can I simplify the parameters?
- Can I hide more complexity inside?

## Principles

- **Depth is a property of the interface, not the implementation.** A deep module can be internally composed of small, mockable, swappable parts; they just aren't part of the interface. A module can have **internal seams** (private to its implementation, used by its own tests) as well as the **external seam** at its interface.
- **The deletion test.** Imagine deleting the module. If complexity vanishes, it was a pass-through. If complexity reappears across N callers, it was earning its keep.
- **The interface is the test surface.** Callers and tests cross the same seam. If you want to test _past_ the interface, the module is probably the wrong shape.
- **One adapter means a hypothetical seam. Two adapters means a real one.** Don't introduce a seam unless something actually varies across it.

## Designing for testability

Good interfaces make testing natural:

1. **Accept dependencies, don't create them.** The module takes its collaborators as inputs rather than constructing them internally. Constructing inside the function makes the collaborator invisible to the test.
2. **Return results, don't produce side effects.** A function that returns a value is trivial to assert on. A function that mutates a passed-in object or a global state requires elaborate setup to observe.
3. **Small surface area.** Fewer methods = fewer tests needed. Fewer params = simpler test setup.

## Relationships

- A **Module** has exactly one **Interface** (the surface it presents to callers and tests).
- **Depth** is a property of a **Module**, measured against its **Interface**.
- A **Seam** is where a **Module**'s **Interface** lives.
- An **Adapter** sits at a **Seam** and satisfies the **Interface**.
- **Depth** produces **Leverage** for callers and **Locality** for maintainers.

## Rejected framings

- **Depth as ratio of implementation-lines to interface-lines** (Ousterhout): rewards padding the implementation. Use depth-as-leverage.
- **"Interface" as a single language keyword or a class's public methods**: too narrow — interface here includes every fact a caller must know.
- **"Boundary"**: overloaded with DDD's bounded context. Say **seam** or **interface**.

## Use in skillgrid

This vocabulary is mandatory in `sdd-propose` when writing the **Architecture Decisions** section of `change.md` and the **Impacted Files Map**. It is also a TDD check during `sdd-apply`: "if you can't test through the interface, the module is probably the wrong shape" (re-shape before adding more tests). See the wiring notes in `.agents/skills/sdd-propose/SKILL.md` and `.agents/skills/sdd-apply/SKILL.md`.
