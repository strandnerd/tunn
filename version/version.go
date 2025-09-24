package version

// Version mirrors the build's semantic version; overridden via ldflags at release time.
var Version = "dev"

// String returns the current version string.
func String() string {
	return Version
}
