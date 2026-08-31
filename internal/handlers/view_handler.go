package handlers

import (
	"net/http"

	"backend/internal/services"

	"github.com/gin-gonic/gin"
)

type ViewHandler struct {
	viewService *services.ViewService
}
type FilterRequest struct {
	ViewName    string `json:"viewName" binding:"required"`
	ColumnName  string `json:"columnName" binding:"required"`
	ColumnValue string `json:"columnValue" binding:"required"`
}

func NewViewHandler(viewService *services.ViewService) *ViewHandler {
	return &ViewHandler{
		viewService: viewService,
	}
}

func ViewTotalHandler(viewService *services.ViewService, viewName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		data, err := viewService.GetViewTotalData(viewName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve data"})
			return
		}
		c.JSON(http.StatusOK, data)
	}
}

func (h *ViewHandler) GetViewFilteredData(c *gin.Context) {
	var request FilterRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Datos inválidos",
		})
		return
	}

	data, err := h.viewService.GetViewFilteredData(
		request.ViewName,
		request.ColumnName,
		request.ColumnValue,
	)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, data)
}
