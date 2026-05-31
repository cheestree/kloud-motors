package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	listingpb "services/listing/proto"
	searchpb "services/search/proto"
	"services/utils"

	"google.golang.org/protobuf/types/known/wrapperspb"
)

func HandleListings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	switch r.Method {
	case http.MethodGet:
		q := r.URL.Query()
		includeSold, err := parseOptionalBoolWrapper(q.Get(queryIncludeSold))
		if err != nil {
			http.Error(w, "Invalid includeSold query parameter", http.StatusBadRequest)
			return
		}

		resp, err := searchClient.Search(ctx, &searchpb.SearchRequest{
			Page:        utils.ParseInt32WithDefault(q.Get(queryPage), 1),
			PageSize:    utils.ParseInt32WithDefault(q.Get(queryPageSizeV2), 20),
			IncludeSold: includeSold,
		})
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, resp)
	case http.MethodPost:
		authUserID, err := authenticatedUserIDFromRequest(r)
		if err != nil {
			http.Error(w, msgUnauthorized, http.StatusUnauthorized)
			return
		}

		var req listingpb.CreateListingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, msgInvalidBody, http.StatusBadRequest)
			return
		}
		req.DealerId = authUserID

		resp, err := listingClient.CreateListing(ctx, &req)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, resp)
	default:
		http.Error(w, msgMethodNotAllowed, http.StatusMethodNotAllowed)
	}
}

func HandleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, msgMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query()
	ctx := r.Context()
	includeSold, err := parseOptionalBoolWrapper(q.Get(queryIncludeSold))
	if err != nil {
		http.Error(w, "Invalid includeSold query parameter", http.StatusBadRequest)
		return
	}

	resp, err := searchClient.Search(ctx, &searchpb.SearchRequest{
		Make:        q.Get(queryMake),
		Model:       q.Get(queryModel),
		Year:        utils.ParseInt32(q.Get(queryYear)),
		MinPrice:    utils.ParseInt64(q.Get(queryMinPrice)),
		MaxPrice:    utils.ParseInt64(q.Get(queryMaxPrice)),
		MaxMileage:  utils.ParseInt32(q.Get(queryMaxMileage)),
		FuelType:    q.Get(queryFuelTypeV2),
		Page:        utils.ParseInt32WithDefault(q.Get(queryPage), 1),
		PageSize:    utils.ParseInt32WithDefault(q.Get(queryPageSizeV2), 20),
		IncludeSold: includeSold,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func parseOptionalBoolWrapper(raw string) (*wrapperspb.BoolValue, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	value, err := strconv.ParseBool(trimmed)
	if err != nil {
		return nil, err
	}
	return wrapperspb.Bool(value), nil
}

func HandleCompare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, msgMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query()
	idStrs := strings.Split(q.Get(queryIDs), ",")
	var ids []int64
	for _, s := range idStrs {
		if s == "" {
			continue
		}
		ids = append(ids, utils.ParseInt64(s))
	}
	ctx := r.Context()
	resp, err := listingClient.CompareListings(ctx, &listingpb.CompareListingsRequest{Ids: ids})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func HandleGetListing(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 || parts[3] == "" {
		http.Error(w, "Missing listing id", http.StatusBadRequest)
		return
	}
	id := utils.ParseInt64(parts[3])
	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		resp, err := listingClient.GetListingDetails(ctx, &listingpb.ListingDetailsRequest{Id: id})
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, resp)
	case http.MethodPut:
		authUserID, err := authenticatedUserIDFromRequest(r)
		if err != nil {
			http.Error(w, msgUnauthorized, http.StatusUnauthorized)
			return
		}

		var req listingpb.UpdateListingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, msgInvalidBody, http.StatusBadRequest)
			return
		}
		req.Id = id
		req.DealerId = authUserID

		resp, err := listingClient.UpdateListing(ctx, &req)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, resp)
	case http.MethodDelete:
		authUserID, err := authenticatedUserIDFromRequest(r)
		if err != nil {
			http.Error(w, msgUnauthorized, http.StatusUnauthorized)
			return
		}

		resp, err := listingClient.DeleteListing(ctx, &listingpb.DeleteListingRequest{Id: id, DealerId: authUserID})
		if err != nil {
			writeServiceError(w, err)
			return
		}
		if !resp.GetDeleted() {
			http.Error(w, msgNotFound, http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, msgMethodNotAllowed, http.StatusMethodNotAllowed)
	}
}
