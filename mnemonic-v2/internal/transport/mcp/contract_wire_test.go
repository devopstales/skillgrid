package mcp

// contract_wire_test.go — the OpenWiki-shaped wire contract (contract.go)
// pinned at the serialization boundary: every registered tool, every
// profile, serves the $schema-identified closed schema + constraints +
// execution.taskSupport + spec-default annotations, and the server
// enforces the schema at call time (a SEP-1303 tool execution error the
// agent can read, not a silent drop).

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// wireTool is the client-visible shape of one registered tool
// (the subset of the MCP tools/list item this contract pins).
type wireTool struct {
	InputSchema struct {
		Schema               string         `json:"$schema"`
		Type                 string         `json:"type"`
		AdditionalProperties any            `json:"additionalProperties"`
		Properties           map[string]any `json:"properties"`
		Required             []string       `json:"required"`
	} `json:"inputSchema"`
	Execution *struct {
		TaskSupport string `json:"taskSupport"`
	} `json:"execution"`
	Annotations struct {
		ReadOnlyHint    *bool `json:"readOnlyHint"`
		DestructiveHint *bool `json:"destructiveHint"`
		IdempotentHint  *bool `json:"idempotentHint"`
		OpenWorldHint   *bool `json:"openWorldHint"`
	} `json:"annotations"`
}

// wireShape parses one registered tool definition the way a client sees
// it (the marshaled tool, not the Go struct — the struct's fields are an
// implementation detail the contract must not depend on).
func wireShape(t *testing.T, def any) wireTool {
	t.Helper()
	raw, err := json.Marshal(def)
	if err != nil {
		t.Fatalf("marshal tool: %v", err)
	}
	var w wireTool
	if err := json.Unmarshal(raw, &w); err != nil {
		t.Fatalf("unmarshal tool: %v", err)
	}
	return w
}

// TestEveryToolServesTheOpenWikiWireContract pins the replication target
// over the COMPLETE registered surface (memory agent+admin profile, the
// code family, the projectwiki family): $schema draft-07, closed schema,
// execution.taskSupport=forbidden, no per-tool hint claims, and
// minLength 1 on every required string.
func TestEveryToolServesTheOpenWikiWireContract(t *testing.T) {
	s, _, _ := newProjectWikiFixture(t, All)
	s.RegisterCodeTools(newTestCodeHandler(t))

	if len(s.toolDefs) == 0 {
		t.Fatal("no tools registered")
	}
	for name, def := range s.toolDefs {
		t.Run(name, func(t *testing.T) {
			w := wireShape(t, def)
			if w.InputSchema.Schema != "http://json-schema.org/draft-07/schema#" {
				t.Errorf("$schema = %q, want draft-07 URI", w.InputSchema.Schema)
			}
			if w.InputSchema.Type != "object" {
				t.Errorf("type = %q, want object", w.InputSchema.Type)
			}
			if w.InputSchema.AdditionalProperties != false {
				t.Errorf("additionalProperties = %v, want false (closed)", w.InputSchema.AdditionalProperties)
			}
			if w.Execution == nil || w.Execution.TaskSupport != "forbidden" {
				t.Errorf("execution = %+v, want taskSupport forbidden", w.Execution)
			}
			if w.Annotations.ReadOnlyHint != nil || w.Annotations.DestructiveHint != nil ||
				w.Annotations.IdempotentHint != nil || w.Annotations.OpenWorldHint != nil {
				t.Errorf("annotations carry per-tool claims, want the spec defaults (all nil): %+v", w.Annotations)
			}
			required := make(map[string]bool, len(w.InputSchema.Required))
			for _, n := range w.InputSchema.Required {
				required[n] = true
			}
			for pname, pany := range w.InputSchema.Properties {
				p, ok := pany.(map[string]any)
				if !ok {
					continue
				}
				if required[pname] && p["type"] == "string" && p["minLength"] != float64(1) {
					t.Errorf("required string %s lacks minLength 1: %v", pname, p)
				}
			}
		})
	}
}

