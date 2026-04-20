//go:build linux

package harden

import (
	"syscall"

	"golang.org/x/sys/unix"
)

func Apply() {
	_ = unix.Prctl(unix.PR_SET_DUMPABLE, 0, 0, 0, 0)
	_ = syscall.Setrlimit(syscall.RLIMIT_CORE, &syscall.Rlimit{Cur: 0, Max: 0})
}
