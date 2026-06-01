// Consistent JSON envelope for all API responses across every service.
// All responses share the same shape so clients have a single contract.

package httputil

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Envelope is the top-level JSON wrapper for every API response.
type Envelope struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

// APIError carries a machine-readable code alongside a human-readable message.
// Clients should switch on Code; Message is for display only.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// OK writes a 200 response with data.
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Envelope{Success: true, Data: data})
}

// Created writes a 201 response with data.
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Envelope{Success: true, Data: data})
}

// Fail writes an error response with the given HTTP status, code, and message.
func Fail(c *gin.Context, status int, code, message string) {
	c.JSON(status, Envelope{
		Success: false,
		Error:   &APIError{Code: code, Message: message},
	})
}

// BadRequest writes a 400.
func BadRequest(c *gin.Context, message string) {
	Fail(c, http.StatusBadRequest, "BAD_REQUEST", message)
}

// Unauthorized writes a 401.
func Unauthorized(c *gin.Context, message string) {
	Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", message)
}

// Forbidden writes a 403.
func Forbidden(c *gin.Context, message string) {
	Fail(c, http.StatusForbidden, "FORBIDDEN", message)
}

// NotFound writes a 404.
func NotFound(c *gin.Context, message string) {
	Fail(c, http.StatusNotFound, "NOT_FOUND", message)
}

// Conflict writes a 409.
func Conflict(c *gin.Context, message string) {
	Fail(c, http.StatusConflict, "CONFLICT", message)
}

// InternalError writes a 500 without leaking internal details.
func InternalError(c *gin.Context) {
	Fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred")
}
