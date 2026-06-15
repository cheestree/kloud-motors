package marketprice

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/schema"
)

func TestBindAndValidateAveragePriceQueryParsesTypedFilters(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/market/average-price?brand=Honda&model=Civic&year_from=2018&year_to=2023", nil)
	query := AveragePriceQuery{}

	if err := BindAndValidateQuery(req, &query); err != nil {
		t.Fatalf("BindAndValidateQuery returned error: %v", err)
	}

	if query.Brand != "Honda" || query.Model != "Civic" {
		t.Fatalf("query brand/model = %q/%q, want Honda/Civic", query.Brand, query.Model)
	}
	if query.YearFrom == nil || *query.YearFrom != 2018 {
		t.Fatalf("YearFrom = %v, want pointer to 2018", query.YearFrom)
	}
	if query.YearTo == nil || *query.YearTo != 2023 {
		t.Fatalf("YearTo = %v, want pointer to 2023", query.YearTo)
	}

	protoReq := BuildAveragePriceRequest(query)
	if protoReq.Brand != "Honda" || protoReq.Model != "Civic" || protoReq.YearFrom != 2018 || protoReq.YearTo != 2023 {
		t.Fatalf("BuildAveragePriceRequest() = %+v, want Honda Civic 2018-2023", protoReq)
	}
}

func TestBindAndValidateAveragePriceQueryRequiresBrandAndModel(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/market/average-price?brand=Honda", nil)
	query := AveragePriceQuery{}

	err := BindAndValidateQuery(req, &query)
	if err == nil {
		t.Fatal("BindAndValidateQuery returned nil, want validation error")
	}

	var validationErrs validator.ValidationErrors
	if !errors.As(err, &validationErrs) {
		t.Fatalf("error = %T, want validator.ValidationErrors", err)
	}

	got := map[string]string{}
	for _, validationErr := range validationErrs {
		got[validationErr.Field()] = validationErr.Tag()
	}
	if got["model"] != "notblank" {
		t.Fatalf("model validation tag = %q, want notblank; all errors: %v", got["model"], got)
	}
}

func TestBindAndValidateAveragePriceQueryRejectsBadYearType(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/market/average-price?brand=Honda&model=Civic&year_from=recent", nil)
	query := AveragePriceQuery{}

	err := BindAndValidateQuery(req, &query)
	if err == nil {
		t.Fatal("BindAndValidateQuery returned nil, want schema conversion error")
	}

	var schemaErrs schema.MultiError
	if !errors.As(err, &schemaErrs) {
		t.Fatalf("error = %T, want schema.MultiError", err)
	}
	if _, ok := schemaErrs["year_from"]; !ok {
		t.Fatalf("schema errors = %v, want year_from conversion error", schemaErrs)
	}
}

func TestBindAndValidateAveragePriceQueryAllowsZeroYearFrom(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/market/average-price?brand=Honda&model=Civic&year_from=0&year_to=2024", nil)
	query := AveragePriceQuery{}

	if err := BindAndValidateQuery(req, &query); err != nil {
		t.Fatalf("BindAndValidateQuery returned error: %v", err)
	}
	if query.YearFrom == nil || *query.YearFrom != 0 {
		t.Fatalf("YearFrom = %v, want pointer to 0", query.YearFrom)
	}
}

func TestBindAndValidateAveragePriceQueryRejectsNegativeYear(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/market/average-price?brand=Honda&model=Civic&year_from=-1", nil)
	query := AveragePriceQuery{}

	err := BindAndValidateQuery(req, &query)
	if err == nil {
		t.Fatal("BindAndValidateQuery returned nil, want validation error")
	}

	var validationErrs validator.ValidationErrors
	if !errors.As(err, &validationErrs) {
		t.Fatalf("error = %T, want validator.ValidationErrors", err)
	}
	if validationErrs[0].Field() != "year_from" || validationErrs[0].Tag() != "gte" {
		t.Fatalf("validation error = field %q tag %q, want year_from gte", validationErrs[0].Field(), validationErrs[0].Tag())
	}
}

func TestBindAndValidateAveragePriceQueryRejectsInvertedYearRange(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/market/average-price?brand=Honda&model=Civic&year_from=2024&year_to=2018", nil)
	query := AveragePriceQuery{}

	err := BindAndValidateQuery(req, &query)
	if err == nil {
		t.Fatal("BindAndValidateQuery returned nil, want validation error")
	}

	var validationErrs validator.ValidationErrors
	if !errors.As(err, &validationErrs) {
		t.Fatalf("error = %T, want validator.ValidationErrors", err)
	}
	if validationErrs[0].Field() != "year_to" || validationErrs[0].Tag() != "gtefield" {
		t.Fatalf("validation error = field %q tag %q, want year_to gtefield", validationErrs[0].Field(), validationErrs[0].Tag())
	}
}
