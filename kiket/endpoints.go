package kiket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Endpoints provides high-level extension API endpoints.
type Endpoints struct {
	Secrets SecretManager

	client       Client
	extensionID  string
	eventVersion string
}

// NewEndpoints creates a new endpoints instance.
func NewEndpoints(client Client, extensionID, eventVersion string) *Endpoints {
	return &Endpoints{
		Secrets:      NewSecretManager(client, extensionID),
		client:       client,
		extensionID:  extensionID,
		eventVersion: eventVersion,
	}
}

// LogEvent logs an event for the extension.
func (e *Endpoints) LogEvent(ctx context.Context, event string, data map[string]interface{}) error {
	if e.extensionID == "" {
		return errors.New("extension ID required for logging events")
	}

	path := fmt.Sprintf("%s/extensions/%s/events", apiPrefix, e.extensionID)
	_, err := e.client.Post(ctx, path, map[string]interface{}{
		"event":     event,
		"version":   e.eventVersion,
		"data":      data,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}, nil)

	return err
}

// GetMetadata retrieves extension metadata.
func (e *Endpoints) GetMetadata(ctx context.Context) (map[string]interface{}, error) {
	if e.extensionID == "" {
		return nil, errors.New("extension ID required for getting metadata")
	}

	path := fmt.Sprintf("%s/extensions/%s", apiPrefix, e.extensionID)
	resp, err := e.client.Get(ctx, path, nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// CustomData returns a custom data client for the given project.
func (e *Endpoints) CustomData(projectID interface{}) CustomDataClient {
	return NewCustomDataClient(e.client, projectID)
}

// SLAEvents returns an SLA events client for the given project.
func (e *Endpoints) SLAEvents(projectID interface{}) SLAEventsClient {
	return NewSLAEventsClient(e.client, projectID)
}

// RateLimit returns the current rate limit status.
func (e *Endpoints) RateLimit(ctx context.Context) (*RateLimitInfo, error) {
	path := fmt.Sprintf("%s/ext/rate_limit", apiPrefix)
	resp, err := e.client.Get(ctx, path, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		RateLimit struct {
			Limit         int `json:"limit"`
			Remaining     int `json:"remaining"`
			WindowSeconds int `json:"window_seconds"`
			ResetIn       int `json:"reset_in"`
		} `json:"rate_limit"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &RateLimitInfo{
		Limit:         result.RateLimit.Limit,
		Remaining:     result.RateLimit.Remaining,
		WindowSeconds: result.RateLimit.WindowSeconds,
		ResetIn:       result.RateLimit.ResetIn,
	}, nil
}
