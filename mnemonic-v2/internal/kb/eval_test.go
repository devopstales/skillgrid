package kb

import (
	"context"
	"testing"
)

func TestEvalRecallAtOne(t *testing.T) {
	ctx := context.Background()
	vault := t.TempDir()
	writeVaultPage(t, vault, "decisions/auth-model.md",
		"---\ntitle: Auth Model\ntype: Decision\nstatus: stable\n---\n\nWe use OAuth2 token rotation.")
	st := openKbStore(t)
	if _, err := Reindex(ctx, st, vault, "proj-a"); err != nil {
		t.Fatalf("reindex: %v", err)
	}
	rep, err := Eval(ctx, st, []EvalCase{{Query: "OAuth2", Expected: []string{"decisions/auth-model.md"}}}, 10, 1.0)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	if rep.Cases != 1 {
		t.Fatalf("cases = %d, want 1", rep.Cases)
	}
	if rep.RecallAtK < 1.0 {
		t.Fatalf("recall@K = %v, want >= 1.0", rep.RecallAtK)
	}
	if rep.Failed {
		t.Fatalf("eval failed, want false at baseline 1.0")
	}
}
