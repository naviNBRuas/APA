//go:build !tinygo

package platform

// TinyGoBuild reports whether this binary was built with the tinygo toolchain.
// Default (standard Go) build.
const TinyGoBuild = false
