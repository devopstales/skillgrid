# Functional Testing

Does the behavior actually run? This area runs the test suite, selects the right layer for each behavior, does the exploratory browser pass when a URL is reachable, and checks the diff for regressions.

**When NOT to run this area:** a pure refactor with no behavioral change where the existing suite is the gate — run the regression check only, not the full layer selection. Record the scope you ran.

## The Test Suite

Run `testing.runner` from config. Capture the full output — the exit code and the failing-test names, not a summary. A test that ran and failed is a FAIL-floor finding.

- **P0 / P1 pass rates:** if the suite tags P0/P1 tests, compute the pass rates against `quality.p0_pass_rate` (default 100) and `quality.p1_pass_rate` (default 95). Any P0 failure is a FAIL.
- **A test that was filtered, skipped, or unregistered is not "covered."** It counts as `pending` until it runs and passes.

## Layer Selection

For each behavior in the change, select the **highest available layer that fits** — but never skip a behavior because a layer is unavailable; degrade to the next available layer.

| What the behavior is | Layer |
|----------------------|-------|
| Pure logic / calculation / data transform | Unit |
| Component rendering / interaction | Integration |
| Multi-component flow / API contract | Integration |
| Persistence / database operation | Integration |
| Critical user journey / full end-to-end path | E2E |
| **Default (always the fallback)** | **Unit** |

Available layers come from `testing.layers` in config (e.g. `[unit, integration, e2e]`). If a layer is not listed, degrade: E2E → Integration → Unit.

**Duplicate Coverage Guard** — before assigning a layer, ask:
1. Is this already tested at a lower level?
2. Can a unit test cover this instead of integration?
3. Can an integration test cover this instead of E2E?

If yes — assign the lower layer. A test at the wrong layer is a slower, flakier, more expensive way to test something a cheaper layer already covers.

**Cadence:**

| Cadence | What runs |
|---------|-----------|
| **Every PR** | All functional tests, if the suite runs in < 15 minutes. Always: unit + integration for the changed area. |
| **Nightly / weekly** | Long-running or expensive suites: full E2E, performance, chaos, large datasets. |

A test that runs nightly but not on PR is not blocking — it's a regression net, not a gate.

## The Browser Pass (when a URL is reachable)

If the change is user-facing and a URL is reachable (local dev server, preview, or a target the human names), do an exploratory pass. Pick the driver in this order: the `agent-browser` MCP if it is wired in the session (tool names `agent_browser_*` — see below), else `playwright-cli` (full command reference: [playwright-cli.md](playwright-cli.md)). The steps are identical either way; the tool names map 1:1 (`open`/`goto` → navigate, `snapshot`, `click`, `fill`, `press`, `eval`, `console`, `screenshot`, `close`).

1. **Plan** — sketch the sitemap under test: landing, nav, the changed flows, forms, edge states (empty, error, 404).
2. **Explore** — `open <url>` → `snapshot` → **check the console (`console`) after every nav + interaction** → exercise interactives via refs from the snapshot (`fill`, `click`, `press`; valid + invalid + empty inputs, keyboard, scroll) → after each interaction re-check console and expected-vs-actual. Search a large snapshot with `find` instead of capturing it all; `eval` for DOM facts the snapshot omits.
3. **Classify** — silent JS errors are the highest-value findings. Classify each by severity × category.

The browser pass finds what the suite does not specify. A genuine correctness bug surfaced here is reported as a separate "found a bug" note, not folded into cleanup. Close the session when done (`close`). If `playwright-cli` is not installed, fall back per [playwright-cli.md](playwright-cli.md) § Installation (`npx playwright cli`, else `npm install -g @playwright/cli@latest`) — and if no browser tooling is available at all, say so and stop: don't narrate a test you can't run.

**agent-browser MCP (preferred when wired):** the `agent-browser` MCP server (run as `agent-browser mcp --tools all`) drives a real browser with the same snapshot-ref model as `playwright-cli`. Prefer it over the CLI when present: `agent_browser_open` / `agent_browser_goto` → `agent_browser_snapshot` (stable element refs) → `agent_browser_click` / `agent_browser_type` / `agent_browser_press` on refs → `agent_browser_console` after every nav + interaction → `agent_browser_screenshot` for evidence. Use `agent_browser_tab_list` to track multiple tabs and the `session` parameter to keep the pass in an isolated browser session. If the MCP is wired but a call fails (browser not launchable, no display), say so and fall back to `playwright-cli` for the same steps — don't silently skip the pass.

**Evidence:** capture `screenshot` (or `video-start`/`video-stop`) of the failure state when a bug is found, and `--raw snapshot` before/after for behavioral deltas — the report section below carries these as the pass's evidence. Attach a screenshot or short video to the PR when it saves the reviewer a checkout ([pr-attachments.md](playwright/pr-attachments.md)).

## The Regression Check

Review the diff for behavioral differences against the pre-change code, per `skillgrid:regression-dog`. Focus on logic/behavior/data — the suite and linters are covered elsewhere. A behavior that changed without a covering test is a regression gap (a FAIL-floor finding if it is on a path the change touched).

## Writing the Report Section

```
## Functional
Suite: <command> → <exit code>, <N> tests, <K> failing
P0: <pass %>/<floor>   P1: <pass %>/<floor>
| Behavior | Layer | Test | Ran? | Result |
|---|---|---|---|---|
| <behavior> | Integration | <test name> | yes | pass |

Browser pass: <ran / N/A — no reachable URL> — <findings>
Regression: <N> behavioral deltas, <K> without a covering test (<names>)
```
