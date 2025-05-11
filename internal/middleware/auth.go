package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	CodeAuthHeaderMissing = "auth_header_missing"
	CodeInvalidAuthHeader = "invalid_auth_header"
	CodeInvalidToken      = "invalid_token"
)

func AuthMiddleware(jwtService *JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "authorization header is required",
				"code":  CodeAuthHeaderMissing,
			})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid authorization header format",
				"code":  CodeInvalidAuthHeader,
			})
			return
		}

		token := parts[1]
		claims, err := jwtService.ValidateToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid token",
				"code":  CodeInvalidToken,
			})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Next()
	}
}
