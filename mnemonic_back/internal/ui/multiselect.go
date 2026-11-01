package ui

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
)

type Option struct {
	Label   string
	Value   string
	Default bool
	Hint    string
}

// ErrCancelled is returned when the user cancels the selection (ctrl+c).
var ErrCancelled = errors.New("cancelled")

// MultiSelect renders a checklist using charmbracelet/huh.
//
// Falls back to a plain letter-list prompt when the TTY is unavailable
// or the terminal is incapable (TERM=dumb/linux, CI, pipes).
func MultiSelect(title string, opts []Option) ([]string, bool, error) {
	if !Interactive() {
		return plainFallback(title, opts)
	}

	var values []string
	huhOpts := make([]huh.Option[string], 0, len(opts))
	for _, o := range opts {
		huhOpts = append(huhOpts, huh.NewOption(o.Label, o.Value))
	}

	m := huh.NewMultiSelect[string]().
		Title(title).
		Value(&values).
		Options(huhOpts...)

	if err := m.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			fmt.Fprintln(os.Stderr, "cancelled")
			return nil, false, ErrCancelled
		}
		return plainFallback(title, opts)
	}

	if len(values) == 0 {
		return nil, true, nil
	}
	return values, true, nil
}

func plainFallback(title string, opts []Option) ([]string, bool, error) {
	fmt.Printf("\n== %s ==\n", title)
	fmt.Println("Enter the letters of the options you want ('a' for all), or blank to skip:")
	for i, o := range opts {
		line := fmt.Sprintf("  %c  %s", Letter(i), o.Label)
		fmt.Println(line)
	}
	fmt.Print("\n> ")
	raw := strings.ToLower(strings.ReplaceAll(Ask(), " ", ""))
	if raw == "" {
		return nil, true, nil
	}
	if raw == "a" {
		out := make([]string, len(opts))
		for i, o := range opts {
			out[i] = o.Value
		}
		return out, true, nil
	}
	want := map[rune]struct{}{}
	for _, r := range raw {
		want[r] = struct{}{}
	}
	var out []string
	for i, o := range opts {
		if _, ok := want[Letter(i)]; ok {
			out = append(out, o.Value)
		}
	}
	return out, true, nil
}
