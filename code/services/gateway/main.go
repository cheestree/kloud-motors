package main

import (
	"log"
	"net/http"
	"os"

	auctionpb "services/auction/proto"
	authpb "services/auth/proto"
	chatpb "services/chat/proto"
	"services/gateway/handlers"
	geopb "services/geographic-maket-insights/proto"
	listingpb "services/listing/proto"
	marketpricepb "services/marketprice/proto"
	searchpb "services/search/proto"
	sellerpb "services/seller/proto"
	userpb "services/user/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func registerRoutes() {
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
	http.HandleFunc(routeChatOpen, handlers.HandleChatOpen)
	http.HandleFunc(routeChatByID, handlers.HandleChatHistory)
}

func registerMarketRoutes() {
	http.HandleFunc(routeMarketAggregates, handlers.HandleMarketAggregates)
	http.HandleFunc(routeMarketPriceComparison, handlers.HandleMarketPriceComparison)
	http.HandleFunc(routeMarketAveragePrice, handlers.HandleAveragePrice)
}

func registerAuctionRoutes() {
	http.HandleFunc(routeAuctions, handlers.HandleAuctions)
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
	authConn, err := grpc.NewClient(os.Getenv("AUTH_GRPC_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to auth service: %v", err)
	}
	defer authConn.Close()
	authClient := authpb.NewAuthServiceClient(authConn)

	listingConn, err := grpc.NewClient(os.Getenv("LISTING_GRPC_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to listing service: %v", err)
	}
	defer listingConn.Close()
	listingClient := listingpb.NewListingServiceClient(listingConn)

	searchConn, err := grpc.NewClient(os.Getenv("SEARCH_GRPC_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to search service: %v", err)
	}
	defer searchConn.Close()
	searchClient := searchpb.NewSearchServiceClient(searchConn)

	userConn, err := grpc.NewClient(os.Getenv("USER_GRPC_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to user service: %v", err)
	}
	defer userConn.Close()
	userClient := userpb.NewUserServiceClient(userConn)

	sellerConn, err := grpc.NewClient(os.Getenv("SELLER_GRPC_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to seller service: %v", err)
	}
	defer sellerConn.Close()
	sellerClient := sellerpb.NewSellerServiceClient(sellerConn)

	chatConn, err := grpc.NewClient(os.Getenv("CHAT_GRPC_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to chat service: %v", err)
	}
	defer chatConn.Close()
	chatClient := chatpb.NewChatServiceClient(chatConn)

	geoConn, err := grpc.NewClient(os.Getenv("GEO_GRPC_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to geo-market-insights service: %v", err)
	}
	defer geoConn.Close()
	geoClient := geopb.NewGeoMarketInsightsServiceClient(geoConn)

	auctionConn, err := grpc.NewClient(os.Getenv("AUCTION_GRPC_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to auction service: %v", err)
	}
	defer auctionConn.Close()
	auctionClient := auctionpb.NewAuctionServiceClient(auctionConn)

	marketpriceConn, err := grpc.NewClient(os.Getenv("MARKETPRICE_GRPC_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to marketprice service: %v", err)
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

	registerRoutes()

	log.Println("Gateway listening on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
