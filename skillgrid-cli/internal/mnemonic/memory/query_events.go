package memory

import (
	"context"
	"encoding/json"
	"strings"
)

// QueryEventOpts filters a read over session_events.
type QueryEventOpts struct {
	Action           string
	File             string // LIKE pattern (e.g. "src/%")
	Since            string // RFC3339 lower bound
	Until            string // RFC3339 upper bound
	Session          string
	IncludeSensitive bool
	CountOnly        bool
}

// QueryEvents returns matching session_events (sequence order) + total count.
func (s *Service) QueryEvents(ctx context.Context, opts QueryEventOpts) ([]Event, int, error) {
	where, args := buildEventWhere(opts)
	count, err := s.countEvents(ctx, where, args)
	if err != nil {
		return nil, 0, err
	}
	if opts.CountOnly || count == 0 {
		return nil, count, nil
	}
	events, err := s.listEvents(ctx, where, args)
	return events, count, err
}

// ExportEvents returns JSONL (one JSON object per line) of matching events.
func (s *Service) ExportEvents(ctx context.Context, opts QueryEventOpts) (string, error) {
	events, _, err := s.QueryEvents(ctx, opts)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for i, e := range events {
		if i > 0 {
			b.WriteByte('\n')
		}
		raw, _ := json.Marshal(e)
		b.Write(raw)
	}
	return b.String(), nil
}

func buildEventWhere(opts QueryEventOpts) (string, []any) {
	var clauses []string
	var args []any
	if opts.Session != "" {
		clauses = append(clauses, "session_id = ?")
		args = append(args, opts.Session)
	}
	if opts.Action != "" {
		clauses = append(clauses, "action_type = ?")
		args = append(args, opts.Action)
	}
	if opts.File != "" {
		clauses = append(clauses, "path LIKE ?")
		args = append(args, opts.File)
	}
	if opts.Since != "" {
		clauses = append(clauses, "timestamp >= ?")
		args = append(args, opts.Since)
	}
	if opts.Until != "" {
		clauses = append(clauses, "timestamp <= ?")
		args = append(args, opts.Until)
	}
	if !opts.IncludeSensitive {
		clauses = append(clauses, "is_sensitive = 0")
	}
	where := ""
	if len(clauses) > 0 {
		where = " WHERE " + strings.Join(clauses, " AND ")
	}
	return where, args
}

func (s *Service) countEvents(ctx context.Context, where string, args []any) (int, error) {
	var n int
	err := s.store.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM session_events"+where, args...).Scan(&n)
	return n, err
}

func (s *Service) listEvents(ctx context.Context, where string, args []any) ([]Event, error) {
	rows, err := s.store.DB.QueryContext(ctx,
		`SELECT id, session_id, project, sequence, action_type,
		        COALESCE(result_status, 'success'), COALESCE(is_sensitive, 0),
		        COALESCE(tool_name, ''), COALESCE(path, ''), COALESCE(command, ''),
		        COALESCE("commit", ''), COALESCE(payload, ''), timestamp
		 FROM session_events`+where+`
		 ORDER BY session_id, sequence`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []Event
	for rows.Next() {
		var e Event
		var sensitive int
		if err := rows.Scan(&e.ID, &e.SessionID, &e.Project, &e.Sequence, &e.ActionType,
			&e.ResultStatus, &sensitive, &e.ToolName, &e.Path, &e.Command,
			&e.Commit, &e.Payload, &e.Timestamp); err != nil {
			return nil, err
		}
		e.IsSensitive = sensitive != 0
		events = append(events, e)
	}
	return events, rows.Err()
}
