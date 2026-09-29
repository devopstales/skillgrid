package wiki

import (
	"strings"
	"testing"
	"time"
)

var ts = time.Date(2026, 9, 29, 14, 0, 0, 0, time.UTC)

// R1.1: minimal page — only type, title, body. Output begins "---\n",
// contains "type: ADR", and the body appears verbatim after the closing "---".
func TestEmit_MinimalPage(t *testing.T) {
	c := Concept{
		Type:  "ADR",
		Title: "Use Go stdlib for YAML",
		Body:  "# ADR: Use Go stdlib for YAML\n\nDecided.",
	}
	out, err := Emit(c)
	if err != nil {
		t.Fatalf("Emit() error: %v", err)
	}
	if !strings.HasPrefix(out, "---\n") {
		t.Fatalf("output must begin with ---\\n, got prefix %q", out[:min(5, len(out))])
	}
	if !strings.Contains(out, "type: ADR") {
		t.Errorf("output must contain 'type: ADR'\noutput:\n%s", out)
	}
	// Body must appear verbatim after the closing delimiter.
	closing := strings.LastIndex(out, "---\n")
	rest := out[closing+len("---\n"):]
	if !strings.Contains(rest, c.Body) {
		t.Errorf("body not found verbatim after closing ---\noutput:\n%s", out)
	}
}

// R1.3: no-null omission — a concept with no tags, no verified, no resource,
// no description, no sources, no generated, no status, no stale_after, no
// source_path must not emit any of those keys at all (no "tags: null" etc.).
func TestEmit_NoNullOmission(t *testing.T) {
	c := Concept{
		Type:  "ADR",
		Title: "Bare minimum",
		Body:  "body text",
	}
	out, err := Emit(c)
	if err != nil {
		t.Fatalf("Emit() error: %v", err)
	}
	for _, key := range []string{"description", "resource", "tags", "sources", "generated", "verified", "status", "stale_after", "source_path"} {
		if strings.Contains(out, key+":") {
			t.Errorf("zero-value key %q must be omitted entirely, found in output:\n%s", key, out)
		}
	}
}

// Timestamps: every timestamp-valued key uses absolute UTC ISO 8601 "…T…Z".
func TestEmit_TimestampFormat(t *testing.T) {
	c := Concept{
		Type:       "Spec",
		Title:      "Timestamped",
		Body:       "b",
		StaleAfter: ts,
		Generated:  Verifier{By: "skillgrid", At: ts},
		Verified:   []Verifier{{By: "team:x", At: ts}},
		Sources:    []SourceRef{{Resource: "https://example.com", LastModified: ts}},
	}
	out, err := Emit(c)
	if err != nil {
		t.Fatalf("Emit() error: %v", err)
	}
	const want = "2026-09-29T14:00:00Z"
	if !strings.Contains(out, want) {
		t.Errorf("output must contain %q\noutput:\n%s", want, out)
	}
	// No naive (non-UTC) rendering may leak through.
	if strings.Contains(out, "2026-09-29 14:00:00") || strings.Contains(out, "2026-09-29T14:00:00+00:00") {
		t.Errorf("timestamps must be absolute UTC RFC3339 (…Z), got:\n%s", out)
	}
}

// Non-UTC input times must be normalized to UTC in the output.
func TestEmit_TimestampNormalizedToUTC(t *testing.T) {
	local := time.FixedZone("UTC+2", 2*3600)
	c := Concept{
		Type:       "Spec",
		Title:      "NonUTC",
		Body:       "b",
		StaleAfter: time.Date(2026, 9, 29, 16, 0, 0, 0, local), // == 14:00Z
		Generated:  Verifier{By: "skillgrid", At: time.Date(2026, 9, 29, 16, 0, 0, 0, local)},
	}
	out, err := Emit(c)
	if err != nil {
		t.Fatalf("Emit() error: %v", err)
	}
	if !strings.Contains(out, "2026-09-29T14:00:00Z") {
		t.Errorf("non-UTC time must render as 2026-09-29T14:00:00Z\noutput:\n%s", out)
	}
}

// Sources: rendered as a YAML list of maps with the entry keys resource,
// author, title, last_modified (last_modified omitted when zero).
func TestEmit_SourcesList(t *testing.T) {
	c := Concept{
		Type:  "Spec",
		Title: "Sourced",
		Body:  "b",
		Sources: []SourceRef{
			{Resource: "https://example.com", Author: "team:x", Title: "Doc", LastModified: ts},
			{Resource: "internal/note.md"}, // no author/title/last_modified
		},
	}
	out, err := Emit(c)
	if err != nil {
		t.Fatalf("Emit() error: %v", err)
	}
	want := `sources:
  - resource: "https://example.com"
    author: "team:x"
    title: Doc
    last_modified: "2026-09-29T14:00:00Z"
  - resource: internal/note.md`
	if !strings.Contains(out, want) {
		t.Errorf("sources block mismatch\nwant contains:\n%s\n---\nactual:\n%s", want, out)
	}
	// The bare source must not carry empty keys.
	bare := "  - resource: internal/note.md\n"
	idx := strings.Index(out, bare)
	if idx < 0 {
		t.Fatalf("bare source entry not found\noutput:\n%s", out)
	}
	after := out[idx+len(bare):]
	if len(after) == 0 {
		return
	}
	next := after[:strings.IndexByte(after, '\n')+1]
	if strings.Contains(next, "author") || strings.Contains(next, "title") || strings.Contains(next, "last_modified") {
		t.Errorf("bare source must not carry empty keys, got %q", next)
	}
}

