---
name: trivy
description: Trivy comprehensive security scanner — vulnerabilities, misconfigurations, secrets, and SBOM generation for filesystem, containers, IaC, and repos.
---

# Trivy Security Skill

## When to Activate

- After adding or updating dependencies
- Before deploying containers or infrastructure
- Reviewing Dockerfiles, Terraform, or K8s manifests
- Scanning for exposed secrets or credentials
- Generating SBOMs for compliance
- CI/CD security gates

## When Used From Skillgrid

Trivy is an automated evidence source for Skillgrid gates:

- Use it during `/skillgrid-test` for dependency, filesystem, container, IaC, or secret baselines.
- Escalate unresolved Critical or High findings to `/skillgrid-security` or `/skillgrid-validate`.
- Store long scan summaries under `.skillgrid/tasks/research/<change-id>/` when they exceed a short handoff note.

## Core Capabilities

### 1. Vulnerability Scanning

Detect known CVEs in:
- Application dependencies (npm, pip, Maven, Go, etc.)
- Container images
- OS packages
- Language-specific packages

### 2. Misconfiguration Detection

Find security issues in IaC:
- Dockerfile best practices
- Kubernetes security contexts
- Terraform cloud provider configs
- CloudFormation templates
- Helm charts

### 3. Secret Detection

Identify exposed credentials:
- API keys
- Tokens
- Passwords
- Private keys
- AWS credentials
- Database connection strings

### 4. SBOM Generation

Generate Software Bill of Materials:
- SPDX format
- CycloneDX format
- Dependency inventory for compliance

## Scan Workflow

### Filesystem Scan (default)
```bash
trivy fs --severity CRITICAL,HIGH --exit-code 1 .
```

### With JSON Report
```bash
trivy fs \
  --severity CRITICAL,HIGH \
  --format json \
  --output trivy-report.json \
  .
```

### Container Image Scan
```bash
trivy image --severity CRITICAL,HIGH <image-name>
```

### IaC Config Scan
```bash
trivy config --severity CRITICAL,HIGH <path>
```

### Git Repository Scan
```bash
trivy repo --severity CRITICAL,HIGH <url>
```

### SBOM Generation
```bash
trivy fs --format spdx-json --output sbom.json .
```

## Interpreting Results

### Severity Levels

| Level | Action |
|---|---|
| CRITICAL | Must fix before deploy |
| HIGH | Should fix before deploy |
| MEDIUM | Fix recommended |
| LOW | Optional |

### Vulnerability Types

| Type | Description |
|---|---|
| CVE | Known vulnerability in a package |
| Misconfiguration | Insecure IaC setup |
| Secret | Exposed credential or key |
| License | Non-compliant license detected |

## False Positives

Suppress with `.trivyignore`:
```
# Ignore specific CVE
CVE-2023-12345

# Ignore specific package
npm:lodash
```

## CI/CD Integration

```yaml
# GitHub Actions
- name: Trivy Scan
  run: |
    trivy fs --severity CRITICAL,HIGH --exit-code 1 .
```

## Docker MCP Server

Enable the Trivy MCP server for ongoing monitoring:
```bash
trivy plugin install mcp
```

Config in MCP:
```json
{
  "trivy-command": {
    "command": "trivy",
    "args": ["mcp"]
  }
}
```

## Store results in mnemonic

After a successful scan, persist the findings so they survive the session and become queryable memory. Use the mnemonic MCP tools (fail-open: if the store is unavailable, log and continue — a scan error never blocks the caller):

1. `scan_start` — open a scan row for `tool: "trivy"`, the target, and the scanners used (e.g. `vuln,sbom`). It returns the `scan_id` and the raw-artifact path.
2. Write the scan's `--format json --output trivy-report.json` output to the raw-artifact path.
3. `scan_store_findings` — pass the `scan_id` and `tool: "trivy"`; the parser normalizes each `Vulnerabilities[]` record (CVE, package, version, fixed-version, severity, file) into a stable `dedup_hash` per finding.
4. `dep_ingest` — when the scan produced an SBOM (`--format spdx-json` / `cyclonedx-json`), call `dep_ingest` with the SBOM so the dependency graph upserts by `purl` and soft-retires absent purls.

```
trivy fs --format json --output /tmp/trivy-report.json .
mnemonic: scan_start(tool="trivy", target="/repo", scanners="vuln,sbom")  -> scan_id, raw_path
mnemonic: scan_store_findings(scan_id, tool="trivy")                       # parses raw, upserts findings
mnemonic: dep_ingest(scan_id, sbom=/tmp/sbom.json)                         # when an SBOM was produced
```

## Best Practices

1. **Scan dependencies** — after every `npm install`, `pip install`, etc.
2. **Scan containers** — before pushing to registry
3. **Scan IaC** — before applying Terraform or deploying K8s
4. **Generate SBOMs** — for every release artifact
5. **Block on CRITICAL** — fail CI for critical vulnerabilities
6. **Rotate secrets** — if trivy detects exposed credentials
7. **Keep updated** — trivy's vulnerability database updates daily

## Common Scan Targets

| Target | Command |
|---|---|
| Project dependencies | `trivy fs .` |
| Dockerfile | `trivy config Dockerfile` |
| K8s manifests | `trivy config k8s/` |
| Terraform | `trivy config terraform/` |
| Container image | `trivy image myapp:latest` |
| Git repo | `trivy repo https://github.com/...` |
