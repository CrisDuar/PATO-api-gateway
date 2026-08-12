package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func ProxyHandler(proxy http.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {

		proxy.ServeHTTP(
			c.Writer,
			c.Request,
		)
	}
}
