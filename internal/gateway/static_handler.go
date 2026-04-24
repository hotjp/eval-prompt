// Package gateway implements L5-Gateway layer: TLS termination, protocol adaptation,
// middleware, request routing, and static resource serving.
package gateway

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// StaticFS is the embedded filesystem for React static assets.
//
//go:embed web/dist
var staticFS embed.FS

// WebDist returns the static filesystem as an fs.FS for use with http.FileServer.
func WebDist() fs.FS {
	subFS, err := fs.Sub(staticFS, "web/dist")
	if err != nil {
		// Fallback to root if "web/dist" doesn't exist
		return staticFS
	}
	return subFS
}

// StaticHandler returns an HTTP handler for serving static assets with SPA fallback.
func StaticHandler() http.Handler {
	fs := WebDist()
	fileServer := http.FileServer(http.FS(fs))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the file directly
		filePath := path.Join("/", r.URL.Path)

		// Check if file exists
		if _, err := fs.Open(filePath); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		// SPA fallback: serve index.html for non-file paths
		if !strings.Contains(r.URL.Path, ".") {
			indexPath := path.Join("/", "index.html")
			if f, err := fs.Open(indexPath); err == nil {
				f.Close()
				http.ServeFileFS(w, r, fs, indexPath)
				return
			}
		}

		// Fallback to index.html for SPA routing
		http.ServeFileFS(w, r, fs, "/index.html")
	})
}

// RegisterStaticRoutes registers static file routes on the given mux.
func RegisterStaticRoutes(mux *http.ServeMux) {
	// Serve static assets from embedded filesystem
	mux.Handle("/", StaticHandler())
}
