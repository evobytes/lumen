// Package version exposes build metadata injected with -ldflags -X.
package version

import "fmt"

// Set at build time; see scripts/version and scripts/build-binary.
var (
	Version = "0.0.0-dev"
	Commit  = "unknown"
	Built   = "unknown"
)

// Info is the JSON shape served at /api/v1/version.
type Info struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Built   string `json:"built"`
}

// Get returns the current build info.
func Get() Info { return Info{Version: Version, Commit: Commit, Built: Built} }

// String renders the one-line form printed by -version.
func String() string {
	return fmt.Sprintf("lumen %s (%s, %s)", Version, Commit, Built)
}
