package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

type PredictionService struct {
	baseURL    string
	httpClient *http.Client
}

func NewPredictionService(baseURL string) *PredictionService {
	return &PredictionService{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Tipos soportados por la API de IA (src/api/main.py del servicio de predicción).
var allowedPredictionTypes = map[string]bool{
	"ipm":                   true,
	"privacion":             true,
	"porcentaje":            true,
	"dimension":             true,
	"incidence_by_sex":      true,
	"incidence_by_head_sex": true,
	"contribution_latam":    true,
	"national_poverty":      true,
	"poverty_by_age":        true,
}

var ErrPredictionTypeNotAllowed = errors.New("tipo de predicción no permitido")

type PredictionUpstreamError struct {
	StatusCode int
	Body       string
}

func (e *PredictionUpstreamError) Error() string {
	return fmt.Sprintf("el servicio de IA respondió con error (%d): %s", e.StatusCode, e.Body)
}

func (ps *PredictionService) Predict(predictionType string, payload map[string]interface{}) (map[string]interface{}, error) {
	if !allowedPredictionTypes[predictionType] {
		return nil, ErrPredictionTypeNotAllowed
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("payload inválido: %w", err)
	}

	url := fmt.Sprintf("%s/predict/%s", ps.baseURL, predictionType)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("no se pudo construir la petición al servicio de IA: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := ps.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("no se pudo contactar al servicio de IA: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer la respuesta del servicio de IA: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, &PredictionUpstreamError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("respuesta inválida del servicio de IA: %w", err)
	}

	return result, nil
}
