
- validate test overage at least 80%
- run tests
- code quality test
- security test
  - tivy

## Parallel Review Agents

Four agents run simultaneously, each with a different focus:

- Architecture Compliance: Audit changes against architecture documentation, flag violations with exact rule citations
- Bug Detection: Scan the diff for logic errors, null handling issues, and edge cases
- Security Review: Check for vulnerabilities, injection risks, and unsafe patterns in changed code
- E2E Test: Run an end-to-end test that exercises the new feature from the user's perspective
