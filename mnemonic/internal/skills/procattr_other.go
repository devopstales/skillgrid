//go:build !linux

package skills

import "syscall"

// newSysProcAttr returns the sandbox process attributes. Setpgid puts the
// runner in its own process group so the timeout kill (Kill(-pid, SIGKILL))
// reaches it. Pdeathsig is not portable (Linux-only), so on other platforms
// the process-group kill is the sole reaping mechanism.
func newSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}
