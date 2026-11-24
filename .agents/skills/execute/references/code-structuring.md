# Code Structuring

How execute shapes the code it writes and the code it touches: size a brownfield repo before planning on it, refactor the code it lands in with a scoped health pass (duplication → dead code → complexity), and apply the deep-module vocabulary to the interfaces it creates or reshapes. The three are independent; use the one the task needs, or all three for a brownfield enrollment or a "clean up / improve / refactor" request.

**Announce when running a full pass:** "I'm structuring the code as part of skillgrid:execute."

## When to Use

- **Inspect** — sizing a brownfield repo before planning or slicing on it (line counts calibrate the 400-line budget risk), or when asked "how big is this and what's it made of."
- **Refactor** — a "clean up", "improve", "refactor", or "pay down tech debt in" request on code the change lands in, rather than fixing one specific bug.
- **Design** — when the task creates a new module/interface, or reshapes an existing one (splitting a file, extracting a seam, moving code to a shared module), so the result is a deep module, not a shallow pass-through.

**When NOT to use:** a single-bug fix with no interface change (just fix the bug); duplication-only measurement (`skillgrid:jscpd` / `skillgrid:dry-refactoring` directly); symbol-level orientation / blast radius (the code index, not LOC).

## Part 1 — Inspect (measure before you plan)

Quantitative complement to reading code. Run it **first** when enrolling a brownfield project or before slicing unfamiliar code — the 400-line budget risk and the wave/slicing decisions read differently at 5K vs 500K LOC. Measure first, slice second.

### The command

```bash
uvx pygount --format=summary \
  --folders-to-skip=".git,node_modules,venv,.venv,__pycache__,.cache,dist,build,.next,.tox,.eggs,*.egg-info,vendor,third_party" \
  .
```

**Always pass `--folders-to-skip`.** Without it pygount crawls dependency trees and may take minutes or hang. Tune the skip list to the stack (add `vendor` for Go, `.turbo`/`coverage` for JS/TS, `.mypy_cache` for Python).

Narrow or deepen as needed:

```bash
uvx pygount --suffix=py --format=summary .                    # one language
uvx pygount --folders-to-skip=".git,node_modules,venv" . \
  | sort -t$'\t' -k1 -nr | head -20                       # largest files first
uvx pygount --format=json .                                   # machine-readable
```

### Report (record, don't just chat)

Total LOC, top-3 languages with file counts, code-vs-comment ratio, the 5 largest files, and the tool that produced the numbers. When sizing for slicing, file these under the plan's notes so `tasks.md` calibrates the 400-line budget against measured reality.

**Interpreting:** columns are Language / Files / Code / Comment / %. Pseudo-languages: `__empty__`, `__binary__`, `__generated__`, `__duplicate__`, `__unknown__`. Quirks: Markdown counts as 0 code (all comment — expected); JSON counts conservatively (use `wc -l` for JSON); for large monorepos prefer `--suffix` over a full scan. These are size numbers, not quality judgments — pair with Part 2 (jscpd health) for health.

## Part 2 — Refactor (scoped health pass: duplication → dead code → complexity)

A guided pass over a codebase's three biggest maintenance costs using `jscpd`. Use it on the code the change lands in, scoped to the change's affected areas, not the whole repo unless the user asked for a whole-repo cleanup.

### Start here: measure, then prioritize

```bash
npx jscpd --health --reporters ai <path>
# health 74 B (duplication 75, dead-code 72, complexity 76; 93 code lines)
```

The score is a weighted mix of three sub-scores, each a share of code lines — high is good, 100 clean. **The lowest sub-score names where the codebase hurts most; if it isn't the default order below, do that dimension first.** `n/a` on a dimension (e.g. `dead code n/a`) means jscpd couldn't measure it (dead-code detection needs JS/TS/Python to be a real share of the code) — skip that step, don't read it as clean.

