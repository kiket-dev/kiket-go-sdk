// Package kiket provides the official Go SDK for building Kiket extensions.
package kiket

import (
	"context"
	"time"
)

// WebhookPayload represents a generic webhook payload.
type WebhookPayload map[string]interface{}

// Headers represents HTTP headers.
type Headers map[string]string

// Settings represents extension settings configuration.
type Settings map[string]interface{}

// WebhookHandler is the function signature for webhook handlers.
type WebhookHandler func(ctx context.Context, payload WebhookPayload, handlerCtx *HandlerContext) (interface{}, error)

// HandlerContext provides context to webhook handlers.
type HandlerContext struct {
	// Event name (e.g., "issue.created")
	Event string
	// Event version (e.g., "v1", "v2")
	EventVersion string
	// Request headers
	Headers Headers
	// Kiket API client
	Client Client
	// High-level extension endpoints
	Endpoints *Endpoints
	// Extension settings
	Settings Settings
	// Extension identifier
	ExtensionID string
	// Extension version
	ExtensionVersion string
	// Secret manager
	Secrets SecretManager
}

// Config holds SDK configuration options.
type Config struct {
	// Webhook HMAC secret for signature verification
	WebhookSecret string
	// Workspace token for API authentication
	WorkspaceToken string
	// Extension API key for /api/v1/ext endpoints
	ExtensionAPIKey string
	// Kiket API base URL
	BaseURL string
	// Extension settings
	Settings Settings
	// Extension identifier
	ExtensionID string
	// Extension version
	ExtensionVersion string
	// Path to manifest file (extension.yaml or manifest.yaml)
	ManifestPath string
	// Auto-load secrets from KIKET_SECRET_* environment variables
	AutoEnvSecrets bool
	// Enable telemetry reporting
	TelemetryEnabled bool
	// Telemetry reporting URL
	TelemetryURL string
}

// Manifest represents the extension manifest structure.
type Manifest struct {
	// Extension identifier
	ID string `yaml:"id"`
	// Extension version
	Version string `yaml:"version"`
	// Webhook delivery secret
	DeliverySecret string `yaml:"delivery_secret,omitempty"`
	// Settings with defaults
	Settings []ManifestSetting `yaml:"settings,omitempty"`
}

// ManifestSetting represents a setting definition in the manifest.
type ManifestSetting struct {
	Key     string      `yaml:"key"`
	Default interface{} `yaml:"default,omitempty"`
	Secret  bool        `yaml:"secret,omitempty"`
}

// TelemetryRecord represents a telemetry entry.
type TelemetryRecord struct {
	Event            string                 `json:"event"`
	Version          string                 `json:"version"`
	Status           string                 `json:"status"` // "ok" or "error"
	DurationMs       int64                  `json:"duration_ms"`
	ErrorMessage     string                 `json:"error_message,omitempty"`
	ErrorClass       string                 `json:"error_class,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	ExtensionID      string                 `json:"extension_id,omitempty"`
	ExtensionVersion string                 `json:"extension_version,omitempty"`
	Timestamp        time.Time              `json:"timestamp"`
}

// Client defines the HTTP client interface for API requests.
type Client interface {
	Get(ctx context.Context, path string, opts *RequestOptions) ([]byte, error)
	Post(ctx context.Context, path string, data interface{}, opts *RequestOptions) ([]byte, error)
	Put(ctx context.Context, path string, data interface{}, opts *RequestOptions) ([]byte, error)
	Patch(ctx context.Context, path string, data interface{}, opts *RequestOptions) ([]byte, error)
	Delete(ctx context.Context, path string, opts *RequestOptions) ([]byte, error)
	Close() error
}

// RequestOptions holds options for HTTP requests.
type RequestOptions struct {
	Headers Headers
	Timeout time.Duration
	Params  map[string]string
}

// SecretManager provides methods for managing extension secrets.
type SecretManager interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string) error
	Delete(ctx context.Context, key string) error
	List(ctx context.Context) ([]string, error)
	Rotate(ctx context.Context, key string, newValue string) error
}

// CustomDataClient provides access to custom data operations.
type CustomDataClient interface {
	List(ctx context.Context, moduleKey, table string, opts *CustomDataListOptions) (*CustomDataListResponse, error)
	Get(ctx context.Context, moduleKey, table string, recordID interface{}) (*CustomDataRecordResponse, error)
	Create(ctx context.Context, moduleKey, table string, record map[string]interface{}) (*CustomDataRecordResponse, error)
	Update(ctx context.Context, moduleKey, table string, recordID interface{}, record map[string]interface{}) (*CustomDataRecordResponse, error)
	Delete(ctx context.Context, moduleKey, table string, recordID interface{}) error
}

// SLAEventsClient provides access to SLA event operations.
type SLAEventsClient interface {
	List(ctx context.Context, opts *SLAEventsListOptions) (*SLAEventsListResponse, error)
}

// CustomDataListOptions holds options for listing custom data records.
type CustomDataListOptions struct {
	Limit   int
	Filters map[string]interface{}
}

// CustomDataListResponse represents the response from listing custom data.
type CustomDataListResponse struct {
	Data []map[string]interface{} `json:"data"`
}

// CustomDataRecordResponse represents a single custom data record response.
type CustomDataRecordResponse struct {
	Data map[string]interface{} `json:"data"`
}

// SLAEventsListOptions holds options for listing SLA events.
type SLAEventsListOptions struct {
	IssueID interface{}
	State   string // "imminent", "breached", "recovered"
	Limit   int
}

// SLAEventRecord represents an SLA event.
type SLAEventRecord struct {
	ID          interface{}            `json:"id"`
	IssueID     interface{}            `json:"issue_id"`
	ProjectID   interface{}            `json:"project_id"`
	State       string                 `json:"state"`
	TriggeredAt string                 `json:"triggered_at"`
	ResolvedAt  *string                `json:"resolved_at,omitempty"`
	Definition  map[string]interface{} `json:"definition,omitempty"`
	Metrics     map[string]interface{} `json:"metrics,omitempty"`
}

// SLAEventsListResponse represents the response from listing SLA events.
type SLAEventsListResponse struct {
	Data []SLAEventRecord `json:"data"`
}

// RateLimitInfo contains rate limit metadata.
type RateLimitInfo struct {
	Limit         int `json:"limit"`
	Remaining     int `json:"remaining"`
	WindowSeconds int `json:"window_seconds"`
	ResetIn       int `json:"reset_in"`
}

// HandlerMetadata holds information about a registered handler.
type HandlerMetadata struct {
	Event   string
	Version string
	Handler WebhookHandler
}
