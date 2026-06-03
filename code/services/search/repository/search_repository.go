package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"services/search/domain"
	"services/shared"
)

type SearchRepository struct {
	db *sql.DB
}

type Searcher interface {
	Search(ctx context.Context, filters domain.SearchParams) ([]shared.ListingSummary, int32, error)
}

func NewSearchRepository(db *sql.DB) *SearchRepository {
	return &SearchRepository{db: db}
}

func (r *SearchRepository) Search(ctx context.Context, filters domain.SearchParams) ([]shared.ListingSummary, int32, error) {
	clauses := make([]string, 0)
	args := make([]interface{}, 0)

	// Search returns only active listings by default.
	if !filters.IncludeSold {
		clauses = append(clauses, "COALESCE(ad.is_sold, false) = false")
	}

	if filters.Make != "" {
		args = append(args, "%"+filters.Make+"%")
		clauses = append(clauses, fmt.Sprintf("b.name ILIKE $%d", len(args)))
	}
	if filters.Model != "" {
		args = append(args, "%"+filters.Model+"%")
		clauses = append(clauses, fmt.Sprintf("m.name ILIKE $%d", len(args)))
	}
	if filters.Year > 0 {
		args = append(args, filters.Year)
		clauses = append(clauses, fmt.Sprintf("ad.model_year = $%d", len(args)))
	}
	if filters.MinPrice > 0 {
		args = append(args, filters.MinPrice)
		clauses = append(clauses, fmt.Sprintf("ad.ask_price >= $%d", len(args)))
	}
	if filters.MaxPrice > 0 {
		args = append(args, filters.MaxPrice)
		clauses = append(clauses, fmt.Sprintf("ad.ask_price <= $%d", len(args)))
	}
	if filters.MaxMileage > 0 {
		args = append(args, filters.MaxMileage)
		clauses = append(clauses, fmt.Sprintf("ad.mileage <= $%d", len(args)))
	}
	if filters.FuelType != "" {
		args = append(args, "%"+filters.FuelType+"%")
		clauses = append(clauses, fmt.Sprintf("ft.name ILIKE $%d", len(args)))
	}
	if filters.BodyClass != "" {
		args = append(args, "%"+filters.BodyClass+"%")
		clauses = append(clauses, fmt.Sprintf("bc.name ILIKE $%d", len(args)))
	}
	if filters.DriveType != "" {
		args = append(args, "%"+filters.DriveType+"%")
		clauses = append(clauses, fmt.Sprintf("dt.name ILIKE $%d", len(args)))
	}
	if filters.Transmission != "" {
		args = append(args, "%"+filters.Transmission+"%")
		clauses = append(clauses, fmt.Sprintf("tr.name ILIKE $%d", len(args)))
	}
	if filters.IsNew != nil {
		args = append(args, *filters.IsNew)
		clauses = append(clauses, fmt.Sprintf("ad.is_new = $%d", len(args)))
	}

	if filters.State != "" {
		args = append(args, "%"+filters.State+"%")
		clauses = append(clauses, fmt.Sprintf("ad.state ILIKE $%d", len(args)))
	}
	if filters.District != "" {
		args = append(args, "%"+filters.District+"%")
		clauses = append(clauses, fmt.Sprintf("ad.district ILIKE $%d", len(args)))
	}
	if filters.City != "" {
		args = append(args, "%"+filters.City+"%")
		clauses = append(clauses, fmt.Sprintf("ad.city ILIKE $%d", len(args)))
	}
	if filters.Country != "" {
		args = append(args, "%"+filters.Country+"%")
		clauses = append(clauses, fmt.Sprintf("ad.country ILIKE $%d", len(args)))
	}

	whereSQL := ""
	if len(clauses) > 0 {
		whereSQL = " WHERE " + strings.Join(clauses, " AND ")
	}

	baseSQL := " FROM automotive_data ad" +
		" LEFT JOIN brand b ON ad.brand_id = b.id" +
		" LEFT JOIN model m ON ad.model_id = m.id" +
		" LEFT JOIN fuel_type ft ON ad.fuel_type_id = ft.id" +
		" LEFT JOIN body_class bc ON ad.body_class_id = bc.id" +
		" LEFT JOIN drive_type dt ON ad.drive_type_id = dt.id" +
		" LEFT JOIN transmission tr ON ad.transmission_id = tr.id"

	var total int32
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*)"+baseSQL+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, filters.PageSize, (filters.Page-1)*filters.PageSize)
	limitIdx, offsetIdx := len(args)-1, len(args)

	selectQuery := "SELECT ad.id," +
		" COALESCE(ad.dealer_id, 0)," +
		" COALESCE(b.name, '')," +
		" COALESCE(m.name, '')," +
		" COALESCE(ad.model_year, 0)," +
		" COALESCE(ad.ask_price, 0)," +
		" COALESCE(ad.mileage, 0)," +
		" COALESCE(ft.name, '')," +
		" COALESCE(bc.name, '')," +
		" COALESCE(dt.name, '')," +
		" COALESCE(tr.name, '')," +
		" COALESCE(ad.is_new, false)," +
		" COALESCE(ad.is_sold, false)," +
		" COALESCE(ad.city, '')," +
		" COALESCE(ad.district, '')," +
		" COALESCE(ad.state, '')," +
		" COALESCE(ad.country, '')," +
		" ad.last_seen" +
		baseSQL + whereSQL +
		fmt.Sprintf(" ORDER BY ad.last_seen DESC NULLS LAST, ad.id ASC LIMIT $%d OFFSET $%d", limitIdx, offsetIdx)

	rows, err := r.db.QueryContext(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	listings := make([]shared.ListingSummary, 0)
	for rows.Next() {
		var s shared.ListingSummary
		var dealerID int64
		if err := rows.Scan(
			&s.Id, &dealerID, &s.Make, &s.Model, &s.Year,
			&s.Price, &s.Mileage,
			&s.FuelType, &s.BodyClass, &s.DriveType, &s.Transmission,
			&s.IsNew, &s.IsSold, &s.City, &s.District, &s.State, &s.Country, &s.LastSeen,
		); err != nil {
			return nil, 0, err
		}
		s.SellerId = int32(dealerID)
		listings = append(listings, s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return listings, total, nil
}
