package middlewares

import (
	"fmt"
	"net/http"

	"flexirag-engine/internal/core"

	"github.com/gin-gonic/gin"
)

func RateLimit(limiter core.RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.ClientIP()
		if subjectID, ok := c.Get(ContextSubjectIDKey); ok {
			key = fmt.Sprintf("sub:%v", subjectID)
		}
		if !limiter.Allow(key) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code": http.StatusTooManyRequests,
				"msg":  "请求过于频繁，请稍后再试",
				"data": nil,
			})
			return
		}
		c.Next()
	}
}
