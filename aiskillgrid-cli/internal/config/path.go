package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func WritePathInstructions(baseDir string, writer io.Writer) error {
	bin := filepath.Join(baseDir, "bin")
	npmBin := filepath.Join(baseDir, "npm", ".bin")
	if home, err := os.UserHomeDir(); err == nil {
		bin = strings.Replace(bin, home, "$HOME", 1)
		npmBin = strings.Replace(npmBin, home, "$HOME", 1)
	}
	if runtime.GOOS == "windows" {
		if _, err := fmt.Fprintf(writer, "set PATH=%%PATH%%;%s;%s\n", bin, npmBin); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintf(writer, "export PATH=\"%s:$PATH\"\n", bin); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(writer, "export PATH=\"%s:$PATH\"\n", npmBin); err != nil {
			return err
		}
	}
	return nil
}
