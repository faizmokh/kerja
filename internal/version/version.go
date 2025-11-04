package version

import (
	"fmt"
)

// These variables are populated at build time via -ldflags.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// Info returns a human-friendly version string that surfaces build metadata.
func Info() string {
	return fmt.Sprintf("%s (commit %s, built %s)", Version, Commit, Date)
}
