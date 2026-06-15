package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	userpb "services/user/proto"
	"services/user/repository"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func newMockUserService(t *testing.T) (*UserService, sqlmock.Sqlmock, func()) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
	if err != nil {
		db.Close()
		t.Fatalf("failed to open gorm db: %v", err)
	}

	return NewUserService(repository.NewRepository(gormDB)), mock, func() {
		db.Close()
	}
}

func TestUserService_LoginCallsFirebaseSignInWithPassword(t *testing.T) {
	var gotPath string
	var gotKey string
	var gotBody firebaseAuthRequest

	firebaseServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotKey = r.URL.Query().Get("key")
		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("unexpected content-type: %s", r.Header.Get("Content-Type"))
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"idToken": "id-token",
			"email": "seller7514@mock.local",
			"refreshToken": "refresh-token",
			"expiresIn": "3600",
			"localId": "firebase-uid"
		}`))
	}))
	defer firebaseServer.Close()

	authClient := &firebaseAuthClient{
		apiKey:      "secret-key",
		authBaseURL: firebaseServer.URL,
		client:      firebaseServer.Client(),
	}

	resp, err := authClient.login(context.Background(), "seller7514@mock.local", "password123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotPath != "/accounts:signInWithPassword" {
		t.Fatalf("unexpected firebase path: %s", gotPath)
	}
	if gotKey != "secret-key" {
		t.Fatalf("unexpected firebase key: %s", gotKey)
	}
	if gotBody.Email != "seller7514@mock.local" || gotBody.Password != "password123" || !gotBody.ReturnSecureToken {
		t.Fatalf("unexpected firebase request body: %+v", gotBody)
	}
	if resp.IdToken != "id-token" || resp.RefreshToken != "refresh-token" || resp.LocalId != "firebase-uid" {
		t.Fatalf("unexpected auth response: %+v", resp)
	}
}

func TestUserService_RegisterCallsFirebaseSignUp(t *testing.T) {
	var gotPath string

	firebaseServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"idToken": "new-id-token",
			"email": "new-user@example.test",
			"refreshToken": "new-refresh-token",
			"expiresIn": "3600",
			"localId": "new-firebase-uid"
		}`))
	}))
	defer firebaseServer.Close()

	authClient := &firebaseAuthClient{
		apiKey:      "secret-key",
		authBaseURL: firebaseServer.URL,
		client:      firebaseServer.Client(),
	}

	resp, err := authClient.register(context.Background(), "new-user@example.test", "password123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotPath != "/accounts:signUp" {
		t.Fatalf("unexpected firebase path: %s", gotPath)
	}
	if resp.IdToken != "new-id-token" || resp.Email != "new-user@example.test" {
		t.Fatalf("unexpected auth response: %+v", resp)
	}
}

func TestUserService_AuthMapsFirebaseErrors(t *testing.T) {
	firebaseServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"INVALID_PASSWORD"}}`))
	}))
	defer firebaseServer.Close()

	authClient := &firebaseAuthClient{
		apiKey:      "secret-key",
		authBaseURL: firebaseServer.URL,
		client:      firebaseServer.Client(),
	}

	_, err := authClient.login(context.Background(), "seller7514@mock.local", "wrong")
	if err == nil || !strings.Contains(err.Error(), "invalid email or password") {
		t.Fatalf("expected invalid credentials error, got %v", err)
	}
}

func TestUserService_RefreshTokenCallsFirebaseSecureToken(t *testing.T) {
	var gotPath string
	var gotBody firebaseRefreshRequest

	firebaseServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id_token": "new-id-token",
			"refresh_token": "new-refresh-token",
			"expires_in": "3600",
			"user_id": "firebase-uid"
		}`))
	}))
	defer firebaseServer.Close()

	authClient := &firebaseAuthClient{
		apiKey:             "secret-key",
		secureTokenBaseURL: firebaseServer.URL,
		client:             firebaseServer.Client(),
	}

	resp, err := authClient.refreshToken(context.Background(), "old-refresh-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotPath != "/token" {
		t.Fatalf("unexpected firebase path: %s", gotPath)
	}
	if gotBody.GrantType != "refresh_token" || gotBody.RefreshToken != "old-refresh-token" {
		t.Fatalf("unexpected refresh request body: %+v", gotBody)
	}
	if resp.IdToken != "new-id-token" || resp.RefreshToken != "new-refresh-token" || resp.LocalId != "firebase-uid" {
		t.Fatalf("unexpected refresh response: %+v", resp)
	}
}

func TestUserService_LoginReturnsDatabaseUserID(t *testing.T) {
	svc, mock, cleanup := newMockUserService(t)
	defer cleanup()

	firebaseServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"idToken": "id-token",
			"email": "seller7514@mock.local",
			"refreshToken": "refresh-token",
			"expiresIn": "3600",
			"localId": "firebase-uid"
		}`))
	}))
	defer firebaseServer.Close()

	svc.auth = &firebaseAuthClient{
		apiKey:      "secret-key",
		authBaseURL: firebaseServer.URL,
		client:      firebaseServer.Client(),
	}

	mock.ExpectQuery(`SELECT \* FROM "users" WHERE "users"\."firebase_uid" = \$1 ORDER BY "users"\."id" LIMIT \$2`).
		WithArgs("firebase-uid", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "firebase_uid", "name", "email"}).
			AddRow(int64(42), "firebase-uid", "", "seller7514@mock.local"))

	resp, err := svc.Login(context.Background(), &userpb.AuthRequest{
		Email:    "seller7514@mock.local",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.UserId != 42 || resp.LocalId != "firebase-uid" {
		t.Fatalf("unexpected auth response ids: %+v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

func TestUserService_GetFavoritesMapsListingIDs(t *testing.T) {
	svc, mock, cleanup := newMockUserService(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT \* FROM "favorites" WHERE user_id = \$1`).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "listing_id"}).
			AddRow(uint(1), int64(7), int64(100)).
			AddRow(uint(2), int64(7), int64(200)))

	resp, err := svc.GetFavorites(context.Background(), &userpb.GetFavoritesRequest{UserId: 7})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Favorites) != 2 || resp.Favorites[0] != 100 || resp.Favorites[1] != 200 {
		t.Fatalf("unexpected favorites: %v", resp.Favorites)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

func TestUserService_GetUsersPreview(t *testing.T) {
	svc, mock, cleanup := newMockUserService(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT \* FROM "users" WHERE id IN \(\$1,\$2\)`).
		WithArgs(int64(1), int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "firebase_uid", "name", "email"}).
			AddRow(int64(1), "uid-1", "Alice", "alice@example.test").
			AddRow(int64(2), "uid-2", "Bob", "bob@example.test"))

	resp, err := svc.GetUsersPreview(context.Background(), &userpb.UsersPreviewRequest{UserIds: []int64{1, 2}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Users) != 2 || resp.Users[0].Name != "Alice" || resp.Users[1].Name != "Bob" {
		t.Fatalf("unexpected users preview: %+v", resp.Users)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}
