package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	sellerpb "services/seller/proto"
	"services/utils"
)

func HandleGetSellerProfile(w http.ResponseWriter, r *http.Request) {
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
	ctx := r.Context()
	req := &sellerpb.GetSellerProfileRequest{SellerId: utils.ParseInt64OrZero(sellerID)}
	resp, err := sellerClient.GetSellerProfile(ctx, req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func HandleGetSellersPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, msgMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}
	var req sellerpb.SellersPreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, msgInvalidBody, http.StatusBadRequest)
		return
	}
	ctx := r.Context()
	resp, err := sellerClient.GetSellersPreview(ctx, &req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
