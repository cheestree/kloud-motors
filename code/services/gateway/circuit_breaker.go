package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	auctionpb "services/auction/proto"
	chatpb "services/chat/proto"
	geopb "services/geographic-market-insights/proto"
	listingpb "services/listing/proto"
	marketpricepb "services/marketprice/proto"
	searchpb "services/search/proto"
	sellerpb "services/seller/proto"
	userpb "services/user/proto"

	"github.com/sony/gobreaker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func newServiceCircuitBreaker(name string) *gobreaker.CircuitBreaker {
	return gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        name,
		MaxRequests: 1,
		Interval:    30 * time.Second,
		Timeout:     15 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
	})
}

func withBreaker[T any](breaker *gobreaker.CircuitBreaker, serviceName string, fn func() (T, error)) (T, error) {
	var zero T

	result, err := breaker.Execute(func() (any, error) {
		return fn()
	})
	if err != nil {
		if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
			return zero, status.Error(codes.Unavailable, serviceName+" is unavailable")
		}
		return zero, err
	}

	typed, ok := result.(T)
	if !ok {
		return zero, fmt.Errorf("unexpected breaker result type %T", result)
	}

	return typed, nil
}

type breakerListingClient struct {
	listingpb.ListingServiceClient
	breaker *gobreaker.CircuitBreaker
}

func newBreakerListingClient(inner listingpb.ListingServiceClient) listingpb.ListingServiceClient {
	return &breakerListingClient{
		ListingServiceClient: inner,
		breaker:              newServiceCircuitBreaker("listing-service"),
	}
}

func (c *breakerListingClient) CreateListing(ctx context.Context, in *listingpb.CreateListingRequest, opts ...grpc.CallOption) (*listingpb.ListingDetailsResponse, error) {
	return withBreaker(c.breaker, "listing service", func() (*listingpb.ListingDetailsResponse, error) {
		return c.ListingServiceClient.CreateListing(ctx, in, opts...)
	})
}

func (c *breakerListingClient) CompareListings(ctx context.Context, in *listingpb.CompareListingsRequest, opts ...grpc.CallOption) (*listingpb.CompareListingsResponse, error) {
	return withBreaker(c.breaker, "listing service", func() (*listingpb.CompareListingsResponse, error) {
		return c.ListingServiceClient.CompareListings(ctx, in, opts...)
	})
}

func (c *breakerListingClient) GetListingDetails(ctx context.Context, in *listingpb.ListingDetailsRequest, opts ...grpc.CallOption) (*listingpb.ListingDetailsResponse, error) {
	return withBreaker(c.breaker, "listing service", func() (*listingpb.ListingDetailsResponse, error) {
		return c.ListingServiceClient.GetListingDetails(ctx, in, opts...)
	})
}

func (c *breakerListingClient) UpdateListing(ctx context.Context, in *listingpb.UpdateListingRequest, opts ...grpc.CallOption) (*listingpb.ListingDetailsResponse, error) {
	return withBreaker(c.breaker, "listing service", func() (*listingpb.ListingDetailsResponse, error) {
		return c.ListingServiceClient.UpdateListing(ctx, in, opts...)
	})
}

func (c *breakerListingClient) DeleteListing(ctx context.Context, in *listingpb.DeleteListingRequest, opts ...grpc.CallOption) (*listingpb.DeleteListingResponse, error) {
	return withBreaker(c.breaker, "listing service", func() (*listingpb.DeleteListingResponse, error) {
		return c.ListingServiceClient.DeleteListing(ctx, in, opts...)
	})
}

type breakerSearchClient struct {
	searchpb.SearchServiceClient
	breaker *gobreaker.CircuitBreaker
}

func newBreakerSearchClient(inner searchpb.SearchServiceClient) searchpb.SearchServiceClient {
	return &breakerSearchClient{
		SearchServiceClient: inner,
		breaker:             newServiceCircuitBreaker("search-service"),
	}
}

func (c *breakerSearchClient) Search(ctx context.Context, in *searchpb.SearchRequest, opts ...grpc.CallOption) (*searchpb.SearchResponse, error) {
	return withBreaker(c.breaker, "search service", func() (*searchpb.SearchResponse, error) {
		return c.SearchServiceClient.Search(ctx, in, opts...)
	})
}

type breakerUserClient struct {
	userpb.UserServiceClient
	breaker *gobreaker.CircuitBreaker
}

func newBreakerUserClient(inner userpb.UserServiceClient) userpb.UserServiceClient {
	return &breakerUserClient{
		UserServiceClient: inner,
		breaker:           newServiceCircuitBreaker("user-service"),
	}
}

func (c *breakerUserClient) GetFavorites(ctx context.Context, in *userpb.GetFavoritesRequest, opts ...grpc.CallOption) (*userpb.FavoritesResponse, error) {
	return withBreaker(c.breaker, "user service", func() (*userpb.FavoritesResponse, error) {
		return c.UserServiceClient.GetFavorites(ctx, in, opts...)
	})
}

