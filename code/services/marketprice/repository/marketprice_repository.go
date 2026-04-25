package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type Repository struct {
	DB *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{DB: db}
}

func (r *Repository) GetAverageMarketPrice(
	ctx context.Context,
	brand string,
	model string,
	yearFrom int32,
	yearTo int32,
) (float64, float64, float64, int32, error) {
	query := `SELECT 
		COALESCE(AVG(ad.ask_price), 0), 
		COALESCE(MIN(ad.ask_price), 0), 
		COALESCE(MAX(ad.ask_price), 0), 
		COUNT(ad.ask_price) 
		FROM automotive_data ad
		JOIN brand b ON ad.brand_id = b.id
		JOIN model m ON ad.model_id = m.id
		WHERE 1=1`

	var args []interface{}
	argId := 1

	if brand != "" {
		query += fmt.Sprintf(` AND b.name = $%d`, argId)
		args = append(args, strings.ToUpper(brand))
		argId++
	}
	if model != "" {
		query += fmt.Sprintf(` AND m.name = $%d`, argId)
		args = append(args, model)
		argId++
	}
	if yearFrom != 0 {
		query += fmt.Sprintf(` AND ad.model_year >= $%d`, argId)
		args = append(args, yearFrom)
		argId++
	}
	if yearTo != 0 {
		query += fmt.Sprintf(` AND ad.model_year <= $%d`, argId)
		args = append(args, yearTo)
		argId++
	}

	var avgPrice, minPrice, maxPrice float64
	var count int32

	err := r.DB.QueryRowContext(ctx, query, args...).Scan(&avgPrice, &minPrice, &maxPrice, &count)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	return avgPrice, minPrice, maxPrice, count, nil
}
