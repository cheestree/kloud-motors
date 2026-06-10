package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strconv"
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

type ListingListQuery struct {
	Page        int32 `schema:"page" validate:"gte=1"`
	PageSize    int32 `schema:"pageSize" validate:"gte=1,lte=100"`
	IncludeSold *bool `schema:"includeSold" validate:"omitempty"`
}

type ListingMutationBody struct {
	Vin          string  `json:"vin" validate:"notblank"`
	Make         string  `json:"make" validate:"notblank"`
	Model        string  `json:"model" validate:"notblank"`
	Year         int32   `json:"year" validate:"gte=1886,lte=2100"`
	Price        float64 `json:"price" validate:"gte=0"`
	Mileage      int32   `json:"mileage" validate:"gte=0"`
	City         string  `json:"city" validate:"omitempty"`
	District     string  `json:"district" validate:"omitempty"`
	State        string  `json:"state" validate:"omitempty"`
	Country      string  `json:"country" validate:"omitempty"`
	FuelType     string  `json:"fuel_type" validate:"omitempty"`
	BodyClass    string  `json:"body_class" validate:"omitempty"`
	DriveType    string  `json:"drive_type" validate:"omitempty"`
	Transmission string  `json:"transmission" validate:"omitempty"`
	Trim         string  `json:"trim" validate:"omitempty"`
	Color        string  `json:"color" validate:"omitempty"`
	IsNew        bool    `json:"is_new" validate:"omitempty"`
	IsSold       bool    `json:"is_sold" validate:"omitempty"`
}

type ListingIDPath struct {
	ID int64 `schema:"id" validate:"gt=0"`
}

type ListingCompareQuery struct {
	IDs []int64 `schema:"ids" validate:"min=1,dive,gt=0"`
}

var errMissingListingID = errors.New("missing listing id")

var (
	queryDecoder   = schema.NewDecoder()
	queryValidator = validator.New()
)

func init() {
	queryDecoder.IgnoreUnknownKeys(true)
	queryValidator.RegisterValidation("notblank", func(field validator.FieldLevel) bool {
		return strings.TrimSpace(field.Field().String()) != ""
	})
	queryValidator.RegisterTagNameFunc(func(field reflect.StructField) string {
		for _, tag := range []string{"schema", "json"} {
			name := strings.SplitN(field.Tag.Get(tag), ",", 2)[0]
			if name == "-" {
				return ""
			}
			if name != "" {
				return name
			}
		}
		return ""
	})
}

func HandleListings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	switch r.Method {
	case http.MethodGet:
		query := ListingListQuery{
			Page:     1,
			PageSize: 20,
		}
		if err := bindAndValidateQuery(r, &query); err != nil {
			writeRequestError(w, "Invalid query parameters", err)
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

		var body ListingMutationBody
		if err := bindAndValidateJSON(r, &body); err != nil {
			writeRequestError(w, msgInvalidBody, err)
			return
		}
		req := body.CreateListingRequest()
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

func bindAndValidateJSON(r *http.Request, target interface{}) error {
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		return err
	}
	return queryValidator.Struct(target)
}

func (q ListingListQuery) SearchRequest() *searchpb.SearchRequest {
	return &searchpb.SearchRequest{
		Page:        q.Page,
		PageSize:    q.PageSize,
		IncludeSold: utils.BoolPtrToProtoBoolValue(q.IncludeSold),
	}
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

func (b ListingMutationBody) CreateListingRequest() listingpb.CreateListingRequest {
	return listingpb.CreateListingRequest{
		Vin:          b.Vin,
		Make:         b.Make,
		Model:        b.Model,
		Year:         b.Year,
		Price:        b.Price,
		Mileage:      b.Mileage,
		City:         b.City,
		District:     b.District,
		State:        b.State,
		Country:      b.Country,
		FuelType:     b.FuelType,
		BodyClass:    b.BodyClass,
		DriveType:    b.DriveType,
		Transmission: b.Transmission,
		Trim:         b.Trim,
		Color:        b.Color,
		IsNew:        b.IsNew,
		IsSold:       b.IsSold,
	}
}

func (b ListingMutationBody) UpdateListingRequest() listingpb.UpdateListingRequest {
	return listingpb.UpdateListingRequest{
		Vin:          b.Vin,
		Make:         b.Make,
		Model:        b.Model,
		Year:         b.Year,
		Price:        b.Price,
		Mileage:      b.Mileage,
		City:         b.City,
		District:     b.District,
		State:        b.State,
		Country:      b.Country,
		FuelType:     b.FuelType,
		BodyClass:    b.BodyClass,
		DriveType:    b.DriveType,
		Transmission: b.Transmission,
		Trim:         b.Trim,
		Color:        b.Color,
		IsNew:        b.IsNew,
	}
}

func HandleCompare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, msgMethodNotAllowed, nil)
		return
	}
	ids, err := parseCommaSeparatedInt64s(r.URL.Query().Get(queryIDs))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid query parameters", []fieldError{{
			Field:   queryIDs,
			Message: "must be a comma-separated list of positive integers",
		}})
		return
	}
	query := ListingCompareQuery{IDs: ids}
	if err := queryValidator.Struct(query); err != nil {
		writeRequestError(w, "Invalid query parameters", err)
		return
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
	id, err := listingIDFromPath(r)
	if err != nil {
		if errors.Is(err, errMissingListingID) {
			writeError(w, http.StatusBadRequest, "Missing listing id", nil)
			return
		}
		var validationErrs validator.ValidationErrors
		if errors.As(err, &validationErrs) {
			writeRequestError(w, "Invalid path parameters", err)
			return
		}
		writeError(w, http.StatusBadRequest, "Invalid path parameters", []fieldError{{
			Field:   "id",
			Message: "must be a positive integer",
		}})
		return
	}
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

		var body ListingMutationBody
		if err := bindAndValidateJSON(r, &body); err != nil {
			writeRequestError(w, msgInvalidBody, err)
			return
		}
		req := body.UpdateListingRequest()
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

func listingIDFromPath(r *http.Request) (int64, error) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 || parts[3] == "" {
		return 0, errMissingListingID
	}

	id, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		return 0, err
	}

	path := ListingIDPath{ID: id}
	if err := queryValidator.Struct(path); err != nil {
		return 0, err
	}
	return id, nil
}

func parseCommaSeparatedInt64s(raw string) ([]int64, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	idStrs := strings.Split(raw, ",")
	ids := make([]int64, 0, len(idStrs))
	for _, s := range idStrs {
		trimmed := strings.TrimSpace(s)
		if trimmed == "" {
			return nil, strconv.ErrSyntax
		}
		id, err := strconv.ParseInt(trimmed, 10, 64)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}
