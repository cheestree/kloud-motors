package handlers

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"

	listingpb "services/listing/proto"
	searchpb "services/search/proto"
	"services/utils"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/schema"
)

type ListingSearchQuery struct {
	Make        string `schema:"make" validate:"omitempty"`
	Model       string `schema:"model" validate:"omitempty"`
	Year        *int32 `schema:"year" validate:"omitempty,gte=1886"`
	MinPrice    *int64 `schema:"minPrice" validate:"omitempty,gte=0"`
	MaxPrice    *int64 `schema:"maxPrice" validate:"omitempty,gte=0"`
	MaxMileage  *int32 `schema:"maxMileage" validate:"omitempty,gte=0"`
	FuelType    string `schema:"fuelType" validate:"omitempty"`
	Page        int32  `schema:"page" validate:"gte=1"`
	PageSize    int32  `schema:"pageSize" validate:"gte=1,lte=100"`
	IncludeSold *bool  `schema:"includeSold" validate:"omitempty"`
}

var (
	queryDecoder   = schema.NewDecoder()
	queryValidator = validator.New()
)

func init() {
	queryDecoder.IgnoreUnknownKeys(true)
	queryValidator.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.SplitN(field.Tag.Get("schema"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
}

func HandleListings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	switch r.Method {
	case http.MethodGet:
		q := r.URL.Query()
		includeSold, err := utils.ParseOptionalBoolProtoBoolValue(q.Get(queryIncludeSold))
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid includeSold query parameter", nil)
			return
		}

		resp, err := searchClient.Search(ctx, &searchpb.SearchRequest{
			Page:        utils.ParseInt32OrDefaultIfEmpty(q.Get(queryPage), 1),
			PageSize:    utils.ParseInt32OrDefaultIfEmpty(q.Get(queryPageSizeV2), 20),
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
			writeError(w, http.StatusUnauthorized, msgUnauthorized, nil)
			return
		}

		var req listingpb.CreateListingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, msgInvalidBody, nil)
			return
		}
		req.SellerId = authUserID

		resp, err := listingClient.CreateListing(ctx, &req)
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
	query := ListingSearchQuery{
		Page:     1,
		PageSize: 20,
	}
	if err := bindAndValidateQuery(r, &query); err != nil {
		writeRequestError(w, "Invalid query parameters", err)
		return
	}

	resp, err := searchClient.Search(r.Context(), query.SearchRequest())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func bindAndValidateQuery(r *http.Request, target interface{}) error {
	if err := queryDecoder.Decode(target, r.URL.Query()); err != nil {
		return err
	}
	return queryValidator.Struct(target)
}

func (q ListingSearchQuery) SearchRequest() *searchpb.SearchRequest {
	return &searchpb.SearchRequest{
		Make:        q.Make,
		Model:       q.Model,
		Year:        utils.Int32ValueFromPtrOrZero(q.Year),
		MinPrice:    utils.Int64ValueFromPtrOrZero(q.MinPrice),
		MaxPrice:    utils.Int64ValueFromPtrOrZero(q.MaxPrice),
		MaxMileage:  utils.Int32ValueFromPtrOrZero(q.MaxMileage),
		FuelType:    q.FuelType,
		Page:        q.Page,
		PageSize:    q.PageSize,
		IncludeSold: utils.BoolPtrToProtoBoolValue(q.IncludeSold),
	}
}

func HandleCompare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, msgMethodNotAllowed, nil)
		return
	}
	q := r.URL.Query()
	idStrs := strings.Split(q.Get(queryIDs), ",")
	var ids []int64
	for _, s := range idStrs {
		if s == "" {
			continue
		}
		ids = append(ids, utils.ParseInt64OrZero(s))
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
		writeError(w, http.StatusBadRequest, "Missing listing id", nil)
		return
	}
	id := utils.ParseInt64OrZero(parts[3])
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
			writeError(w, http.StatusUnauthorized, msgUnauthorized, nil)
			return
		}

		var req listingpb.UpdateListingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, msgInvalidBody, nil)
			return
		}
		req.Id = id
		req.SellerId = authUserID

		resp, err := listingClient.UpdateListing(ctx, &req)
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

		resp, err := listingClient.DeleteListing(ctx, &listingpb.DeleteListingRequest{Id: id, SellerId: authUserID})
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
