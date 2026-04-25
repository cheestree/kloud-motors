package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"

	"encoding/base64"

	"github.com/golang-jwt/jwt/v5"
)

type UserClaims struct {
	UserID int64  `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func authenticatedUserIDFromRequest(r *http.Request) (int64, error) {
	authHeader := strings.TrimSpace(r.Header.Get(headerAuth))
	if authHeader == "" {
		return 0, errors.New(errMissingAuthHeader)
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], authSchemeBearer) || strings.TrimSpace(parts[1]) == "" {
		return 0, errors.New(errInvalidAuthHeader)
	}

	b64Key := os.Getenv("JWT_PUBLIC_KEY_B64")
	if b64Key == "" {
		return 0, errors.New("JWT_PUBLIC_KEY_B64 is not configured")
	}

	keyBytes, err := base64.StdEncoding.DecodeString(b64Key)
	if err != nil {
		return 0, errors.New("failed to decode base64 public key")
	}

	pubKey, err := jwt.ParseRSAPublicKeyFromPEM(keyBytes)
	if err != nil {
		return 0, errors.New("failed to parse RSA public key")
	}

	claims := &UserClaims{}
	token, err := jwt.ParseWithClaims(parts[1], claims, func(token *jwt.Token) (interface{}, error) {
		return pubKey, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}))
	if err != nil || !token.Valid {
		return 0, errors.New(errInvalidToken)
	}

	if claims.UserID > 0 {
		return claims.UserID, nil
	}

	if claims.Subject != "" {
		userID, err := strconv.ParseInt(claims.Subject, 10, 64)
		if err == nil && userID > 0 {
			return userID, nil
		}
	}

	return 0, errors.New(errUserIDNotInToken)
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set(headerContentType, contentTypeJSON)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
