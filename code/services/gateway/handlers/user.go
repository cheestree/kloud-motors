package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	userpb "services/user/proto"
	"services/utils"

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
		http.Error(w, msgMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	var req userpb.RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, msgInvalidBody, http.StatusBadRequest)
		return
	}
	if req.RefreshToken == "" {
		http.Error(w, "refresh_token is required", http.StatusBadRequest)
		return
	}

	resp, err := userClient.RefreshToken(r.Context(), &req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func handleUserAuth(w http.ResponseWriter, r *http.Request, authFn func(context.Context, *userpb.AuthRequest, ...grpc.CallOption) (*userpb.AuthResponse, error)) {
	if r.Method != http.MethodPost {
		http.Error(w, msgMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	var req userpb.AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, msgInvalidBody, http.StatusBadRequest)
		return
	}
	if req.Email == "" || req.Password == "" {
		http.Error(w, "email and password are required", http.StatusBadRequest)
		return
	}

	resp, err := authFn(r.Context(), &req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func HandleGetFavorites(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, msgMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}
	authUserID, err := authenticatedUserIDFromRequest(r)
	if err != nil {
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}
	ctx := r.Context()
	req := &userpb.GetFavoritesRequest{UserId: authUserID}
	resp, err := userClient.GetFavorites(ctx, req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func HandleFavoriteListing(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 6 || parts[5] == "" {
		http.Error(w, "Missing listing id", http.StatusBadRequest)
		return
	}
	listingID := utils.ParseInt64(parts[5])
	authUserID, err := authenticatedUserIDFromRequest(r)
	if err != nil {
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}
	ctx := r.Context()
	switch r.Method {
	case http.MethodPost:
		req := &userpb.AddFavoriteRequest{UserId: authUserID, ListingId: listingID}
		resp, err := userClient.AddFavorite(ctx, req)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, resp)
	case http.MethodDelete:
		req := &userpb.RemoveFavoriteRequest{UserId: authUserID, ListingId: listingID}
		resp, err := userClient.RemoveFavorite(ctx, req)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, resp)
	default:
		http.Error(w, msgMethodNotAllowed, http.StatusMethodNotAllowed)
	}
}

func HandleGetUsersPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, msgMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}
	var req userpb.UsersPreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, msgInvalidBody, http.StatusBadRequest)
		return
	}
	ctx := r.Context()
	resp, err := userClient.GetUsersPreview(ctx, &req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
