package public

import (
	"embed"
	"io/fs"
)

//go:embed dist/*

var distFS embed.FS

var (
	Dist, _ = fs.Sub(distFS, "dist")
)
