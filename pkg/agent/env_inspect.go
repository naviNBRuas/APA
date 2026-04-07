package agent

import (
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/naviNBRuas/APA/pkg/obfuscation"
)

// EnvProfile captures runtime environment signals.
type EnvProfile struct {
	Arch             string
	CPUCount         int
	Virtualized      bool
	DebuggerAttached bool
	Uptime           time.Duration
	EntropyAvailable bool
	HardwareHint     string
}

// EnvInspector gathers environment attributes to choose execution paths.
type EnvInspector struct {
}

// Inspect collects environment signals.
func (EnvInspector) Inspect() EnvProfile {
	prof := EnvProfile{Arch: runtime.GOARCH, CPUCount: runtime.NumCPU()}
	prof.Virtualized = detectVirtualization()
	prof.DebuggerAttached = obfuscation.NewAntiAnalysis(nil).DetectDebugger()
	prof.Uptime = readUptime()
	prof.EntropyAvailable = entropyReady()
	prof.HardwareHint = hardwareHint()
	return prof
}

// ShouldPreferLowProfile returns true when environment suggests constrained or observable execution.
func (p EnvProfile) ShouldPreferLowProfile() bool {
	return p.Virtualized || p.DebuggerAttached || p.CPUCount <= 2
}

var uptimeReader = func() (time.Duration, bool) {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, false
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return 0, false
	}
	sec, err := time.ParseDuration(fields[0] + "s")
	if err != nil {
		return 0, false
	}
	return sec, true
}

func readUptime() time.Duration {
	if d, ok := uptimeReader(); ok {
		return d
	}
	return 0
}

var entropyCheck = func() bool {
	if _, err := os.Stat("/dev/random"); err == nil {
		return true
	}
	return false
}

func entropyReady() bool {
	return entropyCheck()
}

var virtCheck = func() bool {
	if data, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		if strings.Contains(strings.ToLower(string(data)), "hypervisor") {
			return true
		}
	}
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	return false
}

func detectVirtualization() bool { return virtCheck() }

var hwHintReader = func() string {
	if data, err := os.ReadFile("/sys/class/dmi/id/product_name"); err == nil {
		return strings.TrimSpace(string(data))
	}
	return ""
}

func hardwareHint() string { return hwHintReader() }
