//go:build !minimal

package platform

// MinimalBuild reports whether the build excludes heavier/optional features.
// This default file is used when the `minimal` build tag is NOT set.
const MinimalBuild = false
