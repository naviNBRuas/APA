package platform

import (
	"log/slog"
	"runtime"
	"sync"
	"time"
)

type PlatformDetector struct {
	logger      *slog.Logger
	cache       *PlatformProfile
	cacheMutex  sync.RWMutex
	cacheExpiry time.Duration
}

func NewPlatformDetector(logger *slog.Logger, cacheExpiry time.Duration) *PlatformDetector {
	return &PlatformDetector{
		logger:      logger,
		cacheExpiry: cacheExpiry,
	}
}

func detectCurrentPlatform() PlatformType {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	switch {
	case goos == "linux" && goarch == "amd64":
		return PlatformLinuxAMD64
	case goos == "linux" && goarch == "arm64":
		return PlatformLinuxARM64
	case goos == "linux" && goarch == "arm":
		return PlatformLinuxARM
	case goos == "linux" && goarch == "386":
		return PlatformLinux386
	case goos == "linux" && goarch == "riscv64":
		return PlatformLinuxRISCV64
	case goos == "windows" && goarch == "amd64":
		return PlatformWindowsAMD64
	case goos == "windows" && goarch == "arm64":
		return PlatformWindowsARM64
	case goos == "windows" && goarch == "386":
		return PlatformWindows386
	case goos == "darwin" && goarch == "amd64":
		return PlatformDarwinAMD64
	case goos == "darwin" && goarch == "arm64":
		return PlatformDarwinARM64
	case goos == "freebsd" && goarch == "amd64":
		return PlatformFreeBSDAMD64
	default:
		return PlatformUnknown
	}
}

func (pd *PlatformDetector) DetectPlatform() (*PlatformProfile, error) {
	numCPU := runtime.NumCPU()

	return &PlatformProfile{
		OS: OperatingSystem{
			Name:   runtime.GOOS,
			Version: "detected",
			Kernel: runtime.GOOS,
			Family: runtime.GOOS,
			Build:  runtime.GOARCH,
		},
		Architecture: Architecture{
			Type:       runtime.GOARCH,
			NumCPUs:    numCPU,
			NumCores:   numCPU,
			NumThreads: numCPU,
			CacheLine:  64,
			PageSize:   4096,
		},
		Runtime: RuntimeEnvironment{
			GoVersion:  runtime.Version(),
			GoOS:       runtime.GOOS,
			GoArch:     runtime.GOARCH,
			Compiler:   runtime.Compiler,
			CGOEnabled: false,
			GOMAXPROCS: numCPU,
		},
		Hardware: HardwareSpecs{
			CPU: CPUInfo{
				Cores:   numCPU,
				Threads: numCPU,
			},
			Memory: MemoryInfo{
				Total:     1 << 30,
				Available: 1 << 30,
			},
		},
		ProfileTimestamp: time.Now(),
		ConfidenceScore:  0.95,
	}, nil
}
