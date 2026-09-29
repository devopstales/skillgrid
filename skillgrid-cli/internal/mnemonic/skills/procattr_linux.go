package skills

import "syscall"

// newSysProcAttr returns the sandbox process attributes. Setpgid puts the
// runner in its own process group so the timeout kill (Kill(-pid, SIGKILL))
// reaches it; Pdeathsig adds the belt to that suspenders: when the runner
// dies, the kernel kills its children too, so a `go run` compiled binary
// (a different process group) can never outlive the sandbox.
func newSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true, Pdeathsig: syscall.SIGKILL}
}
