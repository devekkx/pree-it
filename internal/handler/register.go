package handler

import (
	"errors"
	"net/http"

	"github.com/devekkx/pree-it-auth/internal/service"
	"github.com/gin-gonic/gin"
)

type registerRequest struct {
	Email    string `json:"email"    binding:"required,email,max=254"`
	Password string `json:"password" binding:"required,min=12,max=128"`
}

// Register handles POST /api/v1/auth/register.
// Deliberately returns identical error messages for taken and available
// emails to prevent account enumeration.
func Register(svc *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req registerRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		user, err := svc.Register(c.Request.Context(), req.Email, req.Password)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrEmailTaken):
				// Return 201 with a generic message - don't reveal whether
				// the email is already registered (prevents enumeration).
				c.JSON(http.StatusCreated, gin.H{
					"message": "if this email is not already registered, you will receive a verification email shortly",
				})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "registration failed"})
			}
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"id":    user.ID.String(),
			"email": user.Email,
		})
	}
}
