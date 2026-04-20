package handlers

import (
	authpb "services/auth/proto"
	auctionpb "services/auction/proto"
	chatpb "services/chat/proto"
	geopb "services/geographic-maket-insights/proto"
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
