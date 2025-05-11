package auth

import (
	"Test/config"
	"Test/internal/repository"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	CodeInvalidRequest     = "invalid_request"
	CodeInvalidPassword    = "invalid_password"
	CodeEmailExists        = "email_exists"
	CodeInvalidCredentials = "invalid_credentials"
	CodeInternalError      = "internal_error"
	CodeLoginRequired      = "login_required"
)

type AuthHandler struct {
	service *AuthService
	cfg     *config.Config
}

func NewAuthHandler(repo repository.UserRepository, cfg *config.Config) *AuthHandler {
	jwt := NewJWTService(cfg.JWT.SecretKey, cfg.JWT.Expiration)
	service := NewAuthService(repo, jwt)
	return &AuthHandler{service: service, cfg: cfg}
}

// Регистрация
func (h *AuthHandler) RegisterHandler(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, CodeInvalidRequest, "invalid request format", err.Error(), "")
		return
	}

	if len(req.Password) < 8 {
		h.sendError(c, http.StatusBadRequest, CodeInvalidPassword, "password must be at least 8 characters", "", "password")
		return
	}

	err := h.service.Register(c.Request.Context(), req.Username, req.Password, req.Email)
	if err != nil {
		h.handleServiceError(c, err, req.Email)
		return
	}

	token, err := h.service.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		log.Printf("Auto-login failed: %v", err)
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "registration successful, but login required",
			"code":    CodeLoginRequired,
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":    true,
		"token":      token,
		"expires_in": h.cfg.JWT.Expiration / time.Second,
		"user": gin.H{
			"username": req.Username,
			"email":    req.Email,
		},
	})
}

func (h *AuthHandler) LoginHandler(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, CodeInvalidRequest, "invalid request format", err.Error(), "")
		return
	}

	token, err := h.service.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		h.handleServiceError(c, err, req.Email)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":      token,
		"expires_in": h.cfg.JWT.Expiration / time.Second,
	})
}

func (h *AuthHandler) handleServiceError(c *gin.Context, err error, email string) {
	switch {
	case strings.Contains(err.Error(), "email already exists"):
		h.sendError(c, http.StatusConflict, CodeEmailExists, fmt.Sprintf("email '%s' already in use", email), "", "email")
	case strings.Contains(err.Error(), "password too short"):
		h.sendError(c, http.StatusBadRequest, CodeInvalidPassword, "password must be at least 8 characters", "", "password")
	case strings.Contains(err.Error(), "invalid credentials"):
		h.sendError(c, http.StatusUnauthorized, CodeInvalidCredentials, "invalid email or password", "", "password")
	default:
		log.Printf("Unexpected error: %v", err)
		h.sendError(c, http.StatusInternalServerError, CodeInternalError, "internal server error", err.Error(), "")
	}
}

func (h *AuthHandler) sendError(
	c *gin.Context,
	status int,
	code string,
	message string,
	details string,
	field string,
) {
	response := gin.H{
		"error": message,
		"code":  code,
	}
	if details != "" {
		response["details"] = details
	}
	if field != "" {
		response["field"] = field
	}
	c.JSON(status, response)
}
