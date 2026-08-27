//go:build unix

package ui

import (
	"golang.org/x/term"
)

// makeRaw puts stdin (fd 0) into raw mode and returns its old state.
func makeRaw() (*term.State, error) {
	st, err := term.MakeRaw(0)
	if err != nil {
		return nil, err
	}
	return st, nil
}

// restore returns stdin to its previous state.
func restore(old *term.State) {
	if old != nil {
		_ = term.Restore(0, old)
	}
}
