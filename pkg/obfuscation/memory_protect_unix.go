//go:build !windows

package obfuscation

import (
	"golang.org/x/sys/unix"
)

func (a *AntiTampering) ProtectMemory() error {
	if err := unix.Mlockall(unix.MCL_CURRENT | unix.MCL_FUTURE); err != nil {
		a.logger.Warn("mlockall failed", "error", err)
		return nil
	}
	return nil
}
