package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	firebase "firebase.google.com/go/v4"
	auctionpb "services/auction/proto"
	chatpb "services/chat/proto"
	"services/gateway/handlers"
	geopb "services/geographic-market-insights/proto"
	listingpb "services/listing/proto"
	marketpricepb "services/marketprice/proto"
	"services/observability"
	searchpb "services/search/proto"
	sellerpb "services/seller/proto"
	userpb "services/user/proto"



	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"google.golang.org/api/option"
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
	ctx := context.Background()
	shutdownTracing := observability.InitTracing(ctx, Logger, "gateway")
	defer func() {
		if err := shutdownTracing(ctx); err != nil {
			Logger.Error("failed to shutdown tracing", "error", err)
		}
	}()

	firebaseProjectID := os.Getenv("FIREBASE_PROJECT_ID")

	var firebaseOpts []option.ClientOption
	if credFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); credFile != "" {
		firebaseOpts = append(firebaseOpts, option.WithCredentialsFile(credFile))
	}

	fbApp, err := firebase.NewApp(ctx, &firebase.Config{
		ProjectID: firebaseProjectID,
	}, firebaseOpts...)
	if err != nil {
		Logger.Error("failed to initialise Firebase app", "error", err)
		return
	}

	fbAuthClient, err := fbApp.Auth(ctx)
	if err != nil {
		Logger.Error("failed to create Firebase Auth client", "error", err)
		return
	}
	handlers.SetFirebaseAuthClient(fbAuthClient)

	listingConn, err := grpc.NewClient(
		os.Getenv("LISTING_GRPC_ADDR"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		Logger.Error("failed to connect to listing service", "error", err)
		return
	}
	defer listingConn.Close()
	listingClient := listingpb.NewListingServiceClient(listingConn)

	searchConn, err := grpc.NewClient(
		os.Getenv("SEARCH_GRPC_ADDR"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		Logger.Error("failed to connect to search service", "error", err)
		return
	}
	defer searchConn.Close()
	searchClient := searchpb.NewSearchServiceClient(searchConn)

	userConn, err := grpc.NewClient(
		os.Getenv("USER_GRPC_ADDR"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		Logger.Error("failed to connect to user service", "error", err)
		return
	}
	defer userConn.Close()
	userClient := userpb.NewUserServiceClient(userConn)

	sellerConn, err := grpc.NewClient(
		os.Getenv("SELLER_GRPC_ADDR"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		Logger.Error("failed to connect to seller service", "error", err)
		return
	}
	defer sellerConn.Close()
	sellerClient := sellerpb.NewSellerServiceClient(sellerConn)

	chatConn, err := grpc.NewClient(
		os.Getenv("CHAT_GRPC_ADDR"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		Logger.Error("failed to connect to chat service", "error", err)
		return
	}
	defer chatConn.Close()
	chatClient := chatpb.NewChatServiceClient(chatConn)

	geoConn, err := grpc.NewClient(
		os.Getenv("GEO_GRPC_ADDR"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		Logger.Error("failed to connect to geo-market-insights service", "error", err)
		return
	}
	defer geoConn.Close()
	geoClient := geopb.NewGeoMarketInsightsServiceClient(geoConn)

	auctionConn, err := grpc.NewClient(
		os.Getenv("AUCTION_GRPC_ADDR"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		Logger.Error("failed to connect to auction service", "error", err)
		return
	}
	defer auctionConn.Close()
	auctionClient := auctionpb.NewAuctionServiceClient(auctionConn)

	marketpriceConn, err := grpc.NewClient(
		os.Getenv("MARKETPRICE_GRPC_ADDR"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		Logger.Error("failed to connect to marketprice service", "error", err)
		return
	}
	defer marketpriceConn.Close()
	marketpriceClient := marketpricepb.NewMarketPriceServiceClient(marketpriceConn)

	handlers.SetClients(
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

	RegisterMetrics()
	mux := http.DefaultServeMux
	handler := MetricsMiddleware(mux)

	Logger.Info("Gateway listening on :8080...")

	handler = otelhttp.NewHandler(handler, "gateway-http")

	if err := http.ListenAndServe(":8080", handler); err != nil {
		Logger.Error("failed to start HTTP server", "error", err)
	}
}
