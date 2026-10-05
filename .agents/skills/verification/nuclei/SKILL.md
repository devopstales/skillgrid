---
name: nuclei
description: Runs Nuclei, the fast template-based vulnerability scanner, against a running target and triages its findings. Use when scanning a live web app, API, host, or domain for known CVEs, exposures, misconfigurations, and network issues; when you need fast active vulnerability checks against a reachable URL; or when verifying a specific CVE or template against a deployment.
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
  based_on: projectdiscovery/nuclei
---

# Nuclei

**Announce at start:** "I'm using the skillgrid:nuclei skill to scan this target and triage the findings."

Nuclei fires a large library of YAML templates (CVE checks, exposure scans, misconfigurations, network probes) at a **running** target and reports matches with a severity per finding. It is the fast, broad-net DAST option: thousands of templates in seconds, each tagged by severity and category. A match is a *lead*, not a confirmed exploit — every finding still needs the reachability triage below before it is reported.

## When to Use

- You have a live URL, host, or domain and want broad known-CVE / exposure / misconfig coverage quickly.
- Verifying whether a specific CVE or template applies to a deployment.
- Continuous or pre-release scanning of a reachable service.
- Network-layer checks (open ports, exposed services, SSL, DNS).

**When NOT to use:**
- Static review of code, config, or dependencies that is not running — use `skillgrid:owasp-security` (code/config) or a SAST/SCA tool.
- You need deep, evidence-backed injection/DOM analysis of a single app — use `skillgrid:akca` or `skillgrid:wapiti`.
- The target is not reachable, or is a stateful flow behind multi-step auth that Nuclei's request-based templates can't traverse well.

## Quick Start

```bash
nuclei -u https://example.com                       # all templates, auto-downloaded on first run
nuclei -u https://example.com -s high,critical -json-export findings.json
nuclei -u https://example.com -tags cve              # known CVEs only
nuclei -l targets.txt -severity medium,high,critical # many targets from a file
```

First run pulls the `nuclei-templates` repo. Confirm the install before scanning. If `nuclei` is not in `$PATH`, check `$(go env GOPATH)/bin/nuclei` — add that directory to `$PATH` or use the full path.

```bash
nuclei -version                # or $(go env GOPATH)/bin/nuclei -version
nuclei -health-check
```

## Scan Workflow

```
1. Confirm binary + templates        nuclei -version && nuclei -health-check
2. Scope the target                  live URL/host/CIDR; authorized; prefer test env
3. Pick the net                      severity / tags / templates / workflows (see flags)
4. Run and capture                   add -json-export (or -sarif-export) for machine review
5. Triage every match                reachability rubric below; drop the unreachable
6. Report                              highest severity first, in the standard finding block
```

### Choosing the net

Default (all templates) is thorough but noisy. Narrow by intent:

| Goal | Flags |
|------|-------|
| Fast risk signal | `-s high,critical` |
| Known vulnerabilities only | `-tags cve` (add `-s medium,high,critical` to drop info/low) |
| Exposed artifacts/configs | `-t http/exposures/` |
| Specific CVE | `-id CVE-2021-44228` or `-t http/cves/2021/CVE-2021-44228.yaml` |
| Tech-stack specific | `-as` (auto-detect via Wappalyzer) or `-tags wordpress,react` |
| Multi-step logic | `-w workflows/wordpress-workflow.yaml` |
| Network / non-HTTP | `-type network,dns,ssl` |
| Exclude noisy categories | `-etags dos,fuzzing` |

Use `-as` (automatic scan) when the stack is unknown; use explicit `-tags`/`-t` when you know what to look for. Prefer explicit over auto for repeatable runs.

### Rate limit the target

Active scanning sends real requests. Throttle before you flood a production or rate-limited host:

```bash
nuclei -u https://example.com -rate-limit 50 -concurrency 10
```

### Output for triage

Always export machine-readable output and read it; do not triage from the scrolling terminal:

```bash
nuclei -u https://example.com -s high,critical -json-export findings.json -silent
nuclei -u https://example.com -sarif-export findings.sarif     # for code-analysis platforms
```

Each JSONL/SARIF record carries the template id, matcher name, severity, target, and the matched request/response excerpt.

## Triage a Match

A template match is not a vulnerability. Run the same rubric as `skillgrid:owasp-security` ("Before Reporting a Finding") and drop or downgrade what fails it:

