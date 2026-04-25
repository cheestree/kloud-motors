package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	auctionpb "services/auction/proto"
	"services/utils"

	"github.com/gorilla/websocket"
)

var auctionWSUpstream string

func SetAuctionWSUpstream(url string) {
	auctionWSUpstream = url
}

func HandleAuctions(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	switch r.Method {
	case http.MethodGet:
		q := r.URL.Query()
		req := &auctionpb.ListAuctionsRequest{
			Status: q.Get(queryStatus),
			Page:   utils.ParseInt32WithDefault(q.Get(queryPage), 1),
			Limit:  utils.ParseInt32WithDefault(q.Get(queryPageSize), 20),
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
				Page:      utils.ParseInt32WithDefault(q.Get(queryPage), 1),
				Limit:     utils.ParseInt32WithDefault(q.Get(queryPageSize), 20),
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

func HandleAuctionWebSocket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, msgMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	userID, err := authenticatedUserIDFromRequest(r)
	if err != nil {
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 || parts[4] == "" {
		http.Error(w, "Missing auction id", http.StatusBadRequest)
		return
	}
	auctionID := parts[4]

	upstreamURL, err := auctionWSProxyURL(auctionID, r.URL.RawQuery)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	upstreamHeader := make(http.Header)
	upstreamHeader.Set("X-User-ID", strconv.FormatInt(userID, 10))

	upstreamConn, resp, err := websocket.DefaultDialer.Dial(upstreamURL, upstreamHeader)
	if err != nil {
		status := http.StatusBadGateway
		if resp != nil {
			status = resp.StatusCode
			_ = resp.Body.Close()
		}
		http.Error(w, "failed to connect auction websocket upstream", status)
		return
	}

	clientConn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		_ = upstreamConn.Close()
		log.Printf("ws upgrade error: %v", err)
		return
	}

	errCh := make(chan error, 2)

	// Server -> Client (Sends out notices of new bids without accepting new requests)
	go proxyWebSocket(upstreamConn, clientConn, errCh)

	// Client -> Server (Read and discard strictly for processing Ping/Pong/Close control frames)
	go readAndDiscardWebSocket(clientConn, errCh)

	<-errCh
	_ = clientConn.Close()
	_ = upstreamConn.Close()
}

func auctionWSProxyURL(auctionID, rawQuery string) (string, error) {
	if auctionWSUpstream == "" {
		return "", errors.New("auction websocket upstream is not configured")
	}

	baseURL, err := url.Parse(auctionWSUpstream)
	if err != nil {
		return "", err
	}

	baseURL.Path = "/ws/auction/" + auctionID
	baseURL.RawQuery = rawQuery
	return baseURL.String(), nil
}

func readAndDiscardWebSocket(src *websocket.Conn, errCh chan<- error) {
	for {
		_, _, err := src.ReadMessage()
		if err != nil {
			errCh <- err
			return
		}
	}
}
