# Patterns for effective skill content

Reusable techniques for structuring skill bodies. Use the ones that fit the task; not every pattern applies.

## Gotchas section (highest value)

A list of concrete corrections to mistakes the agent will make without being told. Not general advice ("handle errors appropriately") — specific:

```markdown
## Gotchas

- The `users` table uses soft deletes. Queries must include
  `WHERE deleted_at IS NULL` or results will include deactivated accounts.
- The user ID is `user_id` in the DB, `uid` in the auth service,
  `accountId` in the billing API. All three are the same value.
- `/health` returns 200 even if the DB is down. Use `/ready` for full health.
```

Keep in SKILL.md, not a reference file — the agent may not recognize the trigger to load a separate file. Add any correction you made during testing here.

## Output format templates

When output format matters, provide the template — agents pattern-match well against concrete structure. Inline short ones; long or conditional ones go in `assets/`.

```markdown
## Report structure

Use this template, adapting as needed:

# [Analysis Title]

## Executive summary
[One-paragraph overview]

## Key findings
- Finding with supporting data

## Recommendations
1. Specific actionable recommendation
```

## Checklists for multi-step workflows

Prevents skipped steps, especially with dependencies or validation gates:

```markdown
## Form processing workflow

Progress:
- [ ] Step 1: Analyze the form (run `scripts/analyze_form.py`)
- [ ] Step 2: Create field mapping (edit `fields.json`)
- [ ] Step 3: Validate mapping (run `scripts/validate_fields.py`)
- [ ] Step 4: Fill the form (run `scripts/fill_form.py`)
- [ ] Step 5: Verify output (run `scripts/verify_output.py`)
```

## Validation loops

Do → validate → fix → repeat until clean:

```markdown
## Editing workflow

1. Make your edits
2. Run validation: `python scripts/validate.py output/`
3. If validation fails: review errors, fix, re-run
4. Only proceed when validation passes
```

A reference doc can be the "validator": check your work against it before finalizing.

## Plan-validate-execute

For batch or destructive work: build a structured plan, validate it against a source of truth, only then execute. The validator's error output is what lets the agent self-correct:

```markdown
1. Extract form fields → `form_fields.json` (field names, types, required flags)
2. Create `field_values.json` mapping each field to its intended value
3. Validate: `scripts/validate_fields.py form_fields.json field_values.json`
   "Field 'signature_date' not found — available: customer_name, order_total, ..."
4. Revise and re-validate as needed
5. Fill: `scripts/fill_form.py input.pdf field_values.json output.pdf`
```

## Bundling scripts

If the agent independently reinvents the same logic across runs (chart building, format parsing, output validation), write a tested script once in `scripts/` and reference it. Self-contained or clearly documented dependencies.

## Calibrating control per section

- **Freedom** (explain why, not exact steps) when multiple approaches are valid:

  ```markdown
  ## Code review process
  1. Check all database queries for SQL injection (use parameterized queries)
  2. Verify authentication checks on every endpoint
  3. Look for race conditions in concurrent code paths
  ```

- **Prescriptive** (exact command, "do not modify") when operations are fragile or a specific sequence must hold:

  ```markdown
  ## Database migration
  Run exactly this sequence:
  python scripts/migrate.py --verify --backup
  Do not modify the command or add additional flags.
  ```

Most skills have a mix — calibrate each section independently.
