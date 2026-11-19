# Skillgrid router auto-load at session start

> Companion to `blueprint.md` (same directory). Scenarios are the blueprint
> tasks' `SATISFIES:` targets; `# CHECK:` / `# EXPECT:` lines are the runnable
> gates, `# EVIDENCE:` lines are filled at execution/QA (repo pattern).

### Requirement: R1 — One shared bootstrap asset, read live

MUST: a single shared bootstrap asset exists in the skill tree and is the one
source every host injector reads at session start. MUST: a change to the asset
reaches the next session start of every pointer host without reinstall. SHALL
contain: the skillgrid-project notice, the first-action line (read the
using-skillgrid router), the one-line request-class list, the announce line,
and the per-host load-hint variants table. SHALL NOT contain project-specific
content.

#### Scenario: happy path — a one-line edit to the bootstrap asset reaches the next session without reinstall

```gherkin
Given a skillgrid project with the router bootstrap installed
When I edit one line of the bootstrap asset
And I start a fresh cursor session in the project
Then the new session's first-turn context carries the edited line
# CHECK: marker-edit the asset, fresh cursor session, grep the transcript first-turn context
# EXPECT: edited marker line present in the first-turn context
```

#### Scenario: edge — all four pointer hosts read the identical live asset

```gherkin
Given a marker line added to the bootstrap asset once
When I start fresh sessions of pi, opencode, kilo, and cursor in the project
Then each host's first-turn context carries the identical edited bootstrap content
# CHECK: fresh session per host; diff the bootstrap block extracted from each transcript
# EXPECT: four identical blocks, zero diff
```

#### Scenario: failure — the bootstrap asset is missing at session start

```gherkin
Given a skillgrid project whose bootstrap asset is not resolvable
When I start a fresh cursor session in the project
Then the first-turn context carries the static fallback router line
And session start is not blocked
# CHECK: move the asset aside, fresh cursor session, inspect transcript + session log
# EXPECT: fallback line present, session started, no error surfaced
```

### Requirement: R2 — Per-host session-start injection for all five hosts

MUST: every supported host (omp, pi, opencode, Kilo Code, Cursor) receives the
router at session start through its native mechanism. SHALL: on the four
pointer hosts the injected content is the shared bootstrap asset; on omp the
shipped full-content hook carries the full router. SHALL: the injection be
present in the first-turn context and the router be loaded before the first
task action (the agent announces `Using skillgrid:using-skillgrid to …` or
makes its first tool call a read of the router).

#### Scenario: happy path — a cursor session start injects the router bootstrap even with the memory server down

```gherkin
Given a skillgrid project and the memory server stopped
When I start a fresh cursor session in the project
Then the session receives the router bootstrap as session-start additional context
And the first assistant turn announces the router or reads it as its first tool call
# CHECK: stop the mnemonic server, fresh cursor session, inspect first-turn context + first assistant turn
# EXPECT: bootstrap present in additional context; announce or router-read on turn 1
```

#### Scenario: happy path — an omp session start injects the full router content

```gherkin
Given a skillgrid project with the omp agent installed
When I start a fresh omp session in the project
Then the first-turn context carries the full router content
And the first assistant turn announces the router or reads it as its first tool call
# CHECK: fresh omp session, inspect transcript first-turn context + first assistant turn
# EXPECT: full router present (not the short pointer bootstrap); announce or router-read on turn 1
```

#### Scenario: happy path — a pi session start injects the router bootstrap

```gherkin
Given a trusted skillgrid project with the committed pi router extension
When I start a fresh pi session in the project
Then the first-turn system context carries the router bootstrap
And the first assistant turn announces the router or reads it as its first tool call
# CHECK: fresh pi session, inspect transcript first-turn system context + first assistant turn
# EXPECT: bootstrap present in system context; announce or router-read on turn 1
```

#### Scenario: happy path — an opencode session start injects the router bootstrap

