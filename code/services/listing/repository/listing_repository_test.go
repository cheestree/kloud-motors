package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestListingRepository_GetListingSummary(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	lastSeen := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	rows := sqlmock.NewRows([]string{
		"id", "dealer_id", "make", "model", "year", "price", "mileage",
		"fuel_type", "body_class", "drive_type", "transmission", "is_new",
		"is_sold", "city", "district", "state", "country", "last_seen",
	}).AddRow(
		int64(7),
		int64(42),
		"Ford",
		"Fiesta",
		sql.NullInt64{Int64: 2018, Valid: true},
		sql.NullInt64{Int64: 12000, Valid: true},
		sql.NullInt64{Int64: 30000, Valid: true},
		sql.NullString{String: "Gasoline", Valid: true},
		sql.NullString{String: "Sedan", Valid: true},
		sql.NullString{String: "FWD", Valid: true},
		sql.NullString{String: "Auto", Valid: true},
		true,
		false,
		sql.NullString{String: "Porto", Valid: true},
		sql.NullString{String: "Porto", Valid: true},
		sql.NullString{String: "Porto", Valid: true},
		sql.NullString{String: "PT", Valid: true},
		sql.NullTime{Time: lastSeen, Valid: true},
	)
	mock.ExpectQuery("SELECT\\s+ad.id,\\s+ad.dealer_id,.*WHERE ad.id = \\$1").
		WithArgs(int64(7)).
		WillReturnRows(rows)

	repo := NewListingRepository(db)
	summary, err := repo.GetListingSummary(context.Background(), 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary == nil {
		t.Fatalf("expected summary")
	}
	if summary.Id != 7 || summary.SellerId != 42 || summary.Make != "Ford" || summary.Model != "Fiesta" {
		t.Fatalf("unexpected summary identifiers: %+v", summary)
	}
	if summary.LastSeen != "2025-01-02T03:04:05Z" {
		t.Fatalf("unexpected last seen %q", summary.LastSeen)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

func TestListingRepository_CheckListingOpen(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT dealer_id, is_sold FROM automotive_data WHERE id = \\$1 LIMIT 1").
		WithArgs(int64(5)).
		WillReturnRows(sqlmock.NewRows([]string{"dealer_id", "is_sold"}).AddRow(int64(99), false))

	repo := NewListingRepository(db)
	open, dealerID, err := repo.CheckListingOpen(context.Background(), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !open || dealerID != 99 {
		t.Fatalf("expected open listing for dealer 99, got open=%v dealerID=%d", open, dealerID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}
