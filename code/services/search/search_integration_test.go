//go:build integration

package main

import (
	"context"
	"testing"
	"time"

	"services/internal/integrationtest"
	searchpb "services/search/proto"
	"services/utils"

	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSearchIntegration_Basic(t *testing.T) {
	addr := utils.GetEnv("SEARCH_TEST_ADDR", "localhost:15056")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn := integrationtest.DialGRPC(ctx, t, "search", addr)
	client := searchpb.NewSearchServiceClient(conn)
	resp, err := client.Search(ctx, &searchpb.SearchRequest{
		Page:        1,
		PageSize:    5,
		IncludeSold: wrapperspb.Bool(true),
	})
	if err != nil {
		t.Fatalf("search request failed: %v", err)
	}
	if resp.Total <= 0 {
		t.Fatalf("expected total > 0, got %d", resp.Total)
	}
	if len(resp.Listings) == 0 {
		t.Fatalf("expected listings in response")
	}
}
