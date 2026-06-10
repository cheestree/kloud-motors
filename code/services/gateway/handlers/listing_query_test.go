package handlers

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/schema"
)

func TestBindAndValidateListingListQueryKeepsDefaults(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/listings", nil)
	query := ListingListQuery{
		Page:     1,
		PageSize: 20,
	}

	if err := bindAndValidateQuery(req, &query); err != nil {
		t.Fatalf("bindAndValidateQuery returned error: %v", err)
	}

	if query.Page != 1 || query.PageSize != 20 || query.IncludeSold != nil {
		t.Fatalf("query = %+v, want defaults page=1 pageSize=20 includeSold=nil", query)
	}
}

func TestBindAndValidateListingSearchQueryParsesTypedFilters(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/listings/search?make=Honda&model=Civic&year=2020&minPrice=10000&maxPrice=20000&maxMileage=50000&fuelType=gasoline&page=2&pageSize=50&includeSold=true", nil)
	query := ListingSearchQuery{
		Page:     1,
		PageSize: 20,
	}

	if err := bindAndValidateQuery(req, &query); err != nil {
		t.Fatalf("bindAndValidateQuery returned error: %v", err)
	}

	if query.Make != "Honda" || query.Model != "Civic" || query.FuelType != "gasoline" {
		t.Fatalf("string filters were not decoded correctly: %+v", query)
	}
	if query.Year == nil || *query.Year != 2020 {
		t.Fatalf("Year = %v, want pointer to 2020", query.Year)
	}
	if query.MinPrice == nil || *query.MinPrice != 10000 {
		t.Fatalf("MinPrice = %v, want pointer to 10000", query.MinPrice)
	}
	if query.MaxPrice == nil || *query.MaxPrice != 20000 {
		t.Fatalf("MaxPrice = %v, want pointer to 20000", query.MaxPrice)
	}
	if query.MaxMileage == nil || *query.MaxMileage != 50000 {
		t.Fatalf("MaxMileage = %v, want pointer to 50000", query.MaxMileage)
	}
	if query.IncludeSold == nil || *query.IncludeSold != true {
		t.Fatalf("IncludeSold = %v, want pointer to true", query.IncludeSold)
	}
	if query.Page != 2 || query.PageSize != 50 {
		t.Fatalf("pagination = page %d pageSize %d, want page 2 pageSize 50", query.Page, query.PageSize)
	}
}

func TestBindAndValidateListingListQueryRejectsInvalidPagination(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/listings?page=0&pageSize=101", nil)
	query := ListingListQuery{
		Page:     1,
		PageSize: 20,
	}

	err := bindAndValidateQuery(req, &query)
	if err == nil {
		t.Fatal("bindAndValidateQuery returned nil, want validation error")
	}

	var validationErrs validator.ValidationErrors
	if !errors.As(err, &validationErrs) {
		t.Fatalf("error = %T, want validator.ValidationErrors", err)
	}

	got := map[string]string{}
	for _, validationErr := range validationErrs {
		got[validationErr.Field()] = validationErr.Tag()
	}

	if got["page"] != "gte" {
		t.Fatalf("page validation tag = %q, want gte; all errors: %v", got["page"], got)
	}
	if got["pageSize"] != "lte" {
		t.Fatalf("pageSize validation tag = %q, want lte; all errors: %v", got["pageSize"], got)
	}
}

func TestBindAndValidateListingSearchQueryRejectsBadTypes(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/listings/search?year=not-a-year&includeSold=maybe", nil)
	query := ListingSearchQuery{
		Page:     1,
		PageSize: 20,
	}

	err := bindAndValidateQuery(req, &query)
	if err == nil {
		t.Fatal("bindAndValidateQuery returned nil, want schema conversion error")
	}

	var schemaErrs schema.MultiError
	if !errors.As(err, &schemaErrs) {
		t.Fatalf("error = %T, want schema.MultiError", err)
	}
	if _, ok := schemaErrs["year"]; !ok {
		t.Fatalf("schema errors = %v, want year conversion error", schemaErrs)
	}
	if _, ok := schemaErrs["includeSold"]; !ok {
		t.Fatalf("schema errors = %v, want includeSold conversion error", schemaErrs)
	}
}
