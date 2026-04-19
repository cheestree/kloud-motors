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

	chatpb "services/chat/proto"

	"github.com/gorilla/websocket"
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func HandleChatOpen(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, msgMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	userID, err := authenticatedUserIDFromRequest(r)
	if err != nil {
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}

	var req chatpb.OpenChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, msgInvalidBody, http.StatusBadRequest)
		return
	}
	req.UserId = userID

	ctx := context.Background()
	resp, err := chatClient.OpenChat(ctx, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func HandleGetActiveChats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, msgMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	userID, err := authenticatedUserIDFromRequest(r)
	if err != nil {
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}

	ctx := context.Background()
	req := &chatpb.GetActiveChatsRequest{UserId: userID}
	resp, err := chatClient.GetActiveChats(ctx, req)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func HandleChatHistory(w http.ResponseWriter, r *http.Request) {
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
	if len(parts) < 4 || parts[3] == "" {
		http.Error(w, "Missing chat id", http.StatusBadRequest)
		return
	}
	chatID := parts[3]
	ctx := context.Background()
	req := &chatpb.GetChatHistoryRequest{ChatId: chatID, UserId: userID}
	resp, err := chatClient.GetChatHistory(ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func HandleChatWebSocket(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "Missing chat id", http.StatusBadRequest)
		return
	}
	chatID := parts[4]

	upstreamURL, err := chatWSProxyURL(chatID, r.URL.RawQuery)
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
		http.Error(w, "failed to connect chat websocket upstream", status)
		return
	}

	clientConn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		_ = upstreamConn.Close()
		log.Printf("ws upgrade error: %v", err)
		return
	}

	errCh := make(chan error, 2)
	go proxyWebSocket(clientConn, upstreamConn, errCh)
	go proxyWebSocket(upstreamConn, clientConn, errCh)

	<-errCh
	_ = clientConn.Close()
	_ = upstreamConn.Close()
}

func chatWSProxyURL(chatID, rawQuery string) (string, error) {
	if chatWSUpstream == "" {
		return "", errors.New("chat websocket upstream is not configured")
	}

	baseURL, err := url.Parse(chatWSUpstream)
	if err != nil {
		return "", err
	}

	baseURL.Path = "/ws/chat/" + chatID
	baseURL.RawQuery = rawQuery
	return baseURL.String(), nil
}

func proxyWebSocket(src, dst *websocket.Conn, errCh chan<- error) {
	for {
		msgType, msg, err := src.ReadMessage()
		if err != nil {
			errCh <- err
			return
		}

		if err := dst.WriteMessage(msgType, msg); err != nil {
			errCh <- err
			return
		}
	}
}
