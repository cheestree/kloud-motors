package postgres

import (
	"services/chat/repository"
	"context"
	"fmt"
	"regexp"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RelationalRepo struct {
	pool *pgxpool.Pool
	cfg  repository.DBConfig
}

func NewPostgresRepo(ctx context.Context, cfg repository.DBConfig) (*RelationalRepo, error) {
	if cfg.Host == "" {
		return nil, fmt.Errorf("missing POSTGRES_DSN")
	}

	if cfg.Schema == "" {
		cfg.Schema = "chat-db"
	}
	if cfg.Table == "" {
		cfg.Table = "chat"
	}

	pool, err := pgxpool.New(ctx, cfg.Host)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	return &RelationalRepo{pool: pool, cfg: cfg}, nil
}

func (s *RelationalRepo) UpsertChatParticipant(ctx context.Context, userID, sellerId, listingID int64, brand, model string) (string, error) {
	userQuery := fmt.Sprintf(`
		INSERT INTO %s (user_id, listing_id, brand, model)
		VALUES ($1, $2, $3, $4)
		RETURNING chat_id
	`, s.qualifiedTable())

	var chatID string
	if err := s.pool.QueryRow(ctx, userQuery, userID, listingID, brand, model).Scan(&chatID); err != nil {
		return "", fmt.Errorf("insert chat participant: %w", err)
	}

	sellerQuery := fmt.Sprintf(`
		INSERT INTO %s (user_id, listing_id, brand, model, chat_id)
		VALUES ($1, $2, $3, $4, $5)
	`, s.qualifiedTable())

	if err := s.pool.QueryRow(ctx, sellerQuery, sellerId, listingID, brand, model, chatID); err != nil {
		return "", fmt.Errorf("insert chat participant: %w", err)
	}

	return chatID, nil
}

func (s *RelationalRepo) UserCanAccessChat(ctx context.Context, userID int64, listingID string) (bool, error) {
	q := fmt.Sprintf(`SELECT EXISTS(SELECT 1 FROM %s WHERE user_id = $1 AND listing_id = $2);`, s.qualifiedTable())
	var allowed bool
	if err := s.pool.QueryRow(ctx, q, userID, listingID).Scan(&allowed); err != nil {
		return false, fmt.Errorf("check chat access: %w", err)
	}
	return allowed, nil
}

func (s *RelationalRepo) ListUserChats(ctx context.Context, userID int64) ([]repository.ChatSummary, error) {
	q := fmt.Sprintf(`
		SELECT chat_id, listing_id, brand, model
		FROM %s
		WHERE user_id = $1
		ORDER BY chat_id DESC
	`, s.qualifiedTable())

	rows, err := s.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("list user chats: %w", err)
	}
	defer rows.Close()

	chats := make([]repository.ChatSummary, 0)
	for rows.Next() {
		var chat repository.ChatSummary
		if err := rows.Scan(&chat.ChatID, &chat.ListingID, &chat.Brand, &chat.Model); err != nil {
			return nil, fmt.Errorf("scan user chats: %w", err)
		}
		chats = append(chats, chat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user chats: %w", err)
	}

	return chats, nil
}

var identRx = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func safeIdent(v string) string {
	if !identRx.MatchString(v) {
		return ""
	}
	return `"` + v + `"`
}

func (s *RelationalRepo) qualifiedTable() string {
	schema := safeIdent(s.cfg.Schema)
	table := safeIdent(s.cfg.Table)
	if schema == "" || table == "" {
		return `"public"."chat"`
	}
	return schema + "." + table
}

func (s *RelationalRepo) Close() error {
	s.pool.Close()
	return nil
}
