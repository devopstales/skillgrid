// Package wiki is the pure core of the llmwiki-okf-wiki change.
//
// It defines the knowledge-concept model (Concept, SourceRef, Verifier, Edge)
// and the two pure functions that every later ticket builds on:
//
//   - Slugify: turn a human title into a filesystem-safe slug.
//   - TypeDir: map a concept type to its bundle directory.
//
// The package is pure and stdlib-only: no I/O, no database, no imports from
// other internal/ packages. It is the stable type foundation for the concept
// store (TICKET-02), the Markdown renderer (TICKET-03), the source scanner
// (TICKET-04), and the index builder (TICKET-05).
package wiki

import "time"

// Confidence is the provenance label on an Edge. In Pillar 1 every edge is
// EXTRACTED (a direct reference in the source). The type is a string so the
// set of labels can grow (INFERRED, AMBIGUOUS) in later pillars without
// breaking the stored schema.
type Confidence string

// The Confidence labels. Pillar 1 emits only EXTRACTED.
const (
	ConfidenceExtracted Confidence = "EXTRACTED"
	ConfidenceInferred  Confidence = "INFERRED"
	ConfidenceAmbiguous Confidence = "AMBIGUOUS"
)

// Verifier identifies who verified a concept (or its generation) and when.
// Generated is a single Verifier ({by, at}); Verified is a []Verifier, one
// entry per human or agent that has checked the concept since generation.
type Verifier struct {
	By string    // verifier identity (name, agent id, tool)
	At time.Time // when the verification happened
}

// SourceRef points at one upstream source document that a Concept is derived
// from. It carries the resource (origin), a stable id, a display title, the
// author, a usage counter, and the last-modified time of the source. A zero
// LastModified means the modification time is unknown/absent.
type SourceRef struct {
	ID           string    // stable source id (e.g. file hash or doc key)
	Resource     string    // origin resource (repo, url, path)
	Title        string    // display title of the source
	Author       string    // author of the source
	UsageCount   int       // number of times the source has been consumed
	LastModified time.Time // zero value = absent/unknown
}

// Concept is one unit of knowledge in the wiki. A Concept is a typed document
// (ADR, Term, Constraint, Spec, Spike, State, Architecture, Finding) rendered
// to Markdown and cross-linked with Edges. The fields mirror the OKF/llmwiki
// frontmatter plus the rendered body.
type Concept struct {
	ID          string      // stable concept id (slug, unique within a bundle)
	Type        string      // concept type: ADR, Term, Constraint, Spec, Spike, State, Architecture, Finding
	Title       string      // human title
	Description string      // one-paragraph summary
	Resource    string      // origin resource (repo, url, path)
	Tags        []string    // freeform tags
	Sources     []SourceRef // upstream sources this concept derives from
	Generated   Verifier    // who/when generated the concept (single {by, at})
	Verified    []Verifier  // who/when verified the concept (one per verifier)
	Status      string      // concept status (e.g. active, deprecated, superseded)
	StaleAfter  time.Time   // when the concept becomes stale (zero = never)
	SourcePath  string      // project-relative path to the source artifact this concept is bound to
	Body        string      // rendered Markdown body (the document content)
}

// Edge is a typed, directed link between two Concepts. In Pillar 1 every edge
// is EXTRACTED (an explicit reference in a source). Kind describes the
// semantic relationship (e.g. "references", "supersedes", "implements").
type Edge struct {
	From       string     // source concept ID
	To         string     // target concept ID
	Kind       string     // semantic relationship kind
	Confidence Confidence // provenance label (EXTRACTED in Pillar 1)
}
