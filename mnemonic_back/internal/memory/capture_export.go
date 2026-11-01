package memory

// ExtractLearnings is the exported form of the 005 passive-capture extractor
// (extractLearnings). It is the deterministic, LLM-free floor that the layered
// distillation (change 013, step 02) reuses to produce L1 atoms offline: the
// same Key-Learnings / Lesson / Discovery heuristics mem_capture_passive uses,
// so the no-LLM floor is byte-for-byte the proven 005 behaviour.
func ExtractLearnings(text string) []PassiveItem {
	return extractLearnings(text)
}

// ShapePassiveItem is the exported form of shapePassiveItem: it classifies an
// extracted learning by keyword and returns a stable title. The layer package
// uses it to name L1 atoms without re-implementing 005's type heuristics.
func ShapePassiveItem(item PassiveItem) (title, typ string) {
	return shapePassiveItem(item)
}

// ShapePassiveContent is the exported form of shapePassiveContent: it builds a
// What/Why/Where/Learned-shaped body from an extracted item, inventing nothing
// beyond what is present in the source text.
func ShapePassiveContent(item PassiveItem, title, typ string) string {
	return shapePassiveContent(item, title, typ)
}

// IsValidType is the exported type validator used by the layer package to
// confirm a distilled atom's type is a real observation type before saving.
func IsValidTypeExported(typ string) bool {
	return IsValidType(typ)
}
