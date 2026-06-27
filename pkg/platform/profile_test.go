package platform

import (
	"runtime"
	"testing"
)

func TestCurrentProfile(t *testing.T) {
	p := Current()
	if p.Minimal {
		t.Fatal("expected MinimalBuild to be false in default build")
	}
	if p.TinyGo {
		t.Fatal("expected TinyGoBuild to be false in standard Go build")
	}
}

func TestMinimalBuildConst(t *testing.T) {
	if MinimalBuild {
		t.Fatal("expected MinimalBuild to be false without 'minimal' build tag")
	}
}

func TestTinyGoBuildConst(t *testing.T) {
	if TinyGoBuild {
		t.Fatal("expected TinyGoBuild to be false in standard Go build")
	}
}

func TestPlatformTypeConstants(t *testing.T) {
	platforms := []PlatformType{
		PlatformLinuxAMD64, PlatformLinuxARM64, PlatformLinuxARM, PlatformLinux386, PlatformLinuxRISCV64,
		PlatformWindowsAMD64, PlatformWindowsARM64, PlatformWindows386,
		PlatformDarwinAMD64, PlatformDarwinARM64,
		PlatformFreeBSDAMD64, PlatformAndroidARM64, PlatformIOSARM64,
		PlatformUnknown,
	}
	if len(platforms) != 14 {
		t.Fatalf("expected 14 platform constants, got %d", len(platforms))
	}

	current := PlatformUnknown
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "linux/amd64":
		current = PlatformLinuxAMD64
	case "linux/arm64":
		current = PlatformLinuxARM64
	case "linux/arm":
		current = PlatformLinuxARM
	case "linux/386":
		current = PlatformLinux386
	case "linux/riscv64":
		current = PlatformLinuxRISCV64
	case "darwin/amd64":
		current = PlatformDarwinAMD64
	case "darwin/arm64":
		current = PlatformDarwinARM64
	case "windows/amd64":
		current = PlatformWindowsAMD64
	case "windows/arm64":
		current = PlatformWindowsARM64
	case "freebsd/amd64":
		current = PlatformFreeBSDAMD64
	}
	if current == PlatformUnknown {
		t.Logf("running on unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	} else {
		t.Logf("detected platform: %s", current)
	}
}

func TestAcceleratorTypeConstants(t *testing.T) {
	types := []AcceleratorType{AcceleratorGPU, AcceleratorTPU, AcceleratorFPGA, AcceleratorASIC, AcceleratorNeural}
	if len(types) != 5 {
		t.Fatalf("expected 5 accelerator types, got %d", len(types))
	}
	if AcceleratorGPU != "gpu" || AcceleratorNeural != "neural_processor" {
		t.Fatal("unexpected accelerator type values")
	}
}

func TestPlatformProfileStruct(t *testing.T) {
	p := PlatformProfile{
		OS:           OperatingSystem{Name: "linux", Version: "6.8"},
		Architecture: Architecture{Type: "amd64", NumCPUs: 4},
		Runtime:      RuntimeEnvironment{GoVersion: runtime.Version()},
	}
	if p.OS.Name != "linux" {
		t.Fatalf("unexpected OS name: %s", p.OS.Name)
	}
	if p.Runtime.GoVersion != runtime.Version() {
		t.Fatalf("unexpected Go version: %s", p.Runtime.GoVersion)
	}
}
