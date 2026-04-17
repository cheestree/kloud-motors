package main

import (
	"log"
	"net/http"
	"os"

	auctionpb "services/auction/proto"
	chatpb "services/chat/proto"
	geopb "services/geographic-maket-insights/proto"
	listingpb "services/listing/proto"
	searchpb "services/search/proto"
	sellerpb "services/seller/proto"
	userpb "services/user/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	listingClient listingpb.ListingServiceClient
	searchClient  searchpb.SearchServiceClient
	userClient    userpb.UserServiceClient
	sellerClient  sellerpb.SellerServiceClient
	chatClient    chatpb.ChatServiceClient
	geoClient     geopb.GeoMarketInsightsServiceClient
	auctionClient auctionpb.AuctionServiceClient
)

func main() {
	listingConn, err := grpc.NewClient(os.Getenv("LISTING_GRPC_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to listing service: %v", err)
	}
	defer listingConn.Close()
	listingClient = listingpb.NewListingServiceClient(listingConn)

	searchConn, err := grpc.NewClient(os.Getenv("SEARCH_GRPC_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to search service: %v", err)
	}
	defer searchConn.Close()
	searchClient = searchpb.NewSearchServiceClient(searchConn)

	userConn, err := grpc.NewClient(os.Getenv("USER_GRPC_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to user service: %v", err)
	}
	defer userConn.Close()
	userClient = userpb.NewUserServiceClient(userConn)

	sellerConn, err := grpc.NewClient(os.Getenv("SELLER_GRPC_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to seller service: %v", err)
	}
	defer sellerConn.Close()
	sellerClient = sellerpb.NewSellerServiceClient(sellerConn)

	chatConn, err := grpc.NewClient(os.Getenv("CHAT_GRPC_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to chat service: %v", err)
	}
	defer chatConn.Close()
	chatClient = chatpb.NewChatServiceClient(chatConn)

	geoConn, err := grpc.NewClient(os.Getenv("GEO_GRPC_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to geo-market-insights service: %v", err)
	}
	defer geoConn.Close()
	geoClient = geopb.NewGeoMarketInsightsServiceClient(geoConn)

	auctionConn, err := grpc.NewClient(os.Getenv("AUCTION_GRPC_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to auction service: %v", err)
	}
	defer auctionConn.Close()
	auctionClient = auctionpb.NewAuctionServiceClient(auctionConn)

	http.HandleFunc(routeListingsSearch, handleSearch)
	http.HandleFunc(routeListingsCompare, handleCompare)
	http.HandleFunc(routeListingsByID, handleGetListing)
	http.HandleFunc(routeChatOpen, handleChatOpen)
	http.HandleFunc(routeChatByID, handleChatHistory)
	http.HandleFunc(routeMarketAggregates, handleMarketAggregates)
	http.HandleFunc(routeMarketPriceComparison, handleMarketPriceComparison)
	http.HandleFunc(routeListingsStatsByLocation, handleStatsByLocation)
	http.HandleFunc(routeMarketAveragePrice, handleAveragePrice)
	http.HandleFunc(routeAuctions, handleAuctions)
	http.HandleFunc(routeAuctionByID, handleAuctionByIDRoutes)
	http.HandleFunc(routeAuthRegister, handleRegisterUser)
	http.HandleFunc(routeAuthLogin, handleLoginUser)
	http.HandleFunc(routeFavorites, handleGetFavorites)
	http.HandleFunc(routeFavoriteByListingID, handleFavoriteListing)
	http.HandleFunc(routeSellerByID, handleGetSellerProfile)

	log.Println("Gateway listening on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
