package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const headerRequestID = "X-Request-ID"

// RequestID injects a unique request ID into every request.
// If the upstream proxy (Caddy) already set the header, it is reused.
// The ID is stored in the Gin context and echoed back in the response.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(headerRequestID)
		if id == "" {
			id = uuid.NewString()
		}
		c.Set("request_id", id)
		c.Header(headerRequestID, id)
		c.Next()
	}
}
