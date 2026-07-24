// Package web embeds the compiled frontend so a single Go binary can serve the
// entire application (UI + API) with no external web server.
//
// The frontend build output (SvelteKit adapter-static SPA) is copied into
// web/dist during the all-in-one Docker build. When web/dist contains only the
// placeholder .gitkeep (i.e. a plain `go build` of the repo), static serving is
// disabled and the backend behaves as an API-only server — which keeps local
// development (separate `pnpm dev` frontend) working unchanged.
package web

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed all:dist
var dist embed.FS

// Disabled reports whether no real frontend was embedded (only the placeholder),
// meaning static serving should be skipped.
func Disabled() bool {
	return !fileExists("index.html")
}

// fileExists reports whether name exists inside the embedded dist tree.
func fileExists(name string) bool {
	_, err := fs.Stat(dist, path.Join("dist", name))
	return err == nil
}

// indexHTML returns the embedded SPA fallback page.
func indexHTML() ([]byte, error) {
	return dist.ReadFile("dist/index.html")
}

// RegisterSPA serves the embedded single-page application on the Gin engine.
//
// It must be called AFTER all /api routes are registered; a NoRoute handler
// serves static assets when present and falls back to index.html for client
// routes (enabling deep links like /tasks/123 in SPA mode). Paths under /api
// are left to return a JSON 404 so API consumers never receive HTML.
func RegisterSPA(r *gin.Engine) {
	if Disabled() {
		return
	}

	assets, err := fs.Sub(dist, "dist")
	if err != nil {
		return
	}
	fileServer := http.FileServer(http.FS(assets))

	r.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path

		// Never serve HTML for API misses — return a JSON 404 instead.
		if strings.HasPrefix(p, "/api/") || p == "/api" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{"code": "NOT_FOUND", "message": "resource not found"},
			})
			return
		}

		// Serve a real file if it exists (assets, icons, manifest, sw.js, ...).
		// Otherwise fall back to the SPA entry point for client-side routing.
		clean := strings.TrimPrefix(path.Clean(p), "/")
		if clean != "" && clean != "." && !strings.HasPrefix(clean, "..") && fileExists(clean) {
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}

		index, err := indexHTML()
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.Status(http.StatusOK)
		_, _ = c.Writer.Write(index)
	})
}