```gherkin
Given a skillgrid project
When I start a fresh opencode session in the project
Then the first-turn context carries the router bootstrap entry
And the first assistant turn announces the router or reads it as its first tool call
# CHECK: fresh opencode session, inspect transcript first-turn context + first assistant turn
# EXPECT: bootstrap entry present; announce or router-read on turn 1
```

#### Scenario: happy path — a kilo session start injects the router bootstrap

```gherkin
Given a skillgrid project
When I start a fresh kilo session in the project
Then the first user message carries the router bootstrap
And the kilo skills configuration includes the project skills directory
And the first assistant turn announces the router or reads it as its first tool call
# CHECK: fresh kilo session, inspect first user message + skills config + first assistant turn
# EXPECT: bootstrap in first user message; skills path added; announce or router-read on turn 1
```

#### Scenario: edge — re-running the installer does not duplicate the injection

```gherkin
Given a host already receiving the router bootstrap at session start
When I re-run the installer for that host
And I start a fresh session
Then the first-turn context carries exactly one copy of the router bootstrap
# CHECK: re-run installer for the host, fresh session, count bootstrap blocks in the first-turn context
# EXPECT: exactly one block
```

#### Scenario: failure — the cursor session-start hook is absent

```gherkin
Given a cursor session started without the session-start hook configured
When the session begins in a skillgrid project
Then no router bootstrap appears in the first-turn context
And the session starts normally
# CHECK: disable the hook for one session, fresh cursor session, inspect transcript + session log
# EXPECT: no bootstrap, session functional, no error surfaced
```

### Requirement: R3 — Project-conditional, main agent only

MUST: injection is active only where a skillgrid project is detected; the
agent receives nothing in non-skillgrid directories. MUST: only the main agent
receives the bootstrap; subagent sessions skip it. SHALL: an undetectable or
unreadable project configuration be treated as no project (fail-open, no block).

#### Scenario: happy path — a skillgrid project session receives the bootstrap

```gherkin
Given a skillgrid project
When I start a fresh cursor session from the project directory
Then the first-turn context carries the router bootstrap
# CHECK: fresh cursor session from the project root, inspect first-turn context
# EXPECT: bootstrap present
```

#### Scenario: edge — a non-skillgrid directory session receives nothing

```gherkin
Given a directory without a skillgrid configuration
When I start a fresh cursor session from that directory
Then the first-turn context carries no router bootstrap
# CHECK: fresh cursor session from a scratch directory, inspect first-turn context
# EXPECT: no bootstrap content
```

#### Scenario: edge — a subagent session skips the bootstrap

```gherkin
Given a skillgrid project
When the main agent spawns a subagent session
Then the subagent's context carries no router bootstrap
# CHECK: from a fresh main session, spawn a subagent; inspect the subagent's context
# EXPECT: main context has the bootstrap, subagent context does not
# EVIDENCE: subagent-skip probe result (Task 2)
```

#### Scenario: failure — an unreadable skillgrid configuration is treated as no project

```gherkin
Given a project directory whose skillgrid configuration file is unreadable
When I start a fresh cursor session from that directory
Then the first-turn context carries no router bootstrap
And session start is not blocked
# CHECK: chmod the config unreadable, fresh cursor session, inspect transcript + session log
# EXPECT: no bootstrap, session started, no error surfaced
```

### Requirement: R4 — Fail-open with a static fallback

MUST: an injector failure never blocks session start and never surfaces an
error to the user. MUST: when the bootstrap asset is not resolvable, the
injectors use a static fallback that still routes the agent to the router.
SHALL: the installed bootstrap copy take precedence over the checkout asset.

#### Scenario: failure — a session-start hook failure never blocks session start

```gherkin
Given a cursor session whose session-start hook fails before completing
When the session begins
Then the session starts normally
And no injection error is visible to the user
# CHECK: make the hook fail (e.g. point it at a missing file) for one session, inspect session log + transcript
# EXPECT: session functional, no error surfaced
```

#### Scenario: failure — neither the installed copy nor the checkout asset is resolvable

