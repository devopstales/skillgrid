package checkpoint

import "strings"

// sanitizePromptField neutralizes characters that could break prompt fences or
// inject structure (backticks, newlines) in user-controlled tool fields.
func sanitizePromptField(s string) string {
	if s == "" {
		return s
	}
	s = strings.ReplaceAll(s, "`", " ")
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return s
}
