//go:build windows

package controller

import "syscall"

func notifySignal() syscall.Signal {
	return syscall.SIGHUP
}
