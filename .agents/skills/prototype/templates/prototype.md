> Prototype directory: `.skillgrid/prototypes/NNN-name/` — permanently retained (not under the
> spec zone). The `topic:` field links this probe back to the spec that motivated it.

---
prototype: NNN
name: descriptive-name
type: standard            # standard | comparison
topic: YYYY-MM-DD-<topic>
verdict: PENDING          # VALIDATED | INVALIDATED | PARTIAL
tags: [tag1, tag2]
---

# Prototype NNN: Descriptive Name

## What This Validates

> Given [precondition], when [action], then [expected outcome].

[2-3 sentences: what this prototype is, why it matters, the key risk or unknown.]

## Research

[Docs checked, approach comparison table, chosen approach, gotchas. Omit this section
if the prototype is pure logic with no external dependencies.]

| Approach | Tool/Library | Pros | Cons | Status |
|----------|-------------|------|------|--------|
| ... | ... | ... | ... | chosen |

**Chosen approach:** [which one and why]

## How to Run

```bash
[exact command(s)]
```

## What to Expect

[Concrete observable outcomes — what the user should see/hear/feel when the prototype
works, and what would signal it does not.]

## Liftable Module

[Omit if the prototype has no pure module to lift. If present: the file path, what it does,
its input/output signature, and any dependencies. This is the unit the real build
copies and wires I/O around — everything else in this prototype is throwaway.]

## Observability

[Omit if no forensic log layer. If present: what the event log captures, how to export
it, and what the summary reports.]

## Investigation Trail

[Updated as the prototype progresses. Document each iteration: what you tried, what it
revealed, what you tried next. This is the section that makes the prototype worth keeping.]

- **Attempt 1:** [what was tried] → [what it revealed]
- **Attempt 2:** [follow-up / edge case] → [what it revealed]
- [continue for each iteration]

## Results

**Verdict:** [VALIDATED ✓ / INVALIDATED ✗ / PARTIAL ⚠]

**Evidence:** [the specific output, log line, measured number, or screenshot that
proves the verdict. A verdict without evidence is PARTIAL at best.]

**Surprises:** [what did not match the expectation]

**Constraints (if PARTIAL):** [the boundary of what works — input size, concurrency,
workarounds required]

## Signal for the Build

[What to use, what to avoid, what to watch out for. This feeds the topic's
`findings.md` → the blueprint.]
