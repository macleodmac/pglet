package main

import (
	"net/http"
	"strings"
)

// spaHandler wraps the API handler and falls back to the frontend filesystem.
// Non-API paths serve static files; unknown paths serve index.html for SPA routing.
func spaHandler(apiHandler http.Handler, frontendFS http.FileSystem) http.Handler {
	fileServer := http.FileServer(frontendFS)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/api" {
			apiHandler.ServeHTTP(w, r)
			return
		}
		// Try to serve the file directly
		if r.URL.Path != "/" {
			if f, err := frontendFS.Open(r.URL.Path); err == nil {
				f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		// Fallback to index.html for SPA
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}
