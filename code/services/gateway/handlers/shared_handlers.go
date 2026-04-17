package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

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

	secret := os.Getenv(envJWTSecret)
	if secret == "" {
		return 0, errors.New(errJWTNotConfigured)
	}

	claims := &UserClaims{}
	token, err := jwt.ParseWithClaims(parts[1], claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
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

func parseInt32(s string) int32 {
	var v int32
	_, _ = fmt.Sscan(s, &v)
	return v
}

func parseInt32WithDefault(s string, def int32) int32 {
	if s == "" {
		return def
	}
	return parseInt32(s)
}

func parseInt64(s string) int64 {
	var v int64
	_, _ = fmt.Sscan(s, &v)
	return v
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set(headerContentType, contentTypeJSON)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
