//go:build integration

package main

import (
	"context"
	"database/sql"
	"testing"
	"time"

	geopb "services/geographic-market-insights/proto"
	"services/internal/integrationtest"
	"services/utils"

	_ "github.com/lib/pq"
)

func TestGeoIntegration_ByLocation(t *testing.T) {
	dbURL := utils.GetEnv("GEO_TEST_DB_URL", "postgres://listing_user:listing_password@localhost:15432/listing_db?sslmode=disable")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	brand, model, location, err := fetchGeoFixture(ctx, dbURL)
	if err != nil {
		t.Fatalf("failed to fetch geo fixture from db: %v", err)
	}

	addr := utils.GetEnv("GEO_TEST_ADDR", "localhost:15053")
	conn := integrationtest.DialGRPC(ctx, t, "geo", addr)
	client := geopb.NewGeoMarketInsightsServiceClient(conn)

	resp, err := client.ByLocation(ctx, &geopb.ByLocationRequest{
		Brand:    brand,
		Model:    model,
		Location: &location,
	})
	if err != nil {
		t.Fatalf("get geo stats by location failed: %v", err)
	}
	if resp.Stats == nil {
		t.Fatalf("expected stats in response")
	}
	if resp.Stats.MaxPrice <= 0 {
		t.Fatalf("expected max price > 0, got %d", resp.Stats.MaxPrice)
	}
}

func fetchGeoFixture(ctx context.Context, dbURL string) (string, string, string, error) {
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return "", "", "", err
	}
	defer db.Close()

	var brand, model, location string
	row := db.QueryRowContext(ctx, `
		SELECT b.name, m.name, COALESCE(ad.city, ad.district, ad.country, ad.state, '')
		FROM automotive_data ad
		JOIN brand b ON ad.brand_id = b.id
		JOIN model m ON ad.model_id = m.id
		WHERE ad.ask_price > 0
		  AND COALESCE(ad.city, ad.district, ad.country, ad.state, '') <> ''
		ORDER BY ad.id ASC
		LIMIT 1`)
	if err := row.Scan(&brand, &model, &location); err != nil {
		return "", "", "", err
	}
	return brand, model, location, nil
}
