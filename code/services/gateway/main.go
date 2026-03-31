package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	listingpb "services/listing/proto"
	searchpb "services/search/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	listingClient listingpb.ListingServiceClient
	searchClient  searchpb.SearchServiceClient
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

	http.HandleFunc("/api/listings/search", handleSearch)
	http.HandleFunc("/api/listings/compare", handleCompare)
	http.HandleFunc("/api/listings/", handleGetListing)

	log.Println("Gateway listening on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
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
