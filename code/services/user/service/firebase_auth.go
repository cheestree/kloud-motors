package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	proto "services/user/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const defaultFirebaseAuthBaseURL = "https://identitytoolkit.googleapis.com/v1"
const defaultFirebaseSecureTokenBaseURL = "https://securetoken.googleapis.com/v1"

type firebaseAuthClient struct {
	apiKey             string
	authBaseURL        string
	secureTokenBaseURL string
	client             *http.Client
}

type firebaseAuthRequest struct {
	Email             string `json:"email"`
	Password          string `json:"password"`
	ReturnSecureToken bool   `json:"returnSecureToken"`
}

type firebaseAuthResponse struct {
	IDToken      string `json:"idToken"`
	Email        string `json:"email"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    string `json:"expiresIn"`
	LocalID      string `json:"localId"`
}

type firebaseRefreshRequest struct {
	GrantType    string `json:"grant_type"`
	RefreshToken string `json:"refresh_token"`
}

type firebaseRefreshResponse struct {
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    string `json:"expires_in"`
	LocalID      string `json:"user_id"`
}

type firebaseAuthErrorResponse struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

func newFirebaseAuthClientFromEnv() *firebaseAuthClient {
	return &firebaseAuthClient{
		apiKey:             strings.TrimSpace(os.Getenv("FIREBASE_API_KEY")),
		authBaseURL:        strings.TrimRight(os.Getenv("FIREBASE_AUTH_BASE_URL"), "/"),
		secureTokenBaseURL: strings.TrimRight(os.Getenv("FIREBASE_SECURE_TOKEN_BASE_URL"), "/"),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *firebaseAuthClient) login(ctx context.Context, email, password string) (*proto.AuthResponse, error) {
	return c.authenticate(ctx, "accounts:signInWithPassword", email, password)
}

func (c *firebaseAuthClient) register(ctx context.Context, email, password string) (*proto.AuthResponse, error) {
	return c.authenticate(ctx, "accounts:signUp", email, password)
}

func (c *firebaseAuthClient) refreshToken(ctx context.Context, refreshToken string) (*proto.AuthResponse, error) {
	if c.apiKey == "" {
		return nil, status.Error(codes.FailedPrecondition, "firebase api key is not configured")
	}

	body, err := json.Marshal(firebaseRefreshRequest{
		GrantType:    "refresh_token",
		RefreshToken: refreshToken,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to encode refresh token request")
	}

	endpoint := c.secureTokenBaseURL
	if endpoint == "" {
		endpoint = defaultFirebaseSecureTokenBaseURL
	}
	reqURL := fmt.Sprintf("%s/token?key=%s", endpoint, url.QueryEscape(c.apiKey))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(body))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create refresh token request")
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, status.Error(codes.Unavailable, "firebase refresh token request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var firebaseErr firebaseAuthErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&firebaseErr); err == nil && firebaseErr.Error.Message != "" {
			return nil, mapFirebaseAuthError(firebaseErr.Error.Message)
		}
		return nil, status.Error(codes.Unauthenticated, "firebase refresh token failed")
	}

	var refreshResp firebaseRefreshResponse
	if err := json.NewDecoder(resp.Body).Decode(&refreshResp); err != nil {
		return nil, status.Error(codes.Internal, "failed to decode refresh token response")
	}

	return &proto.AuthResponse{
		IdToken:      refreshResp.IDToken,
		RefreshToken: refreshResp.RefreshToken,
		ExpiresIn:    refreshResp.ExpiresIn,
		LocalId:      refreshResp.LocalID,
	}, nil
}

func (c *firebaseAuthClient) authenticate(ctx context.Context, method, email, password string) (*proto.AuthResponse, error) {
	if c.apiKey == "" {
		return nil, status.Error(codes.FailedPrecondition, "firebase api key is not configured")
	}

	body, err := json.Marshal(firebaseAuthRequest{
		Email:             email,
		Password:          password,
		ReturnSecureToken: true,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to encode auth request")
	}

	endpoint := c.authBaseURL
	if endpoint == "" {
		endpoint = defaultFirebaseAuthBaseURL
	}
	reqURL := fmt.Sprintf("%s/%s?key=%s", endpoint, method, url.QueryEscape(c.apiKey))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(body))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create auth request")
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, status.Error(codes.Unavailable, "firebase auth request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var firebaseErr firebaseAuthErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&firebaseErr); err == nil && firebaseErr.Error.Message != "" {
			return nil, mapFirebaseAuthError(firebaseErr.Error.Message)
		}
		return nil, status.Error(codes.Unauthenticated, "firebase auth failed")
	}

	var authResp firebaseAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, status.Error(codes.Internal, "failed to decode auth response")
	}

	return &proto.AuthResponse{
		IdToken:      authResp.IDToken,
		Email:        authResp.Email,
		RefreshToken: authResp.RefreshToken,
		ExpiresIn:    authResp.ExpiresIn,
		LocalId:      authResp.LocalID,
	}, nil
}

func mapFirebaseAuthError(message string) error {
	switch strings.ToUpper(message) {
	case "EMAIL_EXISTS":
		return status.Error(codes.AlreadyExists, "email already exists")
	case "EMAIL_NOT_FOUND", "INVALID_PASSWORD", "INVALID_LOGIN_CREDENTIALS":
		return status.Error(codes.Unauthenticated, "invalid email or password")
	case "TOKEN_EXPIRED", "USER_NOT_FOUND", "INVALID_REFRESH_TOKEN":
		return status.Error(codes.Unauthenticated, "invalid refresh token")
	case "INVALID_EMAIL":
		return status.Error(codes.InvalidArgument, "invalid email")
	case "WEAK_PASSWORD":
		return status.Error(codes.InvalidArgument, "weak password")
	default:
		return status.Error(codes.Unauthenticated, "firebase auth failed")
	}
}
