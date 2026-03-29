package repository

import (
	"context"
	"database/sql"
	"time"

	"listing/domain"

	"github.com/lib/pq"
)

type ListingRepository struct {
	db *sql.DB
}

func NewListingRepository(db *sql.DB) *ListingRepository {
	return &ListingRepository{db: db}
}

func (r *ListingRepository) GetListingDetails(ctx context.Context, id int64) (*domain.ListingDetails, error) {
	query := `
		SELECT
			ad.id,
			b.name,
			m.name,
			ad.model_year,
			ad.ask_price,
			ad.mileage,
			COALESCE(ft.name, ''),
			COALESCE(t.name, ''),
			COALESCE(ad.color, ''),
			ad.first_seen
		FROM automotive_data ad
		JOIN brand b ON ad.brand_id = b.id
		JOIN model m ON ad.model_id = m.id
		LEFT JOIN fuel_type ft ON ad.fuel_type_id = ft.id
		LEFT JOIN transmission t ON ad.transmission_id = t.id
		WHERE ad.id = $1
	`

	var (
		vin          string
		makeName     string
		modelName    string
		yearValue    sql.NullInt64
		priceValue   sql.NullInt64
		mileage      sql.NullInt64
		fuelType     sql.NullString
		transmission sql.NullString
		color        sql.NullString
		listedAt     sql.NullTime
	)

	if err := r.db.QueryRowContext(ctx, query, id).Scan(
		&vin,
		&makeName,
		&modelName,
		&yearValue,
		&priceValue,
		&mileage,
		&fuelType,
		&transmission,
		&color,
		&listedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return buildListingDetails(
		id,
		makeName,
		modelName,
		yearValue,
		priceValue,
		mileage,
		fuelType,
		transmission,
		color,
		listedAt,
	), nil
}

func (r *ListingRepository) CompareListings(ctx context.Context, ids []int64) ([]*domain.ListingDetails, error) {
	if len(ids) == 0 {
		return []*domain.ListingDetails{}, nil
	}

	query := `
		SELECT
			ad.id,
			b.name,
			m.name,
			ad.model_year,
			ad.ask_price,
			ad.mileage,
			COALESCE(ft.name, ''),
			COALESCE(t.name, ''),
			COALESCE(ad.color, ''),
			ad.first_seen
		FROM automotive_data ad
		JOIN brand b ON ad.brand_id = b.id
		JOIN model m ON ad.model_id = m.id
		LEFT JOIN fuel_type ft ON ad.fuel_type_id = ft.id
		LEFT JOIN transmission t ON ad.transmission_id = t.id
		WHERE ad.id = ANY($1)
		ORDER BY array_position($1, ad.vin)
	`

	rows, err := r.db.QueryContext(ctx, query, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	listings := make([]*domain.ListingDetails, 0, len(ids))
	for rows.Next() {
		var (
			id           int64
			makeName     string
			modelName    string
			yearValue    sql.NullInt64
			priceValue   sql.NullInt64
			mileage      sql.NullInt64
			fuelType     sql.NullString
			transmission sql.NullString
			color        sql.NullString
			listedAt     sql.NullTime
		)

		if err := rows.Scan(
			&id,
			&makeName,
			&modelName,
			&yearValue,
			&priceValue,
			&mileage,
			&fuelType,
			&transmission,
			&color,
			&listedAt,
		); err != nil {
			return nil, err
		}

		listings = append(listings, buildListingDetails(
			id,
			makeName,
			modelName,
			yearValue,
			priceValue,
			mileage,
			fuelType,
			transmission,
			color,
			listedAt,
		))
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return listings, nil
}

func buildListingDetails(
	id int64,
	makeName string,
	modelName string,
	yearValue sql.NullInt64,
	priceValue sql.NullInt64,
	mileage sql.NullInt64,
	fuelType sql.NullString,
	transmission sql.NullString,
	color sql.NullString,
	listedAt sql.NullTime,
) *domain.ListingDetails {
	listedAtText := ""
	if listedAt.Valid {
		listedAtText = listedAt.Time.UTC().Format(time.RFC3339)
	}

	return &domain.ListingDetails{
		ID:           id,
		Make:         makeName,
		Model:        modelName,
		Year:         int32(yearValue.Int64),
		Price:        float64(priceValue.Int64),
		Mileage:      int32(mileage.Int64),
		Location:     "",
		FuelType:     fuelType.String,
		Trim:         "",
		Transmission: transmission.String,
		Color:        color.String,
		SellerType:   "",
		Description:  "",
		ListedAt:     listedAtText,
		Images:       []string{},
	}
}

func (r *ListingRepository) CheckListingOwnership(ctx context.Context, listingID int64, dealerID int64) (bool, error) {
	query := `SELECT COUNT(1) FROM automotive_data WHERE id = $1 AND dealer_id = $2`
	var count int
	err := r.db.QueryRowContext(ctx, query, listingID, dealerID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *ListingRepository) CheckListingOpen(ctx context.Context, listingID int64) (bool, error) {
	query := `SELECT 1 FROM automotive_data WHERE id = $1 LIMIT 1`
	var exists int
	err := r.db.QueryRowContext(ctx, query, listingID).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
