# Domain Research Pack

Serves: learn a domain's rules, constraints, and vocabulary before designing in
it — a regulated industry, a protocol's invariants, a market's terminology, the
operational realities a greenfield team doesn't yet feel.

**Dimensions (priority order — prune to the decision):**

1. Rules & invariants — the hard constraints the domain enforces (regulations,
   protocol guarantees, physical limits) that a design cannot violate
2. Vocabulary & mental model — the domain's own terms and how experts reason
   about it; what the lay name hides
3. Operational reality — how the thing actually runs in production, the failure
   modes practitioners warn about, the cost and latency that textbooks omit
4. Players & conventions — who operates in the domain, the established patterns
   and precedents a newcomer is expected to know
5. Edge & failure taxonomy — the known classes of things that go wrong, so the
   design can name them

**Craft (the non-obvious):** read post-mortems and incident write-ups, not
marketing; the domain's standards bodies and RFCs own the invariants; talk to the
failure cases first (what breaks) before the happy path; a domain's jargon is
load-bearing — a term that maps to two things is a bug in the design, not a
nuance; practitioners' "we always do X" is a convention worth recording even
when the reason is historical.

**Freshness bars:** regulations & protocol guarantees ≤ 6 mo · operational
patterns ≤ 12 mo · established conventions ≤ 2 yr (a convention that old is
stable; a regulation that old may be amended).

**Two-source classes (independent second source required):** regulatory or
compliance assertions the design depends on; any invariant the design assumes
holds (e.g. "at-least-once delivery", "the ledger is consistent").

**Feeds:** the spec (constraints and vocabulary — the glossary) · the
architecture (invariants the design must preserve) · the ADRs (why a constraint
forces a particular choice).
