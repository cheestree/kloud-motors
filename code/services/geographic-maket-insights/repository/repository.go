package repository

import "context"

type QueryConfig struct {
	Schema       string
	Table        string
	DefaultLimit int
	MaxLimit     int
}

type Filters struct {
	Brand    string
	Model    string
	YearFrom *int32
	YearTo   *int32
	FuelType *string
}

type AggregateRow struct {
	Location    string
	AvgPrice    int32
	MedianPrice int32
	Count       int32
}

type ComparisonRow struct {
	Location     string
	AveragePrice int32
	ListingCount int32
}

type StatsRow struct {
	MinPrice    int32
	MaxPrice    int32
	AvgPrice    int32
	MedianPrice int32
}

type InsightsRepo interface {
	NormalizePage(limitRaw, skipRaw int32) (int, int)
	FetchAggregates(ctx context.Context, filters Filters, groupCol string, locations []string, limit, skip int) ([]AggregateRow, bool, error)
	FetchPriceComparison(ctx context.Context, filters Filters, groupCol, sortCol, order string, limit, skip int) ([]ComparisonRow, bool, error)
	FetchByLocation(ctx context.Context, filters Filters, location *string) (StatsRow, error)
}
