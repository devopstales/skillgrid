# AI tells in code

The shapes machine-written code tends to take, and why a reviewer should look twice. None of these is proof of bad work; each is a place where the work may be hollow.

When to load: Phase 4 and 5. Role: reviewer.

## Structural tells

- **Abstraction before need.** A factory, a strategy, a base class and an interface for one implementation. Ask what second implementation exists. If none, cut it.
- **The generic layer.** `utils`, `helpers`, `common`, `manager`, `service` holding unrelated functions. A module named for its location, not its job.
- **Uniform files.** Every module the same length and shape, as if generated from a template. Real programs are lumpy.
- **Commentary that restates the code.** `// increment the counter` above `counter += 1`. It signals the writer had nothing to say.
- **Defensive noise.** A null check on a value the type system already excludes, a try/catch that logs and continues, a validation repeated at every layer.
- **Placeholder confidence.** A function that returns a plausible value with no path to the real one, marked TODO in a comment rather than in the tracker.

## Naming tells

- Names that describe the pattern instead of the domain: `dataProcessor`, `handleItem`, `processData`.
- `v2`, `new`, `enhanced`, `improved` in a name, with the old one still present.
- A synonym for a glossary term. Drift, not variety.
- Acronyms the glossary does not define.

## Test tells

- Tests that assert a mock was called, never that the behavior happened.
- One giant test that walks the happy path and calls it coverage.
- Tests named after the function, not the behavior.
- A test that passes when the implementation is deleted.
- Snapshot tests nobody reads.

## Documentation tells

- A README that describes the architecture before it says how to run the program.
- A changelog entry that lists files instead of user-visible changes.
- An error message written for the developer of the code, not the person who hit it.
- A comment explaining a workaround with no link to the reason.

## How to use a tell

1. Quote the passage.
2. Name the consequence: what will cost the next maintainer time.
3. Ask what the writer was avoiding: a decision, a test, a failure path.
4. Propose the smallest move that removes the tell, from `references/specialists/code-craft.md` § Revision moves.

A tell is a question, not a verdict. The reviewer who writes "this is AI slop" has found nothing; the reviewer who writes "this factory has one implementation and no test, so it hides the real seam at `sync.go:88`" has found the defect.
