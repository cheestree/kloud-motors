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

const defaultGRPCCallTimeout = 5 * time.Second

func newServiceCircuitBreaker(name string) *gobreaker.CircuitBreaker {
	return gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        name,
		MaxRequests: 1,
		Interval:    30 * time.Second,
		Timeout:     15 * time.Second,
		IsSuccessful: func(err error) bool {
			return !isBreakerFailure(err)
		},
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
	})
}

func isBreakerFailure(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	switch status.Code(err) {
	case codes.DeadlineExceeded, codes.Unavailable, codes.ResourceExhausted, codes.Internal, codes.Unknown, codes.DataLoss:
		return true
	default:
		return false
	}
}

func withBreaker[T any](ctx context.Context, breaker *gobreaker.CircuitBreaker, serviceName string, fn func(context.Context) (T, error)) (T, error) {
	var zero T

	callCtx, cancel := context.WithTimeout(ctx, defaultGRPCCallTimeout)
	defer cancel()

	result, err := breaker.Execute(func() (any, error) {
		return fn(callCtx)
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
	return withBreaker(ctx, c.breaker, "listing service", func(ctx context.Context) (*listingpb.ListingDetailsResponse, error) {
		return c.ListingServiceClient.CreateListing(ctx, in, opts...)
	})
}

func (c *breakerListingClient) CompareListings(ctx context.Context, in *listingpb.CompareListingsRequest, opts ...grpc.CallOption) (*listingpb.CompareListingsResponse, error) {
	return withBreaker(ctx, c.breaker, "listing service", func(ctx context.Context) (*listingpb.CompareListingsResponse, error) {
		return c.ListingServiceClient.CompareListings(ctx, in, opts...)
	})
}

func (c *breakerListingClient) GetListingDetails(ctx context.Context, in *listingpb.ListingDetailsRequest, opts ...grpc.CallOption) (*listingpb.ListingDetailsResponse, error) {
	return withBreaker(ctx, c.breaker, "listing service", func(ctx context.Context) (*listingpb.ListingDetailsResponse, error) {
		return c.ListingServiceClient.GetListingDetails(ctx, in, opts...)
	})
}

func (c *breakerListingClient) UpdateListing(ctx context.Context, in *listingpb.UpdateListingRequest, opts ...grpc.CallOption) (*listingpb.ListingDetailsResponse, error) {
	return withBreaker(ctx, c.breaker, "listing service", func(ctx context.Context) (*listingpb.ListingDetailsResponse, error) {
		return c.ListingServiceClient.UpdateListing(ctx, in, opts...)
	})
}

func (c *breakerListingClient) DeleteListing(ctx context.Context, in *listingpb.DeleteListingRequest, opts ...grpc.CallOption) (*listingpb.DeleteListingResponse, error) {
	return withBreaker(ctx, c.breaker, "listing service", func(ctx context.Context) (*listingpb.DeleteListingResponse, error) {
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
	return withBreaker(ctx, c.breaker, "search service", func(ctx context.Context) (*searchpb.SearchResponse, error) {
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
	return withBreaker(ctx, c.breaker, "user service", func(ctx context.Context) (*userpb.FavoritesResponse, error) {
		return c.UserServiceClient.GetFavorites(ctx, in, opts...)
	})
}

func (c *breakerUserClient) Login(ctx context.Context, in *userpb.AuthRequest, opts ...grpc.CallOption) (*userpb.AuthResponse, error) {
	return withBreaker(ctx, c.breaker, "user service", func(ctx context.Context) (*userpb.AuthResponse, error) {
		return c.UserServiceClient.Login(ctx, in, opts...)
	})
}

func (c *breakerUserClient) Register(ctx context.Context, in *userpb.AuthRequest, opts ...grpc.CallOption) (*userpb.AuthResponse, error) {
	return withBreaker(ctx, c.breaker, "user service", func(ctx context.Context) (*userpb.AuthResponse, error) {
		return c.UserServiceClient.Register(ctx, in, opts...)
	})
}

func (c *breakerUserClient) RefreshToken(ctx context.Context, in *userpb.RefreshTokenRequest, opts ...grpc.CallOption) (*userpb.AuthResponse, error) {
	return withBreaker(ctx, c.breaker, "user service", func(ctx context.Context) (*userpb.AuthResponse, error) {
		return c.UserServiceClient.RefreshToken(ctx, in, opts...)
	})
}

func (c *breakerUserClient) AddFavorite(ctx context.Context, in *userpb.AddFavoriteRequest, opts ...grpc.CallOption) (*userpb.FavoriteMutationResponse, error) {
	return withBreaker(ctx, c.breaker, "user service", func(ctx context.Context) (*userpb.FavoriteMutationResponse, error) {
		return c.UserServiceClient.AddFavorite(ctx, in, opts...)
	})
}

func (c *breakerUserClient) RemoveFavorite(ctx context.Context, in *userpb.RemoveFavoriteRequest, opts ...grpc.CallOption) (*userpb.FavoriteMutationResponse, error) {
	return withBreaker(ctx, c.breaker, "user service", func(ctx context.Context) (*userpb.FavoriteMutationResponse, error) {
		return c.UserServiceClient.RemoveFavorite(ctx, in, opts...)
	})
}

func (c *breakerUserClient) GetUsersPreview(ctx context.Context, in *userpb.UsersPreviewRequest, opts ...grpc.CallOption) (*userpb.UsersPreviewResponse, error) {
	return withBreaker(ctx, c.breaker, "user service", func(ctx context.Context) (*userpb.UsersPreviewResponse, error) {
		return c.UserServiceClient.GetUsersPreview(ctx, in, opts...)
	})
}

func (c *breakerUserClient) GetOrCreateByFirebaseUID(ctx context.Context, in *userpb.GetOrCreateByFirebaseUIDRequest, opts ...grpc.CallOption) (*userpb.GetOrCreateByFirebaseUIDResponse, error) {
	return withBreaker(ctx, c.breaker, "user service", func(ctx context.Context) (*userpb.GetOrCreateByFirebaseUIDResponse, error) {
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
	return withBreaker(ctx, c.breaker, "seller service", func(ctx context.Context) (*sellerpb.SellerProfileResponse, error) {
		return c.SellerServiceClient.GetSellerProfile(ctx, in, opts...)
	})
}

func (c *breakerSellerClient) GetSellersPreview(ctx context.Context, in *sellerpb.SellersPreviewRequest, opts ...grpc.CallOption) (*sellerpb.SellersPreviewResponse, error) {
	return withBreaker(ctx, c.breaker, "seller service", func(ctx context.Context) (*sellerpb.SellersPreviewResponse, error) {
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
	return withBreaker(ctx, c.breaker, "chat service", func(ctx context.Context) (*chatpb.OpenChatResponse, error) {
		return c.ChatServiceClient.OpenChat(ctx, in, opts...)
	})
}

func (c *breakerChatClient) GetChats(ctx context.Context, in *chatpb.GetChatsRequest, opts ...grpc.CallOption) (*chatpb.GetChatsResponse, error) {
	return withBreaker(ctx, c.breaker, "chat service", func(ctx context.Context) (*chatpb.GetChatsResponse, error) {
		return c.ChatServiceClient.GetChats(ctx, in, opts...)
	})
}

func (c *breakerChatClient) GetChatHistory(ctx context.Context, in *chatpb.GetChatHistoryRequest, opts ...grpc.CallOption) (*chatpb.GetChatHistoryResponse, error) {
	return withBreaker(ctx, c.breaker, "chat service", func(ctx context.Context) (*chatpb.GetChatHistoryResponse, error) {
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
	return withBreaker(ctx, c.breaker, "geo market insights service", func(ctx context.Context) (*geopb.AggregatesResponse, error) {
		return c.GeoMarketInsightsServiceClient.Aggregates(ctx, in, opts...)
	})
}

func (c *breakerGeoClient) PriceComparison(ctx context.Context, in *geopb.PriceComparisonRequest, opts ...grpc.CallOption) (*geopb.PriceComparisonResponse, error) {
	return withBreaker(ctx, c.breaker, "geo market insights service", func(ctx context.Context) (*geopb.PriceComparisonResponse, error) {
		return c.GeoMarketInsightsServiceClient.PriceComparison(ctx, in, opts...)
	})
}

func (c *breakerGeoClient) ByLocation(ctx context.Context, in *geopb.ByLocationRequest, opts ...grpc.CallOption) (*geopb.ByLocationResponse, error) {
	return withBreaker(ctx, c.breaker, "geo market insights service", func(ctx context.Context) (*geopb.ByLocationResponse, error) {
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
	return withBreaker(ctx, c.breaker, "auction service", func(ctx context.Context) (*auctionpb.ListAuctionsResponse, error) {
		return c.AuctionServiceClient.ListAuctions(ctx, in, opts...)
	})
}

func (c *breakerAuctionClient) CreateAuction(ctx context.Context, in *auctionpb.CreateAuctionRequest, opts ...grpc.CallOption) (*auctionpb.CreateAuctionResponse, error) {
	return withBreaker(ctx, c.breaker, "auction service", func(ctx context.Context) (*auctionpb.CreateAuctionResponse, error) {
		return c.AuctionServiceClient.CreateAuction(ctx, in, opts...)
	})
}

func (c *breakerAuctionClient) GetAuctionDetails(ctx context.Context, in *auctionpb.GetAuctionRequest, opts ...grpc.CallOption) (*auctionpb.GetAuctionResponse, error) {
	return withBreaker(ctx, c.breaker, "auction service", func(ctx context.Context) (*auctionpb.GetAuctionResponse, error) {
		return c.AuctionServiceClient.GetAuctionDetails(ctx, in, opts...)
	})
}

func (c *breakerAuctionClient) DeleteAuction(ctx context.Context, in *auctionpb.DeleteAuctionRequest, opts ...grpc.CallOption) (*auctionpb.DeleteAuctionResponse, error) {
	return withBreaker(ctx, c.breaker, "auction service", func(ctx context.Context) (*auctionpb.DeleteAuctionResponse, error) {
		return c.AuctionServiceClient.DeleteAuction(ctx, in, opts...)
	})
}

func (c *breakerAuctionClient) PlaceBid(ctx context.Context, in *auctionpb.PlaceBidRequest, opts ...grpc.CallOption) (*auctionpb.PlaceBidResponse, error) {
	return withBreaker(ctx, c.breaker, "auction service", func(ctx context.Context) (*auctionpb.PlaceBidResponse, error) {
		return c.AuctionServiceClient.PlaceBid(ctx, in, opts...)
	})
}

func (c *breakerAuctionClient) GetAuctionBids(ctx context.Context, in *auctionpb.GetAuctionBidsRequest, opts ...grpc.CallOption) (*auctionpb.GetAuctionBidsResponse, error) {
	return withBreaker(ctx, c.breaker, "auction service", func(ctx context.Context) (*auctionpb.GetAuctionBidsResponse, error) {
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
	return withBreaker(ctx, c.breaker, "market price service", func(ctx context.Context) (*marketpricepb.AveragePriceResponse, error) {
		return c.MarketPriceServiceClient.GetAverageMarketPrice(ctx, in, opts...)
	})
}
