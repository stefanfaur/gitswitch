//go:build !linux

package harden

import "syscall"

func Apply() {
	_ = syscall.Setrlimit(syscall.RLIMIT_CORE, &syscall.Rlimit{Cur: 0, Max: 0})
}
