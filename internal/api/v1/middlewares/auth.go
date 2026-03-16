package middlewares

import (
	"net/http"
	"strings"

	"flexirag-engine/internal/core"

	"github.com/gin-gonic/gin"
)

func Auth(authService core.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearer(c.GetHeader("Authorization"))
		subject, err := authService.ValidateToken(c.Request.Context(), token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": http.StatusUnauthorized,
				"msg":  "未授权，请检查 Bearer Token",
				"data": nil,
			})
			return
		}
		c.Set(ContextSubjectIDKey, subject.ID)
		c.Next()
	}
}

func extractBearer(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
