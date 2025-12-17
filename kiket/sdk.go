package kiket

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// SDK is the main entry point for the Kiket Extension SDK.
type SDK struct {
	config     Config
	client     Client
	endpoints  *Endpoints
	handlers   map[string]*HandlerMetadata
	handlersMu sync.RWMutex
	telemetry  *TelemetryReporter
	manifest   *Manifest
}

// New creates a new SDK instance.
func New(config Config) (*SDK, error) {
	// Load manifest if not provided
	var manifest *Manifest
	if config.ManifestPath != "" || (config.ExtensionID == "" && config.WebhookSecret == "") {
		var err error
		manifest, err = LoadManifest(config.ManifestPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load manifest: %w", err)
		}
	}

	// Apply manifest defaults
	if manifest != nil {
		if config.ExtensionID == "" {
			config.ExtensionID = manifest.ID
		}
		if config.ExtensionVersion == "" {
			config.ExtensionVersion = manifest.Version
		}
		if config.WebhookSecret == "" {
			config.WebhookSecret = manifest.DeliverySecret
		}
		if config.Settings == nil {
			config.Settings = SettingsDefaults(manifest)
		}

		// Apply environment variable overrides for secrets
		if config.AutoEnvSecrets {
			secretKeys := SecretKeys(manifest)
			config.Settings = ApplySecretEnvOverrides(config.Settings, secretKeys)
		}
	}

	// Set default base URL
	if config.BaseURL == "" {
		config.BaseURL = defaultBaseURL
	}

	// Create HTTP client
	clientOpts := []ClientOption{
		WithBaseURL(config.BaseURL),
	}
	if config.ExtensionAPIKey != "" {
		clientOpts = append(clientOpts, WithAPIKey(config.ExtensionAPIKey))
	} else if config.WorkspaceToken != "" {
		clientOpts = append(clientOpts, WithToken(config.WorkspaceToken))
	}
	httpClient := NewHTTPClient(clientOpts...)

	// Create endpoints
	endpoints := NewEndpoints(httpClient, config.ExtensionID, config.ExtensionVersion)

	// Create telemetry reporter
	telemetryOpts := []TelemetryOption{
		WithTelemetryExtension(config.ExtensionID, config.ExtensionVersion),
	}
	if config.TelemetryURL != "" {
		telemetryOpts = append(telemetryOpts, WithTelemetryEndpoint(config.TelemetryURL))
	}
	if config.ExtensionAPIKey != "" {
		telemetryOpts = append(telemetryOpts, WithTelemetryAPIKey(config.ExtensionAPIKey))
	}
	telemetry := NewTelemetryReporter(config.TelemetryEnabled, telemetryOpts...)

	return &SDK{
		config:    config,
		client:    httpClient,
		endpoints: endpoints,
		handlers:  make(map[string]*HandlerMetadata),
		telemetry: telemetry,
		manifest:  manifest,
	}, nil
}

// On registers a webhook handler for an event.
func (s *SDK) On(event string, handler WebhookHandler, versions ...string) {
	version := "v1"
	if len(versions) > 0 {
		version = versions[0]
	}

	key := event + ":" + version

	s.handlersMu.Lock()
	s.handlers[key] = &HandlerMetadata{
		Event:   event,
		Version: version,
		Handler: handler,
	}
	s.handlersMu.Unlock()
}

// GetHandler returns the handler for an event and version.
func (s *SDK) GetHandler(event, version string) *HandlerMetadata {
	key := event + ":" + version

	s.handlersMu.RLock()
	defer s.handlersMu.RUnlock()

	return s.handlers[key]
}

// EventNames returns all registered event names.
func (s *SDK) EventNames() []string {
	s.handlersMu.RLock()
	defer s.handlersMu.RUnlock()

	names := make([]string, 0, len(s.handlers))
	seen := make(map[string]bool)

	for _, h := range s.handlers {
		if !seen[h.Event] {
			names = append(names, h.Event)
			seen[h.Event] = true
		}
	}

	return names
}

// HandleWebhook processes an incoming webhook request.
func (s *SDK) HandleWebhook(ctx context.Context, body []byte, headers Headers) (interface{}, error) {
	// Verify signature
	if err := VerifySignature(s.config.WebhookSecret, body, headers); err != nil {
		return nil, err
	}

	// Parse payload
	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	// Extract event info
	event, _ := payload["event"].(string)
	version := headers["X-Kiket-Event-Version"]
	if version == "" {
		version = headers["x-kiket-event-version"]
	}
	if version == "" {
		version = "v1"
	}

	// Get handler
	handler := s.GetHandler(event, version)
	if handler == nil {
		return nil, fmt.Errorf("no handler registered for event %s (version %s)", event, version)
	}

	// Build handler context
	handlerCtx := &HandlerContext{
		Event:            event,
		EventVersion:     version,
		Headers:          headers,
		Client:           s.client,
		Endpoints:        s.endpoints,
		Settings:         s.config.Settings,
		ExtensionID:      s.config.ExtensionID,
		ExtensionVersion: s.config.ExtensionVersion,
		Secrets:          s.endpoints.Secrets,
	}

	// Execute handler with telemetry
	start := time.Now()
	result, err := handler.Handler(ctx, payload, handlerCtx)
	duration := time.Since(start).Milliseconds()

	// Record telemetry
	status := "ok"
	extras := make(map[string]interface{})
	if err != nil {
		status = "error"
		extras["errorMessage"] = err.Error()
		extras["errorClass"] = fmt.Sprintf("%T", err)
	}
	_ = s.telemetry.Record(ctx, event, version, status, duration, extras)

	return result, err
}

// ServeHTTP implements http.Handler for use with net/http.
func (s *SDK) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Convert headers
	headers := make(Headers)
	for k, v := range r.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	result, err := s.HandleWebhook(r.Context(), body, headers)
	if err != nil {
		if IsAuthenticationError(err) {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if result != nil {
		json.NewEncoder(w).Encode(result)
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	}
}

// Client returns the underlying HTTP client.
func (s *SDK) Client() Client {
	return s.client
}

// Endpoints returns the extension endpoints.
func (s *SDK) Endpoints() *Endpoints {
	return s.endpoints
}

// Config returns the SDK configuration.
func (s *SDK) Config() Config {
	return s.config
}

// Close closes the SDK and releases resources.
func (s *SDK) Close() error {
	return s.client.Close()
}
