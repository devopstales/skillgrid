package main

import (
	"database/sql"
	"fmt"
	"sort"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/hybrid"
)

// RedactionState is the doctor --strict redaction check result (01.4): whether
// any indexed file carries a secret-like pattern (a redaction violation), and
// which files. Output-time redaction (01.3) prevents these from shipping raw,
// but a strict CI run flags their presence so the user knows redaction is
// actively working.
type RedactionState struct {
	Violation bool
	Files     []string
}

// FreshnessState is the doctor --strict freshness check result (01.4): whether
// the index is older than the max age (a freshness violation), with the last
// indexed time and the max age.
type FreshnessState struct {
	Violation     bool
	LastIndexed   string
	MaxAge        time.Duration
	StaleSeconds  int64
}

// DoctorStrictReport is the aggregate doctor --strict state (01.4): the
// redaction + freshness checks. CI-usable — runDoctor exits non-zero when any
// check reports a violation.
type DoctorStrictReport struct {
	Redaction  RedactionState
	Freshness  FreshnessState
}

// Clean reports whether there are no redaction or freshness violations.
func (r DoctorStrictReport) Clean() bool {
	return !r.Redaction.Violation && !r.Freshness.Violation
}

// defaultStrictMaxAge is the index-freshness threshold for doctor --strict.
const defaultStrictMaxAge = 24 * time.Hour

// runDoctorStrictChecks scans the store for the two strict checks:
//
//   - redaction: every indexed chunk is run through the output-time redaction
//     patterns; a chunk whose text changes (a secret is present) flags its file
//     as a redaction violation.
//   - freshness: the newest indexed_at across files is compared to maxAge; an
//     index older than maxAge is a freshness violation.
//
// An empty index is clean (nothing to be stale about, no secrets).
func runDoctorStrictChecks(db *sql.DB, maxAge time.Duration) DoctorStrictReport {
	if maxAge <= 0 {
		maxAge = defaultStrictMaxAge
	}
	rep := DoctorStrictReport{
		Redaction: RedactionState{},
		Freshness: FreshnessState{MaxAge: maxAge},
	}

	// Redaction: scan chunks for secret-like patterns.
	rows, err := db.Query(`
		SELECT f.path, c.text FROM chunks c
		INNER JOIN files f ON f.id = c.file_id
		ORDER BY f.path, c.start_line`)
	if err != nil {
		return rep
	}
	defer rows.Close()
	offender := map[string]bool{}
	for rows.Next() {
		var path, text string
		if err := rows.Scan(&path, &text); err != nil {
			continue
		}
		if hybrid.RedactSecrets(text) != text {
			offender[path] = true
		}
	}
	_ = rows.Err()
	if len(offender) > 0 {
		files := make([]string, 0, len(offender))
		for p := range offender {
			files = append(files, p)
		}
		sort.Strings(files)
		rep.Redaction.Violation = true
		rep.Redaction.Files = files
	}

	// Freshness: the newest indexed_at must be within maxAge.
	var last sql.NullString
	if err := db.QueryRow(`SELECT MAX(indexed_at) FROM files`).Scan(&last); err != nil {
		return rep
	}
	if !last.Valid || last.String == "" {
		// No files indexed → nothing to be stale about (clean).
		return rep
	}
	rep.Freshness.LastIndexed = last.String
	lt, perr := time.Parse(time.RFC3339, last.String)
	if perr != nil {
		// Unparseable timestamp: treat as stale (a loud, conservative state).
		rep.Freshness.Violation = true
		rep.Freshness.StaleSeconds = -1
		return rep
	}
	age := time.Since(lt)
	if age > maxAge {
		rep.Freshness.Violation = true
		rep.Freshness.StaleSeconds = int64(age.Seconds())
	}
	return rep
}

// printStrictReport renders the doctor --strict state as human-readable lines.
func printStrictReport(rep DoctorStrictReport) {
	if rep.Redaction.Violation {
		fmt.Printf("  redaction: VIOLATION — %d indexed file(s) contain a secret-like pattern:\n", len(rep.Redaction.Files))
		for _, f := range rep.Redaction.Files {
			fmt.Printf("    %s\n", f)
		}
	} else {
		fmt.Println("  redaction: clean (no secret-like patterns in indexed files)")
	}
	if rep.Freshness.Violation {
		fmt.Printf("  freshness: VIOLATION — index last indexed %s (older than max age)\n", rep.Freshness.LastIndexed)
	} else if rep.Freshness.LastIndexed != "" {
		fmt.Printf("  freshness: clean (index last indexed %s)\n", rep.Freshness.LastIndexed)
	} else {
		fmt.Println("  freshness: clean (no index yet)")
	}
}
