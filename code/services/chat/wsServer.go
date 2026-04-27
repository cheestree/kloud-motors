package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"services/chat/repository"
	"services/chat/ws"
	listingproto "services/listing/proto"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type wsServer struct {
	hub           *ws.Hub
	messageStore  repository.MessageRepo
	indexStore    repository.ChatIndexRepo
	listingClient listingproto.ListingServiceClient
	logger        *slog.Logger
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

	if s.indexStore == nil {
		http.Error(w, "chat index unavailable", http.StatusInternalServerError)
		return
	}

	if s.listingClient == nil {
		http.Error(w, "listing service unavailable", http.StatusInternalServerError)
		return
	}

	listingID, err := s.indexStore.GetListingIDByChat(r.Context(), userID, chatID)
	if err != nil {
		log.Printf("get listing by chat error chat=%s user=%d err=%v", chatID, userID, err)
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	listingOpen, err := s.listingClient.CheckListingOpen(r.Context(), &listingproto.CheckListingOpenRequest{ListingId: listingID})
	if err != nil {
		log.Printf("listing open check error listing=%d user=%d err=%v", listingID, userID, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !listingOpen.GetIsOpen() {
		http.Error(w, "listing is closed", http.StatusForbidden)
		return
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
	if err := json.Unmarshal(raw, &in); err != nil {
		// Fallback: if it's not valid JSON, treat the whole raw bytes as the message content.
		// This is useful for testing with simple WS clients.
		in.Content = string(raw)
	}

	if in.Content == "" {
		s.logger.Warn("received empty message content", "chat_id", chatID, "user_id", userID)
		return
	}

	out := OutboundMessage{
		ID:       fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		ChatID:   chatID,
		SenderID: userID,
		Content:  in.Content,
		SentAt:   time.Now().UTC(),
	}

	s.logger.Info("received message via websocket", "chat_id", chatID, "user_id", userID, "msg_id", out.ID)

	if s.messageStore != nil {
		s.logger.Info("attempting to save message to firestore", "msg_id", out.ID, "chat_id", chatID)
		err := s.messageStore.SaveMessage(context.Background(), repository.ChatMessage{
			ID:      out.ID,
			ChatID:  out.ChatID,
			UserID:  out.SenderID,
			Message: out.Content,
			Time:    out.SentAt,
		})
		if err != nil {
			s.logger.Error("failed to save message to firestore", "error", err, "chat_id", chatID, "user_id", userID)
			return
		}
		s.logger.Info("message saved successfully to firestore", "msg_id", out.ID)
	} else {
		s.logger.Warn("messageStore is nil, skipping firestore save", "chat_id", chatID)
	}

	payload, err := json.Marshal(out)
	if err != nil {
		s.logger.Error("failed to marshal outbound message", "error", err)
		return
	}

	if err := s.hub.Publish(chatID, payload); err != nil {
		s.logger.Error("failed to publish message to pubsub", "error", err, "chat_id", chatID)
	} else {
		s.logger.Info("message published to pubsub", "chat_id", chatID)
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
