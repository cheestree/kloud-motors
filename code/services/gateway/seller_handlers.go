package main

import (
	"context"
	"net/http"
	"strings"

	sellerpb "services/seller/proto"
)

func handleGetSellerProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, msgMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 || parts[3] == "" {
		http.Error(w, "Missing seller id", http.StatusBadRequest)
		return
	}
	sellerID := parts[3]
	ctx := context.Background()
	req := &sellerpb.GetSellerProfileRequest{SellerId: parseInt64(sellerID)}
	resp, err := sellerClient.GetSellerProfile(ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
