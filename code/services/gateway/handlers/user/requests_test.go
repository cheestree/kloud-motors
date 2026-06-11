package user

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
)

func TestBindAndValidateAuthBodyParsesAndBuildsRequest(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/user/login", strings.NewReader(`{"email":"buyer@example.com","password":"secret"}`))
	body := AuthBody{}

	if err := BindAndValidateJSON(req, &body); err != nil {
		t.Fatalf("BindAndValidateJSON returned error: %v", err)
	}

	protoReq := BuildAuthRequest(body)
	if protoReq.Email != "buyer@example.com" || protoReq.Password != "secret" {
		t.Fatalf("BuildAuthRequest() = %+v, want buyer@example.com/secret", protoReq)
	}
}

func TestBindAndValidateAuthBodyRequiresEmailAndPassword(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/user/login", strings.NewReader(`{"email":"buyer@example.com"}`))
	body := AuthBody{}

	err := BindAndValidateJSON(req, &body)
	if err == nil {
		t.Fatal("BindAndValidateJSON returned nil, want validation error")
	}

	var validationErrs validator.ValidationErrors
	if !errors.As(err, &validationErrs) {
		t.Fatalf("error = %T, want validator.ValidationErrors", err)
	}

	foundPassword := false
	for _, validationErr := range validationErrs {
		if validationErr.Field() == "password" && validationErr.Tag() == "notblank" {
			foundPassword = true
		}
	}
	if !foundPassword {
		t.Fatalf("validation errors = %v, want password notblank", validationErrs)
	}
}

func TestBindAndValidateRefreshTokenBodyParsesAndBuildsRequest(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/user/refresh", strings.NewReader(`{"refresh_token":"refresh-token-value"}`))
	body := RefreshTokenBody{}

	if err := BindAndValidateJSON(req, &body); err != nil {
		t.Fatalf("BindAndValidateJSON returned error: %v", err)
	}

	protoReq := BuildRefreshTokenRequest(body)
	if protoReq.RefreshToken != "refresh-token-value" {
		t.Fatalf("RefreshToken = %q, want refresh-token-value", protoReq.RefreshToken)
	}
}

func TestFavoriteListingIDFromPathParsesAndBuildsRequests(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/users/me/favorites/123", nil)

	listingID, err := FavoriteListingIDFromPath(req)
	if err != nil {
		t.Fatalf("FavoriteListingIDFromPath returned error: %v", err)
	}

	addReq := BuildAddFavoriteRequest(42, listingID)
	if addReq.UserId != 42 || addReq.ListingId != 123 {
		t.Fatalf("BuildAddFavoriteRequest() = %+v, want user 42 listing 123", addReq)
	}

	removeReq := BuildRemoveFavoriteRequest(42, listingID)
	if removeReq.UserId != 42 || removeReq.ListingId != 123 {
		t.Fatalf("BuildRemoveFavoriteRequest() = %+v, want user 42 listing 123", removeReq)
	}
}

func TestFavoriteListingIDFromPathRejectsMissingID(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/users/me/favorites/", nil)

	_, err := FavoriteListingIDFromPath(req)
	if !errors.Is(err, ErrMissingListingID) {
		t.Fatalf("error = %v, want ErrMissingListingID", err)
	}
}

func TestFavoriteListingIDFromPathRejectsInvalidID(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/users/me/favorites/nope", nil)

	_, err := FavoriteListingIDFromPath(req)
	if err == nil {
		t.Fatal("FavoriteListingIDFromPath returned nil, want parse error")
	}
}

func TestFavoriteListingIDFromPathRejectsNonPositiveID(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/users/me/favorites/0", nil)

	_, err := FavoriteListingIDFromPath(req)
	if err == nil {
		t.Fatal("FavoriteListingIDFromPath returned nil, want validation error")
	}

	var validationErrs validator.ValidationErrors
	if !errors.As(err, &validationErrs) {
		t.Fatalf("error = %T, want validator.ValidationErrors", err)
	}
	if validationErrs[0].Field() != "listing_id" || validationErrs[0].Tag() != "gt" {
		t.Fatalf("validation error = field %q tag %q, want listing_id gt", validationErrs[0].Field(), validationErrs[0].Tag())
	}
}

func TestBindAndValidateUsersPreviewBodyParsesAndBuildsRequest(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/users/preview", strings.NewReader(`{"user_ids":[101,102]}`))
	body := UsersPreviewBody{}

	if err := BindAndValidateJSON(req, &body); err != nil {
		t.Fatalf("BindAndValidateJSON returned error: %v", err)
	}

	protoReq := BuildUsersPreviewRequest(body)
	if len(protoReq.UserIds) != 2 || protoReq.UserIds[0] != 101 || protoReq.UserIds[1] != 102 {
		t.Fatalf("UserIds = %v, want [101 102]", protoReq.UserIds)
	}
}

func TestBindAndValidateUsersPreviewBodyAllowsEmptyIDs(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/users/preview", strings.NewReader(`{"user_ids":[]}`))
	body := UsersPreviewBody{}

	if err := BindAndValidateJSON(req, &body); err != nil {
		t.Fatalf("BindAndValidateJSON returned error: %v", err)
	}
	if len(body.UserIDs) != 0 {
		t.Fatalf("UserIDs = %v, want empty", body.UserIDs)
	}
}

func TestBindAndValidateUsersPreviewBodyRejectsNonPositiveIDs(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/users/preview", strings.NewReader(`{"user_ids":[101,0]}`))
	body := UsersPreviewBody{}

	err := BindAndValidateJSON(req, &body)
	if err == nil {
		t.Fatal("BindAndValidateJSON returned nil, want validation error")
	}

	var validationErrs validator.ValidationErrors
	if !errors.As(err, &validationErrs) {
		t.Fatalf("error = %T, want validator.ValidationErrors", err)
	}
	if len(validationErrs) != 1 || !strings.HasPrefix(validationErrs[0].Field(), "user_ids") || validationErrs[0].Tag() != "gt" {
		t.Fatalf("validation errors = %v, want user_ids gt", validationErrs)
	}
}