For the full picture — largest duplicated formats, most complex files, largest dead-code findings, one screen — run `npx jscpd --dashboard` (its `console` output is the only reporter with per-file detail; `--reporters json` if you need it parseable). Don't run all three passes blind on code you've never seen — read the dashboard first so you don't spend equal effort on two dimensions that are already fine.

The three passes are otherwise independent and safe in any order; **duplication → dead code → complexity** is the default because fixing duplication first means step 2 doesn't trace call sites a later extract will move anyway, and clearing dead code before step 3 means the files ranked for simplification are ones actually worth simplifying.

### Step 1 — Duplicated code

```bash
npx jscpd --reporters ai --summary <path>
```

Read each clone, decide whether it's a real copy worth merging, apply the matching strategy:

- **exact clone** → extract function or shared module
- **renamed clone** → parameterize the shared function
- **near-miss clone** → extract the common core, leave the divergent part at each call site
- **structural duplication** (same shape, different types) → base class / template / a generic

A file jscpd flags as a duplication hotspot is the same file that dragged the duplication sub-score down — fixing it is the direct lever.

### Step 2 — Dead code

```bash
npx jscpd --dead-code --reporters ai <path>
```

This is a graph traversal from the project's entry points (`package.json` main/bin/exports/scripts, `pyproject.toml` scripts, framework conventions, or `--entry <glob>`), not a reference count — a helper whose only caller is itself dead is reported too, so **deleting one finding can make another true that wasn't reported yet**.

| Category | Meaning | Usual fix |
|----------|---------|-----------|
| `unused-file` | Never imported, not an entry point | Delete the file |
| `unused-export` | Exported, no other file imports it | Delete, or drop `export` if same-file uses it |
| `unused-symbol` | Module-private, nothing in its file references it | Delete |
| `unused-import` | Imported, never referenced in the file | Delete the import line |

