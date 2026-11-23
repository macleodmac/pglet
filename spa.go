package main

import (
	"bytes"
	"html"
	"io/fs"
	"net/http"
	"strings"
)

// spaHandler serves the API under prefix+"/api/" and the frontend everywhere
// else under prefix, falling back to index.html for unknown paths.
func spaHandler(apiHandler http.Handler, frontend fs.FS, prefix string) (http.Handler, error) {
	index, err := fs.ReadFile(frontend, "index.html")
	if err != nil {
		return nil, err
	}
	// The build uses relative asset URLs, so a <base> makes them resolve
	// under the prefix from any page.
	base := `<head><base href="` + html.EscapeString(prefix+"/") + `">`
	index = bytes.Replace(index, []byte("<head>"), []byte(base), 1)

	fileServer := http.StripPrefix(prefix, http.FileServer(http.FS(frontend)))
	serveIndex := func(w http.ResponseWriter) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Write(index)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if prefix != "" && path == prefix {
			http.Redirect(w, r, prefix+"/", http.StatusMovedPermanently)
			return
		}
		if !strings.HasPrefix(path, prefix+"/") {
			http.NotFound(w, r)
			return
		}
		if strings.HasPrefix(path, prefix+"/api/") {
			apiHandler.ServeHTTP(w, r)
			return
		}
		rel := strings.TrimPrefix(path, prefix+"/")
		if rel != "" && rel != "index.html" {
			if info, err := fs.Stat(frontend, rel); err == nil && !info.IsDir() {
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		serveIndex(w)
	}), nil
}
