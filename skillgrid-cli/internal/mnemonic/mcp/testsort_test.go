package mcp

import "sort"

// sortStrings sorts s in place (test helper; the mcp production code uses its
// own sort where needed).
func sortStrings(s []string) { sort.Strings(s) }
