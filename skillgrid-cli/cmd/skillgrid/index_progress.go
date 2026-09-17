package main

// index_progress.go: live progress rendering for `skillgrid index`.
//
// Two renderers behind one interface:
//   - progressView (bubbletea) when FancyUI: spinner + phase + progress bar
//     + elapsed, one compact line.
//   - plainProgress (stderr lines) for pipes/CI/NO_COLOR: one line per phase
//     transition; stdout stays clean for the final summary.

import (
	"fmt"
	"io"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/codeindex"
)

// progressReporter renders index events; Start begins rendering, Event is
// called from the indexing goroutine (must not block), Done stops rendering.
type progressReporter interface {
	Start()
	Event(codeindex.Event)
	Done()
}

var phaseLabels = map[string]string{
	"scan":             "scanning",
	"extract":          "extracting",
	"embed":            "embedding",
	"community":        "communities",
	"import-cycles":    "import cycles",
	"process":          "processes",
	"knowledge":        "knowledge",
	"lsp":              "lsp",
	"pdg":              "pdg",
	"resolution-audit": "resolution audit",
	"complete":         "done",
}

func phaseLabel(phase string) string {
	if l, ok := phaseLabels[phase]; ok {
		return l
	}
	return phase
}

// ---------- plain (non-TTY) renderer ----------

// plainProgress prints one line per phase transition to w.
type plainProgress struct {
	w     io.Writer
	start time.Time
}

func newPlainProgress(w io.Writer) *plainProgress {
	return &plainProgress{w: w}
}

func (p *plainProgress) Start() {
	p.start = time.Now()
}

func (p *plainProgress) Event(e codeindex.Event) {
	if !e.PhaseStart {
		return
	}
	elapsed := time.Since(p.start).Truncate(time.Millisecond)
	if e.Total > 0 {
		fmt.Fprintf(p.w, "%s: %s (%d items)\n", phaseLabel(e.Phase), elapsed, e.Total)
	} else {
		fmt.Fprintf(p.w, "%s: %s\n", phaseLabel(e.Phase), elapsed)
	}
}

func (p *plainProgress) Done() {}

// ---------- bubbletea renderer ----------

type progressView struct {
	phase  string
	detail string
	done   int
	total  int
	start  time.Time
}

func (v progressView) View() string {
	elapsed := time.Since(v.start).Round(time.Second)
	label := phaseLabel(v.phase)
	if v.detail != "" {
		label += " " + v.detail
	}
	var bar string
	if v.total > 0 {
		width := 20
		filled := 0
		if v.done > 0 {
			filled = v.done * width / v.total
		}
		bar = " " + strings.Repeat("=", filled) + strings.Repeat(" ", width-filled) + " " +
			fmt.Sprintf("%d/%d", v.done, v.total)
	}
	return label + bar + " " + elapsed.String()
}

type tickMsg time.Time
type progressEventMsg codeindex.Event

type progressModel struct {
	view progressView
	quit bool
}

func (m progressModel) Init() tea.Cmd { return tickCmd() }

func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// update applies one message and returns the concrete model (Update, below,
// adapts it to the tea.Model interface).
func (m progressModel) update(msg tea.Msg) (progressModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		return m, tickCmd()
	case progressEventMsg:
		e := codeindex.Event(msg)
		if e.PhaseStart {
			m.view.phase = e.Phase
			m.view.done = 0
			m.view.detail = ""
		}
		if e.Done > 0 || e.Total > 0 {
			m.view.done = e.Done
			m.view.total = e.Total
		}
		if e.Detail != "" {
			m.view.detail = e.Detail
		}
		return m, nil
	case tea.KeyMsg:
		m.quit = true
		return m, tea.Quit
	}
	return m, nil
}

func (m progressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	nm, cmd := m.update(msg)
	return nm, cmd
}

func (m progressModel) View() string {
	return m.view.View()
}

// progressViewTea wraps bubbletea: Event sends a msg to the program, Done
// stops the program and clears the frame.
type progressViewTea struct {
	p     *tea.Program
	start time.Time
}

func newProgressViewTea(w io.Writer) *progressViewTea {
	// Seed with a "starting" phase so the very first rendered frame carries a
	// label + bar area, not a bare elapsed-time cell. Without this, on a warm
	// index the first index event lands after the renderer's first tick and the
	// user only ever sees the ticking seconds (the reported bug).
	m := progressModel{view: progressView{phase: "starting", start: time.Now()}}
	p := tea.NewProgram(m, tea.WithOutput(w), tea.WithInputTTY(), tea.WithoutSignalHandler())
	return &progressViewTea{p: p, start: time.Now()}
}

func (r *progressViewTea) Start() {
	go func() {
		_, _ = r.p.Run()
	}()
}

// Event delivers one index event to the tea program. Program.Send is
// goroutine-safe (it feeds the program's internal message queue), so this can
// be called directly from the indexing goroutine without a relay channel. The
// program is bounded: Run() consumes messages as fast as it renders, so a
// cold-index burst is queued and drained in order.
func (r *progressViewTea) Event(e codeindex.Event) {
	if !e.PhaseStart && e.Done == 0 {
		return
	}
	r.p.Send(progressEventMsg(e))
}

func (r *progressViewTea) Done() {
	r.p.Quit()
}
