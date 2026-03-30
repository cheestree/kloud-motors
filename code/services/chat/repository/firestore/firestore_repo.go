package firestore

import (
	"chat/repository"
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
)

type MessageRepo struct {
	client     *firestore.Client
	collection string
}

func NewFirestoreMessageRepo(ctx context.Context, projectID, collection string) (*MessageRepo, error) {
	if projectID == "" {
		return nil, fmt.Errorf("missing firestore/firestore project id")
	}
	if collection == "" {
		collection = "messages"
	}

	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("create firestore client: %w", err)
	}

	return &MessageRepo{client: client, collection: collection}, nil
}

func (s *MessageRepo) SaveMessage(ctx context.Context, msg repository.ChatMessage) error {
	if msg.ID == "" || msg.ChatID == "" || msg.UserID == "" || msg.Message == "" {
		return fmt.Errorf("invalid message payload: id, chat_id, user_id, and message are required")
	}
	if msg.Time.IsZero() {
		msg.Time = time.Now().UTC()
	}

	_, err := s.client.
		Collection("chats").Doc(msg.ChatID).
		Collection(s.collection).Doc(msg.ID).
		Set(ctx, map[string]any{
			"user_id": msg.UserID,
			"message": msg.Message,
			"time":    msg.Time,
		})
	if err != nil {
		return fmt.Errorf("save firestore message: %w", err)
	}

	return nil
}

func (s *MessageRepo) ListChatMessages(ctx context.Context, chatID string, limit, skip int) ([]repository.ChatMessage, error) {
	if chatID == "" {
		return nil, fmt.Errorf("chat id is required")
	}
	if limit <= 0 {
		limit = 50
	}
	if skip < 0 {
		skip = 0
	}

	iter := s.client.
		Collection("chats").Doc(chatID).
		Collection(s.collection).
		OrderBy("time", firestore.Asc).
		Offset(skip).
		Limit(limit).
		Documents(ctx)
	defer iter.Stop()

	docs, err := iter.GetAll()
	if err != nil {
		return nil, fmt.Errorf("list firestore messages: %w", err)
	}

	messages := make([]repository.ChatMessage, 0, len(docs))
	for _, d := range docs {
		var row struct {
			UserID  string    `firestore:"user_id"`
			Message string    `firestore:"message"`
			Time    time.Time `firestore:"time"`
		}
		if err := d.DataTo(&row); err != nil {
			return nil, fmt.Errorf("decode firestore message: %w", err)
		}
		messages = append(messages, repository.ChatMessage{
			ID:      d.Ref.ID,
			ChatID:  chatID,
			UserID:  row.UserID,
			Message: row.Message,
			Time:    row.Time,
		})
	}

	return messages, nil
}

func (s *MessageRepo) Close() error {
	return s.client.Close()
}
