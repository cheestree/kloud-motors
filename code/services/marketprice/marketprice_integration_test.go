//go:build integration

package main

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"services/internal/integrationtest"
	marketpricepb "services/marketprice/proto"
	"services/utils"

	_ "github.com/lib/pq"
)

func TestMarketPriceIntegration_GetAverageMarketPrice(t *testing.T) {
	dbURL := utils.GetEnv("MARKETPRICE_TEST_DB_URL", "postgres://listing_user:listing_password@localhost:15432/listing_db?sslmode=disable")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	brand, model, err := fetchMarketPriceFixture(ctx, dbURL)
	if err != nil {
		t.Fatalf("failed to fetch market price fixture from db: %v", err)
	}

	addr := utils.GetEnv("MARKETPRICE_TEST_ADDR", "localhost:15055")
	conn := integrationtest.DialGRPC(ctx, t, "marketprice", addr)
	client := marketpricepb.NewMarketPriceServiceClient(conn)

	resp, err := client.GetAverageMarketPrice(ctx, &marketpricepb.AveragePriceRequest{
		Brand: brand,
		Model: model,
	})
	if err != nil {
		t.Fatalf("get average market price failed: %v", err)
	}
	if resp.ListingCount <= 0 {
		t.Fatalf("expected listing count > 0, got %d", resp.ListingCount)
	}
	if resp.AveragePrice <= 0 {
		t.Fatalf("expected average price > 0, got %f", resp.AveragePrice)
	}
}

func fetchMarketPriceFixture(ctx context.Context, dbURL string) (string, string, error) {
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return "", "", err
	}
	defer db.Close()

	var brand, model string
	row := db.QueryRowContext(ctx, `
		SELECT b.name, m.name
		FROM automotive_data ad
		JOIN brand b ON ad.brand_id = b.id
		JOIN model m ON ad.model_id = m.id
		WHERE ad.ask_price > 0
		ORDER BY ad.id ASC
		LIMIT 1`)
	if err := row.Scan(&brand, &model); err != nil {
		return "", "", err
	}
	return brand, model, nil
}
