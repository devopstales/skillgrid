package envelope

// envelope_test.go — the D1 envelope contract: the error-code mapping and
// the degradation flag (INV-3). The envelope is the one shared shape the
// MCP adapter (Task 13) and the CLI adapter (Task 14) both marshal — the
// K6 byte-identical seam — so its tests are the contract's guard.

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/memory"
)

// TestToSnakePinsWireNames pins the CamelCase → snake_case mapping (the
// D1 wire contract: the Go types are CamelCase, the wire is snake_case).
func TestToSnakePinsWireNames(t *testing.T) {
	cases := map[string]string{
		"SyncID":           "sync_id",
		"ID":               "id",
		"JudgmentID":       "judgment_id",
		"MergedIntoSyncID": "merged_into_sync_id",
		"JudgmentRequired": "judgment_required",
		"RevisionCount":    "revision_count",
		"Degraded":         "degraded",
		"candidates":       "candidates",
		"degraded_reason":  "degraded_reason",
	}
	for in, want := range cases {
		if got := toSnake(in); got != want {
			t.Errorf("toSnake(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestMapErrorPinsSentinels pins the sentinel → code mapping (the D1
// error contract: the plugins switch on the code, not the message).
func TestMapErrorPinsSentinels(t *testing.T) {
	cases := []struct {
		err  error
		code string
	}{
		{memory.ErrOperationConflict, CodeOperationConflict},
		{memory.ErrOperationExpired, CodeOperationExpired},
		{memory.ErrSessionEnded, CodeSessionAlreadyEnded},
		{memory.ErrSessionProjectMismatch, CodeProjectMismatch},
		{memory.ErrConsolidateFingerprintMismatch, CodeFingerprintMismatch},
		{memory.ErrConsolidateRefused, CodeConsolidationRefused},
		{errors.New("store: observation 5 not found"), CodeNotFound},
	}
	for _, tc := range cases {
		code, msg := MapError(tc.err)
		if code != tc.code {
			t.Errorf("MapError(%v) = %q, want %q", tc.err, code, tc.code)
		}
		if msg == "" {
			t.Errorf("MapError(%v) returned an empty message", tc.err)
		}
	}
}

// TestWithDegradedNoGhostFlag pins the no-ghost-flag contract: a healthy
// result carries neither the degraded flag nor the reason (INV-3: never a
// silent floor, never a ghost flag).
func TestWithDegradedNoGhostFlag(t *testing.T) {
	e, err := WithDegraded(map[string]any{"id": 1}, false, "")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(e)
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["degraded"]; ok {
		t.Fatalf("healthy envelope must omit the degraded flag: %s", b)
	}
	if _, ok := m["degraded_reason"]; ok {
		t.Fatalf("healthy envelope must omit the degraded_reason: %s", b)
	}
}

// TestWithDegradedCarriesReason pins the degraded contract: a degraded
// result carries the flag + a non-empty reason (INV-3: never a silent
// floor).
func TestWithDegradedCarriesReason(t *testing.T) {
	e, err := WithDegraded(map[string]any{"id": 1}, true, "llm seam absent")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(e)
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if m["degraded"] != true || m["degraded_reason"] == "" {
		t.Fatalf("degraded envelope wrong: %s", b)
	}
}

// TestWithDegradedNormalizesKeys pins the snake_case normalization (the
// D1 wire contract): a CamelCase result is normalized to snake_case keys.
func TestWithDegradedNormalizesKeys(t *testing.T) {
	e, err := WithDegraded(map[string]any{"SyncID": "obs-1", "RevisionCount": 2}, false, "")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(e)
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["sync_id"]; !ok {
		t.Fatalf("SyncID must normalize to sync_id: %s", b)
	}
	if _, ok := m["revision_count"]; !ok {
		t.Fatalf("RevisionCount must normalize to revision_count: %s", b)
	}
}
