package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	chatpb "services/chat/proto"
)

func handleChatOpen(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, msgMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}
	var req chatpb.OpenChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, msgInvalidBody, http.StatusBadRequest)
		return
	}
	ctx := context.Background()
	resp, err := chatClient.OpenChat(ctx, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func handleChatHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, msgMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}
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
	writeJSON(w, http.StatusOK, resp)
}