func (c *breakerUserClient) AddFavorite(ctx context.Context, in *userpb.AddFavoriteRequest, opts ...grpc.CallOption) (*userpb.FavoriteMutationResponse, error) {
	return withBreaker(c.breaker, "user service", func() (*userpb.FavoriteMutationResponse, error) {
		return c.UserServiceClient.AddFavorite(ctx, in, opts...)
	})
}

func (c *breakerUserClient) RemoveFavorite(ctx context.Context, in *userpb.RemoveFavoriteRequest, opts ...grpc.CallOption) (*userpb.FavoriteMutationResponse, error) {
	return withBreaker(c.breaker, "user service", func() (*userpb.FavoriteMutationResponse, error) {
		return c.UserServiceClient.RemoveFavorite(ctx, in, opts...)
	})
}

func (c *breakerUserClient) GetUsersPreview(ctx context.Context, in *userpb.UsersPreviewRequest, opts ...grpc.CallOption) (*userpb.UsersPreviewResponse, error) {
	return withBreaker(c.breaker, "user service", func() (*userpb.UsersPreviewResponse, error) {
		return c.UserServiceClient.GetUsersPreview(ctx, in, opts...)
	})
}

func (c *breakerUserClient) GetOrCreateByFirebaseUID(ctx context.Context, in *userpb.GetOrCreateByFirebaseUIDRequest, opts ...grpc.CallOption) (*userpb.GetOrCreateByFirebaseUIDResponse, error) {
	return withBreaker(c.breaker, "user service", func() (*userpb.GetOrCreateByFirebaseUIDResponse, error) {
		return c.UserServiceClient.GetOrCreateByFirebaseUID(ctx, in, opts...)
	})
}

type breakerSellerClient struct {
	sellerpb.SellerServiceClient
	breaker *gobreaker.CircuitBreaker
}

func newBreakerSellerClient(inner sellerpb.SellerServiceClient) sellerpb.SellerServiceClient {
	return &breakerSellerClient{
		SellerServiceClient: inner,
		breaker:             newServiceCircuitBreaker("seller-service"),
	}
}

func (c *breakerSellerClient) GetSellerProfile(ctx context.Context, in *sellerpb.GetSellerProfileRequest, opts ...grpc.CallOption) (*sellerpb.SellerProfileResponse, error) {
	return withBreaker(c.breaker, "seller service", func() (*sellerpb.SellerProfileResponse, error) {
		return c.SellerServiceClient.GetSellerProfile(ctx, in, opts...)
	})
}

func (c *breakerSellerClient) GetSellersPreview(ctx context.Context, in *sellerpb.SellersPreviewRequest, opts ...grpc.CallOption) (*sellerpb.SellersPreviewResponse, error) {
	return withBreaker(c.breaker, "seller service", func() (*sellerpb.SellersPreviewResponse, error) {
		return c.SellerServiceClient.GetSellersPreview(ctx, in, opts...)
	})
}

type breakerChatClient struct {
	chatpb.ChatServiceClient
	breaker *gobreaker.CircuitBreaker
}

func newBreakerChatClient(inner chatpb.ChatServiceClient) chatpb.ChatServiceClient {
	return &breakerChatClient{
		ChatServiceClient: inner,
		breaker:           newServiceCircuitBreaker("chat-service"),
	}
}

func (c *breakerChatClient) OpenChat(ctx context.Context, in *chatpb.OpenChatRequest, opts ...grpc.CallOption) (*chatpb.OpenChatResponse, error) {
	return withBreaker(c.breaker, "chat service", func() (*chatpb.OpenChatResponse, error) {
		return c.ChatServiceClient.OpenChat(ctx, in, opts...)
	})
}

func (c *breakerChatClient) GetChats(ctx context.Context, in *chatpb.GetChatsRequest, opts ...grpc.CallOption) (*chatpb.GetChatsResponse, error) {
	return withBreaker(c.breaker, "chat service", func() (*chatpb.GetChatsResponse, error) {
		return c.ChatServiceClient.GetChats(ctx, in, opts...)
	})
}

func (c *breakerChatClient) GetChatHistory(ctx context.Context, in *chatpb.GetChatHistoryRequest, opts ...grpc.CallOption) (*chatpb.GetChatHistoryResponse, error) {
	return withBreaker(c.breaker, "chat service", func() (*chatpb.GetChatHistoryResponse, error) {
		return c.ChatServiceClient.GetChatHistory(ctx, in, opts...)
	})
}

type breakerGeoClient struct {
	geopb.GeoMarketInsightsServiceClient
	breaker *gobreaker.CircuitBreaker
}

func newBreakerGeoClient(inner geopb.GeoMarketInsightsServiceClient) geopb.GeoMarketInsightsServiceClient {
	return &breakerGeoClient{
		GeoMarketInsightsServiceClient: inner,
		breaker:                        newServiceCircuitBreaker("geo-market-insights-service"),
	}
}

