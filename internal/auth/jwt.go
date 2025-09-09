package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type JWTServiceImpl struct {
	secret []byte
}

func NewJWTService(secret string) *JWTServiceImpl {
	return &JWTServiceImpl{
		secret: []byte(secret),
	}
}

type JWTHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
}

type JWTClaims struct {
	UserID    string `json:"user_id"`
	ExpiresAt int64  `json:"exp"`
	IssuedAt  int64  `json:"iat"`
}

func (j *JWTServiceImpl) GenerateToken(userID string, expiresAt time.Time) (string, error) {
	header := JWTHeader{
		Algorithm: "HS256",
		Type:      "JWT",
	}

	claims := JWTClaims{
		UserID:    userID,
		ExpiresAt: expiresAt.Unix(),
		IssuedAt:  time.Now().Unix(),
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("failed to marshal header: %w", err)
	}

	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("failed to marshal claims: %w", err)
	}

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	message := headerB64 + "." + claimsB64
	signature := j.createSignature(message)

	token := message + "." + signature
	return token, nil
}

func (j *JWTServiceImpl) ValidateToken(token string) (*TokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	headerB64, claimsB64, signatureB64 := parts[0], parts[1], parts[2]

	// Verify signature
	message := headerB64 + "." + claimsB64
	expectedSignature := j.createSignature(message)
	if signatureB64 != expectedSignature {
		return nil, fmt.Errorf("invalid signature")
	}

	// Decode claims
	claimsJSON, err := base64.RawURLEncoding.DecodeString(claimsB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode claims: %w", err)
	}

	var claims JWTClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, fmt.Errorf("failed to unmarshal claims: %w", err)
	}

	// Check expiration
	if time.Now().Unix() > claims.ExpiresAt {
		return nil, fmt.Errorf("token expired")
	}

	return &TokenClaims{
		UserID:    claims.UserID,
		ExpiresAt: time.Unix(claims.ExpiresAt, 0),
		IssuedAt:  time.Unix(claims.IssuedAt, 0),
	}, nil
}

func (j *JWTServiceImpl) RefreshToken(token string) (string, error) {
	claims, err := j.ValidateToken(token)
	if err != nil {
		return "", err
	}

	// Generate new token with extended expiration
	newExpiresAt := time.Now().Add(24 * time.Hour)
	return j.GenerateToken(claims.UserID, newExpiresAt)
}

func (j *JWTServiceImpl) createSignature(message string) string {
	h := hmac.New(sha256.New, j.secret)
	h.Write([]byte(message))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}