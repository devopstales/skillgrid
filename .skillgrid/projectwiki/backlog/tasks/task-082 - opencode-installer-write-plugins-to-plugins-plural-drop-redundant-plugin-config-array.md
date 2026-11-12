---
id: TASK-082
title: >-
  opencode installer: write plugins to plugins/ (plural) + drop redundant plugin
  config array
status: done
assignee: []
created_date: '2026-10-18 10:35'
updated_date: '2026-10-18 11:08'
labels: []
dependencies: []
priority: medium
type: bug
---

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->

## Comments

<!-- COMMENTS:BEGIN -->
created: 2026-10-18 11:08
---
RESOLVED (b8f719f3): installNativePlugins now copies the 5 plugins + skillgrid-v2sse.ts to ~/.config/<harness>/plugins/ (plural, auto-discovered) and every tools/*.ts to ~/.config/<harness>/tools/ (plural, auto-discovered), and no longer registers the redundant/mis-pathed './plugin/<name>.ts' config array. Added copyToolDir for the one-file-per-tool dir (missing source dir is not an error). Kilo shares the path so it's fixed too. upsertPluginKey retained (now unused) as a documenting helper. Tests updated (opencode_test.go + kilocode_test.go): assert plugins/ plural, tools/mem_save.ts installed, v2sse copied, NO './plugin/<base>.ts' array entries, dry-run writes neither. go build ./... + gofmt clean; go test ./internal/setup/ green.
---
<!-- COMMENTS:END -->
