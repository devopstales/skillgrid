# Envelope Contract (typed handoffs)

The contract for every machine-shaped handoff between pipeline steps — the
envelope. Cite this; do not restate the rules. The schemas live in
`_shared/schemas/pipeline/` (one per envelope family, versioned); the
validator is `_shared/scripts/validate-envelope.mjs`.

## What an envelope is

The **machine shape of a verdict or a pause.** A gate verdict (QA, ship),
a user-gate screen, and the phase status are each emitted as one JSON
envelope. The human-readable screen or one-line verdict is a *rendering* of
the envelope, never a second source of truth. An agent that reports a
verdict the envelope does not carry has self-approved.

Families (v1):

| Family | `schema` const | Emitted by |
|---|---|---|
| gate result | `skillgrid.gate-result/v1` | `test/scripts/verdict-render.mjs` (gate `qa-verdict`), `ship/scripts/ship-gates.mjs` (gate `ship`) |
| gate consent | `skillgrid.gate-consent/v1` | `gate-consent.mjs`, rendered at every named user gate |
| phase status | `skillgrid.phase-status/v1` | the phase authority (a later change — shape frozen now) |

## The rules

1. **Closed shape.** `schema`/`contract` are consts; `additionalProperties:
   false`. **Readers ignore unknown fields**, so a v1 reader parses a v2
   envelope (forward compatibility); writers must conform — an extra field
   on output is a schema violation, caught by the validator.
2. **Frozen subject binding.** A gate envelope carries `subject` — `sha256`
   digests of the exact artifacts it validated. A consumer that re-checks
   against different bytes is `scope-changed`, not a re-verification.
   (gentle-ai: every review is bound to its `subject_hash` the same way.)
3. **In-band continuation.** Every denial (`allowed: false`; phase status
   `kind: "stop"`) carries a `next.step` that is **runnable**: an exact
   command or an exact file edit. A template with `<placeholder>` segments
   is not runnable — the step names the concrete file, area, or command.
   The validator enforces this; it is the check that turns "did the
   workflow tell me what to do" into a number.
4. **One fixture per schema.** `_shared/schemas/pipeline/fixtures/` holds
   at least one valid envelope per family (and per notable state); emitter
   tests validate their real output against the schema with
   `validate-envelope.mjs`, not against a copy of the expected JSON.
5. **Versioning.** A breaking change bumps the family version (`/v2`) and
   adds the new schema; the old one is frozen, not edited in place.
   Non-breaking additions (new optional fields, new enum values a reader
   must tolerate) may edit the current version in place, with a fixture.

## The in-band rule, mechanically

`validate-envelope.mjs` flags a continuation step that:

- is empty, or
- contains a `<placeholder>` segment (`/<[A-Za-z_][A-Za-z0-9_]*>/`), or
- names neither a command (`node|git|wt|bun|npm|npx|uv|uvx|gh|glab` plus an
  argument) nor a file with an extension (optionally `:line`).

Applies to: `gate-result` denials (`next.step`), `gate-consent` choice
`next_step`s and `off_path.step`, `phase-status` stops
(`continuation.step`).

## Validator

```bash
node .agents/skills/_shared/scripts/validate-envelope.mjs \
  .agents/skills/_shared/schemas/pipeline/<family>.schema.json \
  <envelope.json | ->
# exit 0 valid · 1 invalid (findings JSON on stdout) · 2 usage
```

Supported keywords are the subset the pipeline schemas use (see the script
header). A schema that needs a keyword the validator lacks is a change to
the validator plus its test, not to the schema alone.

## Why

Prose handoffs drift: "the verdict is PASS" re-derived run to run is not the
same guarantee as a schema-checked JSON object whose continuation the
validator has proven runnable. The envelope is the cheapest deterministic
boundary the pipeline has — a 200-line validator costs zero tokens at load,
and the denial it rejects is the one that would have sent the agent (or the
user) looking for the next step.
