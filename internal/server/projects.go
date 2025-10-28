package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type projectRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// handleListProjects returns all available projects.
func (s *Server) handleListProjects(c *gin.Context) {
	projects, err := s.store.ListProjects(c.Request.Context())
	if err != nil {
		s.respondError(c, http.StatusInternalServerError, err)
		return
	}
	respondSuccess(c, http.StatusOK, gin.H{"projects": projects})
}

// handleCreateProject creates a new project entity.
func (s *Server) handleCreateProject(c *gin.Context) {
	var req projectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondError(c, http.StatusBadRequest, err)
		return
	}

	project, err := s.store.CreateProject(c.Request.Context(), req.Name, req.Color)
	if err != nil {
		s.respondError(c, http.StatusBadRequest, err)
		return
	}
	respondSuccess(c, http.StatusCreated, gin.H{"project": project})
}

// handleUpdateProject renames or recolors an existing project.
func (s *Server) handleUpdateProject(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}

	var req projectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondError(c, http.StatusBadRequest, err)
		return
	}

	project, err := s.store.UpdateProject(c.Request.Context(), id, req.Name, req.Color)
	if err != nil {
		s.respondError(c, http.StatusBadRequest, err)
		return
	}
	respondSuccess(c, http.StatusOK, gin.H{"project": project})
}

// handleDeleteProject removes a project and all related tasks.
func (s *Server) handleDeleteProject(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	if err := s.store.DeleteProject(c.Request.Context(), id); err != nil {
		s.respondError(c, http.StatusBadRequest, err)
		return
	}
	respondSuccess(c, http.StatusOK, gin.H{"status": "deleted"})
}
