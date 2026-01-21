package kiket

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	algorithm    = "ES256"
	issuer       = "kiket.dev"
	jwksCacheTTL = time.Hour
)

// AuthenticationError represents an authentication failure.
type AuthenticationError struct {
	Message string
}

func (e *AuthenticationError) Error() string {
	return e.Message
}

// JwtPayload contains the decoded JWT claims.
type JwtPayload struct {
	Sub    string   `json:"sub"`
	OrgID  *int     `json:"org_id,omitempty"`
	ExtID  *int     `json:"ext_id,omitempty"`
	ProjID *int     `json:"proj_id,omitempty"`
	PiID   *int     `json:"pi_id,omitempty"`
	Scopes []string `json:"scopes,omitempty"`
	Src    string   `json:"src,omitempty"`
	Iss    string   `json:"iss"`
	Iat    int64    `json:"iat"`
	Exp    int64    `json:"exp"`
	Jti    string   `json:"jti"`
}

// AuthContext contains authentication context from verified JWT.
type AuthContext struct {
	RuntimeToken string
	TokenType    string
	ExpiresAt    *time.Time
	Scopes       []string
	OrgID        *int
	ExtID        *int
	ProjID       *int
}

// JWKS cache
type jwksCacheEntry struct {
	keys      *JWKS
	fetchedAt time.Time
}

var (
	jwksCache   = make(map[string]*jwksCacheEntry)
	jwksCacheMu sync.RWMutex
)

