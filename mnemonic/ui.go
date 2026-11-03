package mnemonic

import (
	"github.com/devopstales/skillgrid/mnemonic/internal/ui"
)

type Option = ui.Option

func Interactive() bool {
	return ui.Interactive()
}

func MultiSelect(title string, opts []Option) ([]string, bool, error) {
	return ui.MultiSelect(title, opts)
}
