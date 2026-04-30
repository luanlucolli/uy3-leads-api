package frontend

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var Files embed.FS

var _ fs.FS = Files
