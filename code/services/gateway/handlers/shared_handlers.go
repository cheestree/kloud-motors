package handlers

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
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
	if strings.TrimSpace(b64Key) == "" {
		return 0, errors.New("JWT_PUBLIC_KEY_B64 is not configured")
	}

	pubKey, err := parseRSAPublicKey(b64Key)
	if err != nil {
		return 0, err
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

func parseRSAPublicKey(value string) (interface{}, error) {
	trimmed := strings.TrimSpace(value)

	// Support raw PEM in env variable.
	if strings.Contains(trimmed, "BEGIN PUBLIC KEY") || strings.Contains(trimmed, "BEGIN RSA PUBLIC KEY") {
		pubKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(trimmed))
		if err != nil {
			return nil, errors.New("failed to parse RSA public key")
		}
		return pubKey, nil
	}

	// Support regular base64 and unpadded base64 payloads.
	decoded, err := base64.StdEncoding.DecodeString(trimmed)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(trimmed)
		if err != nil {
			return nil, errors.New("failed to decode base64 public key")
		}
	}

	pubKey, err := jwt.ParseRSAPublicKeyFromPEM(decoded)
	if err != nil {
		return nil, errors.New("failed to parse RSA public key")
	}

	return pubKey, nil
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set(headerContentType, contentTypeJSON)
	w.WriteHeader(status)

	if msg, ok := payload.(proto.Message); ok {
		marshaler := protojson.MarshalOptions{
			EmitUnpopulated: true,
			UseProtoNames:   true,
		}
		b, err := marshaler.Marshal(msg)
		if err == nil {
			_, _ = w.Write(b)
			return
		}
	}

	_ = json.NewEncoder(w).Encode(payload)
}
