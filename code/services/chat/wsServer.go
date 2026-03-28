package main

import (
	"chat/ws"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type wsServer struct {
	hub *ws.Hub
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

	// TODO: verify user belongs to this chat (MongoDB)

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

	// TODO: persist to MongoDB

	out := OutboundMessage{
		ID:       "msg_" + time.Now().Format("20060102150405"),
		ChatID:   chatID,
		SenderID: userID,
		Content:  in.Content,
		SentAt:   time.Now().UTC(),
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