// TestConstraintSpotChecks pins the named bounds the OpenWiki replication
// targets (the values a working call must never cross — the server
// enforces them, so the pins double as the ceiling contract):
// limit 1..50 (a superset of every tool default), the identifiers'
// maxLength, budget's documented max, confidence 0..1, integer types.
func TestConstraintSpotChecks(t *testing.T) {
	s, _, _ := newProjectWikiFixture(t, Agent)
	s.RegisterCodeTools(newTestCodeHandler(t))

	prop := func(tool, pname string) map[string]any {
		t.Helper()
		w := wireShape(t, s.toolDefs[tool])
		p, ok := w.InputSchema.Properties[pname].(map[string]any)
		if !ok {
			t.Fatalf("%s has no %s property", tool, pname)
		}
		return p
	}

	for _, tool := range []string{"mem_search", "code_search", "projectwiki_query", "projectwiki_search"} {
		p := prop(tool, "limit")
		if p["type"] != "integer" || p["minimum"] != float64(1) || p["maximum"] != float64(50) {
			t.Errorf("%s.limit = %v, want integer 1..50", tool, p)
		}
	}
	if p := prop("mem_context", "budget"); p["type"] != "integer" || p["maximum"] != float64(65536) {
		t.Errorf("mem_context.budget = %v, want integer, max 65536", p)
	}
	if p := prop("mem_judge", "confidence"); p["type"] != "number" || p["minimum"] != float64(0) || p["maximum"] != float64(1) {
		t.Errorf("mem_judge.confidence = %v, want number 0..1", p)
	}
	if p := prop("projectwiki_query", "query"); p["minLength"] != float64(1) || p["maxLength"] != float64(2000) {
		t.Errorf("projectwiki_query.query = %v, want minLength 1 maxLength 2000", p)
	}
	if p := prop("code_read", "path"); p["minLength"] != float64(1) || p["maxLength"] != float64(500) {
		t.Errorf("code_read.path = %v, want minLength 1 maxLength 500", p)
	}
	// empty-string-means-default parameters stay open (no minLength):
	// projectwiki_read.section (empty = the full body),
	// projectwiki_validate.paths (empty = every grounded page).
	if p := prop("projectwiki_read", "section"); p["minLength"] != nil {
		t.Errorf("projectwiki_read.section = %v, must stay open (empty = full body)", p)
	}
	if p := prop("projectwiki_validate", "paths"); p["minLength"] != nil {
		t.Errorf("projectwiki_validate.paths = %v, must stay open (empty = all grounded pages)", p)
	}
}

// stdioToolsCall sends one raw tools/call over the mcp-go stdio path
// (initialize first, the way HandshakeAndCall does) and returns the
// result object of the response.
func stdioToolsCall(t *testing.T, s *Server, name string, args string) map[string]any {
	t.Helper()
	ctx := context.Background()
	initReq := `{"jsonrpc":"2.0","id":0,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}`
	s.mcp.HandleMessage(ctx, json.RawMessage(initReq))
	req := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":%q,"arguments":%s}}`, name, args)
	resp := s.mcp.HandleMessage(ctx, json.RawMessage(req))
	if resp == nil {
		t.Fatalf("no response for %s", name)
	}
	out, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	var rpc struct {
		Result map[string]any `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(out, &rpc); err != nil {
		t.Fatalf("unmarshal response: %v (%s)", err, out)
	}
	if rpc.Error != nil {
		t.Fatalf("protocol-level error for %s: %s", name, rpc.Error.Message)
	}
	return rpc.Result
}

func callResultText(t *testing.T, result map[string]any) (isError bool, text string) {
	t.Helper()
	isError, _ = result["isError"].(bool)
	content, _ := result["content"].([]any)
	for _, c := range content {
		if m, ok := c.(map[string]any); ok {
			text, _ = m["text"].(string)
			break
		}
	}
	return
}

// TestInputSchemaEnforcedOverStdio proves the contract is not
// decorative: the server validates call arguments against the declared
// schema and a violation is a SEP-1303 tool execution error (isError +
// the property named in the message), while a conforming call reaches
// the handler.
func TestInputSchemaEnforcedOverStdio(t *testing.T) {
	s := NewServer(newTestHandler(t), Agent)
	t.Cleanup(s.Close)

	cases := []struct {
		name    string
		args    string
		wantErr bool
		wantSub string
	}{
		{"mem_search", `{"query":"x","limit":0}`, true, "limit"},
		{"mem_search", `{"query":"x","limit":100}`, true, "limit"},
		{"mem_search", `{"query":"","limit":1}`, true, "query"},
		{"mem_search", `{"query":"x","bogus":1}`, true, "bogus"},
		{"mem_context", `{"budget":70000}`, true, "budget"},
		{"mem_search", `{"query":"x","limit":10}`, false, ""},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%s_%s", c.name, c.args), func(t *testing.T) {
			result := stdioToolsCall(t, s, c.name, c.args)
			isError, text := callResultText(t, result)
			if c.wantErr {
				if !isError {
					t.Fatalf("expected a validation error, got success: %s", text)
				}
				if text == "" {
					t.Fatal("validation error carries no message")
				}
				if !strings.Contains(text, "input schema validation failed") {
					t.Errorf("not a schema validation error: %s", text)
				}
				if !strings.Contains(text, c.wantSub) {
					t.Errorf("error does not name the offending property %q: %s", c.wantSub, text)
				}
			} else if isError {
				t.Fatalf("conforming call rejected: %s", text)
			}
		})
	}
}
