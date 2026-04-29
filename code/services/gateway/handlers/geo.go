package handlers

import (
	"context"
	"net/http"
	"strings"

	geopb "services/geographic-market-insights/proto"
	"services/utils"
)

func HandleMarketAggregates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, msgMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query()
	ctx := context.Background()

	var metrics []geopb.MetricType
	for _, m := range q[queryMetrics] {
		switch strings.ToLower(m) {
		case metricAvgPrice:
			metrics = append(metrics, geopb.MetricType_METRIC_TYPE_AVG_PRICE)
		case metricMedianPrice:
			metrics = append(metrics, geopb.MetricType_METRIC_TYPE_MEDIAN_PRICE)
		case metricCount:
			metrics = append(metrics, geopb.MetricType_METRIC_TYPE_COUNT)
		}
	}

	var groupBy geopb.GroupBy
	switch strings.ToLower(q.Get(queryGroupBy)) {
	case groupByDistrict:
		groupBy = geopb.GroupBy_GROUP_BY_DISTRICT
	case groupByCity:
		groupBy = geopb.GroupBy_GROUP_BY_CITY
	case groupByCountry:
		groupBy = geopb.GroupBy_GROUP_BY_COUNTRY
	default:
		groupBy = geopb.GroupBy_GROUP_BY_DISTRICT
	}

	var locations *geopb.Locations
	if locs := q[queryLocations]; len(locs) > 0 {
		locations = &geopb.Locations{Location: locs}
	}

	var yearFrom, yearTo, limit, skip *int32
	if s := q.Get(queryYearFrom); s != "" {
		v := utils.ParseInt32(s)
		yearFrom = &v
	}
	if s := q.Get(queryYearTo); s != "" {
		v := utils.ParseInt32(s)
		yearTo = &v
	}
	if s := q.Get(queryLimit); s != "" {
		v := utils.ParseInt32(s)
		limit = &v
	}
	if s := q.Get(querySkip); s != "" {
		v := utils.ParseInt32(s)
		skip = &v
	}

	var fuelType *string
	if s := q.Get(queryFuelType); s != "" {
		fuelType = &s
	}

	req := &geopb.AggregatesRequest{
		Brand:     q.Get(queryBrand),
		Model:     q.Get(queryModel),
		Metrics:   metrics,
		GroupBy:   groupBy,
		Locations: locations,
		YearFrom:  yearFrom,
		YearTo:    yearTo,
		FuelType:  fuelType,
		Limit:     limit,
		Skip:      skip,
	}
	resp, err := geoClient.Aggregates(ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func HandleMarketPriceComparison(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, msgMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query()
	ctx := context.Background()

	var groupBy geopb.GroupBy
	switch strings.ToLower(q.Get(queryGroupBy)) {
	case groupByDistrict:
		groupBy = geopb.GroupBy_GROUP_BY_DISTRICT
	case groupByCity:
		groupBy = geopb.GroupBy_GROUP_BY_CITY
	case groupByCountry:
		groupBy = geopb.GroupBy_GROUP_BY_COUNTRY
	default:
		groupBy = geopb.GroupBy_GROUP_BY_DISTRICT
	}

	var sortBy *geopb.SortBy
	switch strings.ToLower(q.Get(querySortBy)) {
	case metricAvgPrice:
		v := geopb.SortBy_SORT_BY_AVG_PRICE
		sortBy = &v
	case metricCount:
		v := geopb.SortBy_SORT_BY_COUNT
		sortBy = &v
	}

	var order *geopb.Order
	switch strings.ToLower(q.Get(queryOrder)) {
	case orderAsc:
		v := geopb.Order_ORDER_ASC
		order = &v
	case orderDesc:
		v := geopb.Order_ORDER_DESC
		order = &v
	}

	var limit, skip *int32
	if s := q.Get(queryLimit); s != "" {
		v := utils.ParseInt32(s)
		limit = &v
	}
	if s := q.Get(querySkip); s != "" {
		v := utils.ParseInt32(s)
		skip = &v
	}
	req := &geopb.PriceComparisonRequest{
		Brand:   q.Get(queryBrand),
		Model:   q.Get(queryModel),
		GroupBy: groupBy,
		SortBy:  sortBy,
		Order:   order,
		Limit:   limit,
		Skip:    skip,
	}
	resp, err := geoClient.PriceComparison(ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func HandleStatsByLocation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, msgMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query()
	ctx := context.Background()
	var location *string
	if s := q.Get(queryLocation); s != "" {
		location = &s
	}
	var yearFrom, yearTo *int32
	if s := q.Get(queryYearFrom); s != "" {
		v := utils.ParseInt32(s)
		yearFrom = &v
	}
	if s := q.Get(queryYearTo); s != "" {
		v := utils.ParseInt32(s)
		yearTo = &v
	}
	var fuelType *string
	if s := q.Get(queryFuelType); s != "" {
		fuelType = &s
	}
	req := &geopb.ByLocationRequest{
		Brand:    q.Get(queryBrand),
		Model:    q.Get(queryModel),
		Location: location,
		YearFrom: yearFrom,
		YearTo:   yearTo,
		FuelType: fuelType,
	}
	resp, err := geoClient.ByLocation(ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
