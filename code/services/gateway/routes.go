package main

const (
	routeBaseAPI = "/api"
)

const (
	routePrefixListings = routeBaseAPI + "/listings"
	routePrefixChat     = routeBaseAPI + "/chat"
	routePrefixMarket   = routeBaseAPI + "/market"
	routePrefixAuctions = routeBaseAPI + "/auctions"
	routePrefixAuth     = routeBaseAPI + "/auth"
	routePrefixUsersMe  = routeBaseAPI + "/users/me"
	routePrefixSellers  = routeBaseAPI + "/sellers"
)

const (
	routeListings                = routePrefixListings
	routeListingsSearch          = routePrefixListings + "/search"
	routeListingsCompare         = routePrefixListings + "/compare"
	routeListingsByID            = routePrefixListings + "/"
	routeChatOpen                = routePrefixChat + "/open"
	routeChatByID                = routePrefixChat + "/"
	routeMarketAggregates        = routePrefixMarket + "/insights/aggregates"
	routeMarketPriceComparison   = routePrefixMarket + "/price-comparison"
	routeListingsStatsByLocation = routePrefixListings + "/stats/by-location"
	routeMarketAveragePrice      = routePrefixMarket + "/average-price"
	routeAuctions                = routePrefixAuctions
	routeAuctionByID             = routePrefixAuctions + "/"
	routeAuthRegister            = routePrefixAuth + "/register"
	routeAuthLogin               = routePrefixAuth + "/login"
	routeFavorites               = routePrefixUsersMe + "/favorites"
	routeFavoriteByListingID     = routePrefixUsersMe + "/favorites/"
	routeSellerByID              = routePrefixSellers + "/"
)
