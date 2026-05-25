//go:build integration

package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"services/internal/integrationtest"
	userpb "services/user/proto"
	"services/utils"
)

func TestUserIntegration_FavoritesLifecycle(t *testing.T) {
	addr := utils.GetEnv("USER_TEST_ADDR", "localhost:15058")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn := integrationtest.DialGRPC(ctx, t, "user", addr)
	client := userpb.NewUserServiceClient(conn)

	userID := time.Now().UnixNano()
	createResp, err := client.CreateUserProfile(ctx, &userpb.CreateUserProfileRequest{
		UserId: userID,
		Name:   "Integration User",
		Email:  fmt.Sprintf("integration-%d@example.test", userID),
	})
	if err != nil {
		t.Fatalf("create user profile failed: %v", err)
	}
	if !createResp.Success {
		t.Fatalf("expected create user profile to succeed")
	}

	const listingID int64 = 1
	addResp, err := client.AddFavorite(ctx, &userpb.AddFavoriteRequest{
		UserId:    userID,
		ListingId: listingID,
	})
	if err != nil {
		t.Fatalf("add favorite failed: %v", err)
	}
	if !addResp.Success {
		t.Fatalf("expected add favorite to succeed")
	}

	favorites, err := client.GetFavorites(ctx, &userpb.GetFavoritesRequest{UserId: userID})
	if err != nil {
		t.Fatalf("get favorites failed: %v", err)
	}
	if len(favorites.Favorites) != 1 || favorites.Favorites[0] != listingID {
		t.Fatalf("expected favorites [%d], got %v", listingID, favorites.Favorites)
	}

	removeResp, err := client.RemoveFavorite(ctx, &userpb.RemoveFavoriteRequest{
		UserId:    userID,
		ListingId: listingID,
	})
	if err != nil {
		t.Fatalf("remove favorite failed: %v", err)
	}
	if !removeResp.Success {
		t.Fatalf("expected remove favorite to succeed")
	}
}
