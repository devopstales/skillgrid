package context_harness

// RenderRouting returns the advisory Context Routing tool map injected into
// the prime block. It is deterministic and fixed: no network, model, or store
// query at render time. Keep it under 600 characters.
func RenderRouting() string {
	return "## Context Routing\n" +
		"counts, lists, existence → ctx_query\n" +
		"retrieve, search → ctx search (ctx_search)\n" +
		"index a file → ctx index\n" +
		"memory → mem_*\n"
}
