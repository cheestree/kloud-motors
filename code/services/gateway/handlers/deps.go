package handlers

import (
	auctionpb "services/auction/proto"
	authpb "services/auth/proto"
	chatpb "services/chat/proto"
	geopb "services/geographic-market-insights/proto"
	listingpb "services/listing/proto"
	marketpricepb "services/marketprice/proto"
	searchpb "services/search/proto"
	sellerpb "services/seller/proto"
	userpb "services/user/proto"
)

var (
	authClient        authpb.AuthServiceClient
	listingClient     listingpb.ListingServiceClient
	searchClient      searchpb.SearchServiceClient
	userClient        userpb.UserServiceClient
	sellerClient      sellerpb.SellerServiceClient
	chatClient        chatpb.ChatServiceClient
	geoClient         geopb.GeoMarketInsightsServiceClient
	auctionClient     auctionpb.AuctionServiceClient
	chatWSUpstream    string
	marketpriceClient marketpricepb.MarketPriceServiceClient
)

// SetClients wires service clients from main into the handlers package
func SetClients(
	auth authpb.AuthServiceClient,
	listing listingpb.ListingServiceClient,
	search searchpb.SearchServiceClient,
	user userpb.UserServiceClient,
	seller sellerpb.SellerServiceClient,
	chat chatpb.ChatServiceClient,
	geo geopb.GeoMarketInsightsServiceClient,
	auction auctionpb.AuctionServiceClient,
	marketprice marketpricepb.MarketPriceServiceClient,
) {
	authClient = auth
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
