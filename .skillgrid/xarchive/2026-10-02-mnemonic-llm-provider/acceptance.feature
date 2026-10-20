# Shared LLM Provider
#
# Source: .skillgrid/specs/2026-10-02-mnemonic-llm-provider/
# Trace: briefing requirements 1–7.

## Requirements

### Requirement: LLM config

The system SHALL load an opt-in `mnemonic.llm` config block (enabled, base_url, model, api_key, timeout) defaulting to disabled.

#### Scenario: happy path llm config defaults off

```gherkin
      Given no mnemonic.llm section in config
      When the operator loads mnemonic config
      Then llm.enabled is false
      And no shared LLM client is attached
```

#### Scenario: enabled config requires base_url and model

```gherkin
      Given mnemonic.llm.enabled is true
      And base_url or model is empty
      When the operator starts the mnemonic service
      Then attach is refused with an actionable error
      And all LLM seams remain nil
```

#### Scenario: api key falls back to environment

```gherkin
      Given mnemonic.llm.enabled is true with base_url and model set
      And api_key is empty in config
      And SKILLGRID_LLM_API_KEY is set in the environment
      When the shared LLM client is built
      Then requests carry that bearer token
```

#### Gates
# G1: happy path llm config defaults off
#   CHECK: go test ./skillgrid-cli/internal/mnemonic/config -count=1 -run TestLLMConfigDefaultOff
#   EXPECT: PASS TestLLMConfigDefaultOff
#   EVIDENCE: pending
# G2: enabled config requires base_url and model
#   CHECK: go test ./skillgrid-cli/internal/mnemonic/config -count=1 -run TestLLMConfigRequiresURLAndModel
#   EXPECT: PASS TestLLMConfigRequiresURLAndModel
#   EVIDENCE: pending

### Requirement: OpenAI-compatible client

The system SHALL provide one stdlib HTTP Completer against `{base_url}/chat/completions` with no new LLM SDK dependency.

#### Scenario: happy path openai-compatible complete succeeds

```gherkin
      Given a mock OpenAI-compatible chat completions server
      And a Completer pointed at that server
      When Complete is called with a system and user prompt
      Then the returned string is the assistant message content
```

#### Scenario: non-2xx complete returns an error

```gherkin
      Given a mock server that responds 500
      When Complete is called
      Then an error is returned
      And no empty success string is treated as an answer
```

#### Scenario: timeout complete returns an error

```gherkin
      Given a mock server that exceeds the configured timeout
      When Complete is called
      Then an error is returned
```

#### Gates
# G3: happy path openai-compatible complete succeeds
#   CHECK: go test ./skillgrid-cli/internal/mnemonic/llm -count=1 -run TestCompleteSuccess
#   EXPECT: PASS TestCompleteSuccess
#   EVIDENCE: pending
# G4: non-2xx complete returns an error
#   CHECK: go test ./skillgrid-cli/internal/mnemonic/llm -count=1 -run TestCompleteHTTPError
#   EXPECT: PASS TestCompleteHTTPError
#   EVIDENCE: pending

### Requirement: Single attach wires all seams

The system SHALL attach one Completer to AskLLM, DedupLLM, ExtractionLLM, and Dream LLM from a single boot function.

#### Scenario: happy path attach wires all seams from one client

```gherkin
      Given a valid mnemonic.llm config and a Completer
      When AttachSharedLLM runs
      Then AskLLM, DedupLLM, ExtractionLLM, and Dream LLM seams are non-nil
      And invoking each path that calls Complete increments one shared counter
```

#### Scenario: disabled config clears all seams

```gherkin
      Given seams previously attached
      When AttachSharedLLM runs with llm.enabled false
      Then all four seams are nil
```

#### Scenario: no third-party LLM module in go.mod

```gherkin
      Given the skillgrid-cli module
      When go.mod is inspected for this change
      Then no new LLM SDK dependency was added
```

#### Gates
# G5: happy path attach wires all seams from one client
#   CHECK: go test ./skillgrid-cli/internal/mnemonic/service -count=1 -run TestAttachSharedLLMWiresAllSeams
#   EXPECT: PASS TestAttachSharedLLMWiresAllSeams
#   EVIDENCE: pending
# G6: disabled config clears all seams
#   CHECK: go test ./skillgrid-cli/internal/mnemonic/service -count=1 -run TestAttachSharedLLMDisabledClears
#   EXPECT: PASS TestAttachSharedLLMDisabledClears
#   EVIDENCE: pending

### Requirement: Feature flags stay opt-in

The system SHALL keep extraction.llm and dedup.llm and mem_ask llm-mode opt-in even when a client is attached.

#### Scenario: happy path attached client with flags off uses floors

```gherkin
      Given a Completer is attached
      And mnemonic.extraction.llm is false
      And mnemonic.dedup.llm is false
      When passive extraction and hash dedup run
      Then the Completer is not called
      And the deterministic floors produce results
```

#### Scenario: ask cited mode ignores attached client

```gherkin
      Given a Completer is attached
      When mem_ask runs with mode cited
      Then the Completer is not called
      And citations are returned from the store
```

#### Scenario: flag on with attached client invokes Complete

