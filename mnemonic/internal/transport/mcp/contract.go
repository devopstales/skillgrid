package mcp

// contract.go — the OpenWiki-shaped wire contract for every MCP tool.
//
// Replication target (spec 011-openwiki-port, the OpenWiki MCP server):
// every tool serves a $schema-identified, closed (additionalProperties:
// false), constraint-carrying input schema, plus
// execution.taskSupport="forbidden", and carries no per-tool annotation
// hints (the MCP spec's assumed defaults). finalizeMCPContract is the
// single choke point — Server.registerTool applies it to every tool
// definition of every family and profile, so the finalized definition
// is what the stdio path serves. The server
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

// toolMeta is the per-tool annotation surface (title, MCP hints,
// defer_loading). The mem_* values mirror the engram MCP response 1:1
// (the architecture source for the memory surface — the 2026-11-04
// blueprint; doctor's title drops the engram brand). code_* and
// projectwiki_* follow the same scheme: verb-first titles, correct
// per-tool hints (mcp-go's NewTool defaults — destructive on everything
// — are wrong for the read families), local-first openWorld=false
// (true only where a tool may shell out to an external LLM), and the
// high-frequency retrieval/save set loaded immediately while the
// low-frequency and admin tools defer_loading (engram's split pattern —
// hosts with bounded tool context can skip the tail until needed).
var toolMeta = map[string]struct {
	title      string
	readOnly   bool
	destruct   bool
	idempotent bool
	openWorld  bool
	deferLoad  bool
}{
	// memory family — mirrored from the engram MCP response (2026-11-08 dump)
	"mem_capture_passive":   {title: "Capture Learnings", idempotent: true, deferLoad: true},
	"mem_compare":           {title: "Compare Memory Pair (Persist Semantic Verdict)", idempotent: true},
	"mem_consolidate":       {title: "Consolidate Memories", destruct: true, deferLoad: true},
	"mem_context":           {title: "Get Memory Context", readOnly: true, idempotent: true},
	"mem_current_project":   {title: "Detect Current Project", readOnly: true, idempotent: true},
	"mem_delete":            {title: "Delete Memory", destruct: true, deferLoad: true},
	"mem_doctor":            {title: "Run Doctor", readOnly: true, idempotent: true, deferLoad: true},
	"mem_get_observation":   {title: "Get Observation", readOnly: true, idempotent: true},
	"mem_judge":             {title: "Judge Memory Conflict"},
	"mem_judge_many":        {title: "Judge Memory Relations in Batch", deferLoad: true},
	"mem_list_projects":     {title: "List Known Projects", readOnly: true, idempotent: true},
	"mem_lookup":            {title: "Look Up Memory", readOnly: true, idempotent: true},
	"mem_merge_projects":    {title: "Merge Projects", destruct: true, idempotent: true, deferLoad: true},
	"mem_pin":               {title: "Pin Memory", idempotent: true, deferLoad: true},
	"mem_review":            {title: "Review Memories", deferLoad: true},
	"mem_save":              {title: "Save Memory"},
	"mem_save_prompt":       {title: "Save User Prompt"},
	"mem_search":            {title: "Search Memory", readOnly: true, idempotent: true},
	"mem_search_prompts":    {title: "Search Stored Prompts", readOnly: true, idempotent: true},
	"mem_session_end":       {title: "End Session", idempotent: true, deferLoad: true},
	"mem_session_finalize":  {title: "Finalize Session", idempotent: true, deferLoad: true},
	"mem_session_start":     {title: "Start Session", idempotent: true, deferLoad: true},
	"mem_session_summary":   {title: "Save Session Summary"},
	"mem_stats":             {title: "Memory Stats", readOnly: true, idempotent: true, deferLoad: true},
	"mem_suggest_topic_key": {title: "Suggest Topic Key", readOnly: true, idempotent: true, deferLoad: true},
	"mem_timeline":          {title: "Memory Timeline", readOnly: true, idempotent: true, deferLoad: true},
	"mem_unpin":             {title: "Unpin Memory", idempotent: true, deferLoad: true},
	"mem_update":            {title: "Update Memory", deferLoad: true},
	// code family — the core navigation/retrieval set is eager; the
	// heavy and rarely needed tools defer.
	"code_affected":          {title: "Find Affected Tests", readOnly: true, idempotent: true, deferLoad: true},
	"code_circles":           {title: "Find Dependency Circles", readOnly: true, idempotent: true, deferLoad: true},
	"code_communities":       {title: "List Code Communities", readOnly: true, idempotent: true, deferLoad: true},
	"code_configs":           {title: "List Code Configs", readOnly: true, idempotent: true, deferLoad: true},
	"code_dep_impact":        {title: "Analyze Dependency Impact", readOnly: true, idempotent: true, deferLoad: true},
	"code_deps":              {title: "List Code Dependencies", readOnly: true, idempotent: true, deferLoad: true},
	"code_docs":              {title: "Read Code Docs", readOnly: true, idempotent: true, deferLoad: true},
	"code_embedding_status":  {title: "Embedding Status", readOnly: true, idempotent: true, deferLoad: true},
	"code_explain":           {title: "Explain Code Node", readOnly: true, idempotent: true, deferLoad: true},
	"code_explain_community": {title: "Explain Community", readOnly: true, idempotent: true, deferLoad: true},
	"code_god_nodes":         {title: "Find God Nodes", readOnly: true, idempotent: true, deferLoad: true},
	"code_get_callees":       {title: "Get Symbol Callees", readOnly: true, idempotent: true},
	"code_get_callers":       {title: "Get Symbol Callers", readOnly: true, idempotent: true},
	"code_get_dependents":    {title: "Get Symbol Dependents", readOnly: true, idempotent: true},
	"code_get_hierarchy":     {title: "Get Code Hierarchy", readOnly: true, idempotent: true, deferLoad: true},
	"code_get_implementors":  {title: "Get Implementors", readOnly: true, idempotent: true, deferLoad: true},
	"code_get_tests_for":     {title: "Get Symbol Tests", readOnly: true, idempotent: true, deferLoad: true},
	"code_grep":              {title: "Structural Code Grep", readOnly: true, idempotent: true, deferLoad: true},
	"code_impact":            {title: "Analyze Code Impact", readOnly: true, idempotent: true},
	"code_index":             {title: "Run Code Index", idempotent: true, openWorld: true, deferLoad: true},
	"code_navigates":         {title: "Find Code Navigations", readOnly: true, idempotent: true, deferLoad: true},
	"code_orient":            {title: "Orient on Code", readOnly: true, idempotent: true},
	"code_packages":          {title: "List Code Packages", readOnly: true, idempotent: true, deferLoad: true},
	"code_path":              {title: "Find Code Path", readOnly: true, idempotent: true, deferLoad: true},
	"code_pdg_query":         {title: "Query PDG", readOnly: true, idempotent: true, deferLoad: true},
	"code_process":           {title: "Explain Process", readOnly: true, idempotent: true, deferLoad: true},
	"code_processes":         {title: "List Code Processes", readOnly: true, idempotent: true, deferLoad: true},
	"code_read":              {title: "Read Code", readOnly: true, idempotent: true},
	"code_rename":            {title: "Rename Code Symbol", idempotent: true, deferLoad: true},
	"code_resolve":           {title: "Resolve Symbol", readOnly: true, idempotent: true},
	"code_route":             {title: "Route Code Request", readOnly: true, idempotent: true, deferLoad: true},
	"code_search":            {title: "Search Code", readOnly: true, idempotent: true},
	"code_semantic":          {title: "Semantic Code Search", readOnly: true, idempotent: true, deferLoad: true},
	"code_sql_schema":        {title: "Read SQL Schema", readOnly: true, idempotent: true, deferLoad: true},
	"code_status":            {title: "Code Index Status", readOnly: true, idempotent: true},
	"code_taint":             {title: "Taint Analysis", readOnly: true, idempotent: true, deferLoad: true},
	"code_unresolved":        {title: "List Unresolved Symbols", readOnly: true, idempotent: true, deferLoad: true},
	"code_watch":             {title: "Watch Code Changes", deferLoad: true},
	// projectwiki family — stage-1/2 retrieval + status eager; generation,
	// gates and admin defer.
	"projectwiki_bootstrap":    {title: "Bootstrap ProjectWiki", idempotent: true, deferLoad: true},
	"projectwiki_cite_source":  {title: "Cite ProjectWiki Source", idempotent: true, deferLoad: true},
	"projectwiki_generate":     {title: "Generate ProjectWiki", idempotent: true, openWorld: true, deferLoad: true},
	"projectwiki_index":        {title: "Index ProjectWiki", idempotent: true, deferLoad: true},
	"projectwiki_lint":         {title: "Lint ProjectWiki", idempotent: true, deferLoad: true},
	"projectwiki_query":        {title: "ProjectWiki Query", readOnly: true, idempotent: true},
	"projectwiki_read":         {title: "ProjectWiki Read", readOnly: true, idempotent: true},
	"projectwiki_rebuild_meta": {title: "Rebuild ProjectWiki Meta", idempotent: true, deferLoad: true},
	"projectwiki_reindex":      {title: "Reindex ProjectWiki", idempotent: true, deferLoad: true},
	"projectwiki_search":       {title: "ProjectWiki Search", readOnly: true, idempotent: true},
	"projectwiki_status":       {title: "ProjectWiki Status", readOnly: true, idempotent: true},
	"projectwiki_validate":     {title: "Validate ProjectWiki", readOnly: true, idempotent: true, deferLoad: true},
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
//   - annotations per the engram MCP response's pattern (the memory
//     surface's architecture source): a human-readable title, correct
//     per-tool hint values (mcp-go's NewTool defaults — destructive on
//     everything — are wrong for the read families), and defer_loading on
//     the low-frequency tail;
//   - openWorld=false everywhere (local-first server) except the tools
//     that may call an external LLM.
//
// Idempotent: re-finalizing a finalized definition is a no-op.
func finalizeMCPContract(t mcplib.Tool) mcplib.Tool {
	t.Execution = &mcplib.ToolExecution{TaskSupport: mcplib.TaskSupportForbidden}
	m, hasMeta := toolMeta[t.Name]
	if !hasMeta {
		t.Annotations = mcplib.ToolAnnotation{}
	} else {
		t.Annotations = mcplib.ToolAnnotation{
			Title:           m.title,
			ReadOnlyHint:    mcplib.ToBoolPtr(m.readOnly),
			DestructiveHint: mcplib.ToBoolPtr(m.destruct),
			IdempotentHint:  mcplib.ToBoolPtr(m.idempotent),
			OpenWorldHint:   mcplib.ToBoolPtr(m.openWorld),
		}
		t.DeferLoading = m.deferLoad
	}
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
