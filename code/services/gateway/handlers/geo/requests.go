package geo

import geopb "services/geographic-market-insights/proto"

const (
	groupByDistrict = "district"
	groupByCity     = "city"
	groupByCountry  = "country"

	metricAvgPrice    = "avg_price"
	metricMedianPrice = "median_price"
	metricCount       = "count"

	orderAsc  = "asc"
	orderDesc = "desc"
)

type AggregatesQuery struct {
	Brand     string   `schema:"brand" validate:"notblank"`
	Model     string   `schema:"model" validate:"notblank"`
	Metrics   []string `schema:"metrics" validate:"omitempty,dive,oneof=avg_price median_price count"`
	GroupBy   string   `schema:"group_by" validate:"omitempty,oneof=district city country"`
	Locations []string `schema:"locations" validate:"omitempty,dive,notblank"`
	YearFrom  *int32   `schema:"year_from" validate:"omitempty,gte=1886,lte=2100"`
	YearTo    *int32   `schema:"year_to" validate:"omitempty,gte=1886,lte=2100"`
	FuelType  string   `schema:"fuel_type" validate:"omitempty"`
	Limit     *int32   `schema:"limit" validate:"omitempty,gte=1"`
	Skip      *int32   `schema:"skip" validate:"omitempty,gte=0"`
}

type PriceComparisonQuery struct {
	Brand   string `schema:"brand" validate:"notblank"`
	Model   string `schema:"model" validate:"notblank"`
	GroupBy string `schema:"group_by" validate:"omitempty,oneof=district city country"`
	SortBy  string `schema:"sort_by" validate:"omitempty,oneof=avg_price count"`
	Order   string `schema:"order" validate:"omitempty,oneof=asc desc"`
	Limit   *int32 `schema:"limit" validate:"omitempty,gte=1"`
	Skip    *int32 `schema:"skip" validate:"omitempty,gte=0"`
}

type ByLocationQuery struct {
	Brand    string `schema:"brand" validate:"notblank"`
	Model    string `schema:"model" validate:"notblank"`
	Location string `schema:"location" validate:"omitempty"`
	YearFrom *int32 `schema:"year_from" validate:"omitempty,gte=1886,lte=2100"`
	YearTo   *int32 `schema:"year_to" validate:"omitempty,gte=1886,lte=2100"`
	FuelType string `schema:"fuel_type" validate:"omitempty"`
}

func BuildAggregatesRequest(query AggregatesQuery) *geopb.AggregatesRequest {
	return &geopb.AggregatesRequest{
		Brand:     query.Brand,
		Model:     query.Model,
		Metrics:   buildMetricTypes(query.Metrics),
		GroupBy:   buildGroupBy(query.GroupBy),
		Locations: buildLocations(query.Locations),
		YearFrom:  query.YearFrom,
		YearTo:    query.YearTo,
		FuelType:  stringPtrOrNil(query.FuelType),
		Limit:     query.Limit,
		Skip:      query.Skip,
	}
}

func BuildPriceComparisonRequest(query PriceComparisonQuery) *geopb.PriceComparisonRequest {
	return &geopb.PriceComparisonRequest{
		Brand:   query.Brand,
		Model:   query.Model,
		GroupBy: buildGroupBy(query.GroupBy),
		SortBy:  buildSortBy(query.SortBy),
		Order:   buildOrder(query.Order),
		Limit:   query.Limit,
		Skip:    query.Skip,
	}
}

func BuildByLocationRequest(query ByLocationQuery) *geopb.ByLocationRequest {
	return &geopb.ByLocationRequest{
		Brand:    query.Brand,
		Model:    query.Model,
		Location: stringPtrOrNil(query.Location),
		YearFrom: query.YearFrom,
		YearTo:   query.YearTo,
		FuelType: stringPtrOrNil(query.FuelType),
	}
}

func buildMetricTypes(metrics []string) []geopb.MetricType {
	result := make([]geopb.MetricType, 0, len(metrics))
	for _, metric := range metrics {
		switch metric {
		case metricAvgPrice:
			result = append(result, geopb.MetricType_METRIC_TYPE_AVG_PRICE)
		case metricMedianPrice:
			result = append(result, geopb.MetricType_METRIC_TYPE_MEDIAN_PRICE)
		case metricCount:
			result = append(result, geopb.MetricType_METRIC_TYPE_COUNT)
		}
	}
	return result
}

func buildGroupBy(groupBy string) geopb.GroupBy {
	switch groupBy {
	case groupByCity:
		return geopb.GroupBy_GROUP_BY_CITY
	case groupByCountry:
		return geopb.GroupBy_GROUP_BY_COUNTRY
	default:
		return geopb.GroupBy_GROUP_BY_DISTRICT
	}
}

func buildSortBy(sortBy string) *geopb.SortBy {
	if sortBy == "" {
		return nil
	}

	v := geopb.SortBy_SORT_BY_AVG_PRICE
	if sortBy == metricCount {
		v = geopb.SortBy_SORT_BY_COUNT
	}
	return &v
}

func buildOrder(order string) *geopb.Order {
	if order == "" {
		return nil
	}

	v := geopb.Order_ORDER_ASC
	if order == orderDesc {
		v = geopb.Order_ORDER_DESC
	}
	return &v
}

func buildLocations(locations []string) *geopb.Locations {
	if len(locations) == 0 {
		return nil
	}
	return &geopb.Locations{Location: locations}
}

func stringPtrOrNil(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
