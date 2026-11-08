package mcp

// contract.go — the OpenWiki-shaped wire contract for every MCP tool.
//
// Replication target (spec 011-openwiki-port, the OpenWiki MCP server):
// every tool serves a $schema-identified, closed (additionalProperties:
// false), constraint-carrying input schema, plus
// execution.taskSupport="forbidden", and carries no per-tool annotation
// hints (the MCP spec's assumed defaults). finalizeMCPContract is the
// single choke point — Server.registerTool applies it to every tool
// definition of every family and profile; the re-registration path
// (setInstructions) re-serves the same finalized definitions. The server
// enforces the schema at call time (server.WithInputSchemaValidation):
// a schema violation is a SEP-1303 tool execution error the agent can
// read and correct, never a silent drop.

import (
	mcplib "github.com/mark3labs/mcp-go/mcp"
)

// schemaDialect is the input schema's declared dialect (OpenWiki's SDK
// stamps the same draft-07 URI on every tool).
const schemaDialect = "http://json-schema.org/draft-07/schema#"

// stringMaxLen caps identifier-like string parameters. The server
// enforces these at call time (WithInputSchemaValidation), so the caps
// are superset-safe: a working call never sends a longer identifier.
// Content-carrying parameters (content, text, judgments, ids_to_delete,
// paths, base, changed, current_prompt, summary sections) are
// deliberately uncapped.
var stringMaxLen = map[string]int{
	"query":            2000,
	"pattern":          2000,
	"source_url":       1000,
	"symbol":           500,
	"path":             500,
	"page":             500,
	"file":             500,
	"package":          500,
	"from":             500,
	"to":               500,
	"section":          500,
	"called_from":      500,
	"phrase":           500,
	"topic_key":        500,
	"session_id":       500,
	"sync_id":          500,
	"id":               500,
	"project":          500,
	"expected_project": 500,
	"name":             500,
	"scope":            100,
	"type":             100,
	"mode":             100,
	"match_mode":       100,
	"language":         100,
	"relation":         100,
	"min_confidence":   100,
	"source_commit":    64,
	"version":          64,
}

// numberBounds gives the numeric parameters their floors and ceilings
// (max 0 = no ceiling). limit's ceiling (50) is a superset of every tool
// default (the widest is projectwiki_search's 25). budget's ceiling is
// its documented max.
var numberBounds = map[string]struct {
	min   any
	max   int
	isInt bool
}{
	"limit":       {min: 1, max: 50, isInt: true},
	"depth":       {min: 1, max: 10, isInt: true},
	"max_depth":   {min: 1, max: 10, isInt: true},
	"budget":      {min: 1, max: 65536, isInt: true},
	"id":          {min: 1, max: 0, isInt: true},
	"a_id":        {min: 1, max: 0, isInt: true},
	"b_id":        {min: 1, max: 0, isInt: true},
	"judgment_id": {min: 1, max: 0, isInt: true},
	"start_line":  {min: 1, max: 0, isInt: true},
	"end_line":    {min: 1, max: 0, isInt: true},
	"confidence":  {min: 0, max: 1},
}

// finalizeMCPContract brings one tool definition up to the OpenWiki-shaped
// wire contract:
//   - inputSchema.$schema = draft-07 (declared, as OpenWiki's SDK does);
//   - inputSchema.additionalProperties = false (closed — an undeclared
//     argument is a call error the agent can read, not a silent drop);
//   - per-property constraints (required strings minLength 1; bounded
//     identifiers maxLength; integer-typed counters with floors/ceilings);
//   - execution.taskSupport = "forbidden" (every tool is a synchronous
//     request; none is a long-running task — OpenWiki emits this on all
//     of its tools);
//   - annotations reduced to the spec defaults (all hints nil: the MCP
//     spec's assumed defaults are exactly mcp-go's NewTool hard-coded
//     values — mcp-go emits them explicitly on every tool, OpenWiki's
//     SDK omits the hints entirely; nil claims nothing per tool).
//
// Idempotent: re-finalizing a finalized definition is a no-op.
func finalizeMCPContract(t mcplib.Tool) mcplib.Tool {
	t.Execution = &mcplib.ToolExecution{TaskSupport: mcplib.TaskSupportForbidden}
	t.Annotations = mcplib.ToolAnnotation{}
	sch := t.InputSchema
	if sch.AdditionalFields == nil {
		sch.AdditionalFields = map[string]any{}
	}
	sch.AdditionalFields["$schema"] = schemaDialect
	sch.AdditionalProperties = false
	required := make(map[string]bool, len(sch.Required))
	for _, name := range sch.Required {
		required[name] = true
	}
	for name, propAny := range sch.Properties {
		prop, ok := propAny.(map[string]any)
		if !ok {
			continue
		}
		applyPropertyConstraints(name, required[name], prop)
	}
	t.InputSchema = sch
	return t
}

// applyPropertyConstraints adds the name-derived constraints to one
// property schema (in place). The validator enforces them at call time
// (WithInputSchemaValidation), so every bound is superset-safe against
// the documented behavior.
func applyPropertyConstraints(name string, isRequired bool, prop map[string]any) {
	switch prop["type"] {
	case "string":
		if isRequired {
			prop["minLength"] = 1
		}
		if max, ok := stringMaxLen[name]; ok {
			prop["maxLength"] = max
		}
	case "number":
		b, ok := numberBounds[name]
		if !ok {
			return
		}
		if b.isInt {
			prop["type"] = "integer"
		}
		prop["minimum"] = b.min
		if b.max > 0 {
			prop["maximum"] = b.max
		}
	}
}
