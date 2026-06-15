package repository

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func newMockUserRepository(t *testing.T) (*Repository, sqlmock.Sqlmock, func()) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
	if err != nil {
		db.Close()
		t.Fatalf("failed to open gorm db: %v", err)
	}

	return NewRepository(gormDB), mock, func() {
		db.Close()
	}
}

func TestUserRepository_GetFavorites(t *testing.T) {
	repo, mock, cleanup := newMockUserRepository(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT \* FROM "favorites" WHERE user_id = \$1`).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "listing_id"}).
			AddRow(uint(1), int64(7), int64(100)).
			AddRow(uint(2), int64(7), int64(200)))

	favorites, err := repo.GetFavorites(context.Background(), 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(favorites) != 2 || favorites[0].ListingID != 100 || favorites[1].ListingID != 200 {
		t.Fatalf("unexpected favorites: %+v", favorites)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

func TestUserRepository_RemoveFavorite(t *testing.T) {
	repo, mock, cleanup := newMockUserRepository(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM "favorites" WHERE user_id = \$1 AND listing_id = \$2`).
		WithArgs(int64(7), int64(100)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	rows, err := repo.RemoveFavorite(context.Background(), 7, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rows != 1 {
		t.Fatalf("expected one row removed, got %d", rows)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}
