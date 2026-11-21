# Technical glossary

Architecture, platform, protocol, and design-pattern terms for this project.
One row per term. Definitions stay tight (one or two sentences) and define what
the term IS, not what it does.

| Term | Definition | Use When | Avoid |
| --- | --- | --- | --- |
| Module | Anything with an interface and an implementation; scale-agnostic. | When discussing a unit of code and its boundary. | Unit, component, service |
| Interface | Everything a caller must know to use the module: the signature plus invariants, ordering, error modes, config, performance. | When specifying what a module presents to callers and tests. | API, signature (too narrow) |
| Seam | A place where behaviour can be altered without editing that place; where a module's interface lives. | When deciding where a boundary/interface goes. | Boundary (overloaded with DDD's bounded context) |
| Adapter | A concrete thing that satisfies an interface at a seam; describes role, not substance. | When the seam is the topic. | Wrapper, decorator |
| Depth | Leverage at the interface — behaviour a caller can exercise per unit of interface they must learn. | When judging whether a module's interface earns its keep. | Layering, complexity |
| Leverage | What callers get from depth: more capability per unit of interface learned; one implementation pays back across N call sites and M tests. | When weighing a module's value to its callers. | Benefit, payoff |
| Locality | What maintainers get from depth: change, bugs, knowledge, and verification concentrate in one place. Fix once, fixed everywhere. | When weighing a module's value to its maintainers. | Cohesion (vague) |
| Deletion test | If the module is deleted and complexity vanishes it was a pass-through; if complexity reappears across callers it was earning its keep. | When judging whether a module is worth keeping or is indirection. | — |

## Rules

- **Be opinionated.** When multiple words exist for the same concept, pick the best one and list the others under `Avoid`.
- **Keep definitions tight.** One or two sentences max.
- **Prefer existing project terms** (README, architecture docs, code names) over new jargon.
- **Only include terms specific to this project's context.** General programming concepts that aren't used with a project-specific meaning don't belong here.
- **The glossary is a glossary and nothing else.** No implementation details, no specs, no scratch.
