package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type AuthService struct {
	valkey *redis.Client
}

func NewAuthService(valkey *redis.Client) *AuthService {
	return &AuthService{
		valkey: valkey,
	}
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func sessionKey(token string) string {
	return fmt.Sprintf("session:%s", hashToken(token))
}

func (s *AuthService) ValidateToken(token string) (string, error) {
	userID, err := s.valkey.Get(
		context.Background(),
		sessionKey(token),
	).Result()

	if errors.Is(err, redis.Nil) {
		return "", fmt.Errorf("invalid or expired token")
	}

	if err != nil {
		return "", fmt.Errorf("valkey error: %w", err)
	}

	return userID, nil
}