1. **Is the input/target actually reachable and attacker-relevant?** An exposed internal endpoint behind a network boundary is not exploitable by an external attacker.
2. **Is the match a real exposure or a template false positive?** Info-severity "exposure" templates and version banners are the biggest noise source. Confirm the resource is actually served and sensitive.
3. **Blast radius.** Who can trigger it, what they get, which trust boundary it crosses.
4. **Can the attacker perform every step?** A CVE template can match on a version string that is present but not the vulnerable code path.

Report severity by exploitability, not by the template's nominal severity. Mark confidence as Confirmed / Likely / Needs verification and say what you could not see.

## Key Flags

| Flag | Purpose |
|------|---------|
| `-u`, `-target` | Target URL/host (repeatable) |
| `-l`, `-list` | File of targets, one per line |
| `-t`, `-templates` | Template file or directory (repeatable) |
| `-w`, `-workflows` | Multi-template workflow file |
| `-id` / `-eid` | Include / exclude by template id (CVE-… works) |
| `-tags` / `-etags` / `-itags` | Include / exclude / force-include by tag |
| `-s`, `-severity` | info, low, medium, high, critical, unknown |
| `-es`, `-exclude-severity` | Exclude severities |
| `-pt`, `-type` | Protocol: http, dns, tcp, ssl, websocket, whois, file, headless, workflow |
| `-et`, `-exclude-templates` | Exclude template files/dirs |
| `-as`, `-automatic-scan` | Wappalyzer tech detection → relevant templates |
| `-nt`, `-new-templates` | Only templates added in the latest release |
| `-headless` | Enable templates needing a headless browser (SPAs) |
| `-json-export` / `-jsonl-export` / `-sarif-export` | Machine-readable output |
| `-markdown-export` | Directory of per-finding Markdown reports |
| `-rate-limit` | Max requests/sec (default 150) |
| `-c`, `-concurrency` | Parallel templates (default 25) |
| `-bs`, `-bulk-size` | Hosts in parallel per template (default 25) |
| `-H` | Extra request header (repeatable); use for auth |
| `-p`, `-proxy` | Route through a proxy |
| `-eh`, `-exclude-hosts` | Exclude IPs/CIDR/hosts from input |
| `-sa`, `-scan-all-ips` | Scan all IPs behind a domain |
| `-resume` / `-stream` | Resume interrupted / stream large inputs |
| `-validate` / `-tl` / `-tgl` | Validate templates / list templates / list tags |
| `-up` / `-ut` | Update engine / update templates |
| `-duc`, `-disable-update-check` | Skip the update check (offline/CI) |

## Common Rationalizations

| Rationalization | Reality |
|---|---|
| "The template matched, so the app is vulnerable" | A template match is a lead. Info-severity exposure/version banners are mostly noise; confirm reachability and sensitivity before reporting. |
| "Default all-templates scan is enough" | It's noisy and slow. Narrow by severity, tag, or CVE for the signal you actually need. |
| "Nuclei can scan my code" | It can't. It needs a running target. For source review use `skillgrid:owasp-security`; for deep app analysis use `skillgrid:akca` or `skillgrid:wapiti`. |
| "I'll read the terminal output" | Scroll past it. Export `-json-export` and triage the records so no finding is missed and each is attributable. |
| "Just crank the concurrency to go faster" | You'll flood the target and trigger rate limits/WAFs. Set `-rate-limit` before raising concurrency on anything but a load-balanced test env. |

## Red Flags

- Scanning a production target with default concurrency (150 rps, 25 parallel) and no rate limit.
- Reporting an info/low "exposure" or version banner as a confirmed vulnerability.
- Triaging from scrolling terminal output instead of the exported JSON/SARIF.
- Running all templates against a small, known stack instead of `-tags`/`-id`/`-t`.
- A scan result presented without a severity + reachability + confidence line per finding.
- First run treated as done without confirming `nuclei -health-check` passed.

## Verification

- [ ] `nuclei -version` and `nuclei -health-check` both succeeded before scanning.
- [ ] The target is a reachable, authorized URL/host (prefer a test environment).
- [ ] The scan used an explicit net (severity/tag/template/id) matching the intent, not unscoped "all".
- [ ] Output was exported machine-readable (`-json-export` or `-sarif-export`) and read for triage.
- [ ] Every reported finding states severity, the concrete match, reachability, blast radius, and confidence.
- [ ] No info/low exposure or version banner is reported as a confirmed vulnerability without a demonstrated path.

## References

- [projectdiscovery/nuclei](https://github.com/projectdiscovery/nuclei) — engine source and full flag reference.
- [projectdiscovery/nuclei-templates](https://github.com/projectdiscovery/nuclei-templates) — the template library (CVEs, exposures, technologies, workflows).
