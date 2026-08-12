package middleware

import (
	"net/http"
	"strings"

	"backend/internal/services"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(
	authService *services.AuthService,
) gin.HandlerFunc {

	return func(c *gin.Context) {

		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header required",
				"code":  "UNAUTHORIZED",
			})

			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)

		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization header format",
				"code":  "UNAUTHORIZED",
			})

			c.Abort()
			return
		}

		token := parts[1]

		userID, err := authService.ValidateToken(token)

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
				"code":  "UNAUTHORIZED",
			})

			c.Abort()
			return
		}

		// El Gateway ya sabe quién es el usuario.
		c.Set("userID", userID)

		// Le pasamos la identidad al microservicio.
		c.Request.Header.Set("X-User-ID", userID)

		c.Next()
	}
}
