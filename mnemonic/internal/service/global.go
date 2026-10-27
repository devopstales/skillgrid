// Global operational memory bootstrap (change 2026-10-24-global-llm-wiki,
// Change A, TICKET-01): the global store pseudo-project store at
// <dataDir>/global.sqlite, opened through the same openProject path as
// project stores so it carries the full store schema (migration 072 gives
// facts the category column).
package service

// GlobalProjectID is the pseudo-project id for the cross-project fact store.
// It is a constant, not a project.Resolve result: the global store has no
// directory to resolve against, and its id must survive verbatim on disk.
const GlobalProjectID = "global"

// OpenGlobal opens (creating it on first use) the global pseudo-project
// store and returns its handle, with the usual cleanup function releasing
// the store reference. The id deliberately bypasses project.NormalizeID,
// which would collapse its underscores ("global" -> "-global-"); the
// pseudo-project name is fixed by the spec, so no normalization is applied.
func (s *Service) OpenGlobal() (*ProjectHandle, func(), error) {
	return s.openProject(GlobalProjectID, ".")
}
