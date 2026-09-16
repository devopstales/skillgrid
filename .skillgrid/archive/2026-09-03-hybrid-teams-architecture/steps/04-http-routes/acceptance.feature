# 04-http-routes acceptance
# Source: intent.md + plan.md (docs/skillgrid/changes/001-hybrid-teams-architecture/)

Feature: Teams HTTP routes under teams path
  As an HTTP client
  I want teams CRUD under the teams path prefix
  So that writes are auth-gated without colliding with memory reviews

  @happy @p0
  Scenario: Authenticated write under teams path succeeds
    Given a write token is configured
    When an authenticated client posts a teams resource
    Then the write succeeds

  @edge
  Scenario: Gets stay open and teams paths stay distinct
    Given a write token is configured
    When a client gets a teams resource without a bearer
    Then the read succeeds
    And teams routes do not collide with memory review routes

  @failure @p1
  Scenario: Unauthenticated write returns 401
    Given a write token is configured
    When a client posts a teams resource without a bearer
    Then the response status is 401
