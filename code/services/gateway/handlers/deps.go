package handlers

import (
	"log/slog"

	firebaseauth "firebase.google.com/go/v4/auth"

	auctionpb "services/auction/proto"
	chatpb "services/chat/proto"
	geopb "services/geographic-market-insights/proto"
	listingpb "services/listing/proto"
	marketpricepb "services/marketprice/proto"
	searchpb "services/search/proto"
	sellerpb "services/seller/proto"
	userpb "services/user/proto"
)

var (
	Logger             = slog.Default()
	firebaseAuthClient *firebaseauth.Client
	listingClient      listingpb.ListingServiceClient
	searchClient       searchpb.SearchServiceClient
	userClient         userpb.UserServiceClient
	sellerClient       sellerpb.SellerServiceClient
	chatClient         chatpb.ChatServiceClient
	geoClient          geopb.GeoMarketInsightsServiceClient
	auctionClient      auctionpb.AuctionServiceClient
	chatWSUpstream     string
	marketpriceClient  marketpricepb.MarketPriceServiceClient
)

func SetLogger(l *slog.Logger) {
	if l != nil {
		Logger = l
	}
}

func SetFirebaseAuthClient(c *firebaseauth.Client) {
	firebaseAuthClient = c
}

// SetClients wires service clients from main into the handlers package
func SetClients(
	listing listingpb.ListingServiceClient,
	search searchpb.SearchServiceClient,
	user userpb.UserServiceClient,
	seller sellerpb.SellerServiceClient,
	chat chatpb.ChatServiceClient,
	geo geopb.GeoMarketInsightsServiceClient,
	auction auctionpb.AuctionServiceClient,
	marketprice marketpricepb.MarketPriceServiceClient,
) {
	listingClient = listing
	searchClient = search
	userClient = user
	sellerClient = seller
	chatClient = chat
	geoClient = geo
	auctionClient = auction
	marketpriceClient = marketprice
}

func SetChatWSUpstream(upstream string) {
	chatWSUpstream = upstream
}
