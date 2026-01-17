package kiket

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"
)

// TelemetryReporter handles telemetry reporting.
type TelemetryReporter struct {
	endpoint         string
	enabled          bool
	extensionID      string
	extensionVersion string
	httpClient       *http.Client
}

// TelemetryOption configures the telemetry reporter.
type TelemetryOption func(*TelemetryReporter)

// WithTelemetryEndpoint sets the telemetry endpoint.
func WithTelemetryEndpoint(url string) TelemetryOption {
	return func(r *TelemetryReporter) {
		if url != "" {
			url = strings.TrimSuffix(url, "/")
			if !strings.HasSuffix(url, "/telemetry") {
				url += "/telemetry"
			}
			r.endpoint = url
		}
	}
}

// WithTelemetryExtension sets the extension metadata.
func WithTelemetryExtension(id, version string) TelemetryOption {
	return func(r *TelemetryReporter) {
		r.extensionID = id
		r.extensionVersion = version
	}
}

// NewTelemetryReporter creates a new telemetry reporter.
func NewTelemetryReporter(enabled bool, opts ...TelemetryOption) *TelemetryReporter {
	// Check opt-out environment variable
	optOut := os.Getenv("KIKET_SDK_TELEMETRY_OPTOUT")
	if strings.ToLower(optOut) == "1" {
		enabled = false
	}

	r := &TelemetryReporter{
		enabled: enabled,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// Record records a telemetry event.
func (r *TelemetryReporter) Record(ctx context.Context, event, version, status string, durationMs int64, extras map[string]interface{}) error {
	if !r.enabled {
		return nil
	}

	record := TelemetryRecord{
		Event:            event,
		Version:          version,
		Status:           status,
		DurationMs:       durationMs,
		ExtensionID:      r.extensionID,
		ExtensionVersion: r.extensionVersion,
		Timestamp:        time.Now().UTC(),
	}

	if extras != nil {
		if msg, ok := extras["errorMessage"].(string); ok {
			record.ErrorMessage = msg
		}
		if cls, ok := extras["errorClass"].(string); ok {
			record.ErrorClass = cls
		}
		if meta, ok := extras["metadata"].(map[string]interface{}); ok {
			record.Metadata = meta
		}
	}

	if r.endpoint == "" {
		return nil
	}

	payload := map[string]interface{}{
		"event":             record.Event,
		"version":           record.Version,
		"status":            record.Status,
		"duration_ms":       record.DurationMs,
		"timestamp":         record.Timestamp.Format(time.RFC3339),
		"extension_id":      record.ExtensionID,
		"extension_version": record.ExtensionVersion,
		"error_message":     record.ErrorMessage,
		"error_class":       record.ErrorClass,
		"metadata":          record.Metadata,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		// Best effort - don't fail the handler
		return nil
	}
	defer resp.Body.Close()

	return nil
}
