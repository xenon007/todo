package server

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"todo/internal/storage/sqlite"
)

// Server provides HTTP handlers for the Scrum board backend.
type Server struct {
	engine    *gin.Engine
	store     *sqlite.Store
	logger    *slog.Logger
	staticDir string
}

// New constructs the HTTP server with routes and middleware configured.
func New(store *sqlite.Store, logger *slog.Logger, staticDir string) *Server {
	if logger == nil {
		logger = slog.Default()
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.LoggerWithWriter(gin.DefaultWriter, "/api"))

	srv := &Server{
		engine:    router,
		store:     store,
		logger:    logger,
		staticDir: staticDir,
	}

	srv.registerRoutes()
	return srv
}

// Engine exposes the underlying Gin engine.
func (s *Server) Engine() *gin.Engine {
	return s.engine
}

// registerRoutes wires all API and static handlers together.
func (s *Server) registerRoutes() {
	api := s.engine.Group("/api")
	{
		api.GET("/healthz", s.handleHealth)

		projects := api.Group("/projects")
		{
			projects.GET("", s.handleListProjects)
			projects.POST("", s.handleCreateProject)
			projects.PUT(":id", s.handleUpdateProject)
			projects.DELETE(":id", s.handleDeleteProject)
			projects.GET(":id/tasks", s.handleListTasks)
			projects.POST(":id/tasks", s.handleCreateTask)
		}

		api.PUT("/tasks/:id", s.handleUpdateTask)
		api.DELETE("/tasks/:id", s.handleDeleteTask)
	}

	s.mountStatic()
}

// handleHealth provides a basic readiness endpoint.
func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// parseID converts a path parameter to int64 with error handling.
func parseID(c *gin.Context, name string) (int64, bool) {
	raw := c.Param(name)
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid identifier"})
		return 0, false
	}
	return id, true
}

// respondError logs the error and returns a JSON payload.
func (s *Server) respondError(c *gin.Context, status int, err error) {
	if err != nil {
		s.logger.Error("request failed", slog.String("path", c.FullPath()), slog.String("error", err.Error()))
	}
	c.JSON(status, gin.H{"error": err.Error()})
}

// respondSuccess wraps a payload in a JSON envelope for consistency.
func respondSuccess(c *gin.Context, status int, payload any) {
	if payload == nil {
		c.Status(status)
		return
	}
	c.JSON(status, payload)
}
