package server

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"todo/internal/models"
)

type taskRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
}

// handleListTasks fetches tasks for a project.
func (s *Server) handleListTasks(c *gin.Context) {
	projectID, ok := parseID(c, "id")
	if !ok {
		return
	}

	tasks, err := s.store.ListTasks(c.Request.Context(), projectID)
	if err != nil {
		s.respondError(c, http.StatusInternalServerError, err)
		return
	}
	respondSuccess(c, http.StatusOK, gin.H{"tasks": tasks})
}

// handleCreateTask inserts a new task into a project column.
func (s *Server) handleCreateTask(c *gin.Context) {
	projectID, ok := parseID(c, "id")
	if !ok {
		return
	}

	var req taskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondError(c, http.StatusBadRequest, err)
		return
	}
	if req.Title == nil || *req.Title == "" {
		s.respondError(c, http.StatusBadRequest, fmt.Errorf("title is required"))
		return
	}

	task, err := s.store.CreateTask(c.Request.Context(), models.Task{
		ProjectID:   projectID,
		Title:       *req.Title,
		Description: getString(req.Description),
		Status:      getString(req.Status),
	})
	if err != nil {
		s.respondError(c, http.StatusBadRequest, err)
		return
	}
	respondSuccess(c, http.StatusCreated, gin.H{"task": task})
}

// handleUpdateTask updates task fields such as status or description.
func (s *Server) handleUpdateTask(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}

	var req taskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondError(c, http.StatusBadRequest, err)
		return
	}

	updates := map[string]any{}
	if req.Title != nil && *req.Title != "" {
		updates["title"] = *req.Title
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}

	task, err := s.store.UpdateTask(c.Request.Context(), id, updates)
	if err != nil {
		s.respondError(c, http.StatusBadRequest, err)
		return
	}
	respondSuccess(c, http.StatusOK, gin.H{"task": task})
}

// handleDeleteTask removes a task completely.
func (s *Server) handleDeleteTask(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	if err := s.store.DeleteTask(c.Request.Context(), id); err != nil {
		s.respondError(c, http.StatusBadRequest, err)
		return
	}
	respondSuccess(c, http.StatusOK, gin.H{"status": "deleted"})
}

func getString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
