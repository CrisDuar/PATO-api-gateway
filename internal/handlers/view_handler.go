package handlers

import (
	"net/http"

	"backend/internal/services"

	"github.com/gin-gonic/gin"
)

func ViewHandler(viewService *services.ViewService, viewName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		data, err := viewService.GetViewData(viewName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve data"})
			return
		}
		c.JSON(http.StatusOK, data)
	}
}
