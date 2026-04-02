//go:build prod

package main

import (
	"embed"
	"io/fs"
)

//go:embed ui-dist
var embeddedUIFiles embed.FS

func embeddedUI() fs.FS {
	f, err := fs.Sub(embeddedUIFiles, "ui-dist")
	if err != nil {
		return nil
	}
	return f
}
