## Why

After `skillgrid install`, agents make tool calls with no audit trail. For security-sensitive or compliance use cases, operators need a record of what the agent did — which tools were called, when, and whether any violated policy. [Gryph](https://github.com/safedep/gryph) provides a local-only audit-tail and security-policy engine for AI coding agents. Integrating it into the skillgrid installer means every install gets auditing and policy enforcement by default.

## What Changes

- Add `installGryph` step to `skillgrid-cli/cmd/steps.go` — runs after npm install when `@safedep/gryph` is in `config.d/tools.yaml`
- Run `gryph install --agent opencode` for OpenCode hooks
- Copy OpenCode plugin → Kilo plugins dir (Kilo has no native gryph adapter)
- Run `gryph policy init`, `gryph policy validate`, `gryph config set policy.enabled true`
- Add `cmd/gryph_test.go` with dry-run and real-path tests
- Document Gryph in usage docs and config reference

## Capabilities

### New Capabilities

- `gryph-opencode-hooks`: Install gryph audit hooks into OpenCode via `gryph install --agent opencode`
- `gryph-kilo-plugin-copy`: Copy the OpenCode gryph plugin into Kilo's plugins directory (first-write-wins)
- `gryph-policy-setup`: Scaffold and enable gryph security policy (`policy init`, `validate`, `config set policy.enabled true`)
- `kilo-plugin-copy`: Generic mechanism to copy any plugin into Kilo's plugins directory
- `opencode-hooks`: Generic mechanism to install audit hooks into OpenCode
- `policy-setup`: Generic mechanism to scaffold and enable security policies

### Modified Capabilities

None — gryph integration is a new install-time capability.

## Impact

- **Affected code**: `skillgrid-cli/cmd/steps.go` (new `installGryph` function), `skillgrid-cli/cmd/install.go` (gated call), `config.d/tools.yaml` (gryph entry), `cmd/gryph_test.go` (new test file)
- **Affected systems**: OpenCode (`~/.config/opencode/plugins/gryph.js`), Kilo (`~/.config/kilo/plugins/gryph.js`), gryph SQLite store
- **New dependency**: `@safedep/gryph` npm package (binary downloaded via postinstall)
- **Users**: All skillgrid users who add `@safedep/gryph` to their tools.yaml get auditing + policy enforcement
