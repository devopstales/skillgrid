package codeindex

import (
	"database/sql"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// Edge is one stored graph edge with its temporal window (change 014, step 10).
// ValidFrom is the UNIX second the relationship was first observed; ValidTo is
// NULL (0 here) while the edge is still active, or a past UNIX second once it
// has expired. Status is a computed label (TemporalStatus) so callers do not
// re-derive the window logic.
type Edge struct {
	ID        int64
	Kind      string
	FromID    int64
	FileID    int64
	ToID      sql.NullInt64
	ToName    string
	Confidence string
	Line      int
	ValidFrom int64
	ValidTo   int64 // 0 = NULL (active); otherwise a UNIX second
	Status    string // "active" | "expired" | "pending"
}

// TemporalStatus classifies an edge's window relative to now:
//   - "expired": valid_to <= now  (the relationship has ended)
//   - "pending": valid_from > now  (the relationship has not begun)
//   - "active":  valid_from <= now AND (valid_to == 0 OR valid_to > now)
func TemporalStatus(validFrom, validTo, now int64) string {
	if validTo != 0 && validTo <= now {
		return "expired"
	}
	if validFrom > now {
		return "pending"
	}
	return "active"
}

// QueryEdges returns every edge in the CURRENT graph state: it applies the
// temporal filter so expired and pending edges are hidden. This is the query
// path used for current graph traversal (014 step 10.2). Edges that do not
// meet the window are preserved in the table and reachable via
// QueryEdgesWithHistory.
func QueryEdges(st *store.Store) ([]Edge, error) {
	now := time.Now().Unix()
	rows, err := st.DB.Query(`
		SELECT e.id, e.kind, e.from_id, e.file_id, e.to_id, e.to_name,
		       e.confidence, e.line, e.valid_from, e.valid_to
		FROM edges e
		WHERE e.valid_from <= ? AND (e.valid_to IS NULL OR e.valid_to > ?)
		ORDER BY e.id`, now, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEdges(rows, now)
}

// QueryEdgesWithHistory returns EVERY edge, including expired and pending
// ones (no temporal filter). This is the history read (014 step 10.2): the
// full temporal record, for auditing and for the graph CLI's status display.
func QueryEdgesWithHistory(st *store.Store) ([]Edge, error) {
	now := time.Now().Unix()
	rows, err := st.DB.Query(`
		SELECT e.id, e.kind, e.from_id, e.file_id, e.to_id, e.to_name,
		       e.confidence, e.line, e.valid_from, e.valid_to
		FROM edges e
		ORDER BY e.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEdges(rows, now)
}

// QueryEdgesForSymbol returns the edges touching sym.ID in the current graph
// state (the temporal filter applies), used for per-symbol neighbor traversal.
func QueryEdgesForSymbol(st *store.Store, symID int64) ([]Edge, error) {
	now := time.Now().Unix()
	rows, err := st.DB.Query(`
		SELECT e.id, e.kind, e.from_id, e.file_id, e.to_id, e.to_name,
		       e.confidence, e.line, e.valid_from, e.valid_to
		FROM edges e
		WHERE (e.from_id = ? OR e.to_id = ?)
		  AND e.valid_from <= ? AND (e.valid_to IS NULL OR e.valid_to > ?)
		ORDER BY e.id`, symID, symID, now, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEdges(rows, now)
}

func scanEdges(rows *sql.Rows, now int64) ([]Edge, error) {
	var out []Edge
	for rows.Next() {
		var e Edge
		var toID sql.NullInt64
		var toName sql.NullString
		var validTo sql.NullInt64
		if err := rows.Scan(&e.ID, &e.Kind, &e.FromID, &e.FileID, &toID, &toName,
			&e.Confidence, &e.Line, &e.ValidFrom, &validTo); err != nil {
			return nil, err
		}
		e.ToID = toID
		e.ToName = toName.String
		if validTo.Valid {
			e.ValidTo = validTo.Int64
		}
		e.Status = TemporalStatus(e.ValidFrom, e.ValidTo, now)
		out = append(out, e)
	}
	return out, rows.Err()
}
