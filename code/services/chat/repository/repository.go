package repository

import (
	"context"
	"time"
)

type DBConfig struct {
	Schema       string
	Table        string
	DefaultLimit int
	MaxLimit     int
	Host         string
}

type ChatMessage struct {
	ID      string
	ChatID  string
	UserID  string
	Message string
	Time    time.Time
}

type MessageRepo interface {
	SaveMessage(ctx context.Context, msg ChatMessage) error
	ListChatMessages(ctx context.Context, chatID string, limit, skip int) ([]ChatMessage, error)
	Close() error
}

type ChatIndexRepo interface {
	UpsertChatParticipant(ctx context.Context, userID, listingID, brand, model string) (string, error)
	UserCanAccessChat(ctx context.Context, userID, listingID string) (bool, error)
	Close() error
}
