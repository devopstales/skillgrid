package extract

import (
	"testing"
)

// [knowledge-node-classification] the knowledge-node classifier maps file
// paths to their node kind: .md → doc, .yaml/.toml/.json → config, .sql →
// table; an unmapped extension (.xyz) is skipped, not classified (ok=false).
func TestKnowledgeNodeClassification(t *testing.T) {
	if k, ok := Classify("README.md", "# Title\n"); k != "doc" || !ok {
		t.Fatalf("README.md → doc (ok), got kind=%q ok=%v", k, ok)
	}
	if k, ok := Classify("config.yaml", "a: b\n"); k != "config" || !ok {
		t.Fatalf("config.yaml → config (ok), got kind=%q ok=%v", k, ok)
	}
	if k, ok := Classify("config.toml", "a = b\n"); k != "config" || !ok {
		t.Fatalf("config.toml → config (ok), got kind=%q ok=%v", k, ok)
	}
	if k, ok := Classify("package.json", "{}\n"); k != "config" || !ok {
		t.Fatalf("package.json → config (ok), got kind=%q ok=%v", k, ok)
	}
	if k, ok := Classify("schema.sql", "CREATE TABLE t (id INT);\n"); k != "table" || !ok {
		t.Fatalf("schema.sql → table (ok), got kind=%q ok=%v", k, ok)
	}
	// an unmapped extension is skipped, not classified.
	if k, ok := Classify("x.xyz", "…"); ok {
		t.Fatalf("an unmapped .xyz must be skipped, not classified (kind=%q)", k)
	}
	// a README.sh is not code and not a knowledge node (skipped).
	if k, ok := Classify("README.sh", "#!/bin/sh\n"); ok {
		t.Fatalf("README.sh must be skipped, not classified (kind=%q)", k)
	}
}
