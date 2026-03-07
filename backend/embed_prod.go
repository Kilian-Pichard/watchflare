//go:build embed_frontend

package main

import "embed"

//go:embed all:frontend/dist
var frontendFS embed.FS
