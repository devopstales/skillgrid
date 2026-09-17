package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/codeindex"
)

// TestPhaseLabel: known slugs map to human labels; unknown slugs pass through.
func TestPhaseLabel(t *testing.T) {
	if got := phaseLabel("extract"); got != "extracting" {
		t.Errorf("phaseLabel(extract) = %q, want extracting", got)
	}
	if got := phaseLabel("weird-phase"); got != "weird-phase" {
		t.Errorf("phaseLabel(weird-phase) = %q, want passthrough", got)
	}
}

// TestPlainProgressPhaseLines: one line per phase transition, with item
// counts when known, and the elapsed prefix.
func TestPlainProgressPhaseLines(t *testing.T) {
	var buf bytes.Buffer
	p := newPlainProgress(&buf)
	p.Start()
	p.Event(codeindex.Event{Phase: "scan", PhaseStart: true, Total: 42})
	time.Sleep(5 * time.Millisecond)
	p.Event(codeindex.Event{Phase: "extract", Done: 1, Total: 42})
	p.Event(codeindex.Event{Phase: "extract", Done: 2, Total: 42})
	p.Event(codeindex.Event{Phase: "community", PhaseStart: true})
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines (phase starts only), got %d: %q", len(lines), buf.String())
	}
	if !strings.HasPrefix(lines[0], "scanning:") || !strings.Contains(lines[0], "42 items") {
		t.Errorf("line 0 = %q, want scanning with 42 items", lines[0])
	}
	if !strings.HasPrefix(lines[1], "communities:") || strings.Contains(lines[1], "items") {
		t.Errorf("line 1 = %q, want communities without items", lines[1])
	}
}

// TestProgressViewRender: the TTY view shows the phase label, a progress bar
// with done/total, and the elapsed time.
func TestProgressViewRender(t *testing.T) {
	v := progressView{phase: "extract", done: 10, total: 40, start: time.Now().Add(-2 * time.Second)}
	out := v.View()
	if !strings.Contains(out, "extracting") {
		t.Errorf("view = %q, want extracting label", out)
	}
	if !strings.Contains(out, "10/40") {
		t.Errorf("view = %q, want 10/40", out)
	}
	if !strings.Contains(out, "2s") {
		t.Errorf("view = %q, want 2s elapsed", out)
	}
}

// TestProgressModelEvent: phase start resets the counters; progress events
// update them; the phase label tracks the latest start.
func TestProgressModelEvent(t *testing.T) {
	m := progressModel{view: progressView{start: time.Now()}}
	m, _ = m.update(progressEventMsg(codeindex.Event{Phase: "extract", PhaseStart: true, Total: 5}))
	m, _ = m.update(progressEventMsg(codeindex.Event{Phase: "extract", Done: 3, Total: 5}))
	if m.view.phase != "extract" || m.view.done != 3 || m.view.total != 5 {
		t.Errorf("model = %+v, want extract 3/5", m.view)
	}
	m, _ = m.update(progressEventMsg(codeindex.Event{Phase: "community", PhaseStart: true}))
	if m.view.phase != "community" || m.view.done != 0 {
		t.Errorf("after phase start: %+v, want community 0", m.view)
	}
}

// TestProgressModelDetail: the detail (current file/function) is shown in the
// view and cleared on the next phase start.
func TestProgressModelDetail(t *testing.T) {
	m := progressModel{view: progressView{start: time.Now()}}
	m, _ = m.update(progressEventMsg(codeindex.Event{Phase: "extract", PhaseStart: true, Total: 2}))
	m, _ = m.update(progressEventMsg(codeindex.Event{Phase: "extract", Done: 1, Total: 2, Detail: "main.go"}))
	if !strings.Contains(m.View(), "main.go") {
		t.Errorf("view = %q, want detail main.go", m.View())
	}
	m, _ = m.update(progressEventMsg(codeindex.Event{Phase: "community", PhaseStart: true}))
	if strings.Contains(m.View(), "main.go") {
		t.Errorf("view = %q, detail should reset on phase start", m.View())
	}
}
