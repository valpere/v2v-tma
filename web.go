// Package v2vtma embeds the Mini App so the server ships as one binary.
package v2vtma

import (
	"embed"
	"io/fs"
)

//go:embed web
var files embed.FS

// Client is the contents of web/ (index.html, assets/…).
func Client() fs.FS {
	sub, err := fs.Sub(files, "web")
	if err != nil {
		panic(err) // the directory is embedded at build time
	}
	return sub
}
