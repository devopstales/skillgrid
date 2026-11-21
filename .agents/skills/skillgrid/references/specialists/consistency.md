# Consistency specialist

Keep one set of terms, invariants and contracts across the whole change, and catch the drift before it becomes three dialects of the same idea.

When to load: Phase 2 (start the ledger), every wave in Phase 3, Phase 4. Role: reviewer.

## The ledger

One file in the change folder, `ledger.md`, holding the things that must stay the same:

```text
term          | meaning                          | where used
invariant     | the testable claim               | test that proves it
interface     | name, inputs, outputs, errors    | modules that implement it
error code    | when it is raised                | caller's expected action
config key    | precedence, default, validation  | where read
```

Every work unit that introduces or touches one of these updates the ledger in the same unit. A ledger updated later is a ledger that lies.

## The per-wave check

After each wave, run the check and record the result:

1. **Terms** — every glossary term used in the changed code means what the glossary says. A synonym is a defect.
2. **Invariants** — each invariant has a test, and the test ran in this wave. An invariant with no test is a wish.
3. **Interfaces** — the implemented signature matches the contract. A caller relying on undocumented behavior is a drift finding.
4. **Errors** — no module invents its own error taxonomy. Every error has a code and an action.
5. **Config** — every key has a precedence, a default and a validation. An undocumented key is a trap.
6. **Docs** — the README and help text describe what the code now does, not what it did in Phase 2.

## Output

The check writes a short table in `report.md`:

```text
wave | terms | invariants | interfaces | errors | config | docs
2    | ok    | 2 untested | ok         | ok     | ok     | stale README
```

Every non-ok cell becomes a ticket for the fix editor, fixed before the next wave. A wave that ends with a non-ok cell is not finished.

## Rules

- The ledger is written before the code that depends on it, not after.
- A drift finding quotes both sides: the ledger line and the code line.
- Do not "fix" the ledger to match the code unless the programmer agrees the code is right. The ledger is the contract.
- Consistency is not uniformity. Two different things may look alike; check the meaning, not the shape.
