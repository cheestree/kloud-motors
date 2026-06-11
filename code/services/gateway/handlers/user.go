package handlers

import (
	"context"
	"errors"
	"net/http"

	userrequests "services/gateway/handlers/user"
	userpb "services/user/proto"

	"github.com/go-playground/validator/v10"
	"google.golang.org/grpc"
)

func HandleUserLogin(w http.ResponseWriter, r *http.Request) {
	handleUserAuth(w, r, userClient.Login)
}

func HandleUserRegister(w http.ResponseWriter, r *http.Request) {
	handleUserAuth(w, r, userClient.Register)
}

func HandleUserRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, msgMethodNotAllowed, nil)
		return
	}

	var body userrequests.RefreshTokenBody
	if err := userrequests.BindAndValidateJSON(r, &body); err != nil {
		writeRequestError(w, "Invalid refresh token body", err)
		return
	}

	resp, err := userClient.RefreshToken(r.Context(), userrequests.BuildRefreshTokenRequest(body))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func handleUserAuth(w http.ResponseWriter, r *http.Request, authFn func(context.Context, *userpb.AuthRequest, ...grpc.CallOption) (*userpb.AuthResponse, error)) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, msgMethodNotAllowed, nil)
		return
	}

	var body userrequests.AuthBody
	if err := userrequests.BindAndValidateJSON(r, &body); err != nil {
		writeRequestError(w, "Invalid authentication body", err)
		return
	}

	resp, err := authFn(r.Context(), userrequests.BuildAuthRequest(body))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func HandleGetFavorites(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, msgMethodNotAllowed, nil)
		return
	}
	authUserID, err := authenticatedUserIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, msgUnauthorized, nil)
		return
	}
	ctx := r.Context()
	resp, err := userClient.GetFavorites(ctx, userrequests.BuildGetFavoritesRequest(authUserID))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func HandleFavoriteListing(w http.ResponseWriter, r *http.Request) {
	listingID, err := userrequests.FavoriteListingIDFromPath(r)
	if err != nil {
		if errors.Is(err, userrequests.ErrMissingListingID) {
			writeError(w, http.StatusBadRequest, "Missing listing id", nil)
			return
		}
		var validationErrs validator.ValidationErrors
		if errors.As(err, &validationErrs) {
			writeRequestError(w, "Invalid favorite listing id", err)
			return
		}
		writeError(w, http.StatusBadRequest, "Invalid favorite listing id", []fieldError{{
			Field:   "listing_id",
			Message: "must be a positive integer",
		}})
		return
	}
	authUserID, err := authenticatedUserIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, msgUnauthorized, nil)
		return
	}
	ctx := r.Context()
	switch r.Method {
	case http.MethodPost:
		resp, err := userClient.AddFavorite(ctx, userrequests.BuildAddFavoriteRequest(authUserID, listingID))
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, resp)
	case http.MethodDelete:
		resp, err := userClient.RemoveFavorite(ctx, userrequests.BuildRemoveFavoriteRequest(authUserID, listingID))
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, resp)
	default:
		writeError(w, http.StatusMethodNotAllowed, msgMethodNotAllowed, nil)
	}
}

func HandleGetUsersPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, msgMethodNotAllowed, nil)
		return
	}
	var body userrequests.UsersPreviewBody
	if err := userrequests.BindAndValidateJSON(r, &body); err != nil {
		writeRequestError(w, "Invalid user preview body", err)
		return
	}
	ctx := r.Context()
	resp, err := userClient.GetUsersPreview(ctx, userrequests.BuildUsersPreviewRequest(body))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
