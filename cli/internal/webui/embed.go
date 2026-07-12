// Package webui embeds the built web UI (source in cli/ui, built with
// `npm run build` which emits into this package's dist/) and serves it.
package webui

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:dist
var dist embed.FS

// Handler serves the embedded single-page UI, falling back to index.html for
// unknown paths so client-side routes keep working.
func Handler() http.Handler {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err) // embed layout is fixed at build time
	}
	fileServer := http.FileServerFS(sub)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if p != "/" {
			if _, err := fs.Stat(sub, p[1:]); err != nil {
				r.URL.Path = "/" // SPA fallback
			}
		}
		fileServer.ServeHTTP(w, r)
	})
}
