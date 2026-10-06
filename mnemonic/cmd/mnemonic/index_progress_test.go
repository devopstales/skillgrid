package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/devopstales/skillgrid/mnemonic/internal/codeindex"
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

// TestProgressViewStatusOnOwnLine: the progress bar + done/total + elapsed
// must render on a line by itself, with the phase label (and current-file
// detail) on the line above. The reported "jumping" was the whole single line
// (label+detail+bar+counter) being rewritten on every file change; splitting
// the status onto its own line keeps the bar in a fixed column so only the
// file line moves.
func TestProgressViewStatusOnOwnLine(t *testing.T) {
	v := progressView{phase: "extract", detail: "skillgrid-ui/vite.config.ts", done: 10, total: 40, start: time.Now().Add(-2 * time.Second)}
	lines := strings.Split(v.View(), "\n")
	if len(lines) != 2 {
		t.Fatalf("view = %q, want exactly 2 lines (label line + status line)", v.View())
	}
	// Line 1: phase label + current file detail.
	if !strings.Contains(lines[0], "extracting") || !strings.Contains(lines[0], "vite.config.ts") {
		t.Errorf("line 0 = %q, want phase label + detail", lines[0])
	}
	// Line 2: bar + counter + elapsed, and NO file detail.
	if !strings.Contains(lines[1], "10/40") || !strings.Contains(lines[1], "2s") {
		t.Errorf("line 1 = %q, want done/total + elapsed", lines[1])
	}
	if strings.Contains(lines[1], "vite.config.ts") {
		t.Errorf("line 1 = %q, must not contain the file detail", lines[1])
	}
	// The bar segment is fixed-width (20) regardless of detail length, so it
	// sits in the same column on every frame. Extract the 20-char bar that
	// immediately precedes " 10/40".
	idx := strings.Index(lines[1], " 10/40")
	if idx < 20 {
		t.Fatalf("line 1 = %q, bar should end before the counter", lines[1])
	}
	bar := lines[1][idx-20 : idx]
	if got := len(bar); got != 20 {
		t.Errorf("bar width = %d, want 20 (fixed-width)", got)
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

// TestProgressViewInitialFrame: the very first rendered frame must carry a
// phase label and not be a bare elapsed-time cell. This is the reported bug:
// before the first index event arrives the view was " 0s" (empty phase), so on
// a warm index the user only ever saw the ticking seconds. The view must be
// non-trivial from frame one.
func TestProgressViewInitialFrame(t *testing.T) {
	m := progressModel{view: progressView{phase: "starting", start: time.Now()}}
	lines := strings.Split(m.View(), "\n")
	if len(lines) < 1 || !strings.Contains(lines[0], "starting") {
		t.Errorf("initial frame = %q, want a phase label on line one (not a bare elapsed cell)", m.View())
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

// restoreFake stands in for a bubbletea program. Wait blocks until release
// so a test can prove Done does not return before the terminal is restored.
type restoreFake struct {
	quit        bool
	waits       int
	waitStarted chan struct{}
	release     chan struct{}
}

func (f *restoreFake) Send(tea.Msg) {}

func (f *restoreFake) Run() (tea.Model, error) { return nil, nil }

func (f *restoreFake) Quit() { f.quit = true }

func (f *restoreFake) Wait() {
	f.waits++
	if f.waitStarted != nil {
		select {
		case <-f.waitStarted:
		default:
			close(f.waitStarted)
		}
	}
	if f.release != nil {
		<-f.release
	}
}

// TestProgressDoneWaitsForTerminalRestore: Done must quit the live view and
// block until that program has restored the terminal. Returning earlier lets
// the process exit in raw mode, so the next shell prompt is drawn mid-line.
func TestProgressDoneWaitsForTerminalRestore(t *testing.T) {
	release := make(chan struct{})
	f := &restoreFake{waitStarted: make(chan struct{}), release: release}
	r := &progressViewTea{p: f}
	returned := make(chan struct{})
	go func() {
		r.Done()
		close(returned)
	}()
	select {
	case <-f.waitStarted:
	case <-time.After(time.Second):
		t.Fatal("Done returned without waiting for the program to restore the terminal")
	}
	select {
	case <-returned:
		t.Fatal("Done returned before Wait finished")
	default:
	}
	if !f.quit {
		t.Fatal("Quit was not called before Wait")
	}
	close(release)
	select {
	case <-returned:
	case <-time.After(time.Second):
		t.Fatal("Done did not return after the terminal was restored")
	}
}

// TestProgressDoneOnce: a second Done must not wait again. runIndex restores
// once before printing, and a defer restores again on the way out.
func TestProgressDoneOnce(t *testing.T) {
	f := &restoreFake{}
	r := &progressViewTea{p: f}
	r.Done()
	r.Done()
	if f.waits != 1 || !f.quit {
		t.Fatalf("Quit=%v waits=%d, want one restore", f.quit, f.waits)
	}
}

// orderReporter records whether Done has run. orderWriter flags any write
// that happens before that restore.
type orderReporter struct {
	done bool
}

func (o *orderReporter) Start()                {}
func (o *orderReporter) Event(codeindex.Event) {}
func (o *orderReporter) Done()                 { o.done = true }

type orderWriter struct {
	o     *orderReporter
	buf   bytes.Buffer
	early bool
}

func (w *orderWriter) Write(p []byte) (int, error) {
	if !w.o.done {
		w.early = true
	}
	return w.buf.Write(p)
}

// TestFinishIndexPrintsAfterRestore: the unresolved-member line and the
// indexed summary are written only after the progress view has restored the
// terminal, each starting at column 0 and ending with a newline.
func TestFinishIndexPrintsAfterRestore(t *testing.T) {
	rep := &orderReporter{}
	out := &orderWriter{o: rep}
	errw := &orderWriter{o: rep}
	finishIndex(rep, codeindex.Stats{
		FilesIndexed:      38,
		FilesSkipped:      1,
		ChunksAdded:       61,
		UnresolvedMembers: 81,
		UnresolvedKnown:   true,
	}, out, errw)
	if out.early || errw.early {
		t.Fatal("summary was written before the terminal was restored")
	}
	if strings.Contains(errw.buf.String(), "unresolved") {
		t.Errorf("stderr = %q; unbound members are not an operator error", errw.buf.String())
	}
	if got := out.buf.String(); got != "indexed: 38 files, 61 chunks (+1 skipped, -0 deleted)\n" {
		t.Errorf("stdout = %q", got)
	}
}

// TestFinishIndexOmitsUnknownUnresolved: a failed count query prints no
// unresolved line, and the indexed line is still after restore.
func TestFinishIndexOmitsUnknownUnresolved(t *testing.T) {
	rep := &orderReporter{}
	out := &orderWriter{o: rep}
	errw := &orderWriter{o: rep}
	finishIndex(rep, codeindex.Stats{FilesIndexed: 2, ChunksAdded: 3}, out, errw)
	if out.early || errw.early {
		t.Fatal("summary was written before the terminal was restored")
	}
	if errw.buf.Len() != 0 {
		t.Errorf("stderr = %q, want empty when the count was not read", errw.buf.String())
	}
	if got := out.buf.String(); got != "indexed: 2 files, 3 chunks (+0 skipped, -0 deleted)\n" {
		t.Errorf("stdout = %q", got)
	}
}
