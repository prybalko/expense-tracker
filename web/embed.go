// Package web exposes the embedded frontend bundle that is served by the
// Go binary. The dist directory is produced by `cd web/app && npm run build`.
// A placeholder index.html is committed so that //go:embed succeeds before
// the first real build.
package web

import "embed"

// DistFS is the embedded frontend bundle (web/dist), served by the Go
// binary as the SPA. Imported by cmd/server.
//
//go:embed all:dist
var DistFS embed.FS
