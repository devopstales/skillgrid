package http

import "encoding/json"

// decisionRow is the RED-phase forward declaration of the decisions list row
// (TICKET-01). TICKET-02 replaces this stub with the full type + parseDecision
// implementation in decisions.go and removes this file.
type decisionRow struct {
	ID         int64           `json:"id"`
	TopicKey   string          `json:"topicKey"`
	Title      string          `json:"title"`
	CreatedAt  string          `json:"createdAt"`
	UpdatedAt  string          `json:"updatedAt"`
	Visibility string          `json:"visibility"`
	Content    decisionContent `json:"content"`
	ParseError bool            `json:"parseError,omitempty"`
}

type decisionOption struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Rationale string `json:"rationale,omitempty"`
}

type decisionContent struct {
	Question       string           `json:"question"`
	Options        []decisionOption `json:"options"`
	Recommended    string           `json:"recommended,omitempty"`
	State          string           `json:"state"`
	AnsweredOption string           `json:"answeredOption,omitempty"`
	AnswerNote     string           `json:"answerNote,omitempty"`
	UpdatedBy      string           `json:"updatedBy,omitempty"`
	Visual         string           `json:"visual,omitempty"`
}

// parseDecision is the RED-phase placeholder (always false: content is not
// parsed yet, so every listed row is flagged parseError). TICKET-02 replaces
// it with the real parseDecisionContent in decisions.go.
func parseDecision(raw string) (decisionContent, bool) {
	var c decisionContent
	if err := json.Unmarshal([]byte(raw), &c); err != nil || c.Question == "" {
		return decisionContent{}, false
	}
	_ = c
	return decisionContent{}, false
}
