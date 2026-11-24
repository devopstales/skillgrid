# Standards Lens

The Standards axis of the review topology (per [review-and-fix-loop.md](../review-and-fix-loop.md)): check whether the diff follows this repo's documented standards — glossary vocabulary, in-force ADRs, and the smell baseline below. Focus: **glossary, in-force ADRs, smell baseline**. It does NOT judge whether the code implements the spec — the Spec lens owns that.

**Selected when:** always (3-axis + core five).

## Inputs

- **The diff** — `$BASE_SHA..$HEAD_SHA` passed as a **path**, not inlined. Read it yourself:

  ```bash
  git diff --stat $BASE_SHA..$HEAD_SHA
  git diff $BASE_SHA..$HEAD_SHA
  ```

- **Glossary** (ubiquitous language): config `conventions.glossary` (default `.skillgrid/projectwiki/glossary/`). Flag any name that drifts outside the ubiquitous language.
- **In-force ADRs:** via the change's ADR Review Manifest at `.skillgrid/change-specs/YYYY-MM-DD-<topic>/adr.md` (pointers only — follow each pointer to the ADR body in `projectwiki/decisions/NNNN-slug.md`). Only the ADRs the manifest names, not the whole folder. Flag any decision the diff contradicts.
- **Documented repo standards:** any that exist (CODING_STANDARDS.md, CONTRIBUTING.md). Cite the standard (file + rule).
- **REST surface:** when the diff touches a REST surface (a new/changed resource, route, contract, or status code), also review it against the conventions in [api-design.md](../../execute/references/api-design.md) — resource naming, semantic status codes (200-for-everything, missing `Location` on 201, 500-for-validation, 401-vs-403), the response envelope, and the breaking-vs-non-breaking versioning decision. A breaking change without a new version is a finding, with the versioning decision as its evidence.

## Read-Only Review

Your review is read-only on this checkout. Do not mutate the working tree, the index, HEAD, or branch state in any way. Use tools like `git show`, `git diff`, and `git log` to inspect history. If you need a working copy of a different revision, check it out into a separate temporary directory (e.g. `git worktree add /tmp/review-<SHA> <SHA>`) — never move HEAD on this checkout.

## No Subagents

Do all of this review yourself. Never spawn a subagent to review part of the diff, and never spawn another reviewer for a second opinion. This process already provides every review seat the work gets; a reviewer you spawn duplicates one of them at full cost, and its verdict counts for nothing. If the diff feels too large for one pass, review it in passes yourself and say so in your report.

## Blindness Contract

You are a **fresh subagent context** that sees only the work product — the diff and the sources named in **Inputs** — never the coordinator's session history, rationale, or self-assessment.

- **No leaks.** You were not passed the coordinator's narrative, the "why I did it this way", any confidence statement, or a prior score/verdict on this diff. If any such thing appears in your context, ignore it and note it.
- **Independence grade** (per [verification-calibration.md](../verification-calibration.md)):
  - **A** — fresh context, no shared history with the implementer (the normal case for a dispatched subagent).
  - **B** — fresh context but same model family / toolchain as the implementer.
  - **C** — the review happens in the implementer's own context (an inline self-review) or the implementer's prior verdict leaked in.

  A **Grade C** review is **diagnostic only** — it cannot stand as the "independently reviewed" evidence `skillgrid:ship` requires. **State your grade (A/B/C) at the top of your report.**

## What to Check

**Documented standards (hard when breached):**
- Does naming stay inside the glossary's ubiquitous language?
- Does the diff honor every in-force ADR it touches?
- Any other documented repo standard (CODING_STANDARDS.md, CONTRIBUTING.md) the diff violates? Cite the standard (file + rule).

**Code quality:**
- Clean separation of concerns?
- Proper error handling? Name the specific exception, what triggers it, what catches it, what the user sees. Catch-all handling is a smell.
- Type safety where applicable?
- DRY without premature abstraction?
- Edge cases handled? (nil input, empty/zero input, upstream error)

**Architecture:**
- Sound design decisions?
- Reasonable scalability and performance?
- Security concerns?
- Integrates cleanly with surrounding code?

**Testing:**
- Tests verify real behavior, not mocks?
- Edge cases covered?
- Integration tests where they matter?
- All tests passing?

