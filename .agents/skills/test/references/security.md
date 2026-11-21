# Security

Five tools, each with a different shape. Run the ones the target warrants, not all five unscoped. The split:

- **trivy** — static: dependencies, filesystem, containers, IaC, secrets, SBOM. No running target needed. This is the default for every security run.
- **owasp-security** — code/diff review against OWASP Top 10:2025, ASVS 5.0, LLM Top 10, Agentic Top 10. No running target needed; reviews the change.
- **nuclei** — broad DAST: thousands of templates (CVEs, exposures, misconfigs, network) at a *running* target. Fast, noisy; a match is a lead.
- **wapiti** — lightweight DAST: crawls + fuzzes a *running* web app, no browser, no template library. Fast first report.
- **akca** — deep DAST: fingerprints the stack, discovers the real attack surface, and *verifies* each signal with baselines/controls. Evidence-backed, fewer false positives, browser-backed.

## Selecting the tools

| Target shape | trivy | owasp | nuclei | wapiti | akca |
|---|---|---|---|---|---|
| Code/diff, no running target | **yes** (deps/IaC/secrets) | **yes** (the review) | — | — | — |
| Running web app/API, broad CVE sweep | yes | if a trust boundary is touched | **yes** | optional (quick pass) | if deep evidence is needed |
| Running app, quick pre-release check | yes | if a trust boundary is touched | optional | **yes** (lightweight) | — |
| Running app, SPA / behind auth / need proof | yes | if a trust boundary is touched | optional | — | **yes** (deep, browser-backed) |
| Dependency / IaC / Dockerfile change | **yes** | if config/IaC/CI changed | — | — | — |

**When NOT to run the DAST tools (nuclei / wapiti / akca):** there is no running, reachable, authorized target. They need a live URL. For code/config review without a running target, use trivy + owasp-security.

## trivy

Static evidence source. No running target needed. Per `skillgrid:trivy`:

```bash
# Filesystem (deps + secrets + misconfig) — the default
trivy fs --severity CRITICAL,HIGH --scanners vuln,secret,misconfig .
# IaC / Dockerfile / K8s / Terraform
trivy config --severity CRITICAL,HIGH <path>
# Container image
trivy image --severity CRITICAL,HIGH <image>
# SBOM for compliance
trivy fs --format spdx-json --output sbom.json .
```

Adjust per config: `security.trivy.severities` → `--severity`, `security.trivy.scan_types` → `--scanners`.

**Reading results (trivy MCP, when wired):** the `trivy` MCP server (run as `trivy mcp`) exposes `findings_list` / `findings_get` over the scan results. When it is connected, run the scan as above with a `-json-export`/`-o` output file, then read the findings as structured data via the MCP tools instead of parsing the terminal — `findings_list` to enumerate, `findings_get` for the full detail on a load-bearing one. The MCP reads results; it does not run the scan, so the CLI commands above are still the scan itself. If the MCP is not wired, triage from the exported JSON directly.

**Classify each finding:**

| Scanner | Severity | Classification |
|---------|----------|----------------|
| vuln | CRITICAL | **CRITICAL** — production dependency with a known fix |
| vuln | HIGH | **WARNING** (or **CRITICAL** if `fail_on` includes HIGH) |
| vuln | MEDIUM/LOW | **SUGGESTION** |
| secret | any | **CRITICAL** — a secret was found in a tracked file |
| misconfig | CRITICAL/HIGH | **WARNING** — a Dockerfile/k8s/CI misconfiguration |
| misconfig | MEDIUM/LOW | **SUGGESTION** |

**Gate:** if `security.trivy.fail_on` is set and any finding meets or exceeds that severity, the security gate FAILs. If `fail_on` is empty, Trivy findings are reported but never block. If Trivy is configured but not on `$PATH` at scan time, note "Trivy configured but not found — running manual fallback" and do the manual passes below.

