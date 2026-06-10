package handlers

import (
	"errors"
	"net/http"

	listingrequests "services/gateway/handlers/listing"

	"github.com/go-playground/validator/v10"
)

func HandleListings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	switch r.Method {
	case http.MethodGet:
		query := listingrequests.DefaultListingListQuery()
		if err := listingrequests.BindAndValidateQuery(r, &query); err != nil {
			writeRequestError(w, "Invalid listing pagination parameters", err)
			return
		}

		resp, err := searchClient.Search(ctx, query.SearchRequest())
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, resp)
	case http.MethodPost:
		authUserID, err := authenticatedUserIDFromRequest(r)
		if err != nil {
			writeError(w, http.StatusUnauthorized, msgUnauthorized, nil)
			return
		}

		var body listingrequests.ListingMutationBody
		if err := listingrequests.BindAndValidateJSON(r, &body); err != nil {
			writeRequestError(w, "Invalid listing creation body", err)
			return
		}
		req := listingrequests.BuildCreateListingRequest(body, authUserID)

		resp, err := listingClient.CreateListing(ctx, req)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, resp)
	default:
		writeError(w, http.StatusMethodNotAllowed, msgMethodNotAllowed, nil)
	}
}

func HandleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, msgMethodNotAllowed, nil)
		return
	}
	query := listingrequests.DefaultListingSearchQuery()
	if err := listingrequests.BindAndValidateQuery(r, &query); err != nil {
		writeRequestError(w, "Invalid listing search filters: year must be between at least 1886; minPrice, maxPrice, and maxMileage cannot be negative; page must be at least 1; pageSize must be between 1 and 100", err)
		return
	}

	resp, err := searchClient.Search(r.Context(), query.SearchRequest())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func HandleCompare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, msgMethodNotAllowed, nil)
		return
	}
	ids, err := listingrequests.ParseCommaSeparatedInt64s(r.URL.Query().Get(queryIDs))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid listing comparison parameters", []fieldError{{
			Field:   queryIDs,
			Message: "must be a comma-separated list of positive integers",
		}})
		return
	}
	query := listingrequests.ListingCompareQuery{IDs: ids}
	if err := listingrequests.Validate(query); err != nil {
		writeRequestError(w, "Invalid listing comparison parameters", err)
		return
	}
	ctx := r.Context()
	resp, err := listingClient.CompareListings(ctx, listingrequests.BuildCompareListingsRequest(ids))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func HandleGetListing(w http.ResponseWriter, r *http.Request) {
	id, err := listingrequests.ListingIDFromPath(r)
	if err != nil {
		if errors.Is(err, listingrequests.ErrMissingListingID) {
			writeError(w, http.StatusBadRequest, "Missing listing id", nil)
			return
		}
		var validationErrs validator.ValidationErrors
		if errors.As(err, &validationErrs) {
			writeRequestError(w, "Invalid listing id", err)
			return
		}
		writeError(w, http.StatusBadRequest, "Invalid listing id", []fieldError{{
			Field:   "id",
			Message: "must be a positive integer",
		}})
		return
	}
	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		resp, err := listingClient.GetListingDetails(ctx, listingrequests.BuildListingDetailsRequest(id))
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, resp)
	case http.MethodPut:
		authUserID, err := authenticatedUserIDFromRequest(r)
		if err != nil {
			writeError(w, http.StatusUnauthorized, msgUnauthorized, nil)
			return
		}

		var body listingrequests.ListingMutationBody
		if err := listingrequests.BindAndValidateJSON(r, &body); err != nil {
			writeRequestError(w, "Invalid listing update body", err)
			return
		}
		req := listingrequests.BuildUpdateListingRequest(body, id, authUserID)

		resp, err := listingClient.UpdateListing(ctx, req)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, resp)
	case http.MethodDelete:
		authUserID, err := authenticatedUserIDFromRequest(r)
		if err != nil {
			writeError(w, http.StatusUnauthorized, msgUnauthorized, nil)
			return
		}

		resp, err := listingClient.DeleteListing(ctx, listingrequests.BuildDeleteListingRequest(id, authUserID))
		if err != nil {
			writeServiceError(w, err)
			return
		}
		if !resp.GetDeleted() {
			writeError(w, http.StatusNotFound, msgNotFound, nil)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		writeError(w, http.StatusMethodNotAllowed, msgMethodNotAllowed, nil)
	}
}
