package api

import (
	"crypto/hmac"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ericogr/quimera-cards/internal/constants"
)

type jwtClaims struct {
	Sub  string `json:"sub"`  // email
	Name string `json:"name"` // display name
	Iat  int64  `json:"iat"`
	Exp  int64  `json:"exp"`
}

var devSecret []byte

func getSessionSecret() ([]byte, error) {
	secret := os.Getenv(constants.EnvSessionSecret)
	if secret == "" {
		// Generate an in-memory secret for development if not set
		if len(devSecret) == 0 {
			devSecret = make([]byte, 32)
			if _, err := crand.Read(devSecret); err != nil {
				return nil, errors.New("failed to generate dev session secret")
			}
		}
		return devSecret, nil
	}
	return []byte(secret), nil
}

func b64url(data []byte) string {
	return strings.TrimRight(base64.URLEncoding.EncodeToString(data), "=")
}

func b64urlDecode(s string) ([]byte, error) {
	// pad to multiple of 4
	if m := len(s) % 4; m != 0 {
		s += strings.Repeat("=", 4-m)
	}
	return base64.URLEncoding.DecodeString(s)
}

func signHS256(data string, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(data))
	sig := mac.Sum(nil)
	return b64url(sig)
}

func createSessionToken(email, name string, ttl time.Duration) (string, error) {
	secret, err := getSessionSecret()
	if err != nil {
		return "", err
	}
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	hdrJSON, _ := json.Marshal(header)
	now := time.Now().Unix()
	claims := jwtClaims{Sub: email, Name: name, Iat: now, Exp: now + int64(ttl.Seconds())}
	clJSON, _ := json.Marshal(claims)
	unsigned := fmt.Sprintf("%s.%s", b64url(hdrJSON), b64url(clJSON))
	sig := signHS256(unsigned, secret)
	return unsigned + "." + sig, nil
}

func parseAndValidateSession(token string) (*jwtClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid token format")
	}
	secret, err := getSessionSecret()
	if err != nil {
		return nil, err
	}
	unsigned := parts[0] + "." + parts[1]
	expected := signHS256(unsigned, secret)
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return nil, errors.New("invalid signature")
	}
	payloadBytes, err := b64urlDecode(parts[1])
	if err != nil {
		return nil, err
	}
	var claims jwtClaims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, err
	}
	if time.Now().Unix() > claims.Exp {
		return nil, errors.New("token expired")
	}
	return &claims, nil
}
