package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	auctionpb "services/auction/proto"
)

func HandleAuctions(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	switch r.Method {
	case http.MethodGet:
		q := r.URL.Query()
		req := &auctionpb.ListAuctionsRequest{
			Status: q.Get(queryStatus),
			Page:   parseInt32WithDefault(q.Get(queryPage), 1),
			Limit:  parseInt32WithDefault(q.Get(queryPageSize), 20),
		}
		resp, err := auctionClient.ListAuctions(ctx, req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, resp)
	case http.MethodPost:
		var req auctionpb.CreateAuctionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, msgInvalidBody, http.StatusBadRequest)
			return
		}

		authUserID, err := authenticatedUserIDFromRequest(r)
		if err != nil {
			http.Error(w, msgUnauthorized, http.StatusUnauthorized)
			return
		}
		req.SellerId = authUserID

		resp, err := auctionClient.CreateAuction(ctx, &req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusCreated, resp)
	default:
		http.Error(w, msgMethodNotAllowed, http.StatusMethodNotAllowed)
	}
}

func HandleAuctionByIDRoutes(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 || parts[3] == "" {
		http.Error(w, "Missing auction id", http.StatusBadRequest)
		return
	}

	auctionID := parts[3]
	ctx := context.Background()

	if len(parts) == 4 {
		switch r.Method {
		case http.MethodGet:
			resp, err := auctionClient.GetAuctionDetails(ctx, &auctionpb.GetAuctionRequest{AuctionId: auctionID})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, resp)
		case http.MethodDelete:
			authUserID, err := authenticatedUserIDFromRequest(r)
			if err != nil {
				http.Error(w, msgUnauthorized, http.StatusUnauthorized)
				return
			}
			resp, err := auctionClient.DeleteAuction(ctx, &auctionpb.DeleteAuctionRequest{AuctionId: auctionID, UserId: strconv.FormatInt(authUserID, 10)})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if resp.GetSuccess() {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			http.Error(w, "failed to delete auction", http.StatusInternalServerError)
		default:
			http.Error(w, msgMethodNotAllowed, http.StatusMethodNotAllowed)
		}
		return
	}

	if len(parts) == 5 {
		switch parts[4] {
		case pathActionBid:
			if r.Method != http.MethodPost {
				http.Error(w, msgMethodNotAllowed, http.StatusMethodNotAllowed)
				return
			}
			authUserID, err := authenticatedUserIDFromRequest(r)
			if err != nil {
				http.Error(w, msgUnauthorized, http.StatusUnauthorized)
				return
			}
			var req auctionpb.PlaceBidRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, msgInvalidBody, http.StatusBadRequest)
				return
			}
			req.AuctionId = auctionID
			req.BidderId = strconv.FormatInt(authUserID, 10)
			resp, err := auctionClient.PlaceBid(ctx, &req)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, resp)
		case pathActionBids:
			if r.Method != http.MethodGet {
				http.Error(w, msgMethodNotAllowed, http.StatusMethodNotAllowed)
				return
			}
			q := r.URL.Query()
			req := &auctionpb.GetAuctionBidsRequest{
				AuctionId: auctionID,
				Page:      parseInt32WithDefault(q.Get(queryPage), 1),
				Limit:     parseInt32WithDefault(q.Get(queryPageSize), 20),
			}
			resp, err := auctionClient.GetAuctionBids(ctx, req)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, resp)
		default:
			http.Error(w, msgNotFound, http.StatusNotFound)
		}
		return
	}

	http.Error(w, msgNotFound, http.StatusNotFound)
}
