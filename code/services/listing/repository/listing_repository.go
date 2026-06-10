package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"services/shared"
	"services/utils"

	"github.com/lib/pq"
)

type ListingMutation struct {
	Vin          string
	Make         string
	Model        string
	Year         int32
	Price        float64
	Mileage      int32
	City         string
	District     string
	State        string
	Country      string
	FuelType     string
	BodyClass    string
	DriveType    string
	Transmission string
	Trim         string
	Color        string
	DealerID     int64
	IsNew        bool
	IsSold       bool
}

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
				COALESCE(bc.name, ''),
				COALESCE(dt.name, ''),
				COALESCE(t.name, ''),
				COALESCE(ad.is_new, false),
				COALESCE(ad.is_sold, false),
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
			LEFT JOIN body_class bc ON ad.body_class_id = bc.id
			LEFT JOIN drive_type dt ON ad.drive_type_id = dt.id
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
		bodyClass    sql.NullString
		driveType    sql.NullString
		transmission sql.NullString
		isNew        bool
		isSold       bool
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
		&bodyClass,
		&driveType,
		&transmission,
		&isNew,
		&isSold,
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
		bodyClass,
		driveType,
		trim,
		transmission,
		isNew,
		isSold,
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
			COALESCE(bc.name, ''),
			COALESCE(dt.name, ''),
			COALESCE(t.name, ''),
			COALESCE(ad.is_new, false),
			COALESCE(ad.is_sold, false),
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
		LEFT JOIN body_class bc ON ad.body_class_id = bc.id
		LEFT JOIN drive_type dt ON ad.drive_type_id = dt.id
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
			bodyClass    sql.NullString
			driveType    sql.NullString
			transmission sql.NullString
			isNew        bool
			isSold       bool
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
			&bodyClass,
			&driveType,
			&transmission,
			&isNew,
			&isSold,
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
			bodyClass,
			driveType,
			trim,
			transmission,
			isNew,
			isSold,
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
	bodyClass sql.NullString,
	driveType sql.NullString,
	trim sql.NullString,
	transmission sql.NullString,
	isNew bool,
	isSold bool,
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
		BodyClass:    bodyClass.String,
		DriveType:    driveType.String,
		IsNew:        isNew,
		IsSold:       isSold,
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

func (r *ListingRepository) CreateListing(ctx context.Context, listing ListingMutation) (*shared.ListingDetails, error) {
	brandID, modelID, err := r.resolveBrandModelIDs(ctx, listing.Make, listing.Model)
	if err != nil {
		return nil, err
	}

	fuelTypeID, err := r.lookupOptionalID(ctx, "fuel_type", listing.FuelType)
	if err != nil {
		return nil, err
	}
	bodyClassID, err := r.lookupOptionalID(ctx, "body_class", listing.BodyClass)
	if err != nil {
		return nil, err
	}
	driveTypeID, err := r.lookupOptionalID(ctx, "drive_type", listing.DriveType)
	if err != nil {
		return nil, err
	}
	transmissionID, err := r.lookupOptionalID(ctx, "transmission", listing.Transmission)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	query := `
		INSERT INTO automotive_data (
			vin,
			ask_price,
			mileage,
			model_year,
			trim,
			city,
			district,
			state,
			country,
			color,
			dealer_id,
			is_new,
			is_sold,
			brand_id,
			model_id,
			fuel_type_id,
			transmission_id,
			body_class_id,
			drive_type_id,
			first_seen,
			last_seen
		)
		VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20, $21
		)
		RETURNING id
	`

	var id int64
	err = r.db.QueryRowContext(ctx, query,
		normalizeVIN(listing.Vin),
		utils.SQLNullablePositiveInt64(int64(listing.Price)),
		utils.SQLNullablePositiveInt32(listing.Mileage),
		utils.SQLNullablePositiveInt32(listing.Year),
		utils.SQLNullableNonEmptyString(listing.Trim),
		utils.SQLNullableNonEmptyString(listing.City),
		utils.SQLNullableNonEmptyString(listing.District),
		utils.SQLNullableNonEmptyString(listing.State),
		utils.SQLNullableNonEmptyString(listing.Country),
		utils.SQLNullableNonEmptyString(listing.Color),
		listing.DealerID,
		listing.IsNew,
		listing.IsSold,
		brandID,
		modelID,
		utils.SQLNullableInt64FromPtr(fuelTypeID),
		utils.SQLNullableInt64FromPtr(transmissionID),
		utils.SQLNullableInt64FromPtr(bodyClassID),
		utils.SQLNullableInt64FromPtr(driveTypeID),
		now,
		now,
	).Scan(&id)
	if err != nil {
		return nil, err
	}

	return r.GetListingDetails(ctx, id)
}

