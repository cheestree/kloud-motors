package repository

import (
	"context"
	"time"
)

type DBConfig struct {
	Schema string
	Table  string
	Host   string
}

type ChatMessage struct {
	ID      string
	ChatID  string
	UserID  int64
	Message string
	Time    time.Time
}

type ChatSummary struct {
	ChatID    string
	ListingID int64
	Brand     string
	Model     string
}

type MessageRepo interface {
	SaveMessage(ctx context.Context, msg ChatMessage) error
	ListChatMessages(ctx context.Context, chatID string, limit, skip int32) ([]ChatMessage, error)
	Close() error
}

type ChatIndexRepo interface {
	UpsertChatParticipant(ctx context.Context, userID, sellerID, listingID int64, brand, model string) (string, error)
	ListUserChats(ctx context.Context, userID int64) ([]ChatSummary, error)
	UserCanAccessChat(ctx context.Context, userID int64, chatID string) (bool, error)
	GetListingIDByChat(ctx context.Context, userID int64, chatID string) (int64, error)
	GetChatsFromListingSeller(ctx context.Context, listingID, sellerId int64) ([]string, error)
	Close() error
}
