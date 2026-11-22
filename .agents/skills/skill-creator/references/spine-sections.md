# Spine Sections (fixed section order is load-bearing)

An operational skill — one that walks a process, enforces gates, or renders a
verdict — carries four standard sections in a fixed position, so the agent
finds the same contract at the same place in every sibling skill:

| Section | Position | Content |
|---|---|---|
| **Input Contract** | after the overview, before Decision Gates | the exact inputs the skill expects: file paths, config keys, user parameters. One line per input: name, source, type, required/optional. Skip when the input is already a named section (e.g. a `## Config` table). |
| **Decision Gates** | after the overview / gates section, before the step walk | a `| Condition | Action |` table — one row per branching condition; every row sourced from an existing rule. Routing, not legislation. |
| **Output Contract** | after the process steps, before the tail (verification / references) | the exact final output shape: named fields, file paths, verdict values. |
| **Anti-Patterns** | tail, replacing 2-column "Common Rationalizations" | `| Anti-pattern | Symptom | Fix |` — the symptom column is greppable, names the observable failure, not the excuse. |

Rules:

- **Input Contract pins the left side of the skill** — the Output Contract
  pins the right; together they make the skill's behavior reproducible. An
  input that changes the output and is not in the Input Contract (and not
  scripted or explicitly user-provided) is a reproducibility gap. Skip the
  section when the input is already a named section (e.g. a `## Config`
  table) — cite the existing one, don't duplicate.
- **Decision Gates is a table, not prose** — a "when do I do what" paragraph
  re-derives on every read; the table format is what makes it a lookup.
- **Zero new policy.** Each row cites a rule already in the skill (a gate, a
  hard rule, a step). A row that would introduce unstated behavior is a
  draft rule — write it in the prose first, then tabulate.
- **Output Contract is the deliverable, not the checklist** — the
  Verification checklist proves the work happened; the Output Contract names
  what the user receives. Output already a named section (e.g. a "Return
  Envelope") → skip the section and cite the existing one.
- **Anti-Patterns replaces Common Rationalizations, not joins it** — convert
  2-column rows to 3: excuse → anti-pattern, observable failure → symptom,
  reality → fix. Red Flags that duplicate table rows get trimmed; flags that
  name a *moment to stop* (not a pattern) stay as a list.
- **Judgment-only and reference skills may skip all four** — skipping is
  deliberate, not accidental.
