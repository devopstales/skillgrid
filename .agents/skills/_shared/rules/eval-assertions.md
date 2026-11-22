# Eval Assertions — Deterministic Output Patterns

Complement to the prose `expected_output` in `<skill>/evals/evals.json`.
Where a behavior can be checked by regex on the agent's output, encode it as
a pattern assertion; prose expectations stay for judgment calls.

## Format

Each eval may carry `assertions[]`:

```json
{
  "id": 5,
  "prompt": "...",
  "expected_output": "prose fallback description",
  "assertions": [
    { "type": "required_pattern", "pattern": "...", "message": "why" },
    { "type": "forbidden_pattern", "pattern": "...", "message": "why" }
  ]
}
```

- `required_pattern` — output MUST match; message names the invariant.
- `forbidden_pattern` — output MUST NOT match; use these for "don't do X"
  rules (hallucination guards, scope guards, cost-statement mandates).
  A forbidden assertion is the only way to test *absence of a behavior*.
- Patterns are anchored to the output the eval names (a section, a message,
  the whole reply) — state the scope in `message`.

## When to use

- A skill rule that says "MUST state X" (e.g. a fan-out gate must state the
  worker cap) → `required_pattern` for X.
- A skill rule that says "MUST NOT infer Y from Z" → `forbidden_pattern`
  for Y in the output, paired with a prompt where Y is the tempting answer.
- Keep prose `expected_output` as the human-readable spec; assertions are the
  machine-checkable subset.
