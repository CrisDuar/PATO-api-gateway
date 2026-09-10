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

type CategoryQuery struct {
	ViewName   string `form:"view" binding:"required"`
	ColumnName string `form:"column" binding:"required"`
}

type MultiFilterRequest struct {
	ViewName string `json:"viewName" binding:"required"`
	Filters  []struct {
		ColumnName  string `json:"columnName" binding:"required"`
		ColumnValue string `json:"columnValue" binding:"required"`
	} `json:"filters" binding:"required,min=2,dive"`
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

// GetViewFilteredDataMulti filtra una vista por dos o más categorías a la vez
// (ej. dominio + pais), combinando las condiciones con AND.
func (h *ViewHandler) GetViewFilteredDataMulti(c *gin.Context) {
	var request MultiFilterRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Se requieren 'viewName' y al menos 2 filtros ('columnName', 'columnValue')",
		})
		return
	}

	filters := make([]services.ColumnFilter, 0, len(request.Filters))
	for _, f := range request.Filters {
		filters = append(filters, services.ColumnFilter{
			ColumnName:  f.ColumnName,
			ColumnValue: f.ColumnValue,
		})
	}

	data, err := h.viewService.GetViewFilteredDataMulti(request.ViewName, filters)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, data)
}

// GetCategories devuelve los valores distintos de una columna dentro de una vista,
// usado tanto para categorías IPM (dominio, dimension, variable, privacion) como
// para ubicaciones geográficas (pais, region, departamento, area_geografica).
func (h *ViewHandler) GetCategories(c *gin.Context) {
	var query CategoryQuery

	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Se requieren los parámetros 'view' y 'column'",
		})
		return
	}

	values, err := h.viewService.GetDistinctColumnValues(query.ViewName, query.ColumnName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"values": values,
	})
}
