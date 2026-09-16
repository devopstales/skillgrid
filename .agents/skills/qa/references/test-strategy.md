# Test Strategy

Layer selection, security audit, and code-quality gate. Adapted from BMAD `test-levels-framework` + `step-04-coverage-plan`, gsd-core `TESTING-STANDARDS` + `TESTING-SUITES`, and gstack `cso/audit-phases`.

## Layer Selection

For each behavior in the test plan, select the **highest available layer that fits** — but never skip a behavior because a layer is unavailable; degrade to the next available layer.

| What the behavior is | Layer |
|----------------------|-------|
| Pure logic / calculation / data transform | Unit |
| Component rendering / interaction | Integration |
| Multi-component flow / API contract | Integration |
| Persistence / database operation | Integration |
| Critical user journey / full end-to-end path | E2E |
| **Default (always the fallback)** | **Unit** |

Available layers come from `testing.layers` in `.skillgrid/config.yaml` (e.g. `[unit, integration, e2e]`). If a layer is not listed, degrade: E2E → Integration → Unit.

### Duplicate Coverage Guard

Before assigning a layer, ask:

1. Is this already tested at a lower level?
2. Can a unit test cover this instead of integration?
3. Can an integration test cover this instead of E2E?

If yes — assign the lower layer. A test at the wrong layer is not a test at all; it's a slower, flakier, more expensive way to test something a cheaper layer already covers.

### Cadence

| Cadence | What runs |
|---------|-----------|
| **Every PR** | All functional tests, if the suite runs in < 15 minutes. Always: unit + integration for the changed area. |
| **Nightly / weekly** | Long-running or expensive suites: full E2E, performance, chaos, large datasets. |

Record the cadence per test in the test plan. A test that runs nightly but not on PR is not blocking — it's a regression net, not a gate.

## Security Audit

Two modes: **Trivy** (primary, when configured) and **Manual** (fallback, when Trivy is disabled). Each produces findings classified as CRITICAL / WARNING / SUGGESTION.

### Mode A: Trivy (when `security.trivy.command` is set)

Run the configured Trivy command. The default invocation:

```bash
trivy fs --severity CRITICAL,HIGH --scanners vuln,secret,misconfig .
```

Adjust per `config.yaml`:
- `security.trivy.severities` → `--severity` flag
- `security.trivy.scan_types` → `--scanners` flag
- `security.trivy.target` → the path argument (default `.`)

**Parse the output.** For each finding, classify:

| Trivy scanner | Severity | Classification |
|---------------|----------|----------------|
| vuln | CRITICAL | **CRITICAL** — production dependency with a known fix |
| vuln | HIGH | **WARNING** (or **CRITICAL** if `fail_on: "CRITICAL,HIGH"`) |
| vuln | MEDIUM/LOW | **SUGGESTION** |
| secret | any | **CRITICAL** — a secret was found in a tracked file |
| misconfig | CRITICAL/HIGH | **WARNING** — a Dockerfile/k8s/CI misconfiguration |
| misconfig | MEDIUM/LOW | **SUGGESTION** |

**Gate:** if `security.trivy.fail_on` is set and any finding meets or exceeds that severity, the Security gate **FAILs**. If `fail_on` is empty, Trivy findings are reported but never block.

**If Trivy is not available at scan time** (command not found, even though config says it should be): note "Trivy configured but not found — running manual fallback" and proceed to Mode B.

### Mode B: Manual (when `security.trivy.command` is empty)

Three passes.

#### Pass 1: Secrets Archaeology

| Check | What to look for |
|-------|-----------------|
| Git history | Secret prefixes in commit messages or diffs: `AKIA`, `ghp_`, `sk-`, `xoxb-`, `-----BEGIN.*PRIVATE KEY-----` |
| Tracked files | `.env`, `.env.*`, `*.pem`, `*.key`, `credentials.json`, `id_rsa` tracked in git (not in `.gitignore`) |
| Inline in code | Hardcoded API keys, tokens, passwords, connection strings in source files |
| CI config | Inline secrets in workflow files (not from the secret store) |

**CRITICAL** if a secret is committed to git history or tracked in a file. **WARNING** if a secret is inline in code but not committed.

#### Pass 2: Dependency Audit

| Check | Command (stack-dependent) |
|-------|---------------------------|
| CVE scan | `npm audit` / `go list -m -u all` / `pip-audit` / `cargo audit` |
| Install scripts | `postinstall` / `preinstall` hooks in production dependencies |
| Lockfile integrity | Lockfile present and up-to-date (not hand-edited) |

**CRITICAL** if a production dependency has a HIGH or CRITICAL CVE with a known fix. **WARNING** if a CVE has no fix yet, or an install script runs arbitrary code.

#### Pass 3: OWASP Top 10 Spot-Check

For each changed file that handles user input, check:

| OWASP item | What to look for |
|------------|-----------------|
| A01 Broken Access Control | Missing auth check on a new endpoint or permission |
| A03 Injection | String concatenation in SQL / shell / template (no parameterization) |
| A04 Insecure Design | New feature with no rate limiting, no input validation |
| A07 Auth Failures | Weak password policy, missing MFA, session not invalidated |
| A08 Data Integrity | Deserialization of untrusted input without validation |

This is a **spot-check**, not a full SAST run. Flag suspicious patterns for a human to confirm — do not assert a vulnerability without evidence.

**Note:** Mode B has no OWASP pass when Trivy is configured — Trivy's `misconfig` scanner covers infrastructure misconfigurations, and the OWASP spot-check is a manual code-review task that belongs in `skillgrid:requesting-code-review` (Standards axis), not in the security scanner.

## Code Quality Gate

Run the commands from `.skillgrid/config.yaml`. Each has a configurable threshold under `quality:`.

| Gate | Command | Threshold (config key) | Default | FAIL when |
|------|---------|------------------------|---------|-----------|
| Coverage | `testing.coverage` | `quality.coverage_min` | 80 | Coverage < threshold |
| Mutation | mutation command (if configured) | `quality.mutation_min` | 80 | Mutation score < threshold |
| Lint | `commands.lint` | — | — | Any error (warnings are SUGGESTION) |
| Typecheck | `commands.typecheck` | — | — | Any error |
| P0 pass rate | run P0 tests | `quality.p0_pass_rate` | 100 | Any P0 test fails |
| P1 pass rate | run P1 tests | `quality.p1_pass_rate` | 95 | P1 pass rate < threshold |

**Mutation testing:** if a mutation command is configured (e.g. `npx stryker run`), run it and report the mutation score. A surviving mutant is a concrete specification of missing coverage — treat it as a failing test. If no mutation command is configured, this gate is N/A (not a FAIL).

**Dead code:** if a dead-code tool is configured (e.g. `knip`, `ts-prune`, `golangci-lint` deadcode), run it and report findings as SUGGESTION. Not a hard gate unless `quality.dead_code_min` is set.

**Config resolution:** if `quality:` is absent from `config.yaml`, use the defaults above and note in the report: "quality: not configured — using defaults. Run `skillgrid:onboarding` to set thresholds."
