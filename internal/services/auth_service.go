package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type AuthService struct {
	baseURL string
	client  *http.Client
}

type ValidateTokenRequest struct {
	Token string `json:"token"`
}

type ValidateTokenResponse struct {
	Valid  bool   `json:"valid"`
	UserID string `json:"user_id,omitempty"`
}

func NewAuthService(baseURL string) *AuthService {
	return &AuthService{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (s *AuthService) ValidateToken(token string) (string, error) {
	payload := ValidateTokenRequest{
		Token: token,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf(
			"failed to marshal token validation request: %w",
			err,
		)
	}

	req, err := http.NewRequest(
		http.MethodPost,
		s.baseURL+"/internal/auth/validate",
		bytes.NewReader(body),
	)
	if err != nil {
		return "", fmt.Errorf(
			"failed to create validation request: %w",
			err,
		)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf(
			"auth service request failed: %w",
			err,
		)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf(
			"authentication failed with status %d",
			resp.StatusCode,
		)
	}

	var result ValidateTokenResponse

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf(
			"failed to decode authentication response: %w",
			err,
		)
	}

	if !result.Valid || result.UserID == "" {
		return "", fmt.Errorf("invalid or expired token")
	}

	return result.UserID, nil
}
