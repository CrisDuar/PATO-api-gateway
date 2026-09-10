package handlers

import (
	"errors"
	"net/http"

	"backend/internal/services"

	"github.com/gin-gonic/gin"
)

type PredictionHandler struct {
	predictionService *services.PredictionService
}

func NewPredictionHandler(predictionService *services.PredictionService) *PredictionHandler {
	return &PredictionHandler{
		predictionService: predictionService,
	}
}

// Predict reenvía la solicitud de predicción a la API de IA (POST /predictions/:type)
// y traduce sus fallos a respuestas HTTP consistentes para el cliente.
func (h *PredictionHandler) Predict(c *gin.Context) {
	predictionType := c.Param("type")

	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Datos inválidos",
		})
		return
	}

	data, err := h.predictionService.Predict(predictionType, payload)
	if err != nil {
		var upstreamErr *services.PredictionUpstreamError

		switch {
		case errors.Is(err, services.ErrPredictionTypeNotAllowed):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})

		case errors.As(err, &upstreamErr):
			status := upstreamErr.StatusCode
			if status < 400 || status > 599 {
				status = http.StatusBadGateway
			}
			c.JSON(status, gin.H{
				"error": "El servicio de predicción no pudo generar el resultado",
			})

		default:
			c.JSON(http.StatusBadGateway, gin.H{
				"error": "No se pudo contactar al servicio de predicción",
			})
		}
		return
	}

	c.JSON(http.StatusOK, data)
}
