package service

import (
	"context"
	"testing"

	sellerpb "services/seller/proto"
	"services/seller/repository"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func newMockSellerService(t *testing.T) (*Service, sqlmock.Sqlmock, sqlmock.Sqlmock, func()) {
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

	repo := repository.NewRepository(listingDB, sellerDB)
	return NewService(repo), listingMock, sellerMock, func() {
		listingSQL.Close()
		sellerSQL.Close()
	}
}

func TestSellerService_VerifySellerProfileFalseWhenMissing(t *testing.T) {
	svc, listingMock, sellerMock, cleanup := newMockSellerService(t)
	defer cleanup()

	sellerMock.ExpectQuery(`SELECT \* FROM "sellers" WHERE id = \$1 ORDER BY "sellers"\."id" LIMIT \$2`).
		WithArgs(int64(7), 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "seller_type", "contact_info", "rating"}))

	resp, err := svc.VerifySellerProfile(context.Background(), &sellerpb.VerifySellerRequest{SellerId: 7})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.IsSeller {
		t.Fatalf("expected missing seller to verify false")
	}
	if err := listingMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet listing sqlmock expectations: %v", err)
	}
	if err := sellerMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet seller sqlmock expectations: %v", err)
	}
}

func TestSellerService_CreateSellerAlreadyExists(t *testing.T) {
	svc, listingMock, sellerMock, cleanup := newMockSellerService(t)
	defer cleanup()

	sellerMock.ExpectQuery(`SELECT \* FROM "sellers" WHERE id = \$1 ORDER BY "sellers"\."id" LIMIT \$2`).
		WithArgs(int64(7), 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "seller_type", "contact_info", "rating"}).
			AddRow(int64(7), "Alice", "private_seller", "alice@example.test", 5.0))

	_, err := svc.CreateSeller(context.Background(), &sellerpb.CreateSellerRequest{
		SellerId:    7,
		Name:        "Alice",
		SellerType:  "private_seller",
		ContactInfo: "alice@example.test",
	})
	if status.Code(err) != codes.AlreadyExists {
		t.Fatalf("expected AlreadyExists, got %v", err)
	}
	if err := listingMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet listing sqlmock expectations: %v", err)
	}
	if err := sellerMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet seller sqlmock expectations: %v", err)
	}
}
