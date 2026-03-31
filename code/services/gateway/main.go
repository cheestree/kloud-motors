package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

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

	http.HandleFunc("/api/listings/search", handleSearch)
	http.HandleFunc("/api/listings/compare", handleCompare)
	http.HandleFunc("/api/listings/", handleGetListing)
	http.HandleFunc("/api/chat/open", handleChatOpen)
	http.HandleFunc("/api/chat/", handleChatHistory)
	http.HandleFunc("/api/market/insights/aggregates", handleMarketAggregates)
	http.HandleFunc("/api/market/price-comparison", handleMarketPriceComparison)
	http.HandleFunc("/api/listings/stats/by-location", handleStatsByLocation)
	http.HandleFunc("/api/market/average-price", handleAveragePrice)
	// Missing auction endpoints
	http.HandleFunc("/api/auth/register", handleRegisterUser)
	http.HandleFunc("/api/auth/login", handleLoginUser)
	http.HandleFunc("/api/users/me/favorites", handleGetFavorites)
	http.HandleFunc("/api/users/me/favorites/", handleFavoriteListing)
	http.HandleFunc("/api/sellers/", handleGetSellerProfile)

	log.Println("Gateway listening on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// --- Chat ---
func handleChatOpen(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req chatpb.OpenChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	ctx := context.Background()
	resp, err := chatClient.OpenChat(ctx, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleChatHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Path: /api/chat/{chat_id}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 || parts[3] == "" {
		http.Error(w, "Missing chat id", http.StatusBadRequest)
		return
	}
	chatID := parts[3]
	ctx := context.Background()
	req := &chatpb.GetChatHistoryRequest{ChatId: chatID}
	resp, err := chatClient.GetChatHistory(ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// --- Geo Market Insights ---
func handleMarketAggregates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query()
	ctx := context.Background()
	req := &geopb.AggregatesRequest{
		Brand:     q.Get("brand"),
		Model:     q.Get("model"),
		Metrics:   q["metrics"],
		GroupBy:   q.Get("group_by"),
		Locations: q["locations"],
		YearFrom:  parseInt32(q.Get("year_from")),
		YearTo:    parseInt32(q.Get("year_to")),
		FuelType:  q.Get("fuel_type"),
		Limit:     parseInt32(q.Get("limit")),
		Skip:      parseInt32(q.Get("skip")),
	}
	resp, err := geoClient.Aggregates(ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleMarketPriceComparison(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query()
	ctx := context.Background()
	req := &geopb.PriceComparisonRequest{
		Brand:   q.Get("brand"),
		Model:   q.Get("model"),
		GroupBy: q.Get("group_by"),
		SortBy:  q.Get("sort_by"),
		Order:   q.Get("order"),
		Limit:   parseInt32(q.Get("limit")),
		Skip:    parseInt32(q.Get("skip")),
	}
	resp, err := geoClient.PriceComparison(ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleStatsByLocation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query()
	ctx := context.Background()
	req := &geopb.ByLocationRequest{
		Brand:    q.Get("brand"),
		Model:    q.Get("model"),
		Location: q.Get("location"),
		YearFrom: parseInt32(q.Get("year_from")),
		YearTo:   parseInt32(q.Get("year_to")),
		FuelType: q.Get("fuel_type"),
	}
	resp, err := geoClient.ByLocation(ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleAveragePrice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query()
	ctx := context.Background()
	req := &geopb.AggregatesRequest{
		Brand:    q.Get("brand"),
		Model:    q.Get("model"),
		GroupBy:  q.Get("location"),
		YearFrom: parseInt32(q.Get("year_from")),
		YearTo:   parseInt32(q.Get("year_to")),
		Metrics:  []string{"avg_price"},
	}
	resp, err := geoClient.Aggregates(ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// --- User Auth & Favorites ---
func handleRegisterUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req userpb.RegisterUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	ctx := context.Background()
	resp, err := userClient.RegisterUser(ctx, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleLoginUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req userpb.LoginUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	ctx := context.Background()
	resp, err := userClient.LoginUser(ctx, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleGetFavorites(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx := context.Background()
	req := &userpb.GetFavoritesRequest{} // Add user context if needed
	resp, err := userClient.GetFavorites(ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleFavoriteListing(w http.ResponseWriter, r *http.Request) {
	// Path: /api/users/me/favorites/{listing_id}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 6 || parts[5] == "" {
		http.Error(w, "Missing listing id", http.StatusBadRequest)
		return
	}
	listingID := parts[5]
	ctx := context.Background()
	switch r.Method {
	case http.MethodPost:
		req := &userpb.AddFavoriteRequest{ListingId: listingID}
		resp, err := userClient.AddFavorite(ctx, req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	case http.MethodDelete:
		req := &userpb.RemoveFavoriteRequest{ListingId: listingID}
		resp, err := userClient.RemoveFavorite(ctx, req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// --- Seller ---
func handleGetSellerProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Path: /api/sellers/{seller_id}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 || parts[3] == "" {
		http.Error(w, "Missing seller id", http.StatusBadRequest)
		return
	}
	sellerID := parts[3]
	ctx := context.Background()
	req := &sellerpb.GetSellerProfileRequest{SellerId: sellerID}
	resp, err := sellerClient.GetSellerProfile(ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query()
	ctx := context.Background()
	resp, err := searchClient.Search(ctx, &searchpb.SearchRequest{
		Make:       q.Get("make"),
		Model:      q.Get("model"),
		Year:       parseInt32(q.Get("year")),
		MinPrice:   parseInt64(q.Get("minPrice")),
		MaxPrice:   parseInt64(q.Get("maxPrice")),
		MaxMileage: parseInt32(q.Get("maxMileage")),
		FuelType:   q.Get("fuelType"),
		Page:       parseInt32WithDefault(q.Get("page"), 1),
		PageSize:   parseInt32WithDefault(q.Get("pageSize"), 20),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleCompare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query()
	idStrs := strings.Split(q.Get("ids"), ",")
	var ids []int64
	for _, s := range idStrs {
		if s == "" {
			continue
		}
		ids = append(ids, parseInt64(s))
	}
	ctx := context.Background()
	resp, err := listingClient.CompareListings(ctx, &listingpb.CompareListingsRequest{Ids: ids})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleGetListing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Path: /api/listings/{id}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 || parts[3] == "" {
		http.Error(w, "Missing listing id", http.StatusBadRequest)
		return
	}
	id := parseInt64(parts[3])
	ctx := context.Background()
	resp, err := listingClient.GetListingDetails(ctx, &listingpb.ListingDetailsRequest{Id: id})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func parseInt32(s string) int32 {
	var v int32
	_, _ = fmt.Sscan(s, &v)
	return v
}

func parseInt32WithDefault(s string, def int32) int32 {
	if s == "" {
		return def
	}
	return parseInt32(s)
}

func parseInt64(s string) int64 {
	var v int64
	_, _ = fmt.Sscan(s, &v)
	return v
}
