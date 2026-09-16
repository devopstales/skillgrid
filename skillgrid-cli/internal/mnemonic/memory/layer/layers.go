package layer

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
)

// Layer is one link in the L0→L1→L2→L3 chain as surfaced by Inspect. It carries
// the layer's provenance link to its L0 source (SourceSession + SourceTopic),
// so a reader can verify every distilled record traces back to a resolvable
// session record.
type Layer struct {
	Layer       string `json:"layer"`
	TargetKind  string `json:"target_kind"`
	TargetID    int64  `json:"target_id"`
	Title       string `json:"title,omitempty"`
	Content     string `json:"content,omitempty"`
	SourceSession string `json:"source_session"`
	SourceTopic string `json:"source_topic,omitempty"`
}

// Chain is the result of Inspect: the full L0→L1→L2→L3 ladder for a session,
// with each layer's provenance link. L0Session is the raw session id; L0Title is
// the session's title (when set) so the L0 is human-readable.
type Chain struct {
	L0Session string   `json:"l0_session"`
	L0Title   string   `json:"l0_title,omitempty"`
	L0Summary string   `json:"l0_summary,omitempty"`
	Atoms     []Layer  `json:"atoms,omitempty"` // L1
	Scenarios []Layer  `json:"scenarios,omitempty"` // L2
	Persona   []Layer  `json:"persona,omitempty"` // L3
}

// Inspect returns the L0→L1→L2→L3 chain for a session, with each layer's
// provenance link. It is the read path behind mem_layers. A session with no
// distilled layers returns a chain with only the L0 populated (not an error).
func Inspect(ctx context.Context, svc *memory.Service, sessionID string) (Chain, error) {
	var c Chain
	if svc == nil {
		return c, fmt.Errorf("memory service is nil")
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return c, fmt.Errorf("session_id is required")
	}
	var title, summary sql.NullString
	err := svc.DB().QueryRowContext(ctx, `
		SELECT title, summary FROM sessions WHERE id = ? AND project = ?`,
		sessionID, svc.ProjectID()).Scan(&title, &summary)
	if err != nil {
		if err == sql.ErrNoRows {
			return c, fmt.Errorf("session %s not found", sessionID)
		}
		return c, fmt.Errorf("load session: %w", err)
	}
	c.L0Session = sessionID
	if title.Valid {
		c.L0Title = title.String
	}
	if summary.Valid {
		c.L0Summary = summary.String
	}

	var links []Layer
	rows, err := svc.DB().QueryContext(ctx, `
		SELECT layer, target_kind, target_id, source_session, source_topic
		FROM observation_layers
		WHERE project = ? AND source_session = ?
		ORDER BY layer, target_id`,
		svc.ProjectID(), sessionID)
	if err != nil {
		return c, fmt.Errorf("query layers: %w", err)
	}
	for rows.Next() {
		var l Layer
		var topic sql.NullString
		if err := rows.Scan(&l.Layer, &l.TargetKind, &l.TargetID, &l.SourceSession, &topic); err != nil {
			rows.Close()
			return c, fmt.Errorf("scan layer: %w", err)
		}
		if topic.Valid {
			l.SourceTopic = topic.String
		}
		links = append(links, l)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return c, fmt.Errorf("iterate layers: %w", err)
	}
	rows.Close()

	// Enrich each link with its target's title/content. Done after the result
	// set is fully drained so the open query does not block the same-conn reads.
	for i := range links {
		l := &links[i]
		if l.TargetKind == "observation" {
			var oTitle, oContent string
			if err := svc.DB().QueryRowContext(ctx, `
				SELECT title, content FROM observations
				WHERE id = ? AND project = ? AND deleted_at IS NULL`,
				l.TargetID, svc.ProjectID()).Scan(&oTitle, &oContent); err == nil {
				l.Title = oTitle
				l.Content = oContent
			}
		} else {
			var pTitle, pContent string
			if err := svc.DB().QueryRowContext(ctx, `
				SELECT title, content FROM personas WHERE id = ? AND project = ?`,
				l.TargetID, svc.ProjectID()).Scan(&pTitle, &pContent); err == nil {
				l.Title = pTitle
				l.Content = pContent
			}
		}
		switch l.Layer {
		case "L1":
			c.Atoms = append(c.Atoms, *l)
		case "L2":
			c.Scenarios = append(c.Scenarios, *l)
		case "L3":
			c.Persona = append(c.Persona, *l)
		}
	}
	return c, nil
}

// InspectByTopic returns the chain for a topic: it finds the session(s) whose
// distilled L0 source-topic matches the given topic and returns the first such
// chain. It is the alternative mem_layers entry (by topic_key instead of
// session_id).
func InspectByTopic(ctx context.Context, svc *memory.Service, topic string) (Chain, error) {
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return Chain{}, fmt.Errorf("topic is required")
	}
	var sessionID string
	err := svc.DB().QueryRowContext(ctx, `
		SELECT source_session FROM observation_layers
		WHERE project = ? AND source_topic LIKE ?
		ORDER BY id LIMIT 1`, svc.ProjectID(), "%"+topic+"%").Scan(&sessionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return Chain{}, fmt.Errorf("no layers for topic %q", topic)
		}
		return Chain{}, fmt.Errorf("find session by topic: %w", err)
	}
	return Inspect(ctx, svc, sessionID)
}
