package ui

import (
	"fmt"
	"os"
	"strings"
)

type Option struct {
	Label   string
	Value   string
	Default bool
	Hint    string
}

// Letter returns a/a-letter for the option at index i (a, b, c, ...).
func Letter(i int) rune { return rune('a' + i) }

// MultiSelect renders a terminal checklist using raw ANSI escapes.
// Keys: ↑/↓ or j/k move, space toggle, a selects all, enter confirms, q/cancel.
//
// When the TTY is unavailable, falls back to a plain letter-list prompt.
func MultiSelect(title string, opts []Option) ([]string, bool, error) {
	if !ttyColor {
		return plainFallback(title, opts)
	}
	old, err := makeRaw()
	if err != nil {
		return plainFallback(title, opts)
	}
	defer restore(old)

	selected := make([]bool, len(opts))
	for i, o := range opts {
		selected[i] = o.Default
	}
	cursor := 0
	keys := "↑/↓ or j/k move · space toggle · a all · enter ok · q cancel"

	for {
		draw(title, keys, opts, selected, cursor)
		b, err := readKey()
		if err != nil {
			clearScreen()
			fmt.Println()
			return nil, false, fmt.Errorf("read input: %w", err)
		}
		switch b {
		case 3, 27, 'q': // ctrl+c, esc, q -> cancel
			clearScreen()
			fmt.Println("cancelled")
			return nil, false, nil
		case '\r', '\n': // enter -> confirm
			clearScreen()
			return collect(selected, opts), true, nil
		case '[': // arrow sequence start
			if code, err := readKey(); err == nil {
				switch code {
				case 'A', 'P':
					if cursor > 0 {
						cursor--
					}
				case 'B', 'Q':
					if cursor < len(opts)-1 {
						cursor++
					}
				}
			}
		case 'j':
			if cursor < len(opts)-1 {
				cursor++
			}
		case 'k':
			if cursor > 0 {
				cursor--
			}
		case ' ':
			selected[cursor] = !selected[cursor]
		case 'a':
			for i := range selected {
				selected[i] = true
			}
		default:
			if r := rune(b); r >= 'a' && r <= 'z' {
				idx := int(r - 'a')
				if idx < len(opts) {
					selected[idx] = !selected[idx]
				}
			}
		}
	}
}

func draw(title, keys string, opts []Option, sel []bool, cursor int) {
	clearScreen()
	if ttyColor {
		fmt.Printf("\x1b[7m %s \x1b[0m\n\n", title)
	} else {
		fmt.Printf("== %s ==\n\n", title)
	}
	for i, o := range opts {
		box := " "
		if sel[i] {
			box = "✓"
		}
		line := fmt.Sprintf("  [%s %s]", box, o.Label)
		if o.Hint != "" {
			line += "  " + Dim(o.Hint)
		}
		if i == cursor && ttyColor {
			line = "\x1b[7m " + line + " \x1b[0m"
		}
		fmt.Println(line)
	}
	fmt.Println()
	fmt.Println("  " + Dim(keys))
}

func plainFallback(title string, opts []Option) ([]string, bool, error) {
	fmt.Printf("\n== %s ==\n", title)
	fmt.Println("Enter the letters of the options you want (a single string, no spaces), 'a' for all, or blank for defaults:")
	for i, o := range opts {
		line := fmt.Sprintf("  %c  %s", Letter(i), o.Label)
		if o.Hint != "" {
			line += "  " + Dim(o.Hint)
		}
		def := ""
		if o.Default {
			def = " (default)"
		}
		fmt.Println(line + def)
	}
	fmt.Print("\n> ")
	raw := strings.ToLower(strings.ReplaceAll(Ask(), " ", ""))
	if raw == "" {
		var out []string
		for _, o := range opts {
			if o.Default {
				out = append(out, o.Value)
			}
		}
		return out, true, nil
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

func collect(sel []bool, opts []Option) []string {
	var out []string
	for i, on := range sel {
		if on {
			out = append(out, opts[i].Value)
		}
	}
	return out
}

func clearScreen() { fmt.Print("\x1b[2J\x1b[H") }

func readKey() (byte, error) {
	var b [1]byte
	n, err := os.Stdin.Read(b[:])
	if err != nil || n == 0 {
		return 0, err
	}
	return b[0], nil
}
