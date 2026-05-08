package main

import (
	"log/slog"
	"net/http"
	"os"

	auctionpb "services/auction/proto"
	authpb "services/auth/proto"
	chatpb "services/chat/proto"
	"services/gateway/handlers"
	geopb "services/geographic-market-insights/proto"
	listingpb "services/listing/proto"
	marketpricepb "services/marketprice/proto"
	searchpb "services/search/proto"
	sellerpb "services/seller/proto"
	userpb "services/user/proto"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var Logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

func registerRoutes() {
	http.HandleFunc(routeHealth, handlers.HandleHealth)
	registerListingRoutes()
	registerChatRoutes()
	registerMarketRoutes()
	registerAuctionRoutes()
	registerAuthRoutes()
	registerUserRoutes()
	registerSellerRoutes()
}

func registerListingRoutes() {
	http.HandleFunc(routeListings, handlers.HandleListings)
	http.HandleFunc(routeListingsSearch, handlers.HandleSearch)
	http.HandleFunc(routeListingsCompare, handlers.HandleCompare)
	http.HandleFunc(routeListingsByID, handlers.HandleGetListing)
	http.HandleFunc(routeListingsStatsByLocation, handlers.HandleStatsByLocation)
}

func registerChatRoutes() {
	http.HandleFunc(routeGetChats, handlers.HandleGetChats)
	http.HandleFunc(routeChatOpen, handlers.HandleChatOpen)
	http.HandleFunc(routeChatByID, handlers.HandleChatHistory)
	http.HandleFunc(routeChatWS, handlers.HandleChatWebSocket)
}

func registerMarketRoutes() {
	http.HandleFunc(routeMarketAggregates, handlers.HandleMarketAggregates)
	http.HandleFunc(routeMarketPriceComparison, handlers.HandleMarketPriceComparison)
	http.HandleFunc(routeMarketAveragePrice, handlers.HandleAveragePrice)
}

func registerAuctionRoutes() {
	http.HandleFunc(routeAuctions, handlers.HandleAuctions)
	http.HandleFunc(routeAuctionWS, handlers.HandleAuctionWebSocket)
	http.HandleFunc(routeAuctionByID, handlers.HandleAuctionByIDRoutes)
}

func registerAuthRoutes() {
	http.HandleFunc(routeAuthRegister, handlers.HandleRegisterUser)
	http.HandleFunc(routeAuthLogin, handlers.HandleLoginUser)
}

func registerUserRoutes() {
	http.HandleFunc(routeFavorites, handlers.HandleGetFavorites)
	http.HandleFunc(routeFavoriteByListingID, handlers.HandleFavoriteListing)
	http.HandleFunc(routeUsersPreview, handlers.HandleGetUsersPreview)
}

func registerSellerRoutes() {
	http.HandleFunc(routeSellerByID, handlers.HandleGetSellerProfile)
	http.HandleFunc(routeSellersPreview, handlers.HandleGetSellersPreview)
}

func main() {
	slog.SetDefault(Logger)
	handlers.SetLogger(Logger)

	authConn, err := grpc.NewClient(os.Getenv("AUTH_GRPC_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		Logger.Error("failed to connect to auth service", "error", err)
		return
	}
	defer authConn.Close()
	authClient := authpb.NewAuthServiceClient(authConn)

	listingConn, err := grpc.NewClient(os.Getenv("LISTING_GRPC_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		Logger.Error("failed to connect to listing service", "error", err)
		return
	}
	defer listingConn.Close()
	listingClient := listingpb.NewListingServiceClient(listingConn)

	searchConn, err := grpc.NewClient(os.Getenv("SEARCH_GRPC_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		Logger.Error("failed to connect to search service", "error", err)
		return
	}
	defer searchConn.Close()
	searchClient := searchpb.NewSearchServiceClient(searchConn)

	userConn, err := grpc.NewClient(os.Getenv("USER_GRPC_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		Logger.Error("failed to connect to user service", "error", err)
		return
	}
	defer userConn.Close()
	userClient := userpb.NewUserServiceClient(userConn)

	sellerConn, err := grpc.NewClient(os.Getenv("SELLER_GRPC_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		Logger.Error("failed to connect to seller service", "error", err)
		return
	}
	defer sellerConn.Close()
	sellerClient := sellerpb.NewSellerServiceClient(sellerConn)

	chatConn, err := grpc.NewClient(os.Getenv("CHAT_GRPC_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		Logger.Error("failed to connect to chat service", "error", err)
		return
	}
	defer chatConn.Close()
	chatClient := chatpb.NewChatServiceClient(chatConn)

	geoConn, err := grpc.NewClient(os.Getenv("GEO_GRPC_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		Logger.Error("failed to connect to geo-market-insights service", "error", err)
		return
	}
	defer geoConn.Close()
	geoClient := geopb.NewGeoMarketInsightsServiceClient(geoConn)

	auctionConn, err := grpc.NewClient(os.Getenv("AUCTION_GRPC_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		Logger.Error("failed to connect to auction service", "error", err)
		return
	}
	defer auctionConn.Close()
	auctionClient := auctionpb.NewAuctionServiceClient(auctionConn)

	marketpriceConn, err := grpc.NewClient(os.Getenv("MARKETPRICE_GRPC_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		Logger.Error("failed to connect to marketprice service", "error", err)
		return
	}
	defer marketpriceConn.Close()
	marketpriceClient := marketpricepb.NewMarketPriceServiceClient(marketpriceConn)

	handlers.SetClients(
		authClient,
		listingClient,
		searchClient,
		userClient,
		sellerClient,
		chatClient,
		geoClient,
		auctionClient,
		marketpriceClient,
	)

	handlers.SetChatWSUpstream(os.Getenv("CHAT_WS_ADDR"))
	handlers.SetAuctionWSUpstream(os.Getenv("AUCTION_WS_ADDR"))

	registerRoutes()

	http.Handle("/metrics", promhttp.Handler())

	Logger.Info("Gateway listening on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		Logger.Error("failed to start HTTP server", "error", err)
	}
}
