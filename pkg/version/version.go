package version

import (
	"runtime"
)

var (
	// Version gets defined by the build system.
	Version = "0.0.0-dev"

	// Revision indicates the commit this binary was built from.
	Revision string

	// BuildDate indicates the date this binary was built.
	BuildDate string

	// GoVersion running this binary.
	GoVersion = runtime.Version()
)