// JWKS represents a JSON Web Key Set.
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a JSON Web Key.
type JWK struct {
	Kty string `json:"kty"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	Kid string `json:"kid"`
}

// VerifyRuntimeToken verifies the runtime token JWT from the webhook payload.
func VerifyRuntimeToken(ctx context.Context, payload map[string]interface{}, baseURL string) (*JwtPayload, error) {
	auth, ok := payload["authentication"].(map[string]interface{})
	if !ok {
		return nil, &AuthenticationError{Message: "missing runtime_token in payload"}
	}

	token, ok := auth["runtime_token"].(string)
	if !ok || token == "" {
		return nil, &AuthenticationError{Message: "missing runtime_token in payload"}
	}

	return DecodeJWT(ctx, token, baseURL)
}

// DecodeJWT decodes and verifies a JWT token using the public key from JWKS.
func DecodeJWT(ctx context.Context, tokenString string, baseURL string) (*JwtPayload, error) {
	jwks, err := FetchJWKS(ctx, baseURL)
	if err != nil {
		return nil, err
	}

	// Parse and validate token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if token.Method.Alg() != algorithm {
			return nil, &AuthenticationError{Message: fmt.Sprintf("unexpected signing method: %v", token.Method.Alg())}
		}

		// Get kid from token header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			kid = ""
		}

		// Find matching key
		key, err := findSigningKey(jwks, kid)
		if err != nil {
			return nil, err
		}

		return key, nil
	}, jwt.WithIssuer(issuer), jwt.WithExpirationRequired())

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, &AuthenticationError{Message: "runtime token has expired"}
		}
		if errors.Is(err, jwt.ErrTokenInvalidIssuer) {
			return nil, &AuthenticationError{Message: "invalid token issuer"}
		}
		var authErr *AuthenticationError
		if errors.As(err, &authErr) {
			return nil, authErr
		}
		return nil, &AuthenticationError{Message: fmt.Sprintf("invalid token: %v", err)}
	}

	if !token.Valid {
		return nil, &AuthenticationError{Message: "invalid token"}
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, &AuthenticationError{Message: "invalid token claims"}
	}

	return parseJwtPayload(claims), nil
}

// FetchJWKS fetches JWKS from the well-known endpoint with caching.
func FetchJWKS(ctx context.Context, baseURL string) (*JWKS, error) {
	jwksCacheMu.RLock()
	cached, ok := jwksCache[baseURL]
	if ok && time.Since(cached.fetchedAt) < jwksCacheTTL {
		jwksCacheMu.RUnlock()
		return cached.keys, nil
	}
	jwksCacheMu.RUnlock()

	jwksURL := baseURL + "/.well-known/jwks.json"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURL, nil)
	if err != nil {
		return nil, &AuthenticationError{Message: fmt.Sprintf("failed to create JWKS request: %v", err)}
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, &AuthenticationError{Message: fmt.Sprintf("failed to fetch JWKS: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &AuthenticationError{Message: fmt.Sprintf("failed to fetch JWKS: status %d", resp.StatusCode)}
	}

	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, &AuthenticationError{Message: "invalid JWKS response"}
	}

	jwksCacheMu.Lock()
	jwksCache[baseURL] = &jwksCacheEntry{
		keys:      &jwks,
		fetchedAt: time.Now(),
	}
	jwksCacheMu.Unlock()

	return &jwks, nil
}

// ClearJWKSCache clears the JWKS cache (useful for testing or key rotation).
func ClearJWKSCache() {
	jwksCacheMu.Lock()
	jwksCache = make(map[string]*jwksCacheEntry)
	jwksCacheMu.Unlock()
}

// BuildAuthContext builds authentication context from verified JWT payload.
func BuildAuthContext(jwtPayload *JwtPayload, rawPayload map[string]interface{}) *AuthContext {
	auth, _ := rawPayload["authentication"].(map[string]interface{})
	runtimeToken, _ := auth["runtime_token"].(string)

	var expiresAt *time.Time
	if jwtPayload.Exp > 0 {
		t := time.Unix(jwtPayload.Exp, 0)
		expiresAt = &t
	}

	scopes := jwtPayload.Scopes
	if scopes == nil {
		scopes = []string{}
	}

	return &AuthContext{
		RuntimeToken: runtimeToken,
		TokenType:    "runtime",
		ExpiresAt:    expiresAt,
		Scopes:       scopes,
		OrgID:        jwtPayload.OrgID,
		ExtID:        jwtPayload.ExtID,
		ProjID:       jwtPayload.ProjID,
	}
}

// IsAuthenticationError checks if an error is an AuthenticationError.
func IsAuthenticationError(err error) bool {
	var authErr *AuthenticationError
	return errors.As(err, &authErr)
}

// findSigningKey finds the appropriate signing key from JWKS.
func findSigningKey(jwks *JWKS, kid string) (interface{}, error) {
	for _, key := range jwks.Keys {
		if key.Alg != algorithm || key.Use != "sig" {
			continue
		}
		if kid != "" && key.Kid != kid {
			continue
		}

		// Build EC public key from JWK
		pubKey, err := buildECPublicKey(&key)
		if err != nil {
			continue
		}
		return pubKey, nil
	}

	return nil, &AuthenticationError{Message: "no suitable signing key found in JWKS"}
}

// buildECPublicKey builds an EC public key from JWK parameters.
func buildECPublicKey(jwk *JWK) (*ecdsa.PublicKey, error) {
	if jwk.Kty != "EC" || jwk.Crv != "P-256" {
		return nil, fmt.Errorf("unsupported key type or curve")
	}

	xBytes, err := base64.RawURLEncoding.DecodeString(jwk.X)
	if err != nil {
		return nil, fmt.Errorf("failed to decode X coordinate: %v", err)
	}

	yBytes, err := base64.RawURLEncoding.DecodeString(jwk.Y)
	if err != nil {
		return nil, fmt.Errorf("failed to decode Y coordinate: %v", err)
	}

	pubKey := &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int).SetBytes(xBytes),
		Y:     new(big.Int).SetBytes(yBytes),
	}

	return pubKey, nil
}

// parseJwtPayload converts jwt.MapClaims to JwtPayload.
func parseJwtPayload(claims jwt.MapClaims) *JwtPayload {
	payload := &JwtPayload{}

	if sub, ok := claims["sub"].(string); ok {
		payload.Sub = sub
	}
	if orgID, ok := claims["org_id"].(float64); ok {
		id := int(orgID)
		payload.OrgID = &id
	}
	if extID, ok := claims["ext_id"].(float64); ok {
		id := int(extID)
		payload.ExtID = &id
	}
	if projID, ok := claims["proj_id"].(float64); ok {
		id := int(projID)
		payload.ProjID = &id
	}
	if piID, ok := claims["pi_id"].(float64); ok {
		id := int(piID)
		payload.PiID = &id
	}
	if scopes, ok := claims["scopes"].([]interface{}); ok {
		payload.Scopes = make([]string, 0, len(scopes))
		for _, s := range scopes {
			if str, ok := s.(string); ok {
				payload.Scopes = append(payload.Scopes, str)
			}
		}
	}
	if src, ok := claims["src"].(string); ok {
		payload.Src = src
	}
	if iss, ok := claims["iss"].(string); ok {
		payload.Iss = iss
	}
	if iat, ok := claims["iat"].(float64); ok {
		payload.Iat = int64(iat)
	}
	if exp, ok := claims["exp"].(float64); ok {
		payload.Exp = int64(exp)
	}
	if jti, ok := claims["jti"].(string); ok {
		payload.Jti = jti
	}

	return payload
}
