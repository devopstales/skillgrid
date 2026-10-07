# Mnemonic scan findings + dependency graph
#
# Source: .skillgrid/specs/2026-10-05-mnemonic-scan-findings/
# Trace: change briefing ## Definition of Done; blueprint tasks SATISFIES lines.
#
# Format rules (acceptance-test-authoring):
# - Steps in the ```gherkin fence are indented 6 spaces (verbatim — the extractor
#   does NOT re-indent)
# - No Gherkin tags in the spec — selection/trace is via the SATISFIES field in
#   blueprint/tasks
# - One blank line between the heading and the fence is fine
# - Gates: each requirement carries a `#### Gates` block — the runnable shadow of
#   its happy-path and failure scenarios. Author it BEFORE implementing.

Feature: Mnemonic scan findings + dependency graph
  Scanner output (trivy/vuln+sbom, wapiti, nuclei, semgrep) is stored durably in
  the per-project SQLite store: findings with a stable dedup_hash, a dependency
  graph upserted by purl, and the declared-vs-imported runtime import tree.
  A scan or ingest error never fails the calling session (ADR-0016 floors).
  Raw artifacts live out of repo under the mnemonic data dir. No scan TTL.

  Background:
    Given a mnemonic store is open in a temp data dir
    And the code index has the fixture files indexed

  # --- Req 1: schema ---

  Scenario: happy path scan migration applies idempotently
    Given migration 050 has been applied
    When the store is reopened and migrations re-run
    Then tables scans, findings, dependencies, and dep_edges exist
    And findings_fts is a virtual table
    And no existing table has altered columns

  # --- Req 2: ingest service (trivy tracer) ---

  Scenario: happy path trivy scan ingests findings with stable hash
    Given a trivy artifact with one known CVE "CVE-2024-1234" for package "flask"
    When I start a trivy scan on target "/repo"
    And I store the findings for the scan
    Then a scans row exists with status "ok"
    And the raw artifact is written under ".skillgrid/cache/scans/"
    And exactly one findings row exists
    And the findings row dedup_hash equals the sha256 of tool, cve, pkg, version, file, line
    And the findings row severity is normalized

  Scenario: error path scanner failure records status=error and is fail-open
    Given the trivy binary is missing or errors
    When I start a trivy scan on target "/repo"
    Then the start returns without failing the session
    And a scans row exists with status "error"
    And the scans row error field is non-empty
    And no findings rows exist for the scan

  # --- Req 3: diff ---

  Scenario: happy path unchanged re-scan diffs to zero added
    Given a trivy scan id1 has ingested the fixture artifact
    And a second trivy scan id2 has ingested the same artifact
    When I diff id1 against id2
    Then the diff added set is empty
    And the diff removed set is empty

  Scenario: happy path fixed cve appears as removed in diff
    Given a trivy scan id1 ingested a finding for "CVE-2024-1234"
    And a second trivy scan id2 ingested the artifact without that CVE
    When I diff id1 against id2
    Then the diff removed set contains the dedup_hash for "CVE-2024-1234"
    And the original findings row for id1 is not deleted

  # --- Req 4: dependency graph ---

  Scenario: happy path dep ingest upserts by purl and soft-retires absent
    Given an SBOM with package purl "pkg:pypi/flask@3.0.2"
    When I ingest the SBOM
    Then a dependencies row exists for the purl with retired 0
    When I ingest a second SBOM without that purl
    Then the dependencies row for the purl still exists
    And the dependencies row retired is 1
    And the dependencies row last_seen is preserved

  Scenario: happy path dep_affected returns reverse dependency set
    Given a dependency graph app -> flask -> werkzeug
    When I query affected for the werkzeug purl
    Then the result contains the flask purl
    And the result contains the app purl

  Scenario: happy path dep_runtime flags declared-only and import-only
    Given source file "app/main.py" imports flask and os
    And the manifest declares flask and werkzeug
    When I query runtime for "app/main.py"
    Then flask is in the shared set
    And werkzeug is in the declared-only set
    And os is in the import-only set
    And no new code index extraction pass ran

  # --- Req 5: install (semgrep) ---

  Scenario: happy path semgrep installs via uv
    Given semgrep is absent
    When I run the security-tools install
    Then semgrep is present on PATH
    And it was installed with manager "uv"

  # --- Req 6: skills ---

  Scenario: happy path semgrep skill exists and documents mnemonic store step
    Given the skill registry
    When I list the semgrep skill
    Then a semgrep SKILL.md exists under verification/
    And it documents storing results in mnemonic

  # --- Req 7: MCP registration ---

  Scenario: happy path scan and dep tools are registered
    Given the mnemonic MCP server is booted
    When I list the registered tools
    Then the scan tools scan_start, scan_store_findings, scan_list, scan_get, scan_status, and scan_diff are present
    And the dep tools dep_ingest, dep_list, dep_get, dep_affected, dep_graph, and dep_runtime are present

  # --- Req 8: no behavior change for existing tools ---

  Scenario: happy path existing tool surface is unchanged
    Given the mnemonic MCP server is booted before this change
    When I list the registered tools after this change
    Then every previously registered tool is still present
    And no pre-existing tool has a changed name or signature
