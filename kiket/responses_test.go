package kiket

import (
	"testing"
)

// Allow response tests

func TestAllow_ReturnsProperlyFormattedResponse(t *testing.T) {
	response := Allow().Build()

	if response.Status != "allow" {
		t.Errorf("Expected status 'allow', got '%s'", response.Status)
	}
	if response.Message != "" {
		t.Errorf("Expected empty message, got '%s'", response.Message)
	}
	if len(response.Metadata) != 0 {
		t.Errorf("Expected empty metadata, got %v", response.Metadata)
	}
}

func TestAllow_IncludesMessageWhenProvided(t *testing.T) {
	response := Allow().Message("Success").Build()

	if response.Message != "Success" {
		t.Errorf("Expected message 'Success', got '%s'", response.Message)
	}
}

func TestAllow_IncludesDataInMetadata(t *testing.T) {
	response := Allow().
		Data("routeId", 123).
		Data("email", "test@example.com").
		Build()

	if response.Metadata["routeId"] != 123 {
		t.Errorf("Expected routeId 123, got %v", response.Metadata["routeId"])
	}
	if response.Metadata["email"] != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got %v", response.Metadata["email"])
	}
}

func TestAllow_IncludesDataMapInMetadata(t *testing.T) {
	data := map[string]interface{}{
		"routeId": 456,
		"active":  true,
	}

	response := Allow().DataMap(data).Build()

	if response.Metadata["routeId"] != 456 {
		t.Errorf("Expected routeId 456, got %v", response.Metadata["routeId"])
	}
	if response.Metadata["active"] != true {
		t.Errorf("Expected active true, got %v", response.Metadata["active"])
	}
}

func TestAllow_IncludesOutputFieldsInMetadata(t *testing.T) {
	response := Allow().
		OutputField("inbound_email", "abc@parse.example.com").
		Build()

	outputFields, ok := response.Metadata["output_fields"].(map[string]string)
	if !ok {
		t.Fatal("Expected output_fields to be map[string]string")
	}
	if outputFields["inbound_email"] != "abc@parse.example.com" {
		t.Errorf("Expected inbound_email 'abc@parse.example.com', got '%s'", outputFields["inbound_email"])
	}
}

func TestAllow_IncludesOutputFieldsMapInMetadata(t *testing.T) {
	fields := map[string]string{
		"webhook_url": "https://example.com/hook",
		"api_key":     "sk-xxx",
	}

	response := Allow().OutputFields(fields).Build()

	outputFields, ok := response.Metadata["output_fields"].(map[string]string)
	if !ok {
		t.Fatal("Expected output_fields to be map[string]string")
	}
	if outputFields["webhook_url"] != "https://example.com/hook" {
		t.Errorf("Expected webhook_url, got '%s'", outputFields["webhook_url"])
	}
	if outputFields["api_key"] != "sk-xxx" {
		t.Errorf("Expected api_key 'sk-xxx', got '%s'", outputFields["api_key"])
	}
}

func TestAllow_CombinesDataAndOutputFieldsInMetadata(t *testing.T) {
	response := Allow().
		Message("Configured successfully").
		Data("routeId", 456).
		OutputField("webhook_url", "https://example.com/hook").
		Build()

	if response.Status != "allow" {
		t.Errorf("Expected status 'allow', got '%s'", response.Status)
	}
	if response.Message != "Configured successfully" {
		t.Errorf("Expected message 'Configured successfully', got '%s'", response.Message)
	}
	if response.Metadata["routeId"] != 456 {
		t.Errorf("Expected routeId 456, got %v", response.Metadata["routeId"])
	}

	outputFields, ok := response.Metadata["output_fields"].(map[string]string)
	if !ok {
		t.Fatal("Expected output_fields to be map[string]string")
	}
	if outputFields["webhook_url"] != "https://example.com/hook" {
		t.Errorf("Expected webhook_url, got '%s'", outputFields["webhook_url"])
	}
}

// Deny response tests

func TestDeny_ReturnsProperlyFormattedResponse(t *testing.T) {
	response := Deny("Access denied").Build()

	if response.Status != "deny" {
		t.Errorf("Expected status 'deny', got '%s'", response.Status)
	}
	if response.Message != "Access denied" {
		t.Errorf("Expected message 'Access denied', got '%s'", response.Message)
	}
	if len(response.Metadata) != 0 {
		t.Errorf("Expected empty metadata, got %v", response.Metadata)
	}
}

func TestDeny_IncludesDataInMetadata(t *testing.T) {
	response := Deny("Invalid credentials").
		Data("errorCode", "AUTH_FAILED").
		Build()

	if response.Metadata["errorCode"] != "AUTH_FAILED" {
		t.Errorf("Expected errorCode 'AUTH_FAILED', got %v", response.Metadata["errorCode"])
	}
}

func TestDeny_IncludesDataMapInMetadata(t *testing.T) {
	data := map[string]interface{}{
		"errorCode": "AUTH_FAILED",
		"attempts":  3,
	}

	response := Deny("Invalid credentials").DataMap(data).Build()

	if response.Metadata["errorCode"] != "AUTH_FAILED" {
		t.Errorf("Expected errorCode 'AUTH_FAILED', got %v", response.Metadata["errorCode"])
	}
	if response.Metadata["attempts"] != 3 {
		t.Errorf("Expected attempts 3, got %v", response.Metadata["attempts"])
	}
}

// Pending response tests

func TestPending_ReturnsProperlyFormattedResponse(t *testing.T) {
	response := Pending("Awaiting approval").Build()

	if response.Status != "pending" {
		t.Errorf("Expected status 'pending', got '%s'", response.Status)
	}
	if response.Message != "Awaiting approval" {
		t.Errorf("Expected message 'Awaiting approval', got '%s'", response.Message)
	}
	if len(response.Metadata) != 0 {
		t.Errorf("Expected empty metadata, got %v", response.Metadata)
	}
}

func TestPending_IncludesDataInMetadata(t *testing.T) {
	response := Pending("Processing").
		Data("jobId", "abc123").
		Build()

	if response.Metadata["jobId"] != "abc123" {
		t.Errorf("Expected jobId 'abc123', got %v", response.Metadata["jobId"])
	}
}

// Builder isolation tests

func TestAllow_BuilderDataDoesNotMutateOriginal(t *testing.T) {
	builder := Allow()
	builder.Data("key1", "value1")
	response1 := builder.Build()

	builder.Data("key2", "value2")
	response2 := builder.Build()

	// response1 should have both keys because builder was reused
	// This tests that metadata is copied, not shared
	if response1.Metadata["key1"] != "value1" {
		t.Errorf("Expected key1 in response1")
	}
	if response2.Metadata["key2"] != "value2" {
		t.Errorf("Expected key2 in response2")
	}
}

func TestAllow_EmptyOutputFieldsNotIncluded(t *testing.T) {
	response := Allow().Data("key", "value").Build()

	if _, exists := response.Metadata["output_fields"]; exists {
		t.Error("Expected output_fields to not be present when empty")
	}
}

func TestAllow_NilDataMapIsHandled(t *testing.T) {
	response := Allow().DataMap(nil).Build()

	if len(response.Metadata) != 0 {
		t.Errorf("Expected empty metadata, got %v", response.Metadata)
	}
}

func TestAllow_NilOutputFieldsMapIsHandled(t *testing.T) {
	response := Allow().OutputFields(nil).Build()

	if _, exists := response.Metadata["output_fields"]; exists {
		t.Error("Expected output_fields to not be present when nil map provided")
	}
}
