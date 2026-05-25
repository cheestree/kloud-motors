//go:build integration

package main

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"services/internal/integrationtest"
	listingpb "services/listing/proto"
	"services/utils"

	_ "github.com/lib/pq"
)

func TestListingIntegration_GetSummary(t *testing.T) {
	dbURL := utils.GetEnv("LISTING_TEST_DB_URL", "postgres://listing_user:listing_password@localhost:15432/listing_db?sslmode=disable")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	listingID, err := fetchAnyListingID(ctx, dbURL)
	if err != nil {
		t.Fatalf("failed to fetch listing id from db: %v", err)
	}

	addr := utils.GetEnv("LISTING_TEST_ADDR", "localhost:15054")
	conn := integrationtest.DialGRPC(ctx, t, "listing", addr)
	client := listingpb.NewListingServiceClient(conn)
	summary, err := client.GetListingSummary(ctx, &listingpb.ListingDetailsRequest{Id: listingID})
	if err != nil {
		t.Fatalf("get listing summary failed: %v", err)
	}
	if summary.Id != listingID {
		t.Fatalf("expected listing id %d, got %d", listingID, summary.Id)
	}
	if summary.Make == "" {
		t.Fatalf("expected listing make to be set")
	}
}

func fetchAnyListingID(ctx context.Context, dbURL string) (int64, error) {
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	var id int64
	row := db.QueryRowContext(ctx, "SELECT id FROM automotive_data ORDER BY id ASC LIMIT 1")
	if err := row.Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}
