package graph

import (
	"context"
	"strconv"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// RecordTrail records a retrieval trail for a walker read (M7/T15/INV-5).
// Every walker read (Impact/neighbors/Resolve/Path/Affected/Orient/Explain)
// records a trail via Store.RecordRetrieval into the shared retrieval_trails
// table. The route string becomes the `mnemonic trail show` vocabulary.
//
// IDs are the row ids the read returned (the audit answer to "which rows did
// this read touch?"), serialized as a comma-separated id list for the
// `ids TEXT` column. The degraded reason marks a read that failed (a degraded
// retrieval is still a retrieval worth auditing, INV-5).
func RecordTrail(ctx context.Context, st store.Store, route, query string, degraded string, ids ...int64) {
	idStr := ""
	if len(ids) > 0 {
		parts := make([]string, len(ids))
		for i, id := range ids {
			parts[i] = strconv.FormatInt(id, 10)
		}
		idStr = stringsJoin(parts, ",")
	}
	t := store.RetrievalTrail{
		Route:     route,
		Query:     query,
		Project:   "p", // one DB per project (CI-O2)
		IDs:       idStr,
		Degraded:  degraded != "",
		Timestamp: nowTimestamp(),
	}
	_ = st.RecordRetrieval(ctx, t)
}

// stringsJoin joins the parts with the separator (a local join so the trail
// seam does not pull a strings import into every walker file).
func stringsJoin(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for _, p := range parts[1:] {
		out += sep + p
	}
	return out
}
