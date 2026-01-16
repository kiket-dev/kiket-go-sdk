package kiket

import (
	"os"
	"testing"
)

func TestHandlerContext_Secret_PayloadTakesPriority(t *testing.T) {
	// Set up ENV fallback
	os.Setenv("TEST_SECRET", "env-value")
	defer os.Unsetenv("TEST_SECRET")

	ctx := &HandlerContext{
		payloadSecrets: map[string]string{
			"TEST_SECRET": "payload-value",
		},
	}

	result := ctx.Secret("TEST_SECRET")
	if result != "payload-value" {
		t.Errorf("Expected payload-value, got %s", result)
	}
}

func TestHandlerContext_Secret_FallsBackToEnv(t *testing.T) {
	os.Setenv("ENV_ONLY_SECRET", "from-env")
	defer os.Unsetenv("ENV_ONLY_SECRET")

	ctx := &HandlerContext{
		payloadSecrets: map[string]string{},
	}

	result := ctx.Secret("ENV_ONLY_SECRET")
	if result != "from-env" {
		t.Errorf("Expected from-env, got %s", result)
	}
}

func TestHandlerContext_Secret_ReturnsEmptyWhenNotFound(t *testing.T) {
	ctx := &HandlerContext{
		payloadSecrets: map[string]string{},
	}

	result := ctx.Secret("NONEXISTENT_SECRET")
	if result != "" {
		t.Errorf("Expected empty string, got %s", result)
	}
}

func TestHandlerContext_Secret_IgnoresEmptyPayloadValue(t *testing.T) {
	os.Setenv("TEST_SECRET", "env-value")
	defer os.Unsetenv("TEST_SECRET")

	ctx := &HandlerContext{
		payloadSecrets: map[string]string{
			"TEST_SECRET": "", // Empty payload value
		},
	}

	result := ctx.Secret("TEST_SECRET")
	if result != "env-value" {
		t.Errorf("Expected env-value (fallback), got %s", result)
	}
}

func TestHandlerContext_Secret_NilPayloadSecrets(t *testing.T) {
	os.Setenv("TEST_SECRET", "env-value")
	defer os.Unsetenv("TEST_SECRET")

	ctx := &HandlerContext{
		payloadSecrets: nil,
	}

	result := ctx.Secret("TEST_SECRET")
	if result != "env-value" {
		t.Errorf("Expected env-value, got %s", result)
	}
}

func TestExtractPayloadSecrets(t *testing.T) {
	payload := WebhookPayload{
		"event_type": "test",
		"secrets": map[string]interface{}{
			"API_KEY":    "secret-key",
			"API_SECRET": "secret-value",
		},
	}

	secrets := extractPayloadSecrets(payload)

	if secrets["API_KEY"] != "secret-key" {
		t.Errorf("Expected secret-key, got %s", secrets["API_KEY"])
	}
	if secrets["API_SECRET"] != "secret-value" {
		t.Errorf("Expected secret-value, got %s", secrets["API_SECRET"])
	}
}

func TestExtractPayloadSecrets_NoSecrets(t *testing.T) {
	payload := WebhookPayload{
		"event_type": "test",
	}

	secrets := extractPayloadSecrets(payload)

	if secrets != nil {
		t.Errorf("Expected nil, got %v", secrets)
	}
}

func TestExtractPayloadSecrets_InvalidSecretsType(t *testing.T) {
	payload := WebhookPayload{
		"event_type": "test",
		"secrets":    "not-a-map",
	}

	secrets := extractPayloadSecrets(payload)

	if secrets != nil {
		t.Errorf("Expected nil for invalid type, got %v", secrets)
	}
}
