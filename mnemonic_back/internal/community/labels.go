package community

import (
	"database/sql"
	"fmt"
	"path"
	"strings"
)

// labelForCommunityImpl derives an LLM-free label for a community from its
// top god-node names and file paths. It never calls an API. When no god node
// is found (degree-0 or empty community) it falls back to "community-N" —
// never fabricated.
func labelForCommunityImpl(db *sql.DB, id int, memberIDs []int64, opts Options) ([]string, string) {
	if len(memberIDs) == 0 {
		return nil, fmt.Sprintf("community-%d", id)
	}
	gods, err := RankGodNodes(db, memberIDs, false)
	if err != nil || len(gods) == 0 || gods[0].Degree == 0 {
		// No god node (or all degree-0): deterministic fallback, warn+continue
		// is handled by the caller's warning; here we just never fabricate.
		return []string{}, fmt.Sprintf("community-%d", id)
	}
	top := gods[0]
	names := []string{top.Name}
	// Attach up to 2 more god-node names for disambiguation when the top name
	// is generic.
	for i := 1; i < len(gods) && len(names) < 3; i++ {
		if gods[i].Degree < top.Degree {
			break
		}
		names = append(names, gods[i].Name)
	}
	base := strings.Join(names, " + ")
	// Append the top god node's path dir when it is a useful qualifier.
	dir := path.Dir(top.Path)
	if dir != "" && dir != "." && !strings.HasSuffix(base, dir) {
		base = base + " (" + dir + ")"
	}
	return []string{top.Name}, base
}

// topGodNodeNames returns the top god-node names for a community (used by the
// explain view).
func topGodNodeNames(db *sql.DB, memberIDs []int64, limit int) []string {
	gods, err := RankGodNodes(db, memberIDs, false)
	if err != nil {
		return nil
	}
	var out []string
	for i := 0; i < len(gods) && i < limit; i++ {
		out = append(out, gods[i].Name)
	}
	return out
}

var _ = sql.ErrNoRows
