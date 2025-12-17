package kiket

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"
)

// AuthenticationError represents an authentication failure.
type AuthenticationError struct {
	Message string
}

func (e *AuthenticationError) Error() string {
	return e.Message
}

// VerifySignature verifies the HMAC signature of a webhook payload.
func VerifySignature(secret string, body []byte, headers Headers) error {
	if secret == "" {
		return &AuthenticationError{Message: "webhook secret not configured"}
	}

	signature := headers["X-Kiket-Signature"]
	if signature == "" {
		signature = headers["x-kiket-signature"]
	}
	if signature == "" {
		return &AuthenticationError{Message: "missing X-Kiket-Signature header"}
	}

	timestamp := headers["X-Kiket-Timestamp"]
	if timestamp == "" {
		timestamp = headers["x-kiket-timestamp"]
	}
	if timestamp == "" {
		return &AuthenticationError{Message: "missing X-Kiket-Timestamp header"}
	}

	// Parse and validate timestamp
	requestTime, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return &AuthenticationError{Message: "invalid X-Kiket-Timestamp header"}
	}

	now := time.Now().Unix()
	timeDiff := math.Abs(float64(now - requestTime))
	if timeDiff > 300 {
		return &AuthenticationError{
			Message: fmt.Sprintf("request timestamp too old or too far in future: %.0fs", timeDiff),
		}
	}

	// Compute expected signature
	payload := timestamp + "." + string(body)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	// Constant-time comparison
	if subtle.ConstantTimeCompare([]byte(signature), []byte(expectedSignature)) != 1 {
		return &AuthenticationError{Message: "invalid signature"}
	}

	return nil
}

// GenerateSignature generates an HMAC signature for a payload (for testing).
func GenerateSignature(secret string, body string, timestamp *int64) (signature string, ts string) {
	var tsVal int64
	if timestamp != nil {
		tsVal = *timestamp
	} else {
		tsVal = time.Now().Unix()
	}

	tsStr := strconv.FormatInt(tsVal, 10)
	payload := tsStr + "." + body

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	sig := hex.EncodeToString(mac.Sum(nil))

	return sig, tsStr
}

// IsAuthenticationError checks if an error is an AuthenticationError.
func IsAuthenticationError(err error) bool {
	var authErr *AuthenticationError
	return errors.As(err, &authErr)
}
