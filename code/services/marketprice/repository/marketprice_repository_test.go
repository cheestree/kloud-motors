package repository

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestMarketPriceRepository_GetAverageMarketPrice_WithFilters(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)

	query := "SELECT\\s+COALESCE\\(AVG\\(ad.ask_price\\), 0\\),\\s+COALESCE\\(MIN\\(ad.ask_price\\), 0\\),\\s+COALESCE\\(MAX\\(ad.ask_price\\), 0\\),\\s+COUNT\\(ad.ask_price\\).*WHERE 1=1 AND b.name = \\$1 AND m.name = \\$2 AND ad.model_year >= \\$3 AND ad.model_year <= \\$4"
	mock.ExpectQuery(query).
		WithArgs("FORD", "Fiesta", int32(2010), int32(2020)).
		WillReturnRows(sqlmock.NewRows([]string{"avg", "min", "max", "count"}).
			AddRow(12000.0, 9000.0, 15000.0, int32(3)))

	avg, min, max, count, err := repo.GetAverageMarketPrice(context.Background(), "Ford", "Fiesta", 2010, 2020)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if avg != 12000 || min != 9000 || max != 15000 || count != 3 {
		t.Fatalf("unexpected aggregate values: avg=%f min=%f max=%f count=%d", avg, min, max, count)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}
