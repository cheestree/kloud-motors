package service

import (
	"context"
	"database/sql"
	"fmt"

	marketpricepb "services/marketprice/proto"
	"services/marketprice/repository"
)

type Service struct {
	Repo *repository.Repository
}

func NewService(db *sql.DB) *Service {
	return &Service{Repo: repository.NewRepository(db)}
}

func (s *Service) GetAverageMarketPrice(ctx context.Context, req *marketpricepb.AveragePriceRequest) (*marketpricepb.AveragePriceResponse, error) {
	avgPrice, minPrice, maxPrice, count, err := s.Repo.GetAverageMarketPrice(ctx,
		req.Brand, req.Model, req.YearFrom, req.YearTo)
	if err != nil {
		return nil, fmt.Errorf("failed to get average market price: %w", err)
	}

	return &marketpricepb.AveragePriceResponse{
		Brand:        req.Brand,
		Model:        req.Model,
		AveragePrice: avgPrice,
		MinPrice:     minPrice,
		MaxPrice:     maxPrice,
		ListingCount: count,
	}, nil
}
