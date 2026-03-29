package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type IndexRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresIndexRepo(ctx context.Context, dsn string) (*IndexRepo, error) {
	if dsn == "" {
		return nil, fmt.Errorf("missing POSTGRES_DSN")
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	return &IndexRepo{pool: pool}, nil
}

func (s *IndexRepo) UpsertChatParticipant(ctx context.Context, userID, listingID, brand, model string) (string, error) {
	if userID == "" || listingID == "" || brand == "" || model == "" {
		return "", fmt.Errorf("user_id, listing_id, make, and model are all required")
	}

	const q = `
		INSERT INTO "chat-db".chat (user_id, listing_id, make, model)
		VALUES ($1, $2, $3, $4)
		RETURNING chat_id
	`

	var chatID string
	if err := s.pool.QueryRow(ctx, q, userID, listingID, brand, model).Scan(&chatID); err != nil {
		return "", fmt.Errorf("insert chat participant: %w", err)
	}

	return chatID, nil
}

func (s *IndexRepo) UserCanAccessChat(ctx context.Context, userID, listingID string) (bool, error) {
	if userID == "" || listingID == "" {
		return false, fmt.Errorf("user_id and listing_id are required")
	}

	const q = `SELECT EXISTS(SELECT 1 FROM "chat-db".chat WHERE user_id = $1 AND listing_id = $2);`
	var allowed bool
	if err := s.pool.QueryRow(ctx, q, userID, listingID).Scan(&allowed); err != nil {
		return false, fmt.Errorf("check chat access: %w", err)
	}
	return allowed, nil
}

func (s *IndexRepo) Close() error {
	s.pool.Close()
	return nil
}
