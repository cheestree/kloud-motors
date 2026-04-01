package main

import (
	"errors"
	"log"
	"net/http"
	"services/auction/ws"
	"strings"

	"github.com/gorilla/websocket"
)

type wsServer struct {
	hub *ws.Hub
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (s *wsServer) ServeWS(w http.ResponseWriter, r *http.Request) {
	auctionID := r.PathValue("auctionID")
	if auctionID == "" {
		http.Error(w, "missing auction id", http.StatusBadRequest)
		return
	}

	userID, err := s.userIDFromGateway(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade error: %v", err)
		return
	}

	client := ws.NewClient(auctionID, userID, conn)
	s.hub.Register(auctionID, client)

	go client.WritePump()
	client.ReadPump(s.hub, func(auctionID, userID string, raw []byte) {
		// clients only receive bid updates, they don't send via WS
	})
}

func (s *wsServer) userIDFromGateway(r *http.Request) (string, error) {
	for _, header := range []string{"X-User-ID", "X-User-Id", "X-Authenticated-User-Id", "X-Forwarded-User"} {
		if userID := strings.TrimSpace(r.Header.Get(header)); userID != "" {
			return userID, nil
		}
	}
	return "", errors.New("missing gateway user id")
}
