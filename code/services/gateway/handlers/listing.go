package handlers

import (
	"context"
	"net/http"
	"strings"

	listingpb "services/listing/proto"
	searchpb "services/search/proto"
)

func HandleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, msgMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query()
	ctx := context.Background()
	resp, err := searchClient.Search(ctx, &searchpb.SearchRequest{
		Make:       q.Get(queryMake),
		Model:      q.Get(queryModel),
		Year:       parseInt32(q.Get(queryYear)),
		MinPrice:   parseInt64(q.Get(queryMinPrice)),
		MaxPrice:   parseInt64(q.Get(queryMaxPrice)),
		MaxMileage: parseInt32(q.Get(queryMaxMileage)),
		FuelType:   q.Get(queryFuelTypeV2),
		Page:       parseInt32WithDefault(q.Get(queryPage), 1),
		PageSize:   parseInt32WithDefault(q.Get(queryPageSizeV2), 20),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func HandleCompare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, msgMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query()
	idStrs := strings.Split(q.Get(queryIDs), ",")
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
	writeJSON(w, http.StatusOK, resp)
}

func HandleGetListing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, msgMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}
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
	writeJSON(w, http.StatusOK, resp)
}