// Verified: single entry renders as a bare map; multiple entries as a list.
func TestEmit_VerifiedSingleBareMap(t *testing.T) {
	c := Concept{
		Type:     "Spec",
		Title:    "SingleVerified",
		Body:     "b",
		Verified: []Verifier{{By: "team:x", At: ts}},
	}
	out, err := Emit(c)
	if err != nil {
		t.Fatalf("Emit() error: %v", err)
	}
	want := `verified:
  by: "team:x"
  at: "2026-09-29T14:00:00Z"`
	if !strings.Contains(out, want) {
		t.Errorf("single verified must be a bare map\nwant contains:\n%s\n---\nactual:\n%s", want, out)
	}
}

func TestEmit_VerifiedList(t *testing.T) {
	c := Concept{
		Type:  "Spec",
		Title: "MultiVerified",
		Body:  "b",
		Verified: []Verifier{
			{By: "team:x", At: ts},
			{By: "agent:y", At: ts.Add(time.Hour)},
		},
	}
	out, err := Emit(c)
	if err != nil {
		t.Fatalf("Emit() error: %v", err)
	}
	want := `verified:
  - by: "team:x"
    at: "2026-09-29T14:00:00Z"
  - by: "agent:y"
    at: "2026-09-29T15:00:00Z"`
	if !strings.Contains(out, want) {
		t.Errorf("multi verified must be a list\nwant contains:\n%s\n---\nactual:\n%s", want, out)
	}
}

// Generated: single {by, at} block, never a list.
func TestEmit_Generated(t *testing.T) {
	c := Concept{
		Type:      "Spec",
		Title:     "Generated",
		Body:      "b",
		Generated: Verifier{By: "skillgrid", At: ts},
	}
	out, err := Emit(c)
	if err != nil {
		t.Fatalf("Emit() error: %v", err)
	}
	want := `generated:
  by: skillgrid
  at: "2026-09-29T14:00:00Z"`
	if !strings.Contains(out, want) {
		t.Errorf("generated block mismatch\nwant contains:\n%s\n---\nactual:\n%s", want, out)
	}
}

// SourcePath: the extension key is emitted when set (R3.1/R3.2) and omitted
// when empty.
func TestEmit_SourcePath(t *testing.T) {
	c := Concept{
		Type:       "Spec",
		Title:      "SourcedPath",
		Body:       "b",
		SourcePath: "docs/specs/okf.md",
	}
	out, err := Emit(c)
	if err != nil {
		t.Fatalf("Emit() error: %v", err)
	}
	if !strings.Contains(out, "source_path: docs/specs/okf.md") {
		t.Errorf("source_path must be emitted when set\noutput:\n%s", out)
	}
}

// R3.1/R3.2: the fixed key order is respected for a fully-populated concept.
func TestEmit_FullConceptKeyOrder(t *testing.T) {
	c := Concept{
		ID:          "okf-emit",
		Type:        "ADR",
		Title:       "Full concept",
		Description: "Everything set",
		Resource:    "https://repo.example.com/x",
		Tags:        []string{"okf", "emitter"},
		Sources:     []SourceRef{{Resource: "https://example.com", Author: "team:x", Title: "Doc", LastModified: ts}},
		Generated:   Verifier{By: "skillgrid", At: ts},
		Verified:    []Verifier{{By: "team:x", At: ts}},
		Status:      "active",
		StaleAfter:  ts,
		SourcePath:  "docs/adr.md",
		Body:        "# Full\n\nBody.",
	}
	out, err := Emit(c)
	if err != nil {
		t.Fatalf("Emit() error: %v", err)
	}
	open := strings.Index(out, "---\n")
	close := strings.LastIndex(out, "---\n")
	if open < 0 || close <= open {
		t.Fatalf("frontmatter delimiters missing\noutput:\n%s", out)
	}
	fm := out[open+len("---\n") : close]
	// Top-level keys in strict order.
	idx := map[string]int{}
	for _, line := range strings.Split(fm, "\n") {
		if line == "" || line[0] == ' ' {
			continue
		}
		key := strings.SplitN(line, ":", 2)[0]
		idx[key] = len(idx)
	}
	wantOrder := []string{"type", "title", "description", "resource", "tags", "sources", "generated", "verified", "status", "stale_after", "source_path"}
	var got []string
	for _, k := range wantOrder {
		if _, ok := idx[k]; ok {
			got = append(got, k)
		}
	}
	if len(got) != len(wantOrder) {
		t.Fatalf("not all keys present in full concept; missing: %v\nfrontmatter:\n%s", missingKeys(wantOrder, got), fm)
	}
	for i, k := range got {
		if k != wantOrder[i] {
			t.Errorf("key order wrong at %d: got %q want %q\nfrontmatter:\n%s", i, k, wantOrder[i], fm)
		}
	}
}

