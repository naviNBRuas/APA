//go:build !windows

package injection

import "syscall"

func newProcessAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}
