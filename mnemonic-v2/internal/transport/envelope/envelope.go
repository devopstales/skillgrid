// Package envelope is the D1-pinned response envelope (spec
// 004-transport-mcp §4, INV-3): every tool result is a JSON object that
// carries the result fields plus, when the tool ran degraded, a
// `degraded: true` flag and a non-empty `degraded_reason`. Never a
// silent floor; never a ghost flag.
//
// This is the one shared shape the MCP adapter (Task 13) and the CLI
// adapter (Task 14) both marshal — the K6 byte-identical seam. It knows
// nothing about MCP or CLI; it is a pure value type + a mapping function.
package envelope

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/memory"
)

// Error codes (the (code, message) pair the handler wraps every error
// into). The codes are the D1 contract: the plugins switch on them, so
// they are literal strings, not Go errors.
const (
	CodeInvalidArguments     = "invalid_arguments"
	CodeOperationConflict    = "operation_conflict"    // 409
	CodeOperationExpired     = "operation_expired"     // 410
	CodeSessionAlreadyEnded  = "session_already_ended" // 409
	CodeFingerprintMismatch  = "fingerprint_mismatch"
	CodeConsolidationRefused = "consolidation_refused"
	CodeProjectMismatch      = "project_mismatch"
	CodeNotFound             = "not_found"
	CodeInternal             = "internal_error"
)

// MapError maps a service/store error onto its pinned (code, message).
// Sentinel errors map to their code; anything else is an internal error
// (the Go string is still carried in the message for debuggability, but
// the code never leaks an ad-hoc string).
func MapError(err error) (code, message string) {
	if err == nil {
		return "", ""
	}
	switch {
	case errors.Is(err, memory.ErrOperationConflict):
		return CodeOperationConflict, err.Error()
	case errors.Is(err, memory.ErrOperationExpired):
		return CodeOperationExpired, err.Error()
	case errors.Is(err, memory.ErrSessionEnded), errors.Is(err, store.ErrSessionEnded):
		return CodeSessionAlreadyEnded, err.Error()
	case errors.Is(err, memory.ErrSessionProjectMismatch), errors.Is(err, store.ErrSessionProjectMismatch):
		return CodeProjectMismatch, err.Error()
	case errors.Is(err, memory.ErrConsolidateFingerprintMismatch):
		return CodeFingerprintMismatch, err.Error()
	case errors.Is(err, memory.ErrConsolidateRefused):
		return CodeConsolidationRefused, err.Error()
	case errors.Is(err, store.ErrObservationNotFound), isNotFound(err):
		return CodeNotFound, err.Error()
	case isValidationError(err):
		return CodeInvalidArguments, err.Error()
	default:
		return CodeInternal, err.Error()
	}
}

// isNotFound recognizes the store's not-found family: sql.ErrNoRows (the
// store's GetObservation returns it raw on a missing id) and the store's
// formatted "observation N not found" (the Delete/Update ownership
// assertion's missing-row path).
func isNotFound(err error) bool {
	if errors.Is(err, errNoRows) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "no rows") || strings.Contains(msg, "not found")
}

// isValidationError recognizes the pipeline's validation errors: they are
// plain errors.New with the "memory: … required / unknown …" shape, not
// sentinels. The prefix + the two shapes are the contract; a future
// sentinel (memory.ErrInvalidArguments) would replace this check.
func isValidationError(err error) bool {
	msg := err.Error()
	if !strings.HasPrefix(msg, "memory:") {
		return false
	}
	return strings.Contains(msg, " is required") ||
		strings.Contains(msg, "unknown observation type") ||
		strings.Contains(msg, "unknown scope") ||
		strings.Contains(msg, "must be kebab-case")
}

// errNoRows is the store's no-rows sentinel (database/sql.ErrNoRows — the
// store's GetObservation returns it raw on a missing id).
var errNoRows = sql.ErrNoRows

// WithDegraded merges a service result with the degradation state into the
// D1 envelope shape: the result's fields plus `degraded`/`degraded_reason`
// (both omitempty, so a healthy response carries neither — no ghost flag).
// A non-nil reason with a false flag is a bug: the reason is dropped, never
// half-carried.
//
// The result's keys are normalized to snake_case (the D1 wire format — the
// K6 byte-identical contract is the JSON text, which is snake_case per spec
// 004-transport-mcp). The service's Go types use CamelCase; the envelope is
// the one place that maps them onto the wire shape.
func WithDegraded(result any, degraded bool, reason string) (map[string]any, error) {
	b, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, err
	}
	// Normalize the keys to the D1 wire shape (snake_case) — the Go
	// types are CamelCase, the wire is snake_case (spec 004-transport-mcp).
	// A result that already carries a `degraded` key (the service's result
	// types do) is left in place — the flag is the contract, not a bug.
	v = snakeKeys(v)
	m, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("envelope: result is not a JSON object: %s", b)
	}
	if m == nil {
		m = map[string]any{}
	}
	// A result that already carries a degraded flag (the service's result
	// types — SaveResult.Degraded, SearchResult.Degraded) is the contract:
	// the flag is present, but the reason is not (the service does not
	// carry one). When the flag is present and the reason is empty, the
	// envelope supplies the default reason (INV-3: never a silent floor).
	if d, _ := m["degraded"].(bool); d {
		if _, has := m["degraded_reason"]; !has || m["degraded_reason"] == "" {
			if reason == "" {
				reason = "intelligence seam absent (deterministic floor ran)"
			}
			m["degraded_reason"] = reason
		}
	} else if degraded && reason != "" {
		m["degraded"] = true
		m["degraded_reason"] = reason
	}
	return m, nil
}

// snakeKeys recursively normalizes a JSON value's object keys to snake_case
// (the Go CamelCase → wire snake_case mapping). Non-object values pass
// through; nested objects/arrays are normalized recursively.
func snakeKeys(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			out[toSnake(k)] = snakeKeys(val)
		}
		return out
	case []any:
		for i := range t {
			t[i] = snakeKeys(t[i])
		}
		return t
	default:
		return v
	}
}

// acronymMap are the wire-name overrides for the Go type's acronym
// fields (the D1 contract: SyncID → sync_id, ID → id, JudgmentID →
// judgment_id). An acronym is a run of ≥2 capitals; the generic toSnake
// handles the rest.
var acronymMap = map[string]string{
	"SyncID":     "sync_id",
	"ID":         "id",
	"JudgmentID": "judgment_id",
	"APIKey":     "api_key",
	"DSN":        "dsn",
	"URL":        "url",
}

// toSnake maps a CamelCase key to its snake_case wire form (the D1
// contract: SyncID → sync_id, MergedIntoSyncID → merged_into_sync_id,
// JudgmentRequired → judgment_required). An acronym is a run of ≥2
// capitals; the underscore goes at the run's end (before the following
// lowercase letter), not inside it.
func toSnake(s string) string {
	if v, ok := acronymMap[s]; ok {
		return v
	}
	var b []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			if i > 0 {
				// A capital preceded by a lowercase → a new word.
				if s[i-1] >= 'a' && s[i-1] <= 'z' {
					b = append(b, '_')
				} else if s[i-1] >= 'A' && s[i-1] <= 'Z' && i+1 < len(s) && s[i+1] >= 'a' && s[i+1] <= 'z' {
					// The end of an acronym run (e.g. the 'I' of 'ID' in
					// 'SyncID'): the underscore goes after the run.
					b = append(b, '_')
				}
			}
			b = append(b, c+('a'-'A'))
		} else {
			b = append(b, c)
		}
	}
	return string(b)
}
