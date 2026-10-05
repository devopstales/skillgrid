# Testing & Verification Standards (shared across all Skillgrid skills)

Single source of truth for the testing and verification toolchain: what to run, at which layer, and with which tool. Complements `verification-ladder.md` (depth: L1–L4) and `verification-scope.md` (scope: COMPLETE/TRUNCATED/UNSCOPED/UNREADABLE).

## Test Layers

| Layer | Scope | Tool | When |
|---|---|---|---|
| **Unit** | Single function / class in isolation | Project test runner (pytest, jest, vitest, go test) | Every work unit (L1+) |
| **Integration** | Multiple modules interacting (API + DB, service + broker) | Project test runner with real deps or testcontainers | L2+ (standard changes) |
| **E2E / Acceptance** | Full user-facing flow through the real stack | Playwright (browser) or project e2e runner | L3+ (risky changes) |
| **Security (SAST)** | Static analysis of source + deps | Trivy (filesystem), project linter | L4 (high-risk) |
| **Security (DAST)** | Running app, black-box | Wapiti (fuzzer), Nuclei (templates), AKCA (evidence-driven) | L4 or on-demand |

## Unit & Function Tests

- **TDD is always on.** RED → GREEN → TRIANGULATE → REFACTOR (`skillgrid:test-driven-development`).
- Every public function gets at least one test covering the happy path + one failure mode.
- Tests are the spec: name them by behavior (`test_user_cannot_see_others_profile`), not by implementation.
- No test may depend on execution order, wall-clock time, or network.
- Mock at the boundary, not inside the unit under test.

## Integration Tests

- Verify contract between modules (API shape, DB schema, message format).
- Use real infrastructure where feasible (testcontainers, local Postgres); mock only external SaaS.
- Every migration gets an integration test that runs the migration + a read back.

## E2E / Playwright

- Playwright is the canonical e2e browser. `skillgrid:playwright-cli` wraps it.
- E2E tests encode the `acceptance.feature` BDD scenarios (`skillgrid:acceptance-test-authoring`).
- Keep the e2e suite small: one flow per user story, not one per UI element.
- Screenshots on failure are mandatory evidence for the QA gate.
- Headless by default; headed only for local debug.

## Duplication & Dead Code Detection

| Tool | Detects | Command | Frequency |
|---|---|---|---|
| **jscpd** | Exact, renamed, near-miss code clones (220+ languages) | `npx jscpd --reporters ai <path>` | Before ship (L3+), or on refactor |
| **pygount** | LOC, language breakdown, code-vs-comment ratio | `pygount --format csv <path>` | Project sizing, refactoring |
| **dead code** (jscpd + manual) | Unreachable functions, unused exports | jscpd `--max-gap-lines 1` + project lint | Refactoring passes |

**Thresholds (advisory, project may override in `config.yaml`):**
- Duplication % > 5% → flag for refactoring (extract shared function/module).
- A file > 400 lines with > 30% duplication → split.
- Dead code: remove or comment with a tracking reference.

## Security Scanning

### SAST (Static — on source/deps)

| Tool | Scope | Command | When |
|---|---|---|---|
| **Trivy** | Container images, filesystem, IaC, secrets, misconfigs, license | `trivy fs --scanners vuln,misconfig,secret .` | L4, before ship, CI |
| **Project linter** | Language-specific rules (ruff, eslint, golangci-lint) | Per `config.yaml` `testing.lint` | L3+ (every change) |
| **owasp-security skill** | Code review against OWASP Top 10, LLM & Agent Top 10 | `skillgrid:owasp-security` | L4, auth/crypto/LLM changes |

### DAST (Dynamic — on running app)

| Tool | Scope | Command | When |
|---|---|---|---|
| **Wapiti** | Black-box fuzzer: SQLi, XSS, command injection, XXE, open redirect | `wapiti -t <url> -o report.html` | L4, pre-release, on-demand |
| **Nuclei** | Template-based: CVEs, exposures, misconfigs, network | `nuclei -u <url> -t <template-dir>` | L4, broad-net scan, CVE verification |
| **AKCA** | Evidence-driven DAST: crawls (HTTP + headless), adaptive payloads, replayable proof | `skillgrid:akca` | L4, authenticated apps, false-positive reduction |

**Security rules:**
- L4 changes (auth, data migration, public API, money, concurrency) MUST run at least one SAST + one DAST tool before ship.
- Critical/High findings block ship. Medium findings require a documented risk-acceptance in `review.md`.
- Secrets in code are always Critical. `.gitignore` + `Trivy secret` scan are the gate.

## Verification Scope (mandatory)

Every test/scan result carries its scope (`verification-scope.md`):

- `SCOPE: COMPLETE` — all input seen. Zero is a real answer.
- `SCOPE: TRUNCATED` — stopped early (budget/limit). Zero is a non-answer.
- `SCOPE: UNSCOPED` — no boundary defined. Zero is a non-answer.
- `SCOPE: UNREADABLE` — input present but unreadable. Zero is a non-answer.

A PASS verdict requires `SCOPE: COMPLETE` on every layer that the verification floor demands.

## Config Keys (`.skillgrid/config.yaml`)

```yaml
testing:
  runner: "pytest"            # or "pnpm test", "go test ./...", etc.
  layers: [unit, integration, e2e]
  coverage: "pytest --cov"
  mutation: ""               # empty = disabled
  lint: "ruff check ."
  e2e: "pnpm playwright test"

quality:
  coverage_min: 80           # % (0 = disabled)
  mutation_min: 80           # % (0 = disabled)
  p0_pass_rate: 100          # %
  p1_pass_rate: 95           # %
  duplication_max: 5         # % (jscpd)
  security:
    sast: "trivy fs --scanners vuln,secret ."
    dast: "wapiti -t http://localhost:8080 -o /tmp/wapiti.html"
```

## Red Flags

- A "pass" verdict with no `SCOPE:` line on any test layer.
- L4 change shipped without a SAST or DAST run.
- Duplication > 5% with no refactoring ticket.
- E2E suite that tests UI cosmetics instead of user flows.
- Unit test that hits the network or depends on wall-clock time.
- Security finding marked "won't fix" with no risk-acceptance in `review.md`.
