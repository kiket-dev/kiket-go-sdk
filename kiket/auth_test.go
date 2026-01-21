package kiket

import (
	"testing"
	"time"
)

func TestIsAuthenticationError(t *testing.T) {
	authErr := &AuthenticationError{Message: "test error"}
	if !IsAuthenticationError(authErr) {
		t.Error("Expected true for AuthenticationError")
	}

	regularErr := error(nil)
	if IsAuthenticationError(regularErr) {
		t.Error("Expected false for nil error")
	}
}

func TestAuthenticationError_Error(t *testing.T) {
	err := &AuthenticationError{Message: "test message"}
	if err.Error() != "test message" {
		t.Errorf("Expected 'test message', got '%s'", err.Error())
	}
}

func TestBuildAuthContext(t *testing.T) {
	orgID := 123
	extID := 456
	projID := 789
	exp := time.Now().Add(time.Hour).Unix()

	jwtPayload := &JwtPayload{
		Sub:    "test-subject",
		OrgID:  &orgID,
		ExtID:  &extID,
		ProjID: &projID,
		Scopes: []string{"read", "write"},
		Exp:    exp,
	}

	rawPayload := map[string]interface{}{
		"authentication": map[string]interface{}{
			"runtime_token": "test-token",
		},
	}

	authCtx := BuildAuthContext(jwtPayload, rawPayload)

	if authCtx.RuntimeToken != "test-token" {
		t.Errorf("Expected 'test-token', got '%s'", authCtx.RuntimeToken)
	}
	if authCtx.TokenType != "runtime" {
		t.Errorf("Expected 'runtime', got '%s'", authCtx.TokenType)
	}
	if *authCtx.OrgID != 123 {
		t.Errorf("Expected OrgID 123, got %d", *authCtx.OrgID)
	}
	if *authCtx.ExtID != 456 {
		t.Errorf("Expected ExtID 456, got %d", *authCtx.ExtID)
	}
	if *authCtx.ProjID != 789 {
		t.Errorf("Expected ProjID 789, got %d", *authCtx.ProjID)
	}
	if len(authCtx.Scopes) != 2 || authCtx.Scopes[0] != "read" {
		t.Errorf("Expected scopes ['read', 'write'], got %v", authCtx.Scopes)
	}
	if authCtx.ExpiresAt == nil {
		t.Error("Expected ExpiresAt to be set")
	}
}

func TestBuildAuthContext_NilScopes(t *testing.T) {
	jwtPayload := &JwtPayload{
		Sub:    "test-subject",
		Scopes: nil,
	}

	rawPayload := map[string]interface{}{}

	authCtx := BuildAuthContext(jwtPayload, rawPayload)

	if authCtx.Scopes == nil {
		t.Error("Expected Scopes to be empty slice, not nil")
	}
	if len(authCtx.Scopes) != 0 {
		t.Errorf("Expected empty scopes, got %v", authCtx.Scopes)
	}
}

func TestBuildAuthContext_NoExpiration(t *testing.T) {
	jwtPayload := &JwtPayload{
		Sub: "test-subject",
		Exp: 0,
	}

	rawPayload := map[string]interface{}{}

	authCtx := BuildAuthContext(jwtPayload, rawPayload)

	if authCtx.ExpiresAt != nil {
		t.Error("Expected ExpiresAt to be nil when Exp is 0")
	}
}

func TestClearJWKSCache(t *testing.T) {
	// Add something to cache
	jwksCacheMu.Lock()
	jwksCache["test-url"] = &jwksCacheEntry{
		keys:      &JWKS{},
		fetchedAt: time.Now(),
	}
	jwksCacheMu.Unlock()

	ClearJWKSCache()

	jwksCacheMu.RLock()
	if len(jwksCache) != 0 {
		t.Error("Expected cache to be empty after clear")
	}
	jwksCacheMu.RUnlock()
}

func TestParseJwtPayload(t *testing.T) {
	claims := map[string]interface{}{
		"sub":     "test-sub",
		"org_id":  float64(123),
		"ext_id":  float64(456),
		"proj_id": float64(789),
		"pi_id":   float64(111),
		"scopes":  []interface{}{"read", "write"},
		"src":     "webhook",
		"iss":     "kiket.dev",
		"iat":     float64(1234567890),
		"exp":     float64(1234571490),
		"jti":     "unique-id",
	}

	payload := parseJwtPayload(claims)

	if payload.Sub != "test-sub" {
		t.Errorf("Expected 'test-sub', got '%s'", payload.Sub)
	}
	if *payload.OrgID != 123 {
		t.Errorf("Expected OrgID 123, got %d", *payload.OrgID)
	}
	if *payload.ExtID != 456 {
		t.Errorf("Expected ExtID 456, got %d", *payload.ExtID)
	}
	if *payload.ProjID != 789 {
		t.Errorf("Expected ProjID 789, got %d", *payload.ProjID)
	}
	if *payload.PiID != 111 {
		t.Errorf("Expected PiID 111, got %d", *payload.PiID)
	}
	if len(payload.Scopes) != 2 {
		t.Errorf("Expected 2 scopes, got %d", len(payload.Scopes))
	}
	if payload.Src != "webhook" {
		t.Errorf("Expected 'webhook', got '%s'", payload.Src)
	}
	if payload.Iss != "kiket.dev" {
		t.Errorf("Expected 'kiket.dev', got '%s'", payload.Iss)
	}
	if payload.Iat != 1234567890 {
		t.Errorf("Expected iat 1234567890, got %d", payload.Iat)
	}
	if payload.Exp != 1234571490 {
		t.Errorf("Expected exp 1234571490, got %d", payload.Exp)
	}
	if payload.Jti != "unique-id" {
		t.Errorf("Expected 'unique-id', got '%s'", payload.Jti)
	}
}

func TestParseJwtPayload_EmptyClaims(t *testing.T) {
	claims := map[string]interface{}{}

	payload := parseJwtPayload(claims)

	if payload.Sub != "" {
		t.Errorf("Expected empty sub, got '%s'", payload.Sub)
	}
	if payload.OrgID != nil {
		t.Error("Expected nil OrgID")
	}
	if payload.Scopes != nil {
		t.Errorf("Expected nil scopes, got %v", payload.Scopes)
	}
}
