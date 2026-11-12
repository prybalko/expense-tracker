// Package web embeds the built frontend bundle.
// A placeholder dist/index.html is committed so //go:embed succeeds before the first build.
package web

import "embed"

// DistFS is the embedded web/dist bundle.
//
//go:embed all:dist
var DistFS embed.FS
