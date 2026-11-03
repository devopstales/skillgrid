# Local Ollama Model Catalog
#
# Source: .skillgrid/specs/2026-10-03-local-ollama-models/
# Trace: briefing requirements 1–6.

## Requirements

### Requirement: Six-model local catalog

The system SHALL pull a six-model local Ollama catalog (live chat, system-one, embed, research) on a local `skillgrid install`, gated behind an Ollama version floor for the heavy models.

#### Scenario: happy path catalog pull list

```gherkin
  Given a served Ollama with no models present
  And a version at or above the floor
  When a local install runs the provider step
  Then exactly the six catalog tags are pulled
  And no other tag is pulled
```

#### Scenario: happy path version floor gates heavy models

```gherkin
  Given a served Ollama reporting a version below the floor
  When a local install runs the provider step
  Then the heavy models tev1:0.8b and clef-flash are not pulled
  And a warning names the required Ollama version
  And install still succeeds
```

#### Scenario: happy path version at or above floor pulls all

```gherkin
  Given a served Ollama reporting a version at or above the floor
  When a local install runs the provider step
  Then all six catalog tags are pulled
```

#### Gates
# G1: happy path catalog pull list
#   CHECK: go test ./skillgrid-cli/internal/install -count=1 -run TestCatalogPullList
#   EXPECT: PASS TestCatalogPullList
#   EVIDENCE: PASS (ok internal/install) — six catalog tags pulled, glm:vision-tools absent
# G2: happy path version floor gates heavy models
#   CHECK: go test ./skillgrid-cli/internal/install -count=1 -run TestFloorGatesHeavyModels
#   EXPECT: PASS TestFloorGatesHeavyModels
#   EVIDENCE: PASS (ok internal/install) — below floor, tev1:0.8b + clef-flash skipped with version warning, install succeeds
# G3: happy path version at or above floor pulls all
#   CHECK: go test ./skillgrid-cli/internal/install -count=1 -run TestAtFloorPullsAll
#   EXPECT: PASS TestAtFloorPullsAll
#   EVIDENCE: PASS (ok internal/install) — at/above floor, all six catalog tags pulled

### Requirement: Config wiring

The system SHALL merge the home config so the live-chat LLM model and the Ollama embedder model reference the catalog tags.

#### Scenario: happy path config merge writes new live-chat + embed models

```gherkin
  Given a local install runs the provider step
  When the home indexing.yaml is merged
  Then mnemonic.llm.model is llama3.2:1b
  And mnemonic.embedder.provider is ollama
  And mnemonic.embedder.model is embeddinggemma:300m
  And unrelated operator keys are preserved
```

#### Gates
# G4: happy path config merge writes new live-chat + embed models
#   CHECK: go test ./skillgrid-cli/internal/install -count=1 -run TestConfigMergeWritesNewModels
#   EXPECT: PASS TestConfigMergeWritesNewModels
#   EVIDENCE: PASS (ok internal/install) — llm.model=llama3.2:1b, embedder.provider=ollama, embedder.model=embeddinggemma:300m, base_url=/v1, profile+dimension preserved

### Requirement: Runtime default

The system SHALL default the Ollama embedder runtime model to the catalog embed model.

#### Scenario: happy path ollama default model

```gherkin
  Given the ollama embedder package
  When its default model constant is read
  Then it is embeddinggemma:300m
```

#### Gates
# G5: happy path ollama default model
#   CHECK: go test ./skillgrid-cli/internal/mnemonic/embedder -count=1 -run TestDefaultOllamaModelIsEmbeddinggemma
#   EXPECT: PASS TestDefaultOllamaModelIsEmbeddinggemma
#   EVIDENCE: PASS (ok internal/mnemonic/embedder) — DefaultOllamaModel == embeddinggemma:300m

### Requirement: Typo exclusion

The system SHALL exclude the typo tag `glm:vision-tools` from the catalog entirely.

#### Scenario: happy path glm vision tools excluded

```gherkin
  Given the local catalog
  When a local install runs the provider step
  Then glm:vision-tools is never pulled, listed, or warned about
```