```gherkin
Given a skillgrid checkout whose bootstrap asset is absent
And no installed bootstrap copy present
When I start a fresh cursor session in the checkout
Then the first-turn context carries the static fallback router line
And session start is not blocked
# CHECK: remove the installed copy, move the checkout asset aside, fresh cursor session, inspect transcript
# EXPECT: static fallback present, session started
```

#### Scenario: edge — the installed bootstrap copy takes precedence over the checkout asset

```gherkin
Given an installed bootstrap copy carrying a marker line
And a checkout bootstrap asset carrying a different marker line
When I start a fresh cursor session in the checkout
Then the first-turn context carries the installed copy's marker line
# CHECK: marker both files differently, fresh cursor session, inspect first-turn context
# EXPECT: installed-copy marker present, checkout marker absent
```

### Requirement: R5 — Distribution through the existing installer

MUST: the installer distributes, per host key, exactly the files that host
loads plus the bootstrap copy at the installer-managed location. MUST: an
unknown agent key errors. MUST: `sync-repo` refresh the installed bootstrap
copy from the repo clone. SHALL: a re-run with unchanged content leave the
installed copy unchanged.

#### Scenario: happy path — installing a host key distributes that host's injector files plus the bootstrap

```gherkin
Given a repo checkout containing the router bootstrap asset
When I run the installer for a pointer host key
Then that host's install location gains the host's injector files
And the installed bootstrap copy exists at the installer-managed location
# CHECK: dry-run/install the host key, diff the install location against the expected file set
# EXPECT: exactly the host's injector files + the bootstrap copy
```

#### Scenario: happy path — sync-repo refreshes the installed bootstrap from an edited clone asset

```gherkin
Given an installed bootstrap copy
When I edit the bootstrap asset in the repo clone
And I run sync-repo
Then the installed bootstrap copy carries the edited content
# CHECK: marker-edit the clone asset, run sync-repo, diff the installed copy
# EXPECT: installed copy matches the edited clone asset
```

#### Scenario: edge — re-running the installer with unchanged content leaves the installed copy unchanged

```gherkin
Given an installed bootstrap copy
When I re-run the installer with unchanged asset content
Then the installed bootstrap copy is unchanged
# CHECK: record the installed copy's mtime/hash, re-run installer, compare
# EXPECT: identical bytes, no rewrite
```

#### Scenario: failure — an unknown agent key errors

```gherkin
Given a repo checkout
When I run the installer with an agent key that is not a supported host
Then the installer reports an error naming the unknown key
And no host files are modified
# CHECK: run the installer with a bogus key, inspect exit code + message + install locations
# EXPECT: non-zero exit, key named in the message, install locations untouched
```

### Requirement: R6 — Per-host tool-mapping hint in the bootstrap

MUST: the bootstrap's load-hint variants carry a tool-mapping hint for every
host that already has a mapping reference file (pi, opencode) and a plain
first-action load for hosts without one (kilo, cursor, omp). MUST NOT create
new mapping reference files.

#### Scenario: happy path — hosts with a tool-mapping file get the mapping hint

```gherkin
Given the pi and opencode tool-mapping reference files exist
When I read the bootstrap asset's load-hint variants
Then the pi and opencode variants reference their existing tool-mapping files
# CHECK: inspect the bootstrap asset's load-hint table
# EXPECT: pi and opencode variants point at the existing mapping files
```

#### Scenario: edge — hosts without a tool-mapping file get the plain first-action hint

```gherkin
Given kilo, cursor, and omp have no tool-mapping reference file
When I read the bootstrap asset's load-hint variants
Then their variants carry no tool-mapping reference
And they still direct the first action to the router
# CHECK: inspect the bootstrap asset's load-hint table for the three variants
# EXPECT: no mapping reference, first-action load present
```

#### Scenario: failure — a tool-mapping reference file is missing at session start

```gherkin
Given the pi tool-mapping reference file deleted
When I start a fresh pi session in the project
Then the first-turn context still carries the router bootstrap
And the first action still loads the router
# CHECK: delete the mapping file, fresh pi session, inspect transcript
# EXPECT: bootstrap present, router loaded on first action, no error
```