```gherkin
      Given a Completer is attached
      And mnemonic.dedup.llm is true
      When a save triggers semantic dedup
      Then Complete is called
```

#### Gates
# G7: happy path attached client with flags off uses floors
#   CHECK: go test ./skillgrid-cli/internal/mnemonic/... -count=1 -run TestAttachedClientFlagsOffUsesFloors
#   EXPECT: PASS TestAttachedClientFlagsOffUsesFloors
#   EVIDENCE: pending

### Requirement: Fail-open floors

The system SHALL fall back to deterministic floors when Complete errors.

#### Scenario: happy path llm error fails open to floors

```gherkin
      Given a Completer that returns an error
      And llm feature flags are on
      When mem_ask mode llm, dedup classify, and consolidate run
      Then each path returns a successful floor result
      And the process does not panic
```

#### Scenario: ask llm mode degrades to cited

```gherkin
      Given a Completer that returns an error
      When mem_ask runs with mode llm
      Then the response contains citations
      And degraded or equivalent floor signaling is present
```

#### Scenario: dedup llm error uses hash floor

```gherkin
      Given a Completer that returns an error
      And mnemonic.dedup.llm is true
      When SaveWithAction runs on a hash-duplicate
      Then the action is noop or the hash floor outcome
```

#### Gates
# G8: happy path llm error fails open to floors
#   CHECK: go test ./skillgrid-cli/internal/mnemonic/... -count=1 -run TestLLMErrorFailsOpen
#   EXPECT: PASS TestLLMErrorFailsOpen
#   EVIDENCE: pending

### Requirement: Task-029 closed by this change

The system SHALL retire the FOLLOWUP task-029 debt by implementing the shared client attach described there.

#### Scenario: happy path task-029 superseded by this change

```gherkin
      Given backlog task-029
      When this change is the active LLM-provider spec
      Then task-029 references 2026-10-02-mnemonic-llm-provider
      And its description matches the shared-attach scope
```

#### Scenario: task-029 is not a separate parallel implementation

```gherkin
      Given the llm-provider briefing
      When a reader looks for a second production client design
      Then only this topic owns the shared Completer
```

#### Scenario: closing commit references task-029

```gherkin
      Given the change ships
      When the closing commit or ticket update runs
      Then task-029 is marked done or superseded
```

#### Gates
# G9: happy path task-029 superseded by this change
#   CHECK: rg -n "2026-10-02-mnemonic-llm-provider" ".backlog/tasks/task-029"*
#   EXPECT: match
#   EVIDENCE: pending

### Requirement: Natural provider setup on install

The system SHALL run embedding/LLM provider setup as a normal `skillgrid install` step (Local Ollama ensure + wire, or External OpenAI-compatible wire), because code indexing already needs Ollama or an external embedder. Re-run is ensure (natural update), not a separate opt-in bolt-on.

#### Scenario: happy path install yes ensures local ollama and wires config

```gherkin
      Given skillgrid install with --yes (provider defaults to local)
      And ollama is not on PATH
      And a stubbed OS install + probe succeeds
      When install runs the provider-setup step
      Then Ollama is installed for the host OS
      And the service is probed and started if needed
      And llama3.2:3b and nomic-embed-text are pulled only if absent
      And ~/.skillgrid home config enables mnemonic.llm against http://localhost:11434/v1
      And embedder.provider is ollama with model nomic-embed-text
```

#### Scenario: install provider external wires without ollama

```gherkin
      Given skillgrid install with --provider=external
      And base_url and api_key are available (flag or env)
      When install runs the provider-setup step
      Then no ollama install, start, or pull commands run
      And home config enables mnemonic.llm against that base_url
      And embedder.provider is external against the same host
```

#### Scenario: skip-provider leaves ensure out

```gherkin
      Given skillgrid install with --skip-provider
      When install runs
      Then no provider binary, pull, or home llm/embedder merge runs from this step
```

#### Scenario: ensure skips pull when models already present

```gherkin
      Given provider=local
      And ollama is on PATH and /api/tags lists llama3.2:3b and nomic-embed-text
      When ensure runs
      Then no ollama pull is invoked
      And home llm + embedder config is still merged/ensured
```

#### Scenario: dry-run provider setup writes nothing

```gherkin
      Given --yes and --dry-run
      When install runs
      Then intended Local provider steps are logged
      And no binary install, serve, pull, or config write occurs
```

#### Scenario: provider setup failure is non-fatal

```gherkin
      Given provider=local
      And the OS install or start probe fails
      When install runs
      Then a warning with a manual hint is printed
      And install still exits 0 if earlier steps succeeded
```

#### Gates
# G10: happy path install yes ensures local ollama and wires config
#   CHECK: go test ./skillgrid-cli/internal/install -count=1 -run TestSetupProviderLocal
#   EXPECT: PASS TestSetupProviderLocal
#   EVIDENCE: pending
# G11: skip-provider leaves ensure out
#   CHECK: go test ./skillgrid-cli/internal/install -count=1 -run TestSetupProviderSkipped
#   EXPECT: PASS TestSetupProviderSkipped
#   EVIDENCE: pending
