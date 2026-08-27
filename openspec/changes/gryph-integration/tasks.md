# Tasks: Gryph Integration

## Epic 1: Install step implementation

- [ ] 1-1 Add `@safedep/gryph` entry to `config.d/tools.yaml`
- [ ] 1-2 Implement `installGryph(baseDir string, agents []string, dryRun bool)` in `skillgrid-cli/cmd/steps.go`
- [ ] 1-3 Add gated call to `installGryph` in `cmd/install.go` after the agent-browser block (step 4c)
- [ ] 1-4 Implement binary resolution: prefer `~/.skillgrid/npm/bin/gryph`, fall back to bare `gryph` on PATH
- [ ] 1-5 Implement OpenCode hooks: `gryph install --agent opencode`
- [ ] 1-6 Implement Kilo plugin copy: copy OpenCode plugin → Kilo plugins dir (first-write-wins, warn if source missing)
- [ ] 1-7 Implement policy sub-step: `gryph policy init`, `gryph policy validate`, `gryph config set policy.enabled true`
- [ ] 1-8 Implement dry-run: print all planned actions, perform zero execs/writes

## Epic 2: Testing

- [ ] 2-1 Create `cmd/gryph_test.go` with dry-run test (emits expected log lines, never reports real install)
- [ ] 2-2 Add real-path test: temp HOME + fake `gryph` script on PATH, verify exact argument sequences
- [ ] 2-3 Add Kilo copy source-missing test (logs WARN, no panic)
- [ ] 2-4 Add agent-selection gating test

## Epic 3: Documentation

- [ ] 3-1 Add Gryph section to `docs/02-usage.md`
- [ ] 3-2 Add tools.yaml semantics note to `docs/03-config-reference.md`
- [ ] 3-3 Mark gryph done in `docs/NOTE.md` Usage Data checklist

## Epic 4: Validation

- [ ] 4-1 Run `go build ./... && go test ./...` — all green
- [ ] 4-2 Run `openspec validate gryph-integration --type change --strict` before archive
- [ ] 4-3 Archive via `openspec archive gryph-integration`