Workflow: sort findings high→low confidence (0-100; **below 90 the finding carries a stated reason it may be wrong** — `eval`/`getattr`, an unrecognized decorator, a wildcard re-export, a name only in a string literal, an unparsed file). Confirm each kept finding by hand (grep the symbol across the repo — dynamic access, a string-based router, or a template file jscpd doesn't parse can all hide a real use) before deleting. Run the project's test suite and build after each batch, not only jscpd's re-run: dead-code analysis proves nothing *reaches* the code, not that removing it can't break a build step. **Re-run `--dead-code` after each batch** — the cascade (a file whose only import was the one you just deleted is now dead itself) is expected; keep going until a clean pass. Report what you skipped and why (low confidence, a reason you couldn't rule out, intentionally-kept dead code such as a public library export) separately from what you removed.

### Step 3 — Big and complex files

```bash
npx jscpd --complexity --reporters ai --summary-top 10 <path>
# or ranked alongside duplication and size:
npx jscpd --reporters ai --summary --summary-by complexity <path>
```

`cx` is a language-aware cyclomatic-complexity estimate (one path per function plus one per branch) — read it as a ranking signal, not an exact metric. Prose/data files (Markdown, JSON, YAML, lock files) always score `0`. jscpd's health score treats complexity **≥ 50** as "complex" — a reasonable "worth reading" cutoff when the summary doesn't make an obvious one clear.

Workflow: take the top N by complexity (and by size from `--summary` — a huge low-complexity file, e.g. a big data table, is a different problem than a small high-complexity one). Read each and name the actual driver, then pick the matching strategy:

- **Deep nesting** → guard clauses / early return, flatten the happy path
- **Long function, several responsibilities** → extract a function per responsibility, named for what it does
- **Branching over a fixed case set** (type, enum, status) → lookup table / map, or polymorphism if each case already has its own type
- **File doing too much** → split along its actual seams (one export group per file), not an arbitrary line-count cut

Refactor one file at a time, **keep behavior identical** — a complexity pass, not a rewrite. If a duplication clone (step 1) or dead branch (step 2) turns up while reading, still handle it in its own step so the reason for each change stays traceable. Re-run `--complexity` on the touched files to confirm `cx` actually dropped — a split that only moves code around without reducing branching per function renamed the problem, it didn't fix it.

### Step 4 — Confirm the improvement

```bash
npx jscpd --health --reporters ai <path>
```

Compare the score and grade to the measurement you took before the pass. A dimension that got worse (duplication crept back in while extracting, say) is a signal to revisit that step — re-run the relevant pass (`--summary`, `--dead-code`, `--complexity`) rather than only trusting the one aggregate number. Health scores from two runs are only comparable when built from the same dimensions; if a dimension went from `n/a` to measured (or the reverse) because the file mix changed, say so rather than reading the raw delta as improvement or regression.

### Committing the pass

**Commit each step separately** (duplication fixes, dead-code removal, complexity simplification) per `_shared/work-unit-commits` — a reviewer verifying "did this actually reduce X" wants a diff scoped to X, and each step is independently revertable.

## Part 3 — Design (deep modules for new or reshaped interfaces)

When the task creates a new module/interface or reshapes an existing one (splitting a file, extracting a seam, moving code to a shared module), shape it as a **deep module**: a lot of behaviour behind a small interface, placed at a clean seam, testable through that interface. The aim is leverage for callers, locality for maintainers, testability for everyone.

### The vocabulary (use exactly; don't substitute "component", "service", "API", "boundary")

- **Module** — anything with an interface and an implementation. Scale-agnostic: a function, class, package, or tier-spanning slice. *Avoid:* unit, component, service.
- **Interface** — everything a caller must know to use the module correctly: the type signature, plus invariants, ordering constraints, error modes, required config, performance characteristics. *Avoid:* API, signature (too narrow — those refer only to the type-level surface).
- **Implementation** — what's inside. Distinct from **Adapter**: a concrete thing that satisfies an interface at a seam (describes the slot it fills, not the substance).
- **Depth** — leverage at the interface: behaviour a caller (or test) can exercise per unit of interface it has to learn. **Deep** = lots of behaviour behind a small interface. **Shallow** = interface nearly as complex as the implementation (avoid).
- **Seam** (Feathers) — a place where behaviour can be altered without editing there; the *location* of the interface. *Avoid:* boundary (overloaded with DDD's bounded context).
- **Leverage** / **Locality** — what depth buys: more capability per unit of interface learned; change/bugs/knowledge concentrated in one place, fix once fixed everywhere.

### Deep vs shallow

```
Deep:    small interface + lots of implementation
         ┌─────────────────────┐
         │  Small Interface    │  few methods, simple params
         ├─────────────────────┤
         │  Deep Implementation│  complex logic hidden
         └─────────────────────┘

Shallow: large interface + little implementation (avoid)
         ┌─────────────────────────────────┐
         │  Large Interface                │  many methods, complex params
         ├─────────────────────────────────┤
         │  Thin Implementation            │  just passes through
         └─────────────────────────────────┘
```

When designing an interface, ask: can I reduce the number of methods? Simplify the parameters? Hide more complexity inside?

### Principles

- **Depth is a property of the interface, not the implementation.** A deep module can be internally composed of small, mockable, swappable parts; they just aren't part of the interface. A module can have **internal seams** (private to its implementation, used by its own tests) as well as the **external seam** at its interface.
- **The deletion test.** Imagine deleting the module. If complexity vanishes, it was a pass-through. If complexity reappears across N callers, it was earning its keep.
- **The interface is the test surface.** Callers and tests cross the same seam. If you want to test *past* the interface, the module is probably the wrong shape.
- **One adapter means a hypothetical seam. Two adapters means a real one.** Don't introduce a seam unless something actually varies across it (typically production + test adapters).

### Designing for testability

1. **Accept dependencies, don't create them.** `processOrder(order, paymentGateway)` beats `processOrder(order) { const gw = new StripeGateway() }`.
2. **Return results, don't produce side effects.** `calculateDiscount(cart): Discount` beats `applyDiscount(cart): void { cart.total -= d }`.
3. **Small surface area.** Fewer methods = fewer tests; fewer params = simpler setup.

### Deepening a cluster (when reshaping existing shallow modules)

Classify the candidate's dependencies; the category determines how the deepened module is tested across its seam:

1. **In-process** — pure computation, in-memory state, no I/O. Always deepenable: merge the modules, test through the new interface directly. No adapter needed.
2. **Local-substitutable** — dependencies with local test stand-ins (PGLite for Postgres, in-memory fs). Deepenable if the stand-in exists; test with the stand-in in the suite. The seam is internal; no port at the external interface.
3. **Remote but owned** (Ports & Adapters) — your own services across a network. Define a **port** (interface) at the seam; the deep module owns the logic, transport injected as an **adapter**. Tests use an in-memory adapter; production an HTTP/gRPC/queue adapter.
4. **True external** (Mock) — third-party services you don't control (Stripe, Twilio). Injected port; tests provide a mock adapter.

**Seam discipline:** one adapter = hypothetical seam, two = real one. Don't expose internal seams through the interface just because tests use them.

**Testing strategy — replace, don't layer:** old unit tests on the shallow modules become waste once tests exist at the deepened interface; delete them. Write new tests at the interface, asserting observable outcomes, not internal state. Tests should survive internal refactors — if a test has to change when the implementation changes, it's testing past the interface.

### Design it twice (optional, for a high-stakes interface)

When the interface choice is high-stakes (a new shared module, a seam that will be relied on for a long time), your first idea is unlikely to be the best. Spawn 3+ sub-agents in parallel, each producing a **radically different** interface for the same module, then compare:

- Agent 1 — minimize the interface: 1–3 entry points max, maximize leverage per entry point.
- Agent 2 — maximize flexibility: support many use cases and extension.
- Agent 3 — optimize the most common caller: make the default case trivial.
- Agent 4 (if applicable) — design around ports & adapters for cross-seam dependencies.

Each outputs: the interface (types, methods, params, invariants, ordering, error modes), a usage example, what the implementation hides behind the seam, the dependency strategy + adapters, and trade-offs (where leverage is high vs thin). Present sequentially so the user absorbs each, compare in prose by **depth** (leverage), **locality** (where change concentrates), and **seam placement**, then give your own recommendation — be opinionated; the user wants a strong read, not a menu. If elements combine well, propose a hybrid.

## Red Flags (structuring)

- Planning on a brownfield repo without measuring it first (size numbers drive the 400-line budget and slicing).
- Running pygount with no `--folders-to-skip` (hang risk on dependency trees).
- Quoting Markdown "code lines" as code, or treating LOC as a quality judgment.
- Running all three jscpd passes blind on unfamiliar code without reading `--health`/`--dashboard` first.
- Deleting a low-confidence (<90) dead-code finding without reading its stated reason and grepping for a real use.
- Running the test suite only via jscpd's re-run, not the project's own suite + build, after a batch of deletions.
- A "complexity" split that moves code around without reducing branching per function (cx didn't drop).
- Introducing a seam with only one adapter (hypothetical seam = pure indirection).
- A new module whose interface is nearly as complex as its implementation (shallow — the deletion test would delete it).
- Testing past the interface: a test that has to change when the implementation changes.

## Verification (structuring)

- [ ] Brownfield sizing: numbers came from a stated tool (pygount or named fallback), `--folders-to-skip` covered the dependency/build dirs (no `node_modules`/`vendor` rows), report holds total LOC, top-3 languages, code-vs-comment ratio, 5 largest files.
- [ ] Refactor pass: started from `--health`/`--dashboard`, did the lowest sub-score dimension first, each step committed separately, test suite + build ran after each dead-code batch, re-ran `--health` at the end and a regressed dimension was revisited.
- [ ] Design: the interface follows the deep-module vocabulary (not "component/service/API/boundary"), passed the deletion test (complexity reappears across callers if removed), and is the test surface (tests assert observable outcomes, not internal state).
- [ ] No single-adapter seams introduced; no shallow pass-through modules created.
