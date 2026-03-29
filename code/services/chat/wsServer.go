package main

import (
	"chat/repository"
	"chat/ws"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
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
	SenderID string    `json:"sender_id"`
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
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if s.indexStore != nil {
		// Extract listing_id from request (query param, header, or URL)
		// Example: listingID := r.URL.Query().Get("listing_id")
		// For now, we need you to decide how listing_id comes from the client

		// TODO: Extract listing_id from request
		listingID := r.URL.Query().Get("listing_id")
		if listingID != "" {
			allowed, err := s.indexStore.UserCanAccessChat(r.Context(), userID, listingID)
			if err != nil {
				log.Printf("chat access check error user=%s listing=%s err=%v", userID, listingID, err)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			if !allowed {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
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

func (s *wsServer) onMessage(chatID, userID string, raw []byte) {
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
			ID:       out.ID,
			ChatID:   out.ChatID,
			UserID:   out.SenderID,
			UserName: userID, // Use userID as name, or extract from header if available
			Message:  out.Content,
			Time:     out.SentAt,
		})
		if err != nil {
			log.Printf("save message error chat=%s user=%s err=%v", chatID, userID, err)
			return
		}
	}

	if s.indexStore != nil {
		// Extract listing_id from request context or query param
		listingID := r.URL.Query().Get("listing_id")
		if listingID != "" {
			// TODO: Get brand and model from request or listing service
			brand := r.URL.Query().Get("brand")
			model := r.URL.Query().Get("model")

			if brand == "" || model == "" {
				log.Printf("warning: brand or model missing for user=%s listing=%s", userID, listingID)
			}

			_, err := s.indexStore.UpsertChatParticipant(context.Background(), userID, listingID, brand, model)
			if err != nil {
				log.Printf("index update error user=%s listing=%s err=%v", userID, listingID, err)
				return
			}
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

func (s *wsServer) userIDFromGateway(r *http.Request) (string, error) {
	for _, header := range []string{"X-User-ID", "X-User-Id", "X-Authenticated-User-Id", "X-Forwarded-User"} {
		if userID := strings.TrimSpace(r.Header.Get(header)); userID != "" {
			return userID, nil
		}
	}

	return "", errors.New("missing gateway user id")
}