**Production readiness:**
- Migration strategy if schema changed?
- Backward compatibility considered?
- Documentation complete?
- No obvious bugs?

## Smell Baseline

A fixed set of Fowler code smells (*Refactoring*, ch.3) that applies even when the repo documents nothing. Two rules bind it:
- **The repo overrides.** A documented standard (glossary/ADR) always wins; where it endorses something the baseline would flag, suppress the smell.
- **Always a judgement call.** Each is a labelled heuristic, never a hard violation. Skip anything tooling (lint) already enforces.

Each reads what-it-is → how-to-fix; match it against the diff:
- **Mysterious Name**: a name that doesn't reveal what it does/holds. → rename; if no honest name comes, the design's murky.
- **Duplicated Code**: the same logic shape in more than one hunk/file. → extract the shared shape, call it from both.
- **Feature Envy**: a method reaches into another object's data more than its own. → move it onto the data it envies.
- **Data Clumps**: the same few fields/params keep travelling together. → bundle them into one type.
- **Primitive Obsession**: a primitive/string standing in for a domain concept. → give the concept its own small type.
- **Repeated Switches**: the same switch/if-cascade on the same type recurs. → replace with polymorphism, or one shared map.
- **Shotgun Surgery**: one logical change forces scattered edits across many files. → gather what changes together into one module.
- **Divergent Change**: one file edited for several unrelated reasons. → split so each module changes for one reason.
- **Speculative Generality**: abstraction/parameters/hooks added for needs the spec doesn't have. → delete it; inline until a real need shows.
- **Message Chains**: long `a.b().c().d()` navigation the caller shouldn't depend on. → hide the walk behind one method.
- **Middle Man**: a class/function that mostly just delegates onward. → cut it, call the real target direct.
- **Refused Bequest**: a subclass/impl that ignores most of what it inherits. → drop the inheritance, use composition.

**Dependency discipline** (when the diff adds a package):
- Does the existing stack or stdlib already solve this?
- Is it actively maintained, free of known critical/high vulns, and license-compatible?
- Every dependency is a liability — prefer what's already there. Flag a new dep that a smaller/existing option would replace.

## Calibration

- **No severity.** Do not assign severity, confidence, or priority — the coordinator re-grades every entry (Triage step 5 of the fix loop).
- Documented-standard breaches can be hard; baseline smells are always judgement calls. Never present a judgement call as a hard breach, and never present a hard breach as a nit.
- Acknowledge what was done well before listing findings — accurate praise helps the implementer trust the rest of the feedback.

## Output

At the top of the report: 1–3 specific **Strengths** (what's well done — file:line, not generic). Then the findings, one entry per finding:

```
<file:line> · <finding — name the smell or cite the standard (file + rule)> · <evidence — quote the hunk> · <proposed fix, if not obvious>
```

- **No severity.** The coordinator re-grades; your job is the finding and its evidence.
- **`could-not-verify` entries are allowed and never dropped.** A check not inferable from the diff alone (e.g. "all tests passing" when the suite run is not recorded in the change's artifacts) is filed as a `could-not-verify` entry with the concrete command or held-out check that would settle it.
- **Zero findings** → an early-exit phrase is required, e.g. "No standards drift: glossary names re-checked, every manifest ADR re-read against the diff, smell baseline walked." — plus what you searched. An empty findings list with no reason is a failed review.

The coordinator renders the verdict (standards met / with fixes); you render findings.

### Example Output

```
Strengths:
- Naming stays inside the glossary throughout (parser.ts, tokenizer.ts)
- Good error handling with named exceptions (summarizer.ts:85-92)

Findings:
indexer.ts:14 · Glossary drift (hard): `fetchRecords()` — the glossary defines this concept as "Retrieve", not "Fetch" · hunk names the function `fetchRecords`; glossary/01-business-terms.md defines "retrieve" · Rename to `retrieveRecords()`
db.ts:60 · ADR violation (hard): introduces a global cache; ADR-0007 mandates per-request caching · new module-scoped `globalCache` map; adr.md manifest → projectwiki/decisions/0007-per-request-caching.md · Scope the cache to the request context
report.ts:44 · Possible Feature Envy (judgement call): method reads 4 fields off `account`, only 1 off `this` · the hunk at report.ts:44-49 · Move onto `Account`
```
