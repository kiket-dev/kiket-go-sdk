package kiket

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	defaultTimeout = 30 * time.Second
	defaultBaseURL = "https://kiket.dev"
)

// HTTPClient implements the Client interface using net/http.
type HTTPClient struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
	token      string
}

// ClientOption configures the HTTP client.
type ClientOption func(*HTTPClient)

// WithBaseURL sets the base URL for the client.
func WithBaseURL(url string) ClientOption {
	return func(c *HTTPClient) {
		c.baseURL = url
	}
}

// WithAPIKey sets the extension API key.
func WithAPIKey(key string) ClientOption {
	return func(c *HTTPClient) {
		c.apiKey = key
	}
}

// WithToken sets the bearer token.
func WithToken(token string) ClientOption {
	return func(c *HTTPClient) {
		c.token = token
	}
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *HTTPClient) {
		c.httpClient.Timeout = timeout
	}
}

// NewHTTPClient creates a new HTTP client.
func NewHTTPClient(opts ...ClientOption) *HTTPClient {
	c := &HTTPClient{
		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

func (c *HTTPClient) doRequest(ctx context.Context, method, path string, body interface{}, opts *RequestOptions) ([]byte, error) {
	fullURL := c.baseURL + path

	if opts != nil && len(opts.Params) > 0 {
		params := url.Values{}
		for k, v := range opts.Params {
			params.Set(k, v)
		}
		fullURL += "?" + params.Encode()
	}

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Set authentication
	if c.apiKey != "" {
		req.Header.Set("X-Kiket-API-Key", c.apiKey)
	} else if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	// Apply custom headers
	if opts != nil && opts.Headers != nil {
		for k, v := range opts.Headers {
			req.Header.Set(k, v)
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Body:       string(respBody),
		}
	}

	return respBody, nil
}

// Get performs a GET request.
func (c *HTTPClient) Get(ctx context.Context, path string, opts *RequestOptions) ([]byte, error) {
	return c.doRequest(ctx, http.MethodGet, path, nil, opts)
}

// Post performs a POST request.
func (c *HTTPClient) Post(ctx context.Context, path string, data interface{}, opts *RequestOptions) ([]byte, error) {
	return c.doRequest(ctx, http.MethodPost, path, data, opts)
}

// Put performs a PUT request.
func (c *HTTPClient) Put(ctx context.Context, path string, data interface{}, opts *RequestOptions) ([]byte, error) {
	return c.doRequest(ctx, http.MethodPut, path, data, opts)
}

// Patch performs a PATCH request.
func (c *HTTPClient) Patch(ctx context.Context, path string, data interface{}, opts *RequestOptions) ([]byte, error) {
	return c.doRequest(ctx, http.MethodPatch, path, data, opts)
}

// Delete performs a DELETE request.
func (c *HTTPClient) Delete(ctx context.Context, path string, opts *RequestOptions) ([]byte, error) {
	return c.doRequest(ctx, http.MethodDelete, path, nil, opts)
}

// Close closes the HTTP client.
func (c *HTTPClient) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}

// APIError represents an API error response.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error (status %d): %s", e.StatusCode, e.Body)
}
