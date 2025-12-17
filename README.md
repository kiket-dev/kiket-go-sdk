# Kiket Go SDK

Official Go SDK for building Kiket extensions.

## Installation

```bash
go get github.com/kiket-dev/kiket/sdk/go
```

## Quick Start

```go
package main

import (
    "context"
    "log"
    "net/http"

    "github.com/kiket-dev/kiket/sdk/go/kiket"
)

func main() {
    // Create SDK instance
    sdk, err := kiket.New(kiket.Config{
        WebhookSecret:   "your-webhook-secret",
        ExtensionAPIKey: "your-api-key",
        ExtensionID:     "com.example.my-extension",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer sdk.Close()

    // Register webhook handlers
    sdk.On("issue.created", func(ctx context.Context, payload kiket.WebhookPayload, hctx *kiket.HandlerContext) (interface{}, error) {
        issue := payload["issue"].(map[string]interface{})
        log.Printf("New issue created: %s", issue["title"])

        // Access custom data
        customData := hctx.Endpoints.CustomData(payload["project_id"])
        records, err := customData.List(ctx, "my-module", "my-table", nil)
        if err != nil {
            return nil, err
        }
        log.Printf("Found %d records", len(records.Data))

        return map[string]string{"status": "processed"}, nil
    })

    // Start HTTP server
    log.Println("Starting server on :8080")
    http.Handle("/webhook", sdk)
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## Configuration

The SDK can be configured programmatically or via manifest file:

```go
sdk, err := kiket.New(kiket.Config{
    // Authentication
    WebhookSecret:   "hmac-secret",      // For webhook signature verification
    ExtensionAPIKey: "ext-api-key",      // For extension API calls
    WorkspaceToken:  "workspace-token",  // Alternative: workspace auth

    // Extension info
    ExtensionID:      "com.example.my-ext",
    ExtensionVersion: "1.0.0",

    // Optional
    BaseURL:          "https://kiket.dev",
    Settings:         kiket.Settings{"key": "value"},
    ManifestPath:     "extension.yaml",
    AutoEnvSecrets:   true,
    TelemetryEnabled: true,
})
```

### Manifest File

Create `extension.yaml` in your project root:

```yaml
id: com.example.my-extension
version: 1.0.0
delivery_secret: your-webhook-secret

settings:
  - key: api_token
    secret: true
  - key: default_priority
    default: medium
```

## Webhook Handlers

Register handlers for Kiket events:

```go
// Issue events
sdk.On("issue.created", handleIssueCreated)
sdk.On("issue.updated", handleIssueUpdated)
sdk.On("issue.status_changed", handleStatusChange)
sdk.On("issue.assigned", handleAssignment)
sdk.On("issue.closed", handleClosed)

// Workflow events
sdk.On("workflow.triggered", handleWorkflowTrigger)
sdk.On("workflow.sla_status", handleSLAStatus)
sdk.On("workflow.before_transition", handleBeforeTransition)

// Comment events
sdk.On("comment.created", handleCommentCreated)
```

## Extension Endpoints

### Secrets

```go
// Get a secret
value, err := hctx.Secrets.Get(ctx, "api_token")

// Set a secret
err := hctx.Secrets.Set(ctx, "api_token", "new-value")

// List all secret keys
keys, err := hctx.Secrets.List(ctx)

// Delete a secret
err := hctx.Secrets.Delete(ctx, "api_token")

// Rotate a secret
err := hctx.Secrets.Rotate(ctx, "api_token", "rotated-value")
```

### Custom Data

```go
customData := hctx.Endpoints.CustomData(projectID)

// List records
records, err := customData.List(ctx, "module-key", "table-name", &kiket.CustomDataListOptions{
    Limit: 100,
    Filters: map[string]interface{}{
        "status": "active",
    },
})

// Get a record
record, err := customData.Get(ctx, "module-key", "table-name", recordID)

// Create a record
record, err := customData.Create(ctx, "module-key", "table-name", map[string]interface{}{
    "name": "Test",
    "value": 42,
})

// Update a record
record, err := customData.Update(ctx, "module-key", "table-name", recordID, map[string]interface{}{
    "value": 100,
})

// Delete a record
err := customData.Delete(ctx, "module-key", "table-name", recordID)
```

### SLA Events

```go
slaEvents := hctx.Endpoints.SLAEvents(projectID)

// List SLA events
events, err := slaEvents.List(ctx, &kiket.SLAEventsListOptions{
    State: "breached",
    Limit: 50,
})
```

### Rate Limiting

```go
info, err := hctx.Endpoints.RateLimit(ctx)
log.Printf("Remaining: %d/%d (resets in %ds)",
    info.Remaining, info.Limit, info.ResetIn)
```

## Signature Verification

The SDK automatically verifies webhook signatures. For manual verification:

```go
err := kiket.VerifySignature(secret, body, headers)
if err != nil {
    if kiket.IsAuthenticationError(err) {
        // Invalid signature
    }
}
```

## HTTP Server Integration

The SDK implements `http.Handler`:

```go
// Standard library
http.Handle("/webhook", sdk)

// Gin
router.POST("/webhook", gin.WrapH(sdk))

// Echo
e.POST("/webhook", echo.WrapHandler(sdk))

// Chi
r.Post("/webhook", sdk.ServeHTTP)
```

## Testing

Generate test signatures:

```go
signature, timestamp := kiket.GenerateSignature(secret, body, nil)

headers := kiket.Headers{
    "X-Kiket-Signature":  signature,
    "X-Kiket-Timestamp":  timestamp,
    "X-Kiket-Event-Version": "v1",
}
```

## Environment Variables

- `KIKET_SDK_TELEMETRY_OPTOUT=1` - Disable telemetry
- `KIKET_SECRET_*` - Override secret values (when `AutoEnvSecrets: true`)

## License

MIT License - see [LICENSE](LICENSE) for details.