**Manual fallback** (when Trivy is unavailable): secrets archaeology (git history + tracked files for `AKIA`, `ghp_`, `sk-`, `-----BEGIN.*PRIVATE KEY-----`, `.env`/`*.pem`/`credentials.json`), dependency audit (`npm audit` / `go list -m -u all` / `pip-audit` / `cargo audit`), and an OWASP spot-check (the full review is owasp-security's job).

## nuclei — broad DAST

Thousands of templates at a running target. A match is a *lead*, not a confirmed exploit. Per `skillgrid:nuclei`:

```bash
nuclei -u <url> -s high,critical -json-export findings.json -silent   # fast risk signal
nuclei -u <url> -tags cve                                             # known CVEs only
nuclei -u <url> -id CVE-2021-44228                                    # a specific CVE
```

Confirm `nuclei -version && nuclei -health-check` before scanning. Narrow the net by severity/tag/template — never run unscoped "all" on a small, known stack. Rate-limit a production target (`-rate-limit 50 -concurrency 10`). Triage from the exported JSON, not the scrolling terminal.

## wapiti — lightweight DAST

Black-box fuzzer: crawls a deployed app, attacks the scripts it finds, matches on error strings. No browser, no template library. Per `skillgrid:wapiti` (Python 3.12–3.14):

```bash
wapiti -u <url> -o report.json                    # default attack set
wapiti -u <url> -m sql,xss,exec -o report.json    # only the modules you need
wapiti -u <url> --scope page --max-time 30m       # one page, time-capped
```

Start scope at `page`/`folder`, not `domain`. Narrow with `-m` to the modules matching the change. Add auth for gated apps (`--basic-auth`, `-c`, `--form-script`). Triage every hit — error-based SQLi and file-inclusion are the noisiest; confirm the payload is actually reflected.

## akca — deep DAST

Evidence-first: fingerprint → discover → select → verify with baselines/controls. Browser-backed (needs Chrome/Chromium/Edge). Per `skillgrid:akca` (Go 1.25+):

```bash
akca -u <url> -f json -o findings.json                 # full profile
akca -u <url> -m api -f json -o findings.json          # API change only
akca -u <url> -c "session=..." --request-budget 5000 --time-budget 30m
akca -u <url> --api-spec ./openapi.yaml -m api         # import API scope
```

Prefer the narrowest module set that matches the change over `full`. Set a request/time budget so a partial run is reported as **incomplete coverage**, not a false clean. Triage with evidence: read the proof policy + confidence, and `akca replay --finding N` to confirm a high-severity item.

## owasp-security — code/diff review

Reviews the change (not a running target) against OWASP Top 10:2025, ASVS 5.0, LLM Top 10 (2026), and Agentic Top 10 (2026). Per `skillgrid:owasp-security`:

1. **Map entry points and trust boundaries** — where attacker-controlled data enters (routes, headers, cookies, uploads, webhooks, queue, CLI args, third-party API responses, anything an LLM reads/returns) and where auth/authz is enforced (often centralized middleware).
2. **Load the references** the code needs (language section, config/supply-chain, LLM/Agentic sections).
3. **Sweep** the review checklist for the categories the code touches; trace each candidate from entry point to sink.
4. **Triage** every candidate with the rubric below.
5. **Report** highest severity first.

## The triage rubric (all tools)

A pattern match is not a vulnerability. The most common failure is reporting unreachable or already-mitigated code, which buries the real findings. Confirm all four before reporting:

1. **Is the input actually attacker-controlled and reachable?** Trace it to a real entry point. An internal endpoint behind a network boundary is not externally exploitable.
2. **Is it a real exposure or a false positive?** Confirm the sink is reachable with that input — check validation, an allowlist, an ORM, framework control, or auth middleware that already sits between them.
3. **Blast radius.** Who can trigger it, what they get, which trust boundary it crosses.
4. **Can the attacker perform every step?** Each step of the exploit must be possible from the attacker's position. If a step needs a capability the attacker doesn't have, it's "Needs verification", not High.

**Report severity by exploitability, not by the tool's nominal severity.** Mark every finding **Confirmed / Likely / Needs verification** and say what you could not see. A template match or fuzzer hit without a demonstrated path is a lead, not a Confirmed finding.

## The finding block

One block per finding, highest severity first:

```
[SEVERITY] Title (CWE-###, OWASP A##:2025, LLM##/ASI## Risk Name)
Tool:       trivy | nuclei | wapiti | akca | owasp
Location:   path/to/file.ext:LINE  (code)  /  <url> + <param> (DAST)
Path:       <entry point> -> <intermediate hops> -> <sink>
Impact:     who can trigger it, what they get, which trust boundary it crosses
Fix:        the concrete change, with a snippet when it isn't obvious
Evidence:   the matched request/response, stored transaction, or scanner record
Confidence: Confirmed | Likely | Needs verification (say what you couldn't see)
```

Write every LLM and ASI ID with its risk name (the 2025 and 2026 editions reuse the same numbers for different risks). If the review finds nothing exploitable, say so directly — do not pad with Info items.

| Severity | Meaning |
|---|---|
| Critical | Unauthenticated RCE, auth bypass, or mass data exposure |
| High | Authenticated exploitation crossing a trust boundary (IDOR across tenants, SQLi behind login) |
| Medium | Needs unusual preconditions, or impact limited to the attacker's own data |
| Low | Defense-in-depth gap with no demonstrated exploit path |
| Info | Hardening suggestion; say plainly that it is not a vulnerability |

## Persisting findings

If `mnemonic.enabled: true`, persist each scanner's findings (fail-open — a store error never blocks the caller): `scan_start(tool, target, …)` → write the raw JSON to the returned path → `scan_store_findings(scan_id, tool)`. For trivy with an SBOM, also `dep_ingest`.

## Writing the Report Section

```
## Security
trivy: <command> → <N> findings (<C> critical, <H> high, <S> secrets)
owasp: <N> findings (<severity breakdown>)
nuclei: <ran / N/A — no target> → <N> leads triaged, <K> confirmed
wapiti: <ran / N/A> → <N> hits triaged, <K> confirmed
akca:   <ran / N/A> → <N> verified findings

| # | Sev | Title | Tool | Location | Confidence |
|---|---|---|---|---|---|
| 1 | High | <title> | trivy | <loc> | Confirmed |
```
