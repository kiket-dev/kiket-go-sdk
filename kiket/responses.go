package kiket

// ExtensionResponse represents the standard response format for extension handlers.
//
// Use the helper functions to build properly formatted responses:
//   - Allow() - Build an allow response
//   - Deny(message) - Build a deny response
//   - Pending(message) - Build a pending response
//
// Example usage:
//
//	// Simple allow
//	return Allow().Build(), nil
//
//	// Allow with output fields
//	return Allow().
//		Message("Successfully configured Mailjet").
//		Data("routeId", 123).
//		OutputField("inbound_email", "abc@parse.example.com").
//		Build(), nil
//
//	// Deny with error details
//	return Deny("Invalid credentials").
//		Data("errorCode", "AUTH_FAILED").
//		Build(), nil
type ExtensionResponse struct {
	Status   string                 `json:"status"`
	Message  string                 `json:"message,omitempty"`
	Metadata map[string]interface{} `json:"metadata"`
}

// Allow starts building an allow response.
func Allow() *AllowBuilder {
	return &AllowBuilder{
		data:         make(map[string]interface{}),
		outputFields: make(map[string]string),
	}
}

// Deny starts building a deny response.
func Deny(message string) *ResponseBuilder {
	return &ResponseBuilder{
		status:  "deny",
		message: message,
		data:    make(map[string]interface{}),
	}
}

// Pending starts building a pending response.
func Pending(message string) *ResponseBuilder {
	return &ResponseBuilder{
		status:  "pending",
		message: message,
		data:    make(map[string]interface{}),
	}
}

// AllowBuilder is used to construct allow responses with output fields support.
type AllowBuilder struct {
	message      string
	data         map[string]interface{}
	outputFields map[string]string
}

// Message sets an optional success message.
func (b *AllowBuilder) Message(msg string) *AllowBuilder {
	b.message = msg
	return b
}

// Data adds a key-value pair to the response metadata.
func (b *AllowBuilder) Data(key string, value interface{}) *AllowBuilder {
	b.data[key] = value
	return b
}

// DataMap adds multiple entries to the response metadata.
func (b *AllowBuilder) DataMap(data map[string]interface{}) *AllowBuilder {
	for k, v := range data {
		b.data[k] = v
	}
	return b
}

// OutputField adds an output field to be displayed in the configuration UI.
// The key must match the manifest output_fields schema.
func (b *AllowBuilder) OutputField(key, value string) *AllowBuilder {
	b.outputFields[key] = value
	return b
}

// OutputFields adds multiple output fields to be displayed in the configuration UI.
func (b *AllowBuilder) OutputFields(fields map[string]string) *AllowBuilder {
	for k, v := range fields {
		b.outputFields[k] = v
	}
	return b
}

// Build constructs the final ExtensionResponse.
func (b *AllowBuilder) Build() *ExtensionResponse {
	metadata := make(map[string]interface{})
	for k, v := range b.data {
		metadata[k] = v
	}

	if len(b.outputFields) > 0 {
		fields := make(map[string]string)
		for k, v := range b.outputFields {
			fields[k] = v
		}
		metadata["output_fields"] = fields
	}

	return &ExtensionResponse{
		Status:   "allow",
		Message:  b.message,
		Metadata: metadata,
	}
}

// ResponseBuilder is used to construct deny and pending responses.
type ResponseBuilder struct {
	status  string
	message string
	data    map[string]interface{}
}

// Data adds a key-value pair to the response metadata.
func (b *ResponseBuilder) Data(key string, value interface{}) *ResponseBuilder {
	b.data[key] = value
	return b
}

// DataMap adds multiple entries to the response metadata.
func (b *ResponseBuilder) DataMap(data map[string]interface{}) *ResponseBuilder {
	for k, v := range data {
		b.data[k] = v
	}
	return b
}

// Build constructs the final ExtensionResponse.
func (b *ResponseBuilder) Build() *ExtensionResponse {
	metadata := make(map[string]interface{})
	for k, v := range b.data {
		metadata[k] = v
	}

	return &ExtensionResponse{
		Status:   b.status,
		Message:  b.message,
		Metadata: metadata,
	}
}
