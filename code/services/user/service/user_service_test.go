package service

import (
	"context"
	"testing"

	userpb "services/user/proto"
	"services/user/repository"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func newMockUserService(t *testing.T) (*UserService, sqlmock.Sqlmock, func()) {
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

	return NewUserService(repository.NewRepository(gormDB)), mock, func() {
		db.Close()
	}
}

func TestUserService_GetFavoritesMapsListingIDs(t *testing.T) {
	svc, mock, cleanup := newMockUserService(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT \* FROM "favorites" WHERE user_id = \$1`).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "listing_id"}).
			AddRow(uint(1), int64(7), int64(100)).
			AddRow(uint(2), int64(7), int64(200)))

	resp, err := svc.GetFavorites(context.Background(), &userpb.GetFavoritesRequest{UserId: 7})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Favorites) != 2 || resp.Favorites[0] != 100 || resp.Favorites[1] != 200 {
		t.Fatalf("unexpected favorites: %v", resp.Favorites)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

func TestUserService_GetUsersPreview(t *testing.T) {
	svc, mock, cleanup := newMockUserService(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT \* FROM "users" WHERE id IN \(\$1,\$2\)`).
		WithArgs(int64(1), int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "firebase_uid", "name", "email"}).
			AddRow(int64(1), "uid-1", "Alice", "alice@example.test").
			AddRow(int64(2), "uid-2", "Bob", "bob@example.test"))

	resp, err := svc.GetUsersPreview(context.Background(), &userpb.UsersPreviewRequest{UserIds: []int64{1, 2}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Users) != 2 || resp.Users[0].Name != "Alice" || resp.Users[1].Name != "Bob" {
		t.Fatalf("unexpected users preview: %+v", resp.Users)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}
