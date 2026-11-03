package mcp

import (
	"encoding/json"
	"errors"
	"strings"
	"unicode"

	mcplib "github.com/mark3labs/mcp-go/mcp"
)

// JSONResult marshals v to raw JSON text with no preamble (OCBI convention).
func JSONResult(v any) (*mcplib.CallToolResult, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	text := string(b)
	if err := ValidateRawJSON(text); err != nil {
		return nil, err
	}
	return mcplib.NewToolResultText(text), nil
}

// ValidateRawJSON ensures tool output is JSON only — no leading prose.
func ValidateRawJSON(text string) error {
	trimmed := strings.TrimLeftFunc(text, unicode.IsSpace)
	if trimmed == "" {
		return errors.New("empty response")
	}
	first := trimmed[0]
	if first != '{' && first != '[' {
		return errors.New("response must start with JSON object or array")
	}
	if !json.Valid([]byte(trimmed)) {
		return errors.New("invalid JSON")
	}
	return nil
}

// IsRawJSON reports whether text is valid raw JSON with no leading prose.
func IsRawJSON(text string) bool {
	return ValidateRawJSON(text) == nil
}
