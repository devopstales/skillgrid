package clm

import (
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/config"
)

func TestComputeOverflow_WithinBudget(t *testing.T) {
	m := &Mirror{Version: 1, Revision: 1, Nonce: "n", Baseline: "b"}
	m.Blocks = append(m.Blocks, Block{Index: 0, Role: "user", ID: "m1", Content: "small"})
	cfg := config.CLM{Budget: 1000, Reserve: 2048, Guard: true, Cap: 0}
	d := ComputeOverflow(m, cfg, 4.0)
	if d.Withhold {
		t.Fatalf("within budget should not withhold")
	}
}

func TestComputeOverflow_OverBudget(t *testing.T) {
	m := &Mirror{Version: 1, Revision: 1, Nonce: "n", Baseline: "b"}
	big := Block{Index: 0, Role: "tool", ID: "t1", Content: string(make([]byte, 100000))}
	m.Blocks = append(m.Blocks, big)
	cfg := config.CLM{Budget: 1000, Reserve: 2048, Guard: true, Cap: 0}
	d := ComputeOverflow(m, cfg, 4.0)
	if !d.Withhold {
		t.Fatalf("over budget should withhold")
	}
	if len(d.WithholdIDs) == 0 {
		t.Fatalf("expected withhold ids")
	}
}

func TestComputeOverflow_GuardOff(t *testing.T) {
	m := &Mirror{Version: 1, Revision: 1, Nonce: "n", Baseline: "b"}
	m.Blocks = append(m.Blocks, Block{Index: 0, Role: "tool", ID: "t1", Content: string(make([]byte, 100000))})
	cfg := config.CLM{Budget: 1000, Reserve: 2048, Guard: false, Cap: 0}
	d := ComputeOverflow(m, cfg, 4.0)
	if d.Withhold {
		t.Fatalf("guard off should not withhold")
	}
}
