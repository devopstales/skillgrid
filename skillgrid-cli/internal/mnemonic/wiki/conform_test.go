package wiki

import (
	"strings"
	"testing"
	"time"
)

// conformClean is a fully-populated, conformant page emitted by Emit. It is
// the round-trip baseline: Emit(c) must satisfy Conform with zero errors (R2.3).
func conformClean() string {
	c := Concept{
		Type:        "ADR",
		Title:       "Use stdlib for YAML frontmatter",
		Description: "No external dependencies.",
		Resource:    "https://repo.example.com/x",
		Tags:        []string{"okf", "conformance"},
		Sources: []SourceRef{
			{Resource: "https://example.com/doc", Author: "team:x", Title: "Doc", LastModified: ts},
			{Resource: "internal/note.md"},
		},
		Generated:  Verifier{By: "skillgrid", At: ts},
		Verified:   []Verifier{{By: "team:x", At: ts}},
		Status:     "stable",
		StaleAfter: ts.Add(24 * time.Hour),
		SourcePath: "docs/adr.md",
		Body:       "# ADR\n\nBody text.",
	}
	out, err := Emit(c)
	if err != nil {
		panic(err)
	}
	return out
}

func hasErrorContaining(t *testing.T, errs []error, substr string) bool {
	t.Helper()
	for _, e := range errs {
		if strings.Contains(e.Error(), substr) {
			return true
		}
	}
	t.Errorf("no error mentioning %q in: %v", substr, errs)
	return false
}

// R2.1: missing `type` → at least one error naming the key.
func TestConform_MissingType(t *testing.T) {
	errs := Conform(`---
title: No type
---

body`)
	if len(errs) == 0 {
		t.Fatalf("expected >=1 error for missing type, got none")
	}
	hasErrorContaining(t, errs, "type")
}

// Empty `type` value (present but blank) → error naming it.
func TestConform_EmptyType(t *testing.T) {
	errs := Conform(`---
type: ""
title: Empty type
---

body`)
	if len(errs) == 0 {
		t.Fatalf("expected >=1 error for empty type, got none")
	}
	hasErrorContaining(t, errs, "type")
}

// R2.2: a sources entry with no (or empty) resource → error identifying the
// source entry.
func TestConform_SourceMissingResource(t *testing.T) {
	errs := Conform(`---
type: Spec
title: Bad source
sources:
  - author: "team:x"
  - resource: ""
---

body`)
	if len(errs) == 0 {
		t.Fatalf("expected >=1 error for source without resource, got none")
	}
	hasErrorContaining(t, errs, "resource")
}

// Bad timestamp (space-separated, no T, no Z) on a timestamp-valued key → error.
func TestConform_BadTimestamp(t *testing.T) {
	errs := Conform(`---
type: Spec
title: Bad timestamp
stale_after: "2026-09-29 14:00:00"
---

body`)
	if len(errs) == 0 {
		t.Fatalf("expected >=1 error for bad timestamp, got none")
	}
	hasErrorContaining(t, errs, "stale_after")
}

// Relative (non-absolute) timestamp: offset form is not the OKF absolute-UTC
// Z form → error.
func TestConform_TimestampWithOffset(t *testing.T) {
	errs := Conform(`---
type: Spec
title: Offset timestamp
stale_after: "2026-09-29T14:00:00+02:00"
---

body`)
	if len(errs) == 0 {
		t.Fatalf("expected >=1 error for non-Z timestamp, got none")
	}
	hasErrorContaining(t, errs, "stale_after")
}

// Bad status (not in draft|stable|deprecated) → error naming the value.
func TestConform_BadStatus(t *testing.T) {
	errs := Conform(`---
type: Spec
title: Bad status
status: pending
---

body`)
	if len(errs) == 0 {
		t.Fatalf("expected >=1 error for bad status, got none")
	}
	hasErrorContaining(t, errs, "status")
	hasErrorContaining(t, errs, "pending")
}

// R2.3: a clean, fully-populated page → no errors (round-trips through Emit).
func TestConform_CleanPage(t *testing.T) {
	errs := Conform(conformClean())
	if len(errs) != 0 {
		t.Fatalf("expected 0 errors for clean page, got: %v", errs)
	}
}

// No frontmatter at all → error.
func TestConform_NoFrontmatter(t *testing.T) {
	errs := Conform("# Just a heading\n\nNo frontmatter here.")
	if len(errs) == 0 {
		t.Fatalf("expected >=1 error for page without frontmatter, got none")
	}
}

// An unterminated frontmatter block (opening --- but no closing ---) → error.
func TestConform_UnterminatedFrontmatter(t *testing.T) {
	errs := Conform("---\ntype: Spec\ntitle: Dangling\n\nbody")
	if len(errs) == 0 {
		t.Fatalf("expected >=1 error for unterminated frontmatter, got none")
	}
}

// Unknown keys are ignored: the conformance gate checks the reserved OKF
// surface only, so extra keys must not produce errors.
func TestConform_UnknownKeysIgnored(t *testing.T) {
	errs := Conform(`---
type: Spec
title: Extra keys
custom_thing: whatever
another_key: 42
---

body`)
	if len(errs) != 0 {
		t.Fatalf("unknown keys must be ignored, got: %v", errs)
	}
}

// generated.at and verified[].at and sources[].last_modified are all
// timestamp-valued and must be validated.
func TestConform_TimestampKeysAllValidated(t *testing.T) {
	// generated.at bad
	errs := Conform(`---
type: Spec
title: t
generated:
  by: skillgrid
  at: "not-a-time"
---

body`)
	hasErrorContaining(t, errs, "generated.at")

	// verified[0].at bad (list form)
	errs = Conform(`---
type: Spec
title: t
verified:
  - by: team:x
    at: "also-bad"
---

body`)
	hasErrorContaining(t, errs, "verified[0].at")

	// sources[0].last_modified bad
	errs = Conform(`---
type: Spec
title: t
sources:
  - resource: https://example.com
    last_modified: "2026-09-29"
---

body`)
	hasErrorContaining(t, errs, "sources[0].last_modified")
}

// A page that violates multiple rules reports all violations.
func TestConform_MultipleViolations(t *testing.T) {
	errs := Conform(`---
title: no type
status: pending
stale_after: "2026-09-29 14:00:00"
---

body`)
	if len(errs) < 3 {
		t.Fatalf("expected >=3 errors, got %d: %v", len(errs), errs)
	}
	hasErrorContaining(t, errs, "type")
	hasErrorContaining(t, errs, "status")
	hasErrorContaining(t, errs, "stale_after")
}

// status omitted is fine (only validated when present).
func TestConform_StatusOmittedOK(t *testing.T) {
	errs := Conform(`---
type: Spec
title: No status
---

body`)
	if len(errs) != 0 {
		t.Fatalf("omitted status must be valid, got: %v", errs)
	}
}

// All three valid status values pass.
func TestConform_ValidStatusValues(t *testing.T) {
	for _, s := range []string{"draft", "stable", "deprecated"} {
		errs := Conform("---\ntype: Spec\ntitle: s\nstatus: " + s + "\n---\n\nbody")
		if len(errs) != 0 {
			t.Errorf("status %q must be valid, got: %v", s, errs)
		}
	}
}