#### Gates
# G6: happy path glm vision tools excluded
#   CHECK: go test ./skillgrid-cli/internal/install -count=1 -run TestCatalogPullList
#   EXPECT: PASS TestCatalogPullList (asserts no glm:vision-tools in pull invocations)
#   EVIDENCE: PASS (ok internal/install) — glm:vision-tools absent from pull invocations and catalog

### Requirement: Per-role smoke dispatch (req 3)

The system SHALL run one non-interactive readiness probe per pulled model on the
API that model's role speaks. A failure warns and does not fail the install
(ADR-0016 fail-open).

#### Scenario: happy path smoke dispatch by role

```gherkin
  Given a served Ollama responding on /v1/chat/completions, /api/embed and /v1/systemone
  When the local install runs a per-role smoke probe
  Then the chat role passes on a non-empty first choice from /v1/chat/completions
  And the embed role passes on a /api/embed vector of length greater than zero
  And the systemone role passes on a present answer field from /v1/systemone
  And the research role passes as a no-op
  And a 500 on the embed endpoint fails the embed probe without failing the install
```

#### Gates
# G7: happy path smoke dispatch by role
#   CHECK: go test ./skillgrid-cli/internal/install -count=1 -run TestSmokeProbeDispatchByRole
#   EXPECT: PASS TestSmokeProbeDispatchByRole
#   EVIDENCE: PASS (ok internal/install) — chat/embed/systemone/research dispatch to the right endpoints; embed 500 + unreachable both fail the probe

### Requirement: Embed dimension recording + mismatch warning (reqs 3 + 5)

The system SHALL record the embedder vector length from a successful embed smoke
into `mnemonic.embedder.dimension`. If the operator's existing home config records
a different dimension, the install warns that search is stale until reindex, and the
embedder model still switches (fail-open).

#### Scenario: happy path embed smoke records dimension

```gherkin
  Given a served Ollama whose /api/embed returns a vector of length 384
  When a local install runs the provider step
  Then mnemonic.embedder.model is embeddinggemma:300m
  And mnemonic.embedder.dimension is 384
```

#### Scenario: happy path dimension mismatch warns but still switches

```gherkin
  Given an existing home config recording embedder dimension 768
  And a served Ollama whose /api/embed returns a vector of length 512
  When a local install runs the provider step
  Then a warning names both lengths (768 and 512) and that search is stale until reindex
  And mnemonic.embedder.model is still embeddinggemma:300m
  And mnemonic.embedder.dimension is 512
```

#### Gates
# G8: happy path embed smoke records dimension
#   CHECK: go test ./skillgrid-cli/internal/install -count=1 -run TestEmbedSmokeRecordsDimension
#   EXPECT: PASS TestEmbedSmokeRecordsDimension
#   EVIDENCE: PASS (ok internal/install) — smoke vector length 384 written to mnemonic.embedder.dimension, model=embeddinggemma:300m
# G9: happy path dimension mismatch warns but still switches
#   CHECK: go test ./skillgrid-cli/internal/install -count=1 -run TestEmbedDimensionMismatchWarns
#   EXPECT: PASS TestEmbedDimensionMismatchWarns
#   EVIDENCE: PASS (ok internal/install) — fixture 768 vs smoke 512 warns with both lengths, model still switches to embeddinggemma:300m, dimension=512

### Requirement: Ollama catalog list (req 5)

The system SHALL write `mnemonic.ollama.models` listing the six catalog tags with
their kind (chat / embed / systemone), excluding `glm:vision-tools`.

#### Scenario: happy path home merge writes ollama models list

```gherkin
  Given a local install merges the home indexing.yaml
  When the merge completes
  Then mnemonic.ollama.models has exactly six entries
  And each entry carries the tag name and its kind (chat / embed / systemone)
  And glm:vision-tools is not in the list
  And unrelated operator keys are preserved
```

#### Gates
# G10: happy path home merge writes ollama models list
#   CHECK: go test ./skillgrid-cli/internal/install -count=1 -run TestHomeMergeWritesOllamaModelsList
#   EXPECT: PASS TestHomeMergeWritesOllamaModelsList
#   EVIDENCE: PASS (ok internal/install) — mnemonic.ollama.models has six entries with correct kinds, glm:vision-tools absent, profile preserved
