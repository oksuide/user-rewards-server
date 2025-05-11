package handler

import (
	"Test/internal/service"
	"database/sql"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	CodeUserNotFound   = "user_not_found"
	CodeInvalidRequest = "invalid_request"
	CodeInternalError  = "internal_error"
	CodeUnauthorized   = "unauthorized"
)

type UserHandler struct {
	service *service.UserService
}

func NewUserHandler(service *service.UserService) *UserHandler {
	return &UserHandler{service: service}
}

func (h *UserHandler) sendError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{
		"error": message,
		"code":  code,
	})
}

func (h *UserHandler) GetUserStatus(c *gin.Context) {
	userID := c.Param("id")

	status, err := h.service.GetUserStatus(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.sendError(c, http.StatusNotFound, CodeUserNotFound, "user not found")
		} else {
			log.Printf("GetUserStatus error: %v", err)
			h.sendError(c, http.StatusInternalServerError, CodeInternalError, "failed to get user status")
		}
		return
	}

	c.JSON(http.StatusOK, status)
}

func (h *UserHandler) CompleteTask(c *gin.Context) {
	userID := c.Param("id")

	var req struct {
		TaskName string `json:"task_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, CodeInvalidRequest, "invalid request format")
		return
	}

	if err := h.service.CompleteTask(c.Request.Context(), userID, req.TaskName); err != nil {
		log.Printf("CompleteTask error: %v", err)
		h.sendError(c, http.StatusInternalServerError, CodeInternalError, "failed to complete task")
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (h *UserHandler) SetReferrer(c *gin.Context) {
	userID := c.Param("id")

	var req struct {
		ReferrerID string `json:"referrer_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, CodeInvalidRequest, "invalid request format")
		return
	}

	if err := h.service.SetReferrer(c.Request.Context(), userID, req.ReferrerID); err != nil {
		log.Printf("SetReferrer error: %v", err)
		h.sendError(c, http.StatusInternalServerError, CodeInternalError, "failed to set referrer")
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (h *UserHandler) GetLeaderboard(c *gin.Context) {
	entries, err := h.service.GetLeaderboard(c.Request.Context(), 10)
	if err != nil {
		log.Printf("GetLeaderboard error: %v", err)
		h.sendError(c, http.StatusInternalServerError, CodeInternalError, "failed to get leaderboard")
		return
	}

	c.JSON(http.StatusOK, entries)
}
