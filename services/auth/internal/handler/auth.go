// HTTP layer for the auth service.
// Responsibilities: decode requests, validate input, call the service,
// map domain errors to HTTP status codes, encode responses.
// No business logic lives here.

package handler

import (
	"errors"
	"strings"

	"github.com/devekkx/pree-it/auth/internal/domain"
	"github.com/devekkx/pree-it/auth/internal/service"
	"github.com/devekkx/pree-it/pkg/httputil"
	"github.com/devekkx/pree-it/pkg/middleware"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AuthHandler handles all auth HTTP endpoints.
type AuthHandler struct {
	svc *service.AuthService
	log *zap.Logger
}

// NewAuthHandler constructs an AuthHandler.
func NewAuthHandler(svc *service.AuthService, log *zap.Logger) *AuthHandler {
	return &AuthHandler{svc: svc, log: log}
}

// Request / Response DTOs
// DTOs are defined in the handler layer — they are HTTP concerns, not domain concerns.

type registerRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

type loginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type userResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type authResponse struct {
	User   userResponse       `json:"user"`
	Tokens *service.TokenPair `json:"tokens"`
}

// Handlers
// Register godoc
// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.BadRequest(c, "validation failed: "+err.Error())
		return
	}
	req.Email = normaliseEmail(req.Email)

	user, tokens, err := h.svc.Register(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		h.handleServiceError(c, err, "register")
		return
	}

	httputil.Created(c, authResponse{
		User:   userResponse{ID: user.ID, Email: user.Email},
		Tokens: tokens,
	})
}

// Login godoc
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.BadRequest(c, "validation failed: "+err.Error())
		return
	}
	req.Email = normaliseEmail(req.Email)

	user, tokens, err := h.svc.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		h.handleServiceError(c, err, "login")
		return
	}

	httputil.OK(c, authResponse{
		User:   userResponse{ID: user.ID, Email: user.Email},
		Tokens: tokens,
	})
}

// Refresh godoc
// POST /api/v1/auth/refresh
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.BadRequest(c, "refresh_token is required")
		return
	}

	tokens, err := h.svc.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		h.handleServiceError(c, err, "refresh")
		return
	}

	httputil.OK(c, tokens)
}

// Logout godoc
// POST /api/v1/auth/logout  (requires JWT)
func (h *AuthHandler) Logout(c *gin.Context) {
	userID, ok := c.Get(string(middleware.KeyUserID))
	if !ok {
		httputil.Unauthorized(c, "missing user context")
		return
	}

	if err := h.svc.Logout(c.Request.Context(), userID.(string)); err != nil {
		h.log.Error("logout failed", zap.String("user_id", userID.(string)), zap.Error(err))
		httputil.InternalError(c)
		return
	}

	httputil.OK(c, gin.H{"message": "logged out successfully"})
}

// helpers

// handleServiceError maps domain errors to HTTP responses.
// Unexpected errors are logged at Error level; domain errors are not logged
// (they are expected control-flow, not operational failures).
func (h *AuthHandler) handleServiceError(c *gin.Context, err error, op string) {
	switch {
	case errors.Is(err, domain.ErrEmailTaken):
		httputil.Conflict(c, "email address is already registered")
	case errors.Is(err, domain.ErrInvalidCredentials):
		httputil.Unauthorized(c, "invalid email or password")
	case errors.Is(err, domain.ErrInvalidToken):
		httputil.Unauthorized(c, "invalid or expired token")
	case errors.Is(err, domain.ErrUserNotFound):
		httputil.NotFound(c, "user not found")
	default:
		h.log.Error("unexpected service error",
			zap.String("op", op),
			zap.String("request_id", c.GetString(string(middleware.KeyRequestID))),
			zap.Error(err),
		)
		httputil.InternalError(c)
	}
}

func normaliseEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
