//go:build !prod

package main

import "io/fs"

func embeddedUI() fs.FS { return nil }
