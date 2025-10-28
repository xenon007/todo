package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// mountStatic serves the compiled frontend from the configured directory.
func (s *Server) mountStatic() {
	if s.staticDir == "" {
		s.logger.Warn("static directory not configured; API only mode")
		return
	}

	info, err := os.Stat(s.staticDir)
	if err != nil || !info.IsDir() {
		s.logger.Warn("static directory missing", "path", s.staticDir, "error", err)
		return
	}

	indexPath := filepath.Join(s.staticDir, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		s.logger.Warn("index.html not found", "path", indexPath, "error", err)
	} else {
		s.engine.GET("/", func(c *gin.Context) {
			c.File(indexPath)
		})
		s.engine.NoRoute(func(c *gin.Context) {
			if strings.HasPrefix(c.Request.URL.Path, "/api/") {
				c.JSON(http.StatusNotFound, gin.H{"error": "endpoint not found"})
				return
			}
			c.File(indexPath)
		})
	}

	assetsDir := filepath.Join(s.staticDir, "assets")
	if _, err := os.Stat(assetsDir); err == nil {
		s.engine.StaticFS("/assets", gin.Dir(assetsDir, true))
	}

	favicon := filepath.Join(s.staticDir, "favicon.ico")
	if _, err := os.Stat(favicon); err == nil {
		s.engine.StaticFile("/favicon.ico", favicon)
	}
}
