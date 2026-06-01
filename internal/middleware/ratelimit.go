package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimiter implements a per-IP sliding window counter backed by Redis.
// If Redis is unavailable it fails open - requests are allowed through -
// because dropping all traffic due to a Redis blip is worse than
// temporarily relaxing rate limits.
type RateLimiter struct {
	rdb    *redis.Client
	limit  int
	window time.Duration
}

// NewRateLimiter returns a RateLimiter.
// limit is the maximum number of requests allowed within window.
func NewRateLimiter(rdb *redis.Client, limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{rdb: rdb, limit: limit, window: window}
}

// slidingWindowScript is an atomic Lua script implementing a sliding log.
// Returns 1 if the request is allowed, 0 if it should be rejected.
// Uses ZADD + ZREMRANGEBYSCORE + ZCARD - all in one round trip.
const slidingWindowScript = `
local key      = KEYS[1]
local now      = tonumber(ARGV[1])
local window   = tonumber(ARGV[2])
local limit    = tonumber(ARGV[3])
local expire   = tonumber(ARGV[4])

redis.call('ZREMRANGEBYSCORE', key, 0, now - window)

local count = redis.call('ZCARD', key)
if count >= limit then
    return 0
end

redis.call('ZADD', key, now, now .. '-' .. math.random(1, 1000000))
redis.call('EXPIRE', key, expire)
return 1
`

// Middleware returns a Gin handler that enforces the rate limit.
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	script := redis.NewScript(slidingWindowScript)

	return func(c *gin.Context) {
		ctx := c.Request.Context()
		now := time.Now().UnixMilli()
		windowMs := rl.window.Milliseconds()

		// Key per IP + route - prevents one slow route from penalising others.
		key := fmt.Sprintf("rl:%s:%s", c.ClientIP(), c.FullPath())

		// expire slightly longer than the window so the key is cleaned up
		expireSeconds := int64(rl.window.Seconds()) + 1

		result, err := script.Run(ctx, rl.rdb,
			[]string{key},
			now,
			windowMs,
			rl.limit,
			expireSeconds,
		).Int()

		if err != nil {
			// Redis unavailable - fail open, log the error via context errors.
			_ = c.Error(fmt.Errorf("ratelimit: redis error: %w", err))
			c.Next()
			return
		}

		if result == 0 {
			c.Header("Retry-After", fmt.Sprintf("%d", int(rl.window.Seconds())))
			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", rl.limit))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "too many requests",
			})
			return
		}

		c.Next()
	}
}