// Strings containing YAML special characters are double-quoted; simple
// alphanumeric strings are not.
func TestEmit_StringQuoting(t *testing.T) {
	c := Concept{
		Type:        "Spec",
		Title:       "Simple title",
		Description: "value: with colon and # hash",
		Body:        "b",
	}
	out, err := Emit(c)
	if err != nil {
		t.Fatalf("Emit() error: %v", err)
	}
	if !strings.Contains(out, `title: Simple title`) {
		t.Errorf("simple title must be unquoted\noutput:\n%s", out)
	}
	if !strings.Contains(out, `description: "value: with colon and # hash"`) {
		t.Errorf("special-char description must be double-quoted\noutput:\n%s", out)
	}
}

// Determinism: two calls with the same concept produce identical bytes.
func TestEmit_Deterministic(t *testing.T) {
	c := Concept{
		Type:       "Spec",
		Title:      "Deterministic",
		Body:       "b",
		Generated:  Verifier{By: "skillgrid", At: ts},
		StaleAfter: ts,
	}
	a, err1 := Emit(c)
	b, err2 := Emit(c)
	if err1 != nil || err2 != nil {
		t.Fatalf("Emit() errors: %v %v", err1, err2)
	}
	if a != b {
		t.Errorf("Emit must be deterministic\na:\n%s\nb:\n%s", a, b)
	}
}

// Emit must not mutate its input.
func TestEmit_DoesNotMutate(t *testing.T) {
	c := Concept{
		Type:     "Spec",
		Title:    "Immutable",
		Body:     "b",
		Verified: []Verifier{{By: "team:x", At: ts}},
	}
	cp := c
	_, err := Emit(c)
	if err != nil {
		t.Fatalf("Emit() error: %v", err)
	}
	if c.Verified[0] != cp.Verified[0] {
		t.Errorf("Emit mutated Verified: %v vs %v", c.Verified, cp.Verified)
	}
}

// Quote escaping: embedded double quotes and backslashes are escaped inside
// double-quoted scalars; leading "-" and special leads are quoted.
func TestEmit_QuoteEscaping(t *testing.T) {
	c := Concept{
		Type:        "Spec",
		Title:       `He said "hi"`,
		Description: "back\\slash",
		Body:        "b",
	}
	out, err := Emit(c)
	if err != nil {
		t.Fatalf("Emit() error: %v", err)
	}
	if !strings.Contains(out, `title: "He said \"hi\""`) {
		t.Errorf("embedded double quotes must be escaped\noutput:\n%s", out)
	}
	if !strings.Contains(out, `description: "back\\slash"`) {
		t.Errorf("backslashes must be escaped\noutput:\n%s", out)
	}

	dash := Concept{Type: "Spec", Title: "-leading dash", Body: "b"}
	out2, _ := Emit(dash)
	if !strings.Contains(out2, `title: "-leading dash"`) {
		t.Errorf("leading dash must be quoted\noutput:\n%s", out2)
	}

	tab := Concept{Type: "Spec", Title: "has\ttab", Body: "b"}
	out3, _ := Emit(tab)
	if !strings.Contains(out3, "title: \"has\ttab\"") {
		t.Errorf("tab must force quoting\noutput:\n%s", out3)
	}
}

// Error return: the current pure implementation always returns nil error.
func TestEmit_AlwaysNilError(t *testing.T) {
	c := Concept{Type: "Spec", Title: "Err", Body: "b"}
	_, err := Emit(c)
	if err != nil {
		t.Fatalf("Emit must always return nil error, got: %v", err)
	}
}

// Empty body: the document is still well-formed (frontmatter + blank line,
// then nothing).
func TestEmit_EmptyBody(t *testing.T) {
	out, err := Emit(Concept{Type: "Spec", Title: "NoBody"})
	if err != nil {
		t.Fatalf("Emit() error: %v", err)
	}
	if out != "---\ntype: Spec\ntitle: NoBody\n---\n\n" {
		t.Errorf("empty-body output mismatch:\n%q", out)
	}
}

// Missing type/title: omitted keys, not an error (pure serializer).
func TestEmit_EmptyTypeTitle(t *testing.T) {
	out, err := Emit(Concept{Body: "just body"})
	if err != nil {
		t.Fatalf("Emit() error: %v", err)
	}
	if strings.Contains(out, "type:") || strings.Contains(out, "title:") {
		t.Errorf("empty type/title must be omitted\noutput:\n%s", out)
	}
}

func missingKeys(want, got []string) []string {
	set := map[string]bool{}
	for _, g := range got {
		set[g] = true
	}
	var m []string
	for _, w := range want {
		if !set[w] {
			m = append(m, w)
		}
	}
	return m
}
