package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"services/listing/repository"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestListingService_GetListingSummaryRejectsInvalidID(t *testing.T) {
	svc := NewListingService(repository.NewListingRepository(nil), nil)

	_, err := svc.GetListingSummary(context.Background(), 0)
	if err == nil {
		t.Fatalf("expected invalid id error")
	}
}

func TestListingService_GetListingSummaryMapsMissingListing(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT\\s+ad.id,\\s+ad.dealer_id,.*WHERE ad.id = \\$1").
		WithArgs(int64(10)).
		WillReturnError(sql.ErrNoRows)

	svc := NewListingService(repository.NewListingRepository(db), nil)
	_, err = svc.GetListingSummary(context.Background(), 10)
	if !errors.Is(err, ErrListingNotFound) {
		t.Fatalf("expected ErrListingNotFound, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

func TestListingService_CreateListingValidatesMutation(t *testing.T) {
	svc := NewListingService(repository.NewListingRepository(nil), nil)

	_, err := svc.CreateListing(context.Background(), repository.ListingMutation{
		Make:     "Ford",
		Model:    "Fiesta",
		Year:     2018,
		DealerID: 42,
	})
	if err == nil {
		t.Fatalf("expected validation error")
	}
}
