// Package web provides embedded static file serving
// Author: Done-0
// Created: 2026-01-22
package web

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed static/*
var staticFiles embed.FS

// RegisterStaticRoutes registers routes for serving embedded static files
func RegisterStaticRoutes(r *gin.Engine) {
	// Get the subdirectory
	subFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err)
	}

	// Serve static files
	r.StaticFS("/static", http.FS(subFS))

	// Redirect root to index.html
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/static/index.html")
	})

	// Serve index.html for SPA fallback
	r.NoRoute(func(c *gin.Context) {
		// If it's an API request, return 404
		if len(c.Request.URL.Path) > 4 && c.Request.URL.Path[:4] == "/api" {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		// Otherwise serve the index.html
		c.Redirect(http.StatusMovedPermanently, "/static/index.html")
	})
}
