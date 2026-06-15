package main

import (
	"context"
	"testing"

	geopb "services/geographic-market-insights/proto"
	"services/geographic-market-insights/repository"
)

type fakeInsightsRepo struct {
	lastFilters  repository.Filters
	lastGroupCol string
	lastLocation *string
}

func (f *fakeInsightsRepo) NormalizePage(limitRaw, skipRaw int32) (int, int) {
	if limitRaw == 0 {
		limitRaw = 20
	}
	return int(limitRaw), int(skipRaw)
}

func (f *fakeInsightsRepo) FetchAggregates(ctx context.Context, filters repository.Filters, groupCol string, locations []string, limit, skip int) ([]repository.AggregateRow, bool, error) {
	f.lastFilters = filters
	f.lastGroupCol = groupCol
	return []repository.AggregateRow{{Location: "Porto", AvgPrice: 10000, MedianPrice: 9500, Count: 2}}, false, nil
}

func (f *fakeInsightsRepo) FetchPriceComparison(ctx context.Context, filters repository.Filters, groupCol, sortCol, order string, limit, skip int) ([]repository.ComparisonRow, bool, error) {
	return []repository.ComparisonRow{{Location: "Porto", AveragePrice: 10000, ListingCount: 2}}, false, nil
}

func (f *fakeInsightsRepo) FetchByLocation(ctx context.Context, filters repository.Filters, location *string) (repository.StatsRow, error) {
	f.lastFilters = filters
	f.lastLocation = location
	return repository.StatsRow{MinPrice: 9000, MaxPrice: 12000, AvgPrice: 10000, MedianPrice: 9500}, nil
}

func TestGeoServer_AggregatesRequiresBrandAndModel(t *testing.T) {
	srv := NewGeoServer(&fakeInsightsRepo{})

	_, err := srv.Aggregates(context.Background(), &geopb.AggregatesRequest{Brand: "Ford"})
	if err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestGeoServer_AggregatesMasksUnrequestedMetrics(t *testing.T) {
	repo := &fakeInsightsRepo{}
	srv := NewGeoServer(repo)
	limit := int32(5)

	resp, err := srv.Aggregates(context.Background(), &geopb.AggregatesRequest{
		Brand:   "Ford",
		Model:   "Fiesta",
		GroupBy: geopb.GroupBy_GROUP_BY_CITY,
		Metrics: []geopb.MetricType{geopb.MetricType_METRIC_TYPE_COUNT},
		Limit:   &limit,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.lastGroupCol != "city" {
		t.Fatalf("expected city grouping, got %q", repo.lastGroupCol)
	}
	if len(resp.Aggregates) != 1 {
		t.Fatalf("expected one aggregate, got %d", len(resp.Aggregates))
	}
	got := resp.Aggregates[0]
	if got.Count != 2 || got.AvgPrice != 0 || got.MedianPrice != 0 {
		t.Fatalf("expected only count metric to be populated, got %+v", got)
	}
}

func TestGeoServer_ByLocationPassesFilters(t *testing.T) {
	repo := &fakeInsightsRepo{}
	srv := NewGeoServer(repo)
	location := "Porto"

	resp, err := srv.ByLocation(context.Background(), &geopb.ByLocationRequest{
		Brand:    "Ford",
		Model:    "Fiesta",
		Location: &location,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Stats == nil || resp.Stats.MaxPrice != 12000 {
		t.Fatalf("unexpected stats: %+v", resp.Stats)
	}
	if repo.lastFilters.Brand != "Ford" || repo.lastFilters.Model != "Fiesta" {
		t.Fatalf("unexpected filters: %+v", repo.lastFilters)
	}
	if repo.lastLocation == nil || *repo.lastLocation != "Porto" {
		t.Fatalf("expected location Porto, got %v", repo.lastLocation)
	}
}