func (c *breakerGeoClient) Aggregates(ctx context.Context, in *geopb.AggregatesRequest, opts ...grpc.CallOption) (*geopb.AggregatesResponse, error) {
	return withBreaker(c.breaker, "geo market insights service", func() (*geopb.AggregatesResponse, error) {
		return c.GeoMarketInsightsServiceClient.Aggregates(ctx, in, opts...)
	})
}

func (c *breakerGeoClient) PriceComparison(ctx context.Context, in *geopb.PriceComparisonRequest, opts ...grpc.CallOption) (*geopb.PriceComparisonResponse, error) {
	return withBreaker(c.breaker, "geo market insights service", func() (*geopb.PriceComparisonResponse, error) {
		return c.GeoMarketInsightsServiceClient.PriceComparison(ctx, in, opts...)
	})
}

func (c *breakerGeoClient) ByLocation(ctx context.Context, in *geopb.ByLocationRequest, opts ...grpc.CallOption) (*geopb.ByLocationResponse, error) {
	return withBreaker(c.breaker, "geo market insights service", func() (*geopb.ByLocationResponse, error) {
		return c.GeoMarketInsightsServiceClient.ByLocation(ctx, in, opts...)
	})
}

type breakerAuctionClient struct {
	auctionpb.AuctionServiceClient
	breaker *gobreaker.CircuitBreaker
}

func newBreakerAuctionClient(inner auctionpb.AuctionServiceClient) auctionpb.AuctionServiceClient {
	return &breakerAuctionClient{
		AuctionServiceClient: inner,
		breaker:              newServiceCircuitBreaker("auction-service"),
	}
}

func (c *breakerAuctionClient) ListAuctions(ctx context.Context, in *auctionpb.ListAuctionsRequest, opts ...grpc.CallOption) (*auctionpb.ListAuctionsResponse, error) {
	return withBreaker(c.breaker, "auction service", func() (*auctionpb.ListAuctionsResponse, error) {
		return c.AuctionServiceClient.ListAuctions(ctx, in, opts...)
	})
}

func (c *breakerAuctionClient) CreateAuction(ctx context.Context, in *auctionpb.CreateAuctionRequest, opts ...grpc.CallOption) (*auctionpb.CreateAuctionResponse, error) {
	return withBreaker(c.breaker, "auction service", func() (*auctionpb.CreateAuctionResponse, error) {
		return c.AuctionServiceClient.CreateAuction(ctx, in, opts...)
	})
}

func (c *breakerAuctionClient) GetAuctionDetails(ctx context.Context, in *auctionpb.GetAuctionRequest, opts ...grpc.CallOption) (*auctionpb.GetAuctionResponse, error) {
	return withBreaker(c.breaker, "auction service", func() (*auctionpb.GetAuctionResponse, error) {
		return c.AuctionServiceClient.GetAuctionDetails(ctx, in, opts...)
	})
}

func (c *breakerAuctionClient) DeleteAuction(ctx context.Context, in *auctionpb.DeleteAuctionRequest, opts ...grpc.CallOption) (*auctionpb.DeleteAuctionResponse, error) {
	return withBreaker(c.breaker, "auction service", func() (*auctionpb.DeleteAuctionResponse, error) {
		return c.AuctionServiceClient.DeleteAuction(ctx, in, opts...)
	})
}

func (c *breakerAuctionClient) PlaceBid(ctx context.Context, in *auctionpb.PlaceBidRequest, opts ...grpc.CallOption) (*auctionpb.PlaceBidResponse, error) {
	return withBreaker(c.breaker, "auction service", func() (*auctionpb.PlaceBidResponse, error) {
		return c.AuctionServiceClient.PlaceBid(ctx, in, opts...)
	})
}

func (c *breakerAuctionClient) GetAuctionBids(ctx context.Context, in *auctionpb.GetAuctionBidsRequest, opts ...grpc.CallOption) (*auctionpb.GetAuctionBidsResponse, error) {
	return withBreaker(c.breaker, "auction service", func() (*auctionpb.GetAuctionBidsResponse, error) {
		return c.AuctionServiceClient.GetAuctionBids(ctx, in, opts...)
	})
}

type breakerMarketPriceClient struct {
	marketpricepb.MarketPriceServiceClient
	breaker *gobreaker.CircuitBreaker
}

func newBreakerMarketPriceClient(inner marketpricepb.MarketPriceServiceClient) marketpricepb.MarketPriceServiceClient {
	return &breakerMarketPriceClient{
		MarketPriceServiceClient: inner,
		breaker:                  newServiceCircuitBreaker("marketprice-service"),
	}
}

func (c *breakerMarketPriceClient) GetAverageMarketPrice(ctx context.Context, in *marketpricepb.AveragePriceRequest, opts ...grpc.CallOption) (*marketpricepb.AveragePriceResponse, error) {
	return withBreaker(c.breaker, "market price service", func() (*marketpricepb.AveragePriceResponse, error) {
		return c.MarketPriceServiceClient.GetAverageMarketPrice(ctx, in, opts...)
	})
}