package main

import (
	"services/chat/repository"
	"services/chat/ws"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type wsServer struct {
	hub          *ws.Hub
	messageStore repository.MessageRepo
	indexStore   repository.ChatIndexRepo
}

type InboundMessage struct {
	Content string `json:"content"`
}

type OutboundMessage struct {
	ID       string    `json:"id"`
	ChatID   string    `json:"chat_id"`
	SenderID int64     `json:"sender_id"`
	Content  string    `json:"content"`
	SentAt   time.Time `json:"sent_at"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (s *wsServer) ServeWS(w http.ResponseWriter, r *http.Request) {
	chatID := r.PathValue("chatID")
	if chatID == "" {
		http.Error(w, "missing chat id", http.StatusBadRequest)
		return
	}

	userID, err := s.userIDFromGateway(r)
	if err != nil || userID == 0 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if s.indexStore != nil {
		listingID := r.URL.Query().Get("listing_id")
		if listingID == "" {
			http.Error(w, "missing listing_id", http.StatusBadRequest)
			return
		}

		allowed, err := s.indexStore.UserCanAccessChat(r.Context(), userID, listingID)
		if err != nil {
			log.Printf("chat access check error user=%d listing=%s err=%v", userID, listingID, err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if !allowed {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade error: %v", err)
		return
	}

	client := ws.NewClient(chatID, userID, conn)
	s.hub.Register(chatID, client)

	go client.WritePump()
	client.ReadPump(s.hub, s.onMessage)
}

func (s *wsServer) onMessage(chatID string, userID int64, raw []byte) {
	var in InboundMessage
	if err := json.Unmarshal(raw, &in); err != nil || in.Content == "" {
		return
	}

	out := OutboundMessage{
		ID:       fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		ChatID:   chatID,
		SenderID: userID,
		Content:  in.Content,
		SentAt:   time.Now().UTC(),
	}

	if s.messageStore != nil {
		err := s.messageStore.SaveMessage(context.Background(), repository.ChatMessage{
			ID:      out.ID,
			ChatID:  out.ChatID,
			UserID:  out.SenderID,
			Message: out.Content,
			Time:    out.SentAt,
		})
		if err != nil {
			log.Printf("save message error chat=%s user=%d err=%v", chatID, userID, err)
			return
		}
	}

	payload, err := json.Marshal(out)
	if err != nil {
		return
	}

	if err := s.hub.Publish(chatID, payload); err != nil {
		log.Printf("publish error for chat %s: %v", chatID, err)
	}
}

func (s *wsServer) userIDFromGateway(r *http.Request) (int64, error) {
	userID := strings.TrimSpace(r.Header.Get("X-User-ID"))
	if userID == "" {
		return 0, errors.New("missing gateway user id")
	}

	int64UserID, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return 0, err
	}

	return int64UserID, nil
}
