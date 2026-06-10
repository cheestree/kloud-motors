package geo

import (
	"errors"
	"net/http/httptest"
	"testing"

	geopb "services/geographic-market-insights/proto"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/schema"
)

func TestBindAndValidateAggregatesQueryParsesAndBuildsRequest(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/market/insights/aggregates?brand=Toyota&model=Corolla&metrics=avg_price&metrics=count&group_by=city&locations=Lisbon&locations=Porto&year_from=2018&year_to=2024&fuel_type=Gasoline&limit=25&skip=5", nil)
	query := AggregatesQuery{}

	if err := BindAndValidateQuery(req, &query); err != nil {
		t.Fatalf("BindAndValidateQuery returned error: %v", err)
	}

	protoReq := BuildAggregatesRequest(query)
	if protoReq.Brand != "Toyota" || protoReq.Model != "Corolla" {
		t.Fatalf("brand/model = %q/%q, want Toyota/Corolla", protoReq.Brand, protoReq.Model)
	}
	if protoReq.GroupBy != geopb.GroupBy_GROUP_BY_CITY {
		t.Fatalf("GroupBy = %v, want city", protoReq.GroupBy)
	}
	if len(protoReq.Metrics) != 2 || protoReq.Metrics[0] != geopb.MetricType_METRIC_TYPE_AVG_PRICE || protoReq.Metrics[1] != geopb.MetricType_METRIC_TYPE_COUNT {
		t.Fatalf("Metrics = %v, want avg_price and count", protoReq.Metrics)
	}
	if protoReq.Locations == nil || len(protoReq.Locations.Location) != 2 || protoReq.Locations.Location[1] != "Porto" {
		t.Fatalf("Locations = %+v, want Lisbon and Porto", protoReq.Locations)
	}
	if protoReq.YearFrom == nil || *protoReq.YearFrom != 2018 {
		t.Fatalf("YearFrom = %v, want pointer to 2018", protoReq.YearFrom)
	}
	if protoReq.YearTo == nil || *protoReq.YearTo != 2024 {
		t.Fatalf("YearTo = %v, want pointer to 2024", protoReq.YearTo)
	}
	if protoReq.FuelType == nil || *protoReq.FuelType != "Gasoline" {
		t.Fatalf("FuelType = %v, want pointer to Gasoline", protoReq.FuelType)
	}
	if protoReq.Limit == nil || *protoReq.Limit != 25 {
		t.Fatalf("Limit = %v, want pointer to 25", protoReq.Limit)
	}
	if protoReq.Skip == nil || *protoReq.Skip != 5 {
		t.Fatalf("Skip = %v, want pointer to 5", protoReq.Skip)
	}
}

func TestBuildAggregatesRequestDefaultsGroupByAndOmittedOptionals(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/market/insights/aggregates?brand=Toyota&model=Corolla", nil)
	query := AggregatesQuery{}

	if err := BindAndValidateQuery(req, &query); err != nil {
		t.Fatalf("BindAndValidateQuery returned error: %v", err)
	}

	protoReq := BuildAggregatesRequest(query)
	if protoReq.GroupBy != geopb.GroupBy_GROUP_BY_DISTRICT {
		t.Fatalf("GroupBy = %v, want district default", protoReq.GroupBy)
	}
	if protoReq.Locations != nil || protoReq.YearFrom != nil || protoReq.FuelType != nil || protoReq.Limit != nil || protoReq.Skip != nil {
		t.Fatalf("optional fields should be nil when omitted: %+v", protoReq)
	}
}

func TestBindAndValidateAggregatesQueryRequiresBrandAndModel(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/market/insights/aggregates?brand=Toyota", nil)
	query := AggregatesQuery{}

	err := BindAndValidateQuery(req, &query)
	if err == nil {
		t.Fatal("BindAndValidateQuery returned nil, want validation error")
	}

	var validationErrs validator.ValidationErrors
	if !errors.As(err, &validationErrs) {
		t.Fatalf("error = %T, want validator.ValidationErrors", err)
	}

	foundModel := false
	for _, validationErr := range validationErrs {
		if validationErr.Field() == "model" && validationErr.Tag() == "notblank" {
			foundModel = true
		}
	}
	if !foundModel {
		t.Fatalf("validation errors = %v, want model notblank", validationErrs)
	}
}

