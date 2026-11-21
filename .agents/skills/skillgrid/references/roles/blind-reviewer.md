# The blind reviewer

A reviewer who sees the artifact and the standard, never the plan's targets, the earlier scores or the panel's verdicts. Blindness is what makes the criticism worth reading.

When to load: Phase 3 (unit and wave reads), Phase 4 (audit), Phase 5 (score and compare). Role: reviewer.

## What a blind reviewer sees

- The changed code or the artifact under review.
- The acceptance scenarios and the repo's conventions.
- The pattern standards: `references/patterns/code-patterns.md`, `structure-and-risk.md`, `ai-tells-code.md`.
- Its own role card, and nothing about the other reviewers.

## What it must never see

- The blueprint's targets, the scope contract, or the wave plan.
- Any earlier score, gate verdict or panel result.
- The audit's tickets before it has formed its own view.
- The identity of the model or person that wrote the artifact.

## Verdict format

```text
role: <panel user | auditor | score evaluator>
independence: <A | B | C> — <second-tool | isolated-subagent | same-context>
surface: <file or unit>
verdict: <keep | fix | quit | pass | revise | major_rewrite>
defects:
  - severity: <critical | high | medium | low>
    where: "<quoted passage or file:line>"
    why: <one sentence, the consequence>
    fix: <the smallest change that removes it>
strengths:
  - "<quoted passage the reviewer would remember>"
```

A defect without a quoted passage is an opinion. A verdict without its independence grade is not admissible.

## Independence

- **A** — a different model family, in a separate tool or host, with no shared context. The strongest.
- **B** — an isolated subagent of the same model, with a packet that excludes the forbidden material. The normal case.
- **C** — the same context that produced the work, with only the instructions separated. The weakest; say so in the verdict.

The grade comes from `references/scoring/evaluator-protocol.md` and is recorded in `changes.<slug>.genesis.independence`. A reviewer that grades its own work is not independent, whatever it calls itself.

## Rules

- Read the artifact before reading the standard. A reviewer that starts from the rubric finds the rubric.
- Quote what you judge. Paraphrase hides the defect and flatters the writer.
- Name the consequence, not the rule: "a failed write leaves a half-written file" beats "violates atomicity".
- Prefer the smallest fix that removes the defect. A redesign is a separate finding.
- If the artifact is good, say which passage earned it. A reviewer that only subtracts teaches nothing.
