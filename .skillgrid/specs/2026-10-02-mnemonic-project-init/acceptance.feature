# Project Init
#
# Source: .skillgrid/specs/2026-10-02-mnemonic-project-init/
# Trace: briefing requirements 1–6.

## Requirements

### Requirement: Init command

The system SHALL expose `skillgrid init` that sequences boot-file write, code index, and doc ingest, and SHALL print a result with written / indexed / ingested / skipped / errors.

#### Scenario: happy path init writes boot file and reports counts

```gherkin
      Given a temporary project directory with a README
      When the operator runs skillgrid init
      Then the command exits 0
      And the result reports a boot file path
      And the result reports indexed and ingested counts
```

#### Scenario: init help lists force and docs flags

```gherkin
      Given the skillgrid binary
      When the operator runs skillgrid init -h
      Then the usage names --force
      And the usage names --docs
```

#### Scenario: boot file write failure is the only fatal error

```gherkin
      Given a project directory whose boot file cannot be written
      When the operator runs skillgrid init
      Then the command exits non-zero
      And no success result is printed
```

#### Gates
# G1: happy path init writes boot file and reports counts
#   CHECK: go test ./skillgrid-cli/cmd/skillgrid -count=1 -run TestInitWritesBootFileAndReportsCounts
#   EXPECT: PASS TestInitWritesBootFileAndReportsCounts
#   EVIDENCE: pending
# G2: boot file write failure is the only fatal error
#   CHECK: go test ./skillgrid-cli/cmd/skillgrid -count=1 -run TestInitBootFileWriteFailureIsFatal
#   EXPECT: PASS TestInitBootFileWriteFailureIsFatal
#   EVIDENCE: pending

### Requirement: Boot file

The system SHALL write a short working-agreement preamble above the Skillgrid sentinel and SHALL upsert the sentinel in place so a second run never duplicates it.

#### Scenario: happy path init upserts preamble and sentinel

```gherkin
      Given a temporary project with no AGENTS.md
      When the operator runs skillgrid init
      Then AGENTS.md exists
      And AGENTS.md contains a working-agreement preamble
      And AGENTS.md contains exactly one Skillgrid sentinel pair
```

#### Scenario: second init merges without duplicating the sentinel

```gherkin
      Given a project already initialized
      When the operator runs skillgrid init without --force
      Then AGENTS.md still contains exactly one Skillgrid sentinel pair
      And user text outside the preamble and sentinel is unchanged
```

#### Scenario: force rewrites the preamble only

```gherkin
      Given a project already initialized with a customized preamble
      When the operator runs skillgrid init --force
      Then the preamble is replaced
      And the sentinel is still a single pair
```

#### Gates
# G3: happy path init upserts preamble and sentinel
#   CHECK: go test ./skillgrid-cli/cmd/skillgrid -count=1 -run TestInitUpsertsPreambleAndSentinel
#   EXPECT: PASS TestInitUpsertsPreambleAndSentinel
#   EVIDENCE: pending
# G4: force rewrites the preamble only
#   CHECK: go test ./skillgrid-cli/cmd/skillgrid -count=1 -run TestInitForceRewritesPreamble
#   EXPECT: PASS TestInitForceRewritesPreamble
#   EVIDENCE: pending

### Requirement: Forced code index

The system SHALL run the existing code index pipeline during init so a project with at least one indexable file has a non-empty index.

#### Scenario: happy path init indexes the project

```gherkin
      Given a temporary project with an indexable source file
      When the operator runs skillgrid init
      Then the project index lists at least one file
```

#### Scenario: init reuses the existing indexer

```gherkin
      Given the project init command
      When the operator inspects how init indexes
      Then init uses the same index pipeline as skillgrid index
      And it does not open a second store
```

#### Scenario: index failure after boot file still exits 0

```gherkin
      Given a project whose boot file can be written
      And the index pipeline will fail
      When the operator runs skillgrid init
      Then the command exits 0
      And the result lists the index failure under errors
      And the boot file exists
```

#### Gates
# G5: happy path init indexes the project
#   CHECK: go test ./skillgrid-cli/cmd/skillgrid -count=1 -run TestInitIndexesTheProject
#   EXPECT: PASS TestInitIndexesTheProject
#   EVIDENCE: pending
# G6: index failure after boot file still exits 0
#   CHECK: go test ./skillgrid-cli/cmd/skillgrid -count=1 -run TestInitIndexFailureIsNonFatal
#   EXPECT: PASS TestInitIndexFailureIsNonFatal
#   EVIDENCE: pending