func TestBindAndValidateAggregatesQueryRejectsInvalidEnum(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/market/insights/aggregates?brand=Toyota&model=Corolla&metrics=bogus&group_by=neighborhood", nil)
	query := AggregatesQuery{}

	err := BindAndValidateQuery(req, &query)
	if err == nil {
		t.Fatal("BindAndValidateQuery returned nil, want validation error")
	}

	var validationErrs validator.ValidationErrors
	if !errors.As(err, &validationErrs) {
		t.Fatalf("error = %T, want validator.ValidationErrors", err)
	}
	if len(validationErrs) == 0 {
		t.Fatal("validation errors were empty")
	}
}

func TestBindAndValidateAggregatesQueryRejectsBadIntegerType(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/market/insights/aggregates?brand=Toyota&model=Corolla&limit=many", nil)
	query := AggregatesQuery{}

	err := BindAndValidateQuery(req, &query)
	if err == nil {
		t.Fatal("BindAndValidateQuery returned nil, want schema conversion error")
	}

	var schemaErrs schema.MultiError
	if !errors.As(err, &schemaErrs) {
		t.Fatalf("error = %T, want schema.MultiError", err)
	}
	if _, ok := schemaErrs["limit"]; !ok {
		t.Fatalf("schema errors = %v, want limit conversion error", schemaErrs)
	}
}

func TestBindAndValidatePriceComparisonQueryParsesAndBuildsRequest(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/market/price-comparison?brand=Toyota&model=Corolla&group_by=country&sort_by=count&order=desc&limit=10&skip=20", nil)
	query := PriceComparisonQuery{}

	if err := BindAndValidateQuery(req, &query); err != nil {
		t.Fatalf("BindAndValidateQuery returned error: %v", err)
	}

	protoReq := BuildPriceComparisonRequest(query)
	if protoReq.GroupBy != geopb.GroupBy_GROUP_BY_COUNTRY {
		t.Fatalf("GroupBy = %v, want country", protoReq.GroupBy)
	}
	if protoReq.SortBy == nil || *protoReq.SortBy != geopb.SortBy_SORT_BY_COUNT {
		t.Fatalf("SortBy = %v, want count", protoReq.SortBy)
	}
	if protoReq.Order == nil || *protoReq.Order != geopb.Order_ORDER_DESC {
		t.Fatalf("Order = %v, want desc", protoReq.Order)
	}
	if protoReq.Limit == nil || *protoReq.Limit != 10 {
		t.Fatalf("Limit = %v, want pointer to 10", protoReq.Limit)
	}
	if protoReq.Skip == nil || *protoReq.Skip != 20 {
		t.Fatalf("Skip = %v, want pointer to 20", protoReq.Skip)
	}
}

func TestBindAndValidatePriceComparisonQueryRejectsInvalidSortAndOrder(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/market/price-comparison?brand=Toyota&model=Corolla&sort_by=median_price&order=sideways", nil)
	query := PriceComparisonQuery{}

	err := BindAndValidateQuery(req, &query)
	if err == nil {
		t.Fatal("BindAndValidateQuery returned nil, want validation error")
	}

	var validationErrs validator.ValidationErrors
	if !errors.As(err, &validationErrs) {
		t.Fatalf("error = %T, want validator.ValidationErrors", err)
	}
	if len(validationErrs) != 2 {
		t.Fatalf("validation errors = %v, want sort_by and order errors", validationErrs)
	}
}

func TestBindAndValidateByLocationQueryParsesAndBuildsRequest(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/listings/stats/by-location?brand=Toyota&model=Corolla&location=Lisbon&year_from=2018&year_to=2024&fuel_type=Hybrid", nil)
	query := ByLocationQuery{}

	if err := BindAndValidateQuery(req, &query); err != nil {
		t.Fatalf("BindAndValidateQuery returned error: %v", err)
	}

	protoReq := BuildByLocationRequest(query)
	if protoReq.Location == nil || *protoReq.Location != "Lisbon" {
		t.Fatalf("Location = %v, want pointer to Lisbon", protoReq.Location)
	}
	if protoReq.YearFrom == nil || *protoReq.YearFrom != 2018 {
		t.Fatalf("YearFrom = %v, want pointer to 2018", protoReq.YearFrom)
	}
	if protoReq.YearTo == nil || *protoReq.YearTo != 2024 {
		t.Fatalf("YearTo = %v, want pointer to 2024", protoReq.YearTo)
	}
	if protoReq.FuelType == nil || *protoReq.FuelType != "Hybrid" {
		t.Fatalf("FuelType = %v, want pointer to Hybrid", protoReq.FuelType)
	}
}
