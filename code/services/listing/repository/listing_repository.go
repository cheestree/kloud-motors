package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"services/shared"

	"github.com/lib/pq"
)

type ListingRepository struct {
	db *sql.DB
}

func NewListingRepository(db *sql.DB) *ListingRepository {
	return &ListingRepository{db: db}
}

func (r *ListingRepository) GetListingDetails(ctx context.Context, id int64) (*shared.ListingDetails, error) {
	query := `
			SELECT
				ad.vin,
				b.name,
				m.name,
				ad.model_year,
				ad.ask_price,
				ad.mileage,
				ad.trim,
				COALESCE(ft.name, ''),
				COALESCE(t.name, ''),
				COALESCE(ad.city, ''),
				COALESCE(ad.district, ''),
				COALESCE(ad.state, ''),
				COALESCE(ad.country, ''),
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
		trim         sql.NullString
		fuelType     sql.NullString
		transmission sql.NullString
		city         sql.NullString
		district     sql.NullString
		state        sql.NullString
		country      sql.NullString
		color        sql.NullString
		firstSeen    sql.NullTime
	)

	if err := r.db.QueryRowContext(ctx, query, id).Scan(
		&vin,
		&makeName,
		&modelName,
		&yearValue,
		&priceValue,
		&mileage,
		&trim,
		&fuelType,
		&transmission,
		&city,
		&district,
		&state,
		&country,
		&color,
		&firstSeen,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return buildListingDetails(
		id,
		vin,
		makeName,
		modelName,
		yearValue,
		priceValue,
		mileage,
		fuelType,
		trim,
		transmission,
		city,
		district,
		state,
		country,
		color,
		firstSeen,
	), nil
}

func (r *ListingRepository) CompareListings(ctx context.Context, ids []int64) ([]*shared.ListingDetails, error) {
	if len(ids) == 0 {
		return []*shared.ListingDetails{}, nil
	}

	query := `
		SELECT
		    ad.id,
			ad.vin,
			b.name,
			m.name,
			ad.model_year,
			ad.ask_price,
			ad.mileage,
			ad.trim,
			COALESCE(ft.name, ''),
			COALESCE(t.name, ''),
			COALESCE(ad.city, ''),
			COALESCE(ad.district, ''),
			COALESCE(ad.state, ''),
			COALESCE(ad.country, ''),
			COALESCE(ad.color, ''),
			ad.last_seen
		FROM automotive_data ad
		JOIN brand b ON ad.brand_id = b.id
		JOIN model m ON ad.model_id = m.id
		LEFT JOIN fuel_type ft ON ad.fuel_type_id = ft.id
		LEFT JOIN transmission t ON ad.transmission_id = t.id
		WHERE ad.id = ANY($1)
		ORDER BY array_position($1, ad.id)
	`

	rows, err := r.db.QueryContext(ctx, query, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	listings := make([]*shared.ListingDetails, 0, len(ids))
	for rows.Next() {
		var (
			id           int64
			vin          string
			makeName     string
			modelName    string
			yearValue    sql.NullInt64
			priceValue   sql.NullInt64
			mileage      sql.NullInt64
			trim         sql.NullString
			fuelType     sql.NullString
			transmission sql.NullString
			city         sql.NullString
			district     sql.NullString
			state        sql.NullString
			country      sql.NullString
			color        sql.NullString
			lastSeen     sql.NullTime
		)

		if err := rows.Scan(
			&id,
			&vin,
			&makeName,
			&modelName,
			&yearValue,
			&priceValue,
			&mileage,
			&trim,
			&fuelType,
			&transmission,
			&city,
			&district,
			&state,
			&country,
			&color,
			&lastSeen,
		); err != nil {
			return nil, err
		}

		listings = append(listings, buildListingDetails(
			id,
			vin,
			makeName,
			modelName,
			yearValue,
			priceValue,
			mileage,
			fuelType,
			trim,
			transmission,
			city,
			district,
			state,
			country,
			color,
			lastSeen,
		))
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return listings, nil
}

func buildListingDetails(
	id int64,
	vin string,
	makeName string,
	modelName string,
	yearValue sql.NullInt64,
	priceValue sql.NullInt64,
	mileage sql.NullInt64,
	fuelType sql.NullString,
	trim sql.NullString,
	transmission sql.NullString,
	city sql.NullString,
	district sql.NullString,
	state sql.NullString,
	country sql.NullString,
	color sql.NullString,
	lastSeen sql.NullTime,
) *shared.ListingDetails {
	lastSeenText := ""
	if lastSeen.Valid {
		lastSeenText = lastSeen.Time.UTC().Format(time.RFC3339)
	}
	listing := &shared.ListingDetails{
		Id:           id,
		Vin:          vin,
		Make:         makeName,
		Model:        modelName,
		Year:         int32(yearValue.Int64),
		Price:        float32(priceValue.Int64),
		Mileage:      int32(mileage.Int64),
		City:         city.String,
		District:     district.String,
		State:        state.String,
		Country:      country.String,
		FuelType:     fuelType.String,
		Trim:         trim.String,
		Transmission: transmission.String,
		Color:        color.String,
		SellerType:   "",
		Description:  "",
		Images:       []string{},
		LastSeen:     lastSeenText,
	}
	return listing
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

func (r *ListingRepository) GetListingSummary(ctx context.Context, id int64) (*shared.ListingSummary, error) {
	query := `
			SELECT
				ad.id,
				b.name,
				m.name,
				ad.model_year,
				ad.ask_price,
				ad.mileage,
				COALESCE(ft.name, ''),
				COALESCE(bc.name, ''),
				COALESCE(dt.name, ''),
				COALESCE(t.name, ''),
				COALESCE(ad.is_new, false),
				COALESCE(ad.city, ''),
				COALESCE(ad.district, ''),
				COALESCE(ad.state, ''),
				COALESCE(ad.country, ''),
				ad.last_seen
			FROM automotive_data ad
			JOIN brand b ON ad.brand_id = b.id
			JOIN model m ON ad.model_id = m.id
			LEFT JOIN fuel_type ft ON ad.fuel_type_id = ft.id
			LEFT JOIN body_class bc ON ad.body_class_id = bc.id
			LEFT JOIN drive_type dt ON ad.drive_type_id = dt.id
			LEFT JOIN transmission t ON ad.transmission_id = t.id
			WHERE ad.id = $1
		`

	var (
		idValue      int64
		makeName     string
		modelName    string
		yearValue    sql.NullInt64
		priceValue   sql.NullInt64
		mileage      sql.NullInt64
		fuelType     sql.NullString
		bodyClass    sql.NullString
		driveType    sql.NullString
		transmission sql.NullString
		isNew        bool
		city         sql.NullString
		district     sql.NullString
		state        sql.NullString
		country      sql.NullString
		lastSeen     sql.NullTime
	)

	if err := r.db.QueryRowContext(ctx, query, id).Scan(
		&idValue,
		&makeName,
		&modelName,
		&yearValue,
		&priceValue,
		&mileage,
		&fuelType,
		&bodyClass,
		&driveType,
		&transmission,
		&isNew,
		&city,
		&district,
		&state,
		&country,
		&lastSeen,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	lastSeenText := ""
	if lastSeen.Valid {
		lastSeenText = lastSeen.Time.UTC().Format(time.RFC3339)
	}
	return &shared.ListingSummary{
		Id:           idValue,
		Make:         makeName,
		Model:        modelName,
		Year:         int32(yearValue.Int64),
		Price:        float32(priceValue.Int64),
		Mileage:      int32(mileage.Int64),
		FuelType:     fuelType.String,
		BodyClass:    bodyClass.String,
		DriveType:    driveType.String,
		Transmission: transmission.String,
		IsNew:        isNew,
		City:         city.String,
		District:     district.String,
		State:        state.String,
		Country:      country.String,
		LastSeen:     lastSeenText,
	}, nil
}

func (r *ListingRepository) CheckListingOpen(ctx context.Context, listingID int64) (bool, error) {
	query := `SELECT COUNT(1) FROM automotive_data WHERE id = $1`
	var count int
	err := r.db.QueryRowContext(ctx, query, listingID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
