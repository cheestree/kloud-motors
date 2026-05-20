package repository

import (
	"context"
	"testing"
	"time"

	"services/search/domain"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestSearchRepository_Search_WithFilters(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewSearchRepository(db)
	filters := domain.SearchParams{
		Make:        "Ford",
		Model:       "Fiesta",
		MinPrice:    1000,
		MaxPrice:    20000,
		Page:        1,
		PageSize:    20,
		IncludeSold: false,
	}

	countQuery := "SELECT COUNT\\(\\*\\) FROM automotive_data ad.*WHERE COALESCE\\(ad.is_sold, false\\) = false AND b.name ILIKE \\$1 AND m.name ILIKE \\$2 AND ad.ask_price >= \\$3 AND ad.ask_price <= \\$4"
	mock.ExpectQuery(countQuery).
		WithArgs("%Ford%", "%Fiesta%", int64(1000), int64(20000)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	selectQuery := "SELECT ad.id,.*FROM automotive_data ad.*WHERE COALESCE\\(ad.is_sold, false\\) = false AND b.name ILIKE \\$1 AND m.name ILIKE \\$2 AND ad.ask_price >= \\$3 AND ad.ask_price <= \\$4 ORDER BY ad.last_seen DESC NULLS LAST, ad.id ASC LIMIT \\$5 OFFSET \\$6"
	rows := sqlmock.NewRows([]string{
		"id",
		"dealer_id",
		"make",
		"model",
		"year",
		"price",
		"mileage",
		"fuel_type",
		"body_class",
		"drive_type",
		"transmission",
		"is_new",
		"is_sold",
		"city",
		"district",
		"state",
		"country",
		"last_seen",
	}).AddRow(
		1,
		42,
		"Ford",
		"Fiesta",
		2018,
		12000,
		30000,
		"Gas",
		"Sedan",
		"FWD",
		"Auto",
		true,
		false,
		"Porto",
		"Porto",
		"Porto",
		"PT",
		time.Now(),
	)
	mock.ExpectQuery(selectQuery).
		WithArgs("%Ford%", "%Fiesta%", int64(1000), int64(20000), int32(20), int32(0)).
		WillReturnRows(rows)

	listings, total, err := repo.Search(context.Background(), filters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total 1, got %d", total)
	}
	if len(listings) != 1 {
		t.Fatalf("expected 1 listing, got %d", len(listings))
	}
	if listings[0].Make != "Ford" {
		t.Fatalf("expected listing make Ford, got %q", listings[0].Make)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}
