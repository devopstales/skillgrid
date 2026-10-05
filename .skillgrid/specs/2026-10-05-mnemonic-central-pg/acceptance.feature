Feature: Central PG shared memory state
  A team shares one PostgreSQL database of mnemonic memory state while each
  machine keeps its local SQLite store as the offline-first working store.
  Local saves are never blocked or failed by the sync layer (ADR-0016 floors).

  Background:
    Given a PostgreSQL 14+ server is reachable
    And the admin DSN is configured with role "mem_admin"
    And mnemonic data dir is the temp directory

  # --- Req 8: disabled is a no-op ---

  Scenario: happy path disabled central is a no-op
    Given mnemonic central is disabled
    When I save an observation "fact: we use pgx"
    Then the observation is stored locally
    And no PostgreSQL connection is opened
    And the outbox table is empty
    And "mnemonic central status" reports "disabled"

  # --- Req 1: outbox on shared-entity writes ---

  Scenario: happy path observation save records outbox row
    Given mnemonic central is enabled for project "demo"
    When I save an observation "fact: we use pgx"
    Then exactly one outbox row exists for entity "observation"
    And the outbox row disposition is "pending"
    And the local save returned before any network activity

  Scenario: happy path each shared write site records its outbox row
    Given mnemonic central is enabled for project "demo"
    When I save a prompt, a fact, a skill, and a web cache entry, and end a session, and add a memory edge, and add a team member
    Then the outbox contains one row for each of the 8 entities
    And no local write failed because of the outbox

  Scenario: error path outbox failure never fails the local save
    Given the outbox table is locked by a concurrent writer
    When I save an observation "fact: outbox busy"
    Then the observation is stored locally
    And the save returns success
    And a warning is logged for the missed outbox record

  # --- Req 2: PG schema + migrations ---

  Scenario: happy path pg migrations apply idempotently
    Given an empty database "mnemonic"
    When the sync manager starts for project "demo"
    Then schema "mem_demo" exists with all shared tables
    And the journal table "mem_demo.mem_mutations" exists with a serial seq column
    And re-running migrations changes nothing
    And a migration run interrupted midway resumes without error

  # --- Req 3: sync manager with lease ---

  Scenario: happy path lease guards single worker per machine
    Given two sync managers start on the same machine for project "demo"
    When 70 seconds elapse
    Then exactly one manager held the lease at any time
    And the other manager waited without pushing duplicates

  Scenario: happy path lease expires and another manager takes over
    Given one sync manager holds the lease
    When the holding manager is killed and 70 seconds elapse
    Then the other manager acquires the lease
    And the outbox continues draining

  Scenario: error path auth failure backs off and pauses with reason code
    Given the token role was revoked in PostgreSQL
    When the sync manager attempts 10 consecutive pushes
    Then the manager is paused with reason code "auth_failed"
    And the backoff delay doubled between attempts up to the 5 minute cap
    And "mnemonic central status" shows the pause and reason code

  Scenario: error path pg down queues in outbox and catches up
    Given the PostgreSQL server is unreachable
    When I save 5 observations while offline
    Then the outbox holds 5 pending rows
    And no save was blocked
    When PostgreSQL becomes reachable again
    Then within one sync cycle the outbox is empty
    And PostgreSQL holds all 5 observations
    And the local store still has exactly 5 observations

  # --- Req 4: LWW + tombstones ---

  Scenario: happy path lww converges concurrent updates
    Given observation with sync id "obs-1" exists on machine A and B
    When machine A updates "obs-1" content to "vA" at 10:00:00Z
    And machine B updates "obs-1" content to "vB" at 10:00:01Z
    Then after both machines sync, both local stores hold content "vB"
    And neither machine logged an error

  Scenario: happy path identical re-push is a no-op
    Given observation "obs-2" was already pushed with payload hash H
    When the same payload is pushed again
    Then the journal gains no new row for (observation, obs-2, H)
    And the materialized row is unchanged

  Scenario: happy path delete tombstone propagates and revives on re-save
    Given observation "obs-3" exists on machine A and B
    When machine A deletes "obs-3"
    Then after sync, machine B has "obs-3" soft-deleted
    When machine A re-saves "obs-3" with the same sync id
    Then after sync, "obs-3" is active again on machine B
    And the tombstone does not re-delete it

  Scenario: happy path tombstone prune keeps re-saved sync ids
    Given a tombstone for "obs-4" older than the retention window
    And "obs-4" was re-saved after the tombstone
    When the prune pass runs
    Then the tombstone row is removed
    And "obs-4" remains active

  # --- Req 5: bootstrap ---

  Scenario: happy path enroll bootstraps local history
    Given project "demo" has 20 local observations and is not enrolled
    When I run "mnemonic central enroll demo"
    Then the outbox holds 20 upsert rows for project "demo"
    And after one sync cycle, PostgreSQL holds all 20 observations
    And subsequent saves flow through the normal outbox path

  Scenario: happy path unenroll stops future pushes and preserves data
    Given project "demo" is enrolled with 5 pending outbox rows
    When I run "mnemonic central unenroll demo"
    Then no further outbox rows are pushed for "demo"
    And the 5 pending rows remain in the outbox
    And local data is unchanged

  # --- Req 6: tokens, grants, audit ---

  Scenario: happy path issue token grants only named project
    Given I am authenticated as the admin DSN
    When I run "mnemonic central issue-token --project demo"
    Then a token of the form "egm_dev_XXXXXXXX_<secret>" is printed once
    And a PostgreSQL role "mem_XXXXXXXX" with LOGIN exists
    And the role can SELECT from "mem_demo.mem_observations"
    And the role cannot SELECT from "mem_other.mem_observations"

  Scenario: error path token without grant is denied and audited
    Given a token role granted only to project "demo"
    When the sync manager pushes a row into schema "mem_other"
    Then PostgreSQL denies the write
    And a row is inserted into "mem_auth_audit_log" with reason "policy_denied"

  Scenario: happy path revoke token locks its role
    Given a token with prefix "XXXXXXXX" was issued
    When I run "mnemonic central revoke-token XXXXXXXX"
    Then the PostgreSQL role "mem_XXXXXXXX" no longer has LOGIN
    And "mnemonic central doctor" reports the token as revoked

  # --- Req 7: config + CLI ---

  Scenario: happy path central config defaults off
    Given no mnemonic.store.central block exists
    When the config is loaded
    Then central enabled is false
    And poll interval defaults to 30 seconds
    And push batch defaults to 100
    And tombstone retention defaults to 30 days
    And the environment variable SKILLGRID_MNEMONIC_CENTRAL_DSN wins over the file dsn

  Scenario: happy path central doctor validates connection and grants
    Given a valid central DSN and enrolled project "demo"
    When I run "mnemonic central doctor"
    Then it reports the server version, schema version, and grant status per enrolled project
    And it exits non-zero with a named reason when the DSN is invalid
    And it exits non-zero when the token role lacks a grant for an enrolled project

  # --- Req 2/4 cross-machine end to end ---

  Scenario: happy path two machines share memory end to end
    Given machine A and machine B are enrolled in project "demo" against the same PostgreSQL
    When machine A saves an observation, a fact, and a skill
    Then within one sync cycle, machine B's mem_search finds the observation
    And machine B's local fact and skill tables contain the rows
    And machine B's store has no code-index or embedding rows from machine A
