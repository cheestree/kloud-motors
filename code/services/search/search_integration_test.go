//go:build integration

package main

import (
	"context"
	"testing"
	"time"

	searchpb "services/search/proto"
	"services/utils"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSearchIntegration_Basic(t *testing.T) {
	addr := utils.GetEnv("SEARCH_TEST_ADDR", "localhost:15056")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		t.Fatalf("failed to dial search service at %s: %v", addr, err)
	}
	defer conn.Close()

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
