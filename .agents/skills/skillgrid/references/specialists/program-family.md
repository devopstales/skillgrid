# Program family specialist

For work that is more than one program: a platform, a suite, a CLI plus a service plus a library, or a set of programs that must share a contract.

When to load: only when the intake declares a program family. Role: architect.

## The family model

A family is several programs that share one of: a data format, an API contract, a deployment, a vocabulary, or a release train. Name which one, because it decides what must be centralized and what must not.

| Shared thing | Centralize | Keep local |
|---|---|---|
| Data format | schema, versioning, migration tool | serialization details |
| API contract | the contract file and its tests | transport choices |
| Deployment | the manifest and the health contract | scaling per service |
| Vocabulary | the glossary | internal names |
| Release train | the version and the changelog index | each program's notes |

## Rules

- Each program has its own `SKILL.md`-level purpose, its own Definition of Done, and its own score. A family does not average its members.
- The shared contract has one owner and one test suite. Two copies of a schema are two schemas.
- The tracer bullet runs through the seam between two programs, not inside one. That seam is the risk.
- The panel is per program, plus one **integrator** user who runs the family end to end.
- The release is per program unless the intake declared a release train. A train needs a rollback per member.
- A family change that touches only one member is a single-program change; do not inflate it.

## When not to use

A program with two binaries built from one repo is not automatically a family. Use this specialist only when a contract is shared across a boundary the programmer cannot change unilaterally.