func (r *ListingRepository) UpdateListing(ctx context.Context, id int64, listing ListingMutation) (*shared.ListingDetails, error) {
	brandID, modelID, err := r.resolveBrandModelIDs(ctx, listing.Make, listing.Model)
	if err != nil {
		return nil, err
	}

	fuelTypeID, err := r.lookupOptionalID(ctx, "fuel_type", listing.FuelType)
	if err != nil {
		return nil, err
	}
	bodyClassID, err := r.lookupOptionalID(ctx, "body_class", listing.BodyClass)
	if err != nil {
		return nil, err
	}
	driveTypeID, err := r.lookupOptionalID(ctx, "drive_type", listing.DriveType)
	if err != nil {
		return nil, err
	}
	transmissionID, err := r.lookupOptionalID(ctx, "transmission", listing.Transmission)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	query := `
		UPDATE automotive_data
		SET
			vin = $3,
			ask_price = $4,
			mileage = $5,
			model_year = $6,
			trim = $7,
			city = $8,
			district = $9,
			state = $10,
			country = $11,
			color = $12,
			is_new = $13,
			brand_id = $14,
			model_id = $15,
			fuel_type_id = $16,
			transmission_id = $17,
			body_class_id = $18,
			drive_type_id = $19,
			last_seen = $20
		WHERE id = $1 AND dealer_id = $2
		RETURNING id
	`

	var updatedID int64
	err = r.db.QueryRowContext(ctx, query,
		id,
		listing.DealerID,
		normalizeVIN(listing.Vin),
		utils.SQLNullablePositiveInt64(int64(listing.Price)),
		utils.SQLNullablePositiveInt32(listing.Mileage),
		utils.SQLNullablePositiveInt32(listing.Year),
		utils.SQLNullableNonEmptyString(listing.Trim),
		utils.SQLNullableNonEmptyString(listing.City),
		utils.SQLNullableNonEmptyString(listing.District),
		utils.SQLNullableNonEmptyString(listing.State),
		utils.SQLNullableNonEmptyString(listing.Country),
		utils.SQLNullableNonEmptyString(listing.Color),
		listing.IsNew,
		brandID,
		modelID,
		utils.SQLNullableInt64FromPtr(fuelTypeID),
		utils.SQLNullableInt64FromPtr(transmissionID),
		utils.SQLNullableInt64FromPtr(bodyClassID),
		utils.SQLNullableInt64FromPtr(driveTypeID),
		now,
	).Scan(&updatedID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return r.GetListingDetails(ctx, updatedID)
}

func (r *ListingRepository) SetListingSoldStatus(ctx context.Context, id int64, dealerID int64, isSold bool) (*shared.ListingDetails, error) {
	query := `UPDATE automotive_data SET is_sold = $3 WHERE id = $1 AND dealer_id = $2 RETURNING id`
	var updatedID int64
	err := r.db.QueryRowContext(ctx, query, id, dealerID, isSold).Scan(&updatedID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.GetListingDetails(ctx, updatedID)
}

func (r *ListingRepository) DeleteListing(ctx context.Context, id int64, dealerID int64) (bool, error) {
	result, err := r.db.ExecContext(ctx, `DELETE FROM automotive_data WHERE id = $1 AND dealer_id = $2`, id, dealerID)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rowsAffected > 0, nil
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
				ad.dealer_id,
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
				COALESCE(ad.is_sold, false),
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
		dealerID     int64
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
		isSold       bool
		city         sql.NullString
		district     sql.NullString
		state        sql.NullString
		country      sql.NullString
		lastSeen     sql.NullTime
	)

	if err := r.db.QueryRowContext(ctx, query, id).Scan(
		&idValue,
		&dealerID,
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
		&isSold,
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
		SellerId:     dealerID,
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
		IsSold:       isSold,
		City:         city.String,
		District:     district.String,
		State:        state.String,
		Country:      country.String,
		LastSeen:     lastSeenText,
	}, nil
}

func (r *ListingRepository) GetListingSummaries(ctx context.Context, ids []int64) ([]*shared.ListingSummary, error) {
	if len(ids) == 0 {
		return []*shared.ListingSummary{}, nil
	}

	query := `
		SELECT
			ad.id,
			ad.dealer_id,
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
			COALESCE(ad.is_sold, false),
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
		WHERE ad.id = ANY($1)
		ORDER BY array_position($1, ad.id)
	`

	rows, err := r.db.QueryContext(ctx, query, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	listings := make([]*shared.ListingSummary, 0, len(ids))
	for rows.Next() {
		var (
			idValue      int64
			dealerID     int64
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
			isSold       bool
			city         sql.NullString
			district     sql.NullString
			state        sql.NullString
			country      sql.NullString
			lastSeen     sql.NullTime
		)

		if err := rows.Scan(
			&idValue,
			&dealerID,
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
			&isSold,
			&city,
			&district,
			&state,
			&country,
			&lastSeen,
		); err != nil {
			return nil, err
		}

		lastSeenText := ""
		if lastSeen.Valid {
			lastSeenText = lastSeen.Time.UTC().Format(time.RFC3339)
		}

		listings = append(listings, &shared.ListingSummary{
			Id:           idValue,
			SellerId:     dealerID,
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
			IsSold:       isSold,
			City:         city.String,
			District:     district.String,
			State:        state.String,
			Country:      country.String,
			LastSeen:     lastSeenText,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return listings, nil
}

func (r *ListingRepository) CheckListingOpen(ctx context.Context, listingID int64) (bool, int64, error) {
	query := `SELECT dealer_id, is_sold FROM automotive_data WHERE id = $1 LIMIT 1`
	var dealerID int64
	var isSold bool
	err := r.db.QueryRowContext(ctx, query, listingID).Scan(&dealerID, &isSold)
	if err != nil {
		return false, 0, err
	}
	return !isSold, dealerID, nil
}

func (r *ListingRepository) resolveBrandModelIDs(ctx context.Context, makeName string, modelName string) (int64, int64, error) {
	brandID, err := r.lookupRequiredID(ctx, "brand", makeName)
	if err != nil {
		return 0, 0, err
	}

	modelID, err := r.ensureModelID(ctx, brandID, modelName)
	if err != nil {
		return 0, 0, err
	}

	return brandID, modelID, nil
}

func (r *ListingRepository) ensureModelID(ctx context.Context, brandID int64, modelName string) (int64, error) {
	trimmed := strings.TrimSpace(modelName)
	if trimmed == "" {
		return 0, fmt.Errorf("model name is required")
	}

	modelQuery := `SELECT id FROM model WHERE brand_id = $1 AND LOWER(name) = LOWER($2) LIMIT 1`
	var modelID int64
	if err := r.db.QueryRowContext(ctx, modelQuery, brandID, trimmed).Scan(&modelID); err == nil {
		return modelID, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}

	insertQuery := `
		INSERT INTO model (brand_id, name)
		VALUES ($1, $2)
		ON CONFLICT (brand_id, name) DO NOTHING
		RETURNING id
	`
	if err := r.db.QueryRowContext(ctx, insertQuery, brandID, trimmed).Scan(&modelID); err == nil {
		return modelID, nil
	}

	if err := r.db.QueryRowContext(ctx, modelQuery, brandID, trimmed).Scan(&modelID); err != nil {
		return 0, err
	}
	return modelID, nil
}

func (r *ListingRepository) lookupRequiredID(ctx context.Context, tableName string, value string) (int64, error) {
	query, err := queryForLookupTable(tableName)
	if err != nil {
		return 0, err
	}

	var id int64
	if err := r.db.QueryRowContext(ctx, query, strings.TrimSpace(value)).Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("unknown %s: %s", tableName, value)
		}
		return 0, err
	}
	return id, nil
}

func (r *ListingRepository) lookupOptionalID(ctx context.Context, tableName string, value string) (*int64, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}
	id, err := r.lookupRequiredID(ctx, tableName, trimmed)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func queryForLookupTable(tableName string) (string, error) {
	switch tableName {
	case "brand":
		return `SELECT id FROM brand WHERE LOWER(name) = LOWER($1) LIMIT 1`, nil
	case "fuel_type":
		return `SELECT id FROM fuel_type WHERE LOWER(name) = LOWER($1) LIMIT 1`, nil
	case "transmission":
		return `SELECT id FROM transmission WHERE LOWER(name) = LOWER($1) LIMIT 1`, nil
	case "body_class":
		return `SELECT id FROM body_class WHERE LOWER(name) = LOWER($1) LIMIT 1`, nil
	case "drive_type":
		return `SELECT id FROM drive_type WHERE LOWER(name) = LOWER($1) LIMIT 1`, nil
	default:
		return "", fmt.Errorf("unsupported lookup table: %s", tableName)
	}
}

func normalizeVIN(vin string) string {
	return strings.ToUpper(strings.TrimSpace(vin))
}
