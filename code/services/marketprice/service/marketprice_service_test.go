package service

import (
	"context"
	"testing"

	marketpricepb "services/marketprice/proto"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestMarketPriceService_GetAverageMarketPrice(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	query := "SELECT\\s+COALESCE\\(AVG\\(ad.ask_price\\), 0\\),\\s+COALESCE\\(MIN\\(ad.ask_price\\), 0\\),\\s+COALESCE\\(MAX\\(ad.ask_price\\), 0\\),\\s+COUNT\\(ad.ask_price\\).*WHERE 1=1 AND b.name = \\$1 AND m.name = \\$2"
	mock.ExpectQuery(query).
		WithArgs("FORD", "Fiesta").
		WillReturnRows(sqlmock.NewRows([]string{"avg", "min", "max", "count"}).
			AddRow(12000.0, 9000.0, 15000.0, int32(3)))

	svc := NewService(db)
	resp, err := svc.GetAverageMarketPrice(context.Background(), &marketpricepb.AveragePriceRequest{
		Brand: "Ford",
		Model: "Fiesta",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Brand != "Ford" || resp.Model != "Fiesta" {
		t.Fatalf("unexpected echo fields: brand=%q model=%q", resp.Brand, resp.Model)
	}
	if resp.AveragePrice != 12000 || resp.MinPrice != 9000 || resp.MaxPrice != 15000 || resp.ListingCount != 3 {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}
