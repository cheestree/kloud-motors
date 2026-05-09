package main

const (
	routeBaseAPI = "/api"
)

const (
	routePrefixListings = routeBaseAPI + "/listings"
	routePrefixChat     = routeBaseAPI + "/chat"
	routePrefixMarket   = routeBaseAPI + "/market"
	routePrefixAuctions = routeBaseAPI + "/auctions"
	routePrefixUsers    = routeBaseAPI + "/users"
	routePrefixSellers  = routeBaseAPI + "/sellers"
	routePrefixHealth   = routeBaseAPI + "/health"
)

const (
	routeListings                = routePrefixListings
	routeListingsSearch          = routePrefixListings + "/search"
	routeListingsCompare         = routePrefixListings + "/compare"
	routeListingsByID            = routePrefixListings + "/"
	routeGetChats                = routePrefixChat
	routeChatOpen                = routePrefixChat + "/open"
	routeChatWS                  = routePrefixChat + "/ws/{chatID}"
	routeChatByID                = routePrefixChat + "/{chatID}"
	routeMarketAggregates        = routePrefixMarket + "/insights/aggregates"
	routeMarketPriceComparison   = routePrefixMarket + "/price-comparison"
	routeListingsStatsByLocation = routePrefixListings + "/stats/by-location"
	routeMarketAveragePrice      = routePrefixMarket + "/average-price"
	routeAuctions                = routePrefixAuctions
	routeAuctionWS               = routePrefixAuctions + "/ws/{auctionID}"
	routeAuctionByID             = routePrefixAuctions + "/"
	routeFavorites               = routePrefixUsers + "/me/favorites"
	routeFavoriteByListingID     = routePrefixUsers + "/me/favorites/"
	routeSellerByID              = routePrefixSellers + "/"
	routeUsersPreview            = routePrefixUsers + "/preview"
	routeSellersPreview          = routePrefixSellers + "/preview"
	routeHealth                  = routePrefixHealth
)
