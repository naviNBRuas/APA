package platform

// RuntimeProfile reports build-time feature toggles.
type RuntimeProfile struct {
	Minimal bool
	TinyGo  bool
}

// Current returns the runtime profile for the active build.
func Current() RuntimeProfile {
	return RuntimeProfile{
		Minimal: MinimalBuild,
		TinyGo:  TinyGoBuild,
	}
}
