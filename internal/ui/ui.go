// Package ui embeds the web pages the application's windows (and any browser,
// OBS browser source or phone) load: the control page and the output page.
// They are static HTML, CSS and native ES modules: there is no build step.
package ui

import (
	"embed"
	"io/fs"
)

//go:embed assets
var assets embed.FS

// Assets returns the embedded files rooted at the assets directory.
func Assets() fs.FS {
	sub, err := fs.Sub(assets, "assets")
	if err != nil {
		panic(err) // the directory is embedded at build time; a failure is a programmer error
	}
	return sub
}
