package repository

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func newMockSellerRepository(t *testing.T) (*Repository, sqlmock.Sqlmock, sqlmock.Sqlmock, func()) {
	t.Helper()

	listingSQL, listingMock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create listing sqlmock: %v", err)
	}
	sellerSQL, sellerMock, err := sqlmock.New()
	if err != nil {
		listingSQL.Close()
		t.Fatalf("failed to create seller sqlmock: %v", err)
	}

	listingDB, err := gorm.Open(postgres.New(postgres.Config{Conn: listingSQL}), &gorm.Config{})
	if err != nil {
		listingSQL.Close()
		sellerSQL.Close()
		t.Fatalf("failed to open listing gorm db: %v", err)
	}
	sellerDB, err := gorm.Open(postgres.New(postgres.Config{Conn: sellerSQL}), &gorm.Config{})
	if err != nil {
		listingSQL.Close()
		sellerSQL.Close()
		t.Fatalf("failed to open seller gorm db: %v", err)
	}

	return NewRepository(listingDB, sellerDB), listingMock, sellerMock, func() {
		listingSQL.Close()
		sellerSQL.Close()
	}
}

func TestSellerRepository_GetSellersPreview(t *testing.T) {
	repo, listingMock, sellerMock, cleanup := newMockSellerRepository(t)
	defer cleanup()

	sellerMock.ExpectQuery(`SELECT \* FROM "sellers" WHERE id IN \(\$1,\$2\)`).
		WithArgs(int64(1), int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "seller_type", "contact_info", "rating"}).
			AddRow(int64(1), "Alice", "private_seller", "alice@example.test", 5.0).
			AddRow(int64(2), "Bob", "professional_dealer", "bob@example.test", 4.5))

	previews, err := GetSellersPreview(context.Background(), repo.SellerDB, []int64{1, 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(previews) != 2 || previews[0].Name != "Alice" || previews[1].Name != "Bob" {
		t.Fatalf("unexpected previews: %+v", previews)
	}
	if err := listingMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet listing sqlmock expectations: %v", err)
	}
	if err := sellerMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet seller sqlmock expectations: %v", err)
	}
}
