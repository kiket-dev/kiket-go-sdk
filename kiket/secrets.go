package kiket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

const apiPrefix = "/api/v1"

// secretManager implements the SecretManager interface.
type secretManager struct {
	client      Client
	extensionID string
}

// NewSecretManager creates a new secret manager.
func NewSecretManager(client Client, extensionID string) SecretManager {
	return &secretManager{
		client:      client,
		extensionID: extensionID,
	}
}

func (s *secretManager) Get(ctx context.Context, key string) (string, error) {
	if s.extensionID == "" {
		return "", errors.New("extension ID required for secret operations")
	}

	path := fmt.Sprintf("%s/extensions/%s/secrets/%s", apiPrefix, s.extensionID, key)
	resp, err := s.client.Get(ctx, path, nil)
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			return "", nil
		}
		return "", err
	}

	var result struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Value, nil
}

func (s *secretManager) Set(ctx context.Context, key string, value string) error {
	if s.extensionID == "" {
		return errors.New("extension ID required for secret operations")
	}

	path := fmt.Sprintf("%s/extensions/%s/secrets/%s", apiPrefix, s.extensionID, key)
	_, err := s.client.Post(ctx, path, map[string]string{"value": value}, nil)
	return err
}

func (s *secretManager) Delete(ctx context.Context, key string) error {
	if s.extensionID == "" {
		return errors.New("extension ID required for secret operations")
	}

	path := fmt.Sprintf("%s/extensions/%s/secrets/%s", apiPrefix, s.extensionID, key)
	_, err := s.client.Delete(ctx, path, nil)
	return err
}

func (s *secretManager) List(ctx context.Context) ([]string, error) {
	if s.extensionID == "" {
		return nil, errors.New("extension ID required for secret operations")
	}

	path := fmt.Sprintf("%s/extensions/%s/secrets", apiPrefix, s.extensionID)
	resp, err := s.client.Get(ctx, path, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Keys []string `json:"keys"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Keys, nil
}

func (s *secretManager) Rotate(ctx context.Context, key string, newValue string) error {
	// Delete old value, then set new one
	if err := s.Delete(ctx, key); err != nil {
		return err
	}
	return s.Set(ctx, key, newValue)
}
