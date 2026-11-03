package clm

import (
	"context"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

func TestValidateRevision(t *testing.T) {
	cases := []struct {
		name     string
		edit     *Edit
		expected string
		wantErr  bool
	}{
		{
			name:     "match nonce and non-empty passes",
			edit:     &Edit{Nonce: "abc", Messages: []string{"m1", "m2"}},
			expected: "abc",
			wantErr:  false,
		},
		{
			name:     "single message passes",
			edit:     &Edit{Nonce: "xyz", Messages: []string{"only"}},
			expected: "xyz",
			wantErr:  false,
		},
		{
			name:     "mismatched nonce errors",
			edit:     &Edit{Nonce: "abc", Messages: []string{"m1"}},
			expected: "def",
			wantErr:  true,
		},
		{
			name:     "empty messages errors",
			edit:     &Edit{Nonce: "abc", Messages: nil},
			expected: "abc",
			wantErr:  true,
		},
		{
			name:     "empty message slice errors",
			edit:     &Edit{Nonce: "abc", Messages: []string{}},
			expected: "abc",
			wantErr:  true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateRevision(tc.edit, tc.expected)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ValidateRevision() error = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}

func TestPersistRevision_RoundTrip(t *testing.T) {
	st, err := store.Open(t.TempDir(), "rproj")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()
	ctx := context.Background()

	rev := &Revision{
		SessionID:         "s1",
		ProjectID:         "rproj",
		Revision:          1,
		AnchorCount:       3,
		AnchorDigest:      "deadbeef",
		Messages:          "m1\nm2",
		SizeEstimate:      42,
		CalibrationFactor: 4.0,
		WithholdIDs:       "t1,t2",
		EditTrace:         "trace-1",
	}
	if err := PersistRevision(ctx, st.DB, rev); err != nil {
		t.Fatalf("persist: %v", err)
	}

	got, err := Resume(ctx, st.DB, "s1")
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if got == nil {
		t.Fatalf("expected a revision, got nil")
	}
	if got.Revision != 1 {
		t.Fatalf("expected revision 1, got %d", got.Revision)
	}
	if got.AnchorCount != 3 || got.AnchorDigest != "deadbeef" || got.Messages != "m1\nm2" {
		t.Fatalf("fields did not round-trip: %+v", got)
	}
	if got.SizeEstimate != 42 || got.WithholdIDs != "t1,t2" || got.EditTrace != "trace-1" {
		t.Fatalf("fields did not round-trip: %+v", got)
	}

	// A second persist must yield a higher max(id).
	rev2 := &Revision{SessionID: "s1", ProjectID: "rproj", Revision: 2, AnchorDigest: "second"}
	if err := PersistRevision(ctx, st.DB, rev2); err != nil {
		t.Fatalf("persist second: %v", err)
	}
	got2, err := Resume(ctx, st.DB, "s1")
	if err != nil {
		t.Fatalf("resume second: %v", err)
	}
	if got2 == nil || got2.Revision != 2 || got2.AnchorDigest != "second" {
		t.Fatalf("expected revision 2 after second persist, got %+v", got2)
	}
	var maxID int
	if err := st.DB.QueryRow("SELECT max(id) FROM context_revisions").Scan(&maxID); err != nil {
		t.Fatalf("query max id: %v", err)
	}
	if maxID < 2 {
		t.Fatalf("expected max(id) >= 2 after two persists, got %d", maxID)
	}
}
