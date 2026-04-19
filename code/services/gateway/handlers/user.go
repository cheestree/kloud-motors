package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	userpb "services/user/proto"
)


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
	ctx := context.Background()
	req := &userpb.GetFavoritesRequest{UserId: authUserID}
	resp, err := userClient.GetFavorites(ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
	listingID := parseInt64(parts[5])
	authUserID, err := authenticatedUserIDFromRequest(r)
	if err != nil {
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}
	ctx := context.Background()
	switch r.Method {
	case http.MethodPost:
		req := &userpb.AddFavoriteRequest{UserId: authUserID, ListingId: listingID}
		resp, err := userClient.AddFavorite(ctx, req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, resp)
	case http.MethodDelete:
		req := &userpb.RemoveFavoriteRequest{UserId: authUserID, ListingId: listingID}
		resp, err := userClient.RemoveFavorite(ctx, req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
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
	ctx := context.Background()
	resp, err := userClient.GetUsersPreview(ctx, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