### Requirement: Default ingest

The system SHALL upsert observations for each existing default path under topic key `init/docs/<relpath>` and SHALL skip missing defaults without failing.

#### Scenario: happy path init ingests default paths

```gherkin
      Given a temporary project with README.md and one file under docs
      When the operator runs skillgrid init
      Then an observation exists with topic key init/docs/README.md
      And an observation exists with topic key init/docs/docs/<that-file>
```

#### Scenario: missing docs directory is skipped

```gherkin
      Given a temporary project with README.md and no docs directory
      When the operator runs skillgrid init
      Then the command exits 0
      And the result lists docs/ under skipped
      And an observation exists with topic key init/docs/README.md
```

#### Scenario: second init upserts the same topic keys

```gherkin
      Given a project already ingested
      When the operator runs skillgrid init again
      Then no second observation is created for init/docs/README.md
      And the existing observation is updated
```

#### Gates
# G7: happy path init ingests default paths
#   CHECK: go test ./skillgrid-cli/cmd/skillgrid -count=1 -run TestInitIngestsDefaultPaths
#   EXPECT: PASS TestInitIngestsDefaultPaths
#   EVIDENCE: pending
# G8: missing docs directory is skipped
#   CHECK: go test ./skillgrid-cli/cmd/skillgrid -count=1 -run TestInitSkipsMissingDocs
#   EXPECT: PASS TestInitSkipsMissingDocs
#   EVIDENCE: pending

### Requirement: Extra docs flag

The system SHALL ingest each existing `--docs` path the same way as defaults and SHALL list a missing extra path as a non-fatal error.

#### Scenario: happy path init ingests extra --docs path

```gherkin
      Given a temporary project with extra.md outside the defaults
      When the operator runs skillgrid init --docs extra.md
      Then an observation exists with topic key init/docs/extra.md
```

#### Scenario: repeated --docs flags ingest each path

```gherkin
      Given a temporary project with a.md and b.md
      When the operator runs skillgrid init --docs a.md --docs b.md
      Then observations exist for init/docs/a.md and init/docs/b.md
```

#### Scenario: missing extra docs path is listed and non-fatal

```gherkin
      Given a temporary project whose boot file can be written
      When the operator runs skillgrid init --docs missing.md
      Then the command exits 0
      And the result lists missing.md under errors
      And the boot file exists
```

#### Gates
# G9: happy path init ingests extra --docs path
#   CHECK: go test ./skillgrid-cli/cmd/skillgrid -count=1 -run TestInitIngestsExtraDocs
#   EXPECT: PASS TestInitIngestsExtraDocs
#   EVIDENCE: pending
# G10: missing extra docs path is listed and non-fatal
#   CHECK: go test ./skillgrid-cli/cmd/skillgrid -count=1 -run TestInitMissingExtraDocsIsNonFatal
#   EXPECT: PASS TestInitMissingExtraDocsIsNonFatal
#   EVIDENCE: pending

### Requirement: Skill hands off to CLI

The onboarding skill SHALL finish by running `skillgrid init` and SHALL NOT re-specify an ingest procedure.

#### Scenario: happy path onboarding skill calls skillgrid init

```gherkin
      Given the onboarding skill
      When a reader looks at the write-and-verify steps
      Then the skill names skillgrid init as the deterministic finish
```

#### Scenario: skill passes extra docs through

```gherkin
      Given the user named extra documentation paths in the interview
      When the skill runs skillgrid init
      Then it passes those paths as --docs flags
```

#### Scenario: skill does not contain a second ingest procedure

```gherkin
      Given the onboarding skill
      When a reader searches for a file-walk ingest contract
      Then the only ingest procedure is the skillgrid init command
```

#### Gates
# G11: happy path onboarding skill calls skillgrid init
#   CHECK: go test ./skillgrid-cli/cmd/skillgrid -count=1 -run TestOnboardingSkillCallsSkillgridInit
#   EXPECT: PASS TestOnboardingSkillCallsSkillgridInit
#   EVIDENCE: pending
# G12: skill does not contain a second ingest procedure
#   CHECK: go test ./skillgrid-cli/cmd/skillgrid -count=1 -run TestOnboardingSkillHasNoSecondIngest
#   EXPECT: PASS TestOnboardingSkillHasNoSecondIngest
#   EVIDENCE: pending
