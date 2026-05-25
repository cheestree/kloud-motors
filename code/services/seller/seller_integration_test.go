//go:build integration

package main

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"services/internal/integrationtest"
	sellerpb "services/seller/proto"
	"services/utils"

	_ "github.com/lib/pq"
)

func TestSellerIntegration_GetAndVerifyProfile(t *testing.T) {
	dbURL := utils.GetEnv("SELLER_TEST_DB_URL", "postgres://seller_user:seller_password@localhost:15437/seller_db?sslmode=disable")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sellerID, err := fetchAnySellerID(ctx, dbURL)
	if err != nil {
		t.Fatalf("failed to fetch seller id from db: %v", err)
	}

	addr := utils.GetEnv("SELLER_TEST_ADDR", "localhost:15057")
	conn := integrationtest.DialGRPC(ctx, t, "seller", addr)
	client := sellerpb.NewSellerServiceClient(conn)

	profile, err := client.GetSellerProfile(ctx, &sellerpb.GetSellerProfileRequest{SellerId: sellerID})
	if err != nil {
		t.Fatalf("get seller profile failed: %v", err)
	}
	if profile.SellerId != sellerID {
		t.Fatalf("expected seller id %d, got %d", sellerID, profile.SellerId)
	}
	if profile.Name == "" {
		t.Fatalf("expected seller name to be set")
	}

	verify, err := client.VerifySellerProfile(ctx, &sellerpb.VerifySellerRequest{SellerId: sellerID})
	if err != nil {
		t.Fatalf("verify seller profile failed: %v", err)
	}
	if !verify.IsSeller {
		t.Fatalf("expected seller %d to verify as seller", sellerID)
	}
}

func fetchAnySellerID(ctx context.Context, dbURL string) (int64, error) {
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	var id int64
	row := db.QueryRowContext(ctx, "SELECT id FROM sellers ORDER BY id ASC LIMIT 1")
	if err := row.Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}
