package handlers

import (
	"errors"
	"net/http"

	sellerrequests "services/gateway/handlers/seller"

	"github.com/go-playground/validator/v10"
)

func HandleGetSellerProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, msgMethodNotAllowed, nil)
		return
	}

	sellerID, err := sellerrequests.SellerIDFromPath(r)
	if err != nil {
		if errors.Is(err, sellerrequests.ErrMissingSellerID) {
			writeError(w, http.StatusBadRequest, "Missing seller id", nil)
			return
		}
		var validationErrs validator.ValidationErrors
		if errors.As(err, &validationErrs) {
			writeRequestError(w, "Invalid seller id", err)
			return
		}
		writeError(w, http.StatusBadRequest, "Invalid seller id", []fieldError{{
			Field:   "seller_id",
			Message: "must be a positive integer",
		}})
		return
	}
	ctx := r.Context()

	resp, err := sellerClient.GetSellerProfile(ctx, sellerrequests.BuildGetSellerProfileRequest(sellerID))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func HandleGetSellersPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, msgMethodNotAllowed, nil)
		return
	}
	var body sellerrequests.SellersPreviewBody
	if err := sellerrequests.BindAndValidateJSON(r, &body); err != nil {
		writeRequestError(w, "Invalid seller preview body", err)
		return
	}
	ctx := r.Context()

	resp, err := sellerClient.GetSellersPreview(ctx, sellerrequests.BuildSellersPreviewRequest(body))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
