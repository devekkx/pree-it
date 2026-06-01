package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Logger is a structured Zap middleware.
// It deliberately omits the Authorization header, cookie values,
// and request bodies to prevent credential leakage in logs.
func Logger(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path // captured before c.Next() may mutate it

		c.Next()

		// Errors accumulated by handlers via c.Error()
		var errMessages []string
		for _, e := range c.Errors {
			errMessages = append(errMessages, e.Error())
		}

		fields := []zap.Field{
			zap.String("request_id", c.GetString("request_id")),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
			zap.Int("response_bytes", c.Writer.Size()),
		}

		if len(errMessages) > 0 {
			fields = append(fields, zap.Strings("errors", errMessages))
		}

		switch {
		case c.Writer.Status() >= 500:
			log.Error("request", fields...)
		case c.Writer.Status() >= 400:
			log.Warn("request", fields...)
		default:
			log.Info("request", fields...)
		}
	}
}
